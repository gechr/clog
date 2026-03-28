package fx

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/level"
)

// FieldAlignment controls optional group-level alignment behavior.
type FieldAlignment int

const (
	// FieldAlignmentNone disables group-level field alignment padding.
	FieldAlignmentNone FieldAlignment = iota
	// FieldAlignmentMessage aligns the first field column after the message.
	FieldAlignmentMessage
)

// Group manages a set of concurrent animations rendered as a multi-line
// block. Create one with root clog's Group function, add animations with
// [Group.Add], then call [Group.Wait] to run the render loop.
type Group struct {
	Ctx context.Context //nolint:containedctx // Group shares a single ctx with all child goroutines
	Mu  sync.Mutex

	FieldAlignment FieldAlignment
	Footer         *GroupStatus
	Header         *GroupStatus
	HideDone       bool
	Log            Logger
	Monotonic      bool
	Parallelism    int
	SyncAnimations bool
	Tasks          []*GroupTask

	sem chan struct{}
}

// GroupStatusFunc is called each render tick with the number of completed
// and total tasks. Use the [Update] to set the message and fields for
// that tick.
type GroupStatusFunc func(done, total int, u *Update)

// GroupStatus pairs a [Builder] (for initial config like level, symbol,
// parts) with a [GroupStatusFunc] callback that updates it each tick.
type GroupStatus struct {
	Builder  *Builder
	Callback GroupStatusFunc
}

// GroupTask holds per-animation state for the group render loop.
// This is exported so the root clog rendering code can access it.
type GroupTask struct {
	Builder        *Builder
	DoneErr        chan error // buffered(1); goroutine sends result here
	Err            error      // populated by Wait() after DoneErr is drained
	FieldsPtr      *atomic.Pointer[[]core.Field]
	FinishedAt     atomic.Int64
	LevelPtr       *atomic.Int64
	MsgPtr         *atomic.Pointer[string]
	StartTime      time.Time
	StartedAt      atomic.Int64
	SymbolOverride atomic.Bool // true when SetSymbol has been called; disables animated spinner
	SymbolPtr      *atomic.Pointer[string]
}

// Started reports whether the task has begun executing.
func (t *GroupTask) Started() bool {
	return !t.startTime().IsZero()
}

// Duration returns the elapsed execution time, or zero while the task is queued.
func (t *GroupTask) Duration(now time.Time) time.Duration {
	start := t.startTime()
	if start.IsZero() {
		return 0
	}
	return now.Sub(start)
}

// MarkStarted records the actual task start time.
func (t *GroupTask) MarkStarted(now time.Time) {
	if now.IsZero() {
		return
	}
	t.StartedAt.Store(now.UnixNano())
}

// FinishTime returns the actual finish time, or the zero value if unfinished.
func (t *GroupTask) FinishTime() time.Time {
	if finishedAt := t.FinishedAt.Load(); finishedAt > 0 {
		return time.Unix(0, finishedAt)
	}
	return time.Time{}
}

// MarkFinished records the actual task finish time.
func (t *GroupTask) MarkFinished(now time.Time) {
	if now.IsZero() {
		return
	}
	t.FinishedAt.Store(now.UnixNano())
}

func (t *GroupTask) startTime() time.Time {
	if startedAt := t.StartedAt.Load(); startedAt > 0 {
		return time.Unix(0, startedAt)
	}
	return t.StartTime
}

// Add registers an animation builder with the group and returns a
// [GroupEntry] for starting the task.
func (g *Group) Add(b *Builder) *GroupEntry {
	if b.Log == nil {
		b.Log = g.Log
	}

	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]core.Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msgPtr.Store(&b.Message)
	fieldsPtr.Store(&b.Fields)
	sym := b.SymbolIcon
	if sym == "" {
		sym = DefaultSymbol
	}
	symbolPtr.Store(&sym)

	levelPtr := &atomic.Int64{}
	levelPtr.Store(int64(level.Unset))

	gt := &GroupTask{
		Builder:   b,
		DoneErr:   make(chan error, 1),
		FieldsPtr: fieldsPtr,
		LevelPtr:  levelPtr,
		MsgPtr:    msgPtr,
		SymbolPtr: symbolPtr,
	}

	g.Mu.Lock()
	g.Tasks = append(g.Tasks, gt)
	g.Mu.Unlock()

	return &GroupEntry{task: gt, group: g}
}

// Wait runs the render loop, blocking until all tasks complete or the context
// is cancelled. The returned [GroupResult] can be used to log a single summary line.
func (g *Group) Wait() *GroupResult {
	g.Mu.Lock()
	tasks := g.Tasks
	g.Mu.Unlock()

	l := g.Log

	result := &GroupResult{
		resultBase: resultBase[GroupResult]{
			Log:          l,
			SuccessLevel: level.Info,
			LevelError:   level.Error,
		},
		group: g,
	}
	result.InitSelf(result)

	if len(tasks) == 0 {
		return result
	}

	// Delegate the actual render loop to the Logger implementation.
	_ = l.RunGroup(g.Ctx, g)
	return result
}

func (g *Group) acquireSlot(ctx context.Context) error {
	if g.Parallelism <= 0 {
		return nil
	}

	sem := g.semaphore()
	select {
	case sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (g *Group) releaseSlot() {
	if g.Parallelism <= 0 {
		return
	}

	select {
	case <-g.semaphore():
	default:
	}
}

func (g *Group) semaphore() chan struct{} {
	g.Mu.Lock()
	defer g.Mu.Unlock()

	if g.Parallelism <= 0 {
		return nil
	}
	if g.sem == nil {
		g.sem = make(chan struct{}, g.Parallelism)
	}
	return g.sem
}

// GroupEntry is returned by [Group.Add] and provides [Run] and [Progress]
// methods to start a task within the group.
type GroupEntry struct {
	task  *GroupTask
	group *Group
}

// Run starts a simple task (no progress updates) and returns a [TaskResult].
func (ge *GroupEntry) Run(task TaskFunc) *TaskResult {
	return ge.Progress(func(ctx context.Context, _ *Update) error {
		return task(ctx)
	})
}

// Progress starts a task with progress update capability and returns a [TaskResult].
func (ge *GroupEntry) Progress(task UpdateFunc) *TaskResult {
	t := ge.task
	b := t.Builder
	g := ge.group

	update := &Update{
		MsgText:           b.Message,
		MsgPtr:            t.MsgPtr,
		FieldsPtr:         t.FieldsPtr,
		Base:              b.Fields,
		LevelPtr:          t.LevelPtr,
		SymbolOverridePtr: &t.SymbolOverride,
		SymbolPtr:         t.SymbolPtr,
	}
	if b.Mode == AnimationBar {
		update.ProgressPtr = b.BarProgressPtr
		update.TotalPtr = b.BarTotalPtr
	}
	update.InitSelf(update)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.DoneErr <- fmt.Errorf("panic: %v", r)
			}
		}()
		if err := g.acquireSlot(g.Ctx); err != nil {
			t.DoneErr <- err
			return
		}
		defer g.releaseSlot()
		t.MarkStarted(time.Now())
		err := task(g.Ctx, update)
		t.MarkFinished(time.Now())
		t.DoneErr <- err
	}()

	l := b.Log
	if l == nil {
		l = g.Log
	}

	r := &TaskResult{
		resultBase: resultBase[TaskResult]{
			Log:          b.IndentedLogger(l),
			PartOverride: b.PartOverrides,
			SuccessLevel: b.Level,
			LevelError:   level.Error,
		},
		task: t,
	}
	r.InitSelf(r)
	return r
}

// TaskResult holds the result of a group animation task.
type TaskResult struct {
	resultBase[TaskResult]

	task *GroupTask
}

// Err returns the error, logging success or failure using the original message.
func (r *TaskResult) Err() error {
	return r.Send()
}

// Msg logs at success level with the given message on success, or at error
// level with the error string on failure. Returns the error.
func (r *TaskResult) Msg(msg string) error {
	r.SuccessMsg = msg
	return r.Send()
}

// Send finalises the result, logging at the configured success or error level.
func (r *TaskResult) Send() error {
	t := r.task
	err := t.Err

	msg := r.SuccessMsg
	if msg == "" {
		msg = *t.MsgPtr.Load()
	}

	finalFields := t.Builder.ResolveDynamicFields(*t.FieldsPtr.Load(), t.Duration(time.Now()))
	if len(r.Fields) > 0 {
		finalFields = core.MergeFields(finalFields, r.Fields)
	}

	sendResult(
		r.Log,
		finalFields,
		r.PartOverride,
		r.SymbolStr,
		r.SuccessLevel,
		r.LevelError,
		msg,
		r.ErrorMsg,
		err,
	)
	return err
}

// Silent returns just the error without logging anything.
func (r *TaskResult) Silent() error {
	return r.task.Err
}

// GroupResult holds the aggregate result of a [Group.Wait].
type GroupResult struct {
	resultBase[GroupResult]

	group *Group
}

// Err returns the joined error, logging success or failure.
func (r *GroupResult) Err() error {
	return r.Send()
}

// Msg logs at success level with the given message if all tasks succeeded.
func (r *GroupResult) Msg(msg string) error {
	r.SuccessMsg = msg
	return r.Send()
}

// Send finalises the result.
func (r *GroupResult) Send() error {
	err := r.joinErrors()
	sendResult(
		r.Log,
		r.Fields,
		r.PartOverride,
		r.SymbolStr,
		r.SuccessLevel,
		r.LevelError,
		r.SuccessMsg,
		r.ErrorMsg,
		err,
	)
	return err
}

// Silent returns the joined error without logging anything.
func (r *GroupResult) Silent() error {
	return r.joinErrors()
}

func (r *GroupResult) joinErrors() error {
	var errs []error
	for _, t := range r.group.Tasks {
		if t.Err != nil {
			errs = append(errs, t.Err)
		}
	}
	return errors.Join(errs...)
}

// sendResult logs a success or error event.
func sendResult(
	l Logger,
	fields []core.Field,
	parts *[]core.Part,
	symbol *string,
	successLevel, errorLevel core.Level,
	successMsg string,
	errorMsg *string,
	err error,
) {
	var lvl core.Level
	var msg string
	var errField error

	switch {
	case err == nil:
		lvl = successLevel
		msg = successMsg
	case errorMsg != nil:
		lvl = errorLevel
		msg = *errorMsg
		errField = err
	default:
		lvl = errorLevel
		msg = err.Error()
	}

	evt := DoneEvent{
		Level:  lvl,
		Fields: fields,
		Parts:  parts,
		Symbol: symbol,
		Msg:    msg,
		Err:    errField,
	}
	l.Done(evt)
}
