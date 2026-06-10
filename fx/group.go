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
// block. Create one with [NewGroup] (or root clog's Group function), add
// animations with [Group.Add], then call [Group.Wait] to run the render loop.
type Group struct {
	ctx context.Context //nolint:containedctx // Group shares a single ctx with all child goroutines
	mu  sync.Mutex

	clearOnCancel    bool
	fieldAlignment   FieldAlignment
	footer           *groupStatus
	header           *groupStatus
	hideDone         bool
	log              Logger
	maxLines         int
	maxHeightPercent float64
	monotonic        bool
	parallelism      int
	renderDelay      time.Duration
	syncAnimations   bool
	tasks            []*GroupTask
	transientFooter  bool
	transientHeader  bool

	sem chan struct{}
}

// NewGroup creates a new animation group bound to ctx and log. Configure it
// with [GroupOption] values; animations in a group share a common epoch
// unless [WithoutSyncAnimations] is given.
func NewGroup(ctx context.Context, log Logger, opts ...GroupOption) *Group {
	g := &Group{ctx: ctx, log: log, syncAnimations: true}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// GroupStatusFunc is called each render tick with the number of completed
// and total tasks. Use the [Update] to set the message and fields for
// that tick.
type GroupStatusFunc func(done, total int, u *Update)

// groupStatus pairs a [Builder] (for initial config like level, symbol,
// parts) with a [GroupStatusFunc] callback that updates it each tick.
type groupStatus struct {
	builder  *Builder
	callback GroupStatusFunc
}

// GroupTask holds per-animation state for the group render loop.
type GroupTask struct {
	builder        *Builder
	doneErr        chan error // buffered(1); goroutine sends result here
	err            error      // populated by Wait() after doneErr is drained
	fieldsPtr      *atomic.Pointer[[]core.Field]
	finishedAt     atomic.Int64
	levelPtr       *atomic.Int64
	msgPtr         *atomic.Pointer[string]
	start          time.Time
	startedAt      atomic.Int64
	symbolOverride atomic.Bool // true when SetSymbol has been called; disables animated spinner
	symbolPtr      *atomic.Pointer[string]
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
	t.startedAt.Store(now.UnixNano())
}

// FinishTime returns the actual finish time, or the zero value if unfinished.
func (t *GroupTask) FinishTime() time.Time {
	if finishedAt := t.finishedAt.Load(); finishedAt > 0 {
		return time.Unix(0, finishedAt)
	}
	return time.Time{}
}

// MarkFinished records the actual task finish time.
func (t *GroupTask) MarkFinished(now time.Time) {
	if now.IsZero() {
		return
	}
	t.finishedAt.Store(now.UnixNano())
}

func (t *GroupTask) startTime() time.Time {
	if startedAt := t.startedAt.Load(); startedAt > 0 {
		return time.Unix(0, startedAt)
	}
	return t.start
}

// Add registers an animation builder with the group and returns a
// [GroupEntry] for starting the task.
func (g *Group) Add(b *Builder) *GroupEntry {
	if b.Log == nil {
		b.Log = g.log
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
		builder:   b,
		doneErr:   make(chan error, 1),
		fieldsPtr: fieldsPtr,
		levelPtr:  levelPtr,
		msgPtr:    msgPtr,
		symbolPtr: symbolPtr,
	}

	g.mu.Lock()
	g.tasks = append(g.tasks, gt)
	g.mu.Unlock()

	return &GroupEntry{task: gt, group: g}
}

// Wait runs the render loop, blocking until all tasks complete or the context
// is cancelled. The returned [GroupResult] can be used to log a single summary line.
func (g *Group) Wait() *GroupResult {
	g.mu.Lock()
	tasks := g.tasks
	g.mu.Unlock()

	result := &GroupResult{
		resultBase: resultBase[GroupResult]{
			Log:          g.log,
			SuccessLevel: level.Info,
			LevelError:   level.Error,
		},
		group: g,
	}
	result.InitSelf(result)

	if len(tasks) == 0 {
		return result
	}

	_ = runGroupLoop(g.ctx, g)
	return result
}

func (g *Group) acquireSlot(ctx context.Context) error {
	if g.parallelism <= 0 {
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
	if g.parallelism <= 0 {
		return
	}

	select {
	case <-g.semaphore():
	default:
	}
}

func (g *Group) semaphore() chan struct{} {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.parallelism <= 0 {
		return nil
	}
	if g.sem == nil {
		g.sem = make(chan struct{}, g.parallelism)
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
	b := t.builder
	g := ge.group

	update := &Update{
		MsgText:           b.Message,
		MsgPtr:            t.msgPtr,
		FieldsPtr:         t.fieldsPtr,
		Base:              b.Fields,
		LevelPtr:          t.levelPtr,
		SymbolOverridePtr: &t.symbolOverride,
		SymbolPtr:         t.symbolPtr,
	}
	if b.Mode == AnimationBar {
		update.ProgressPtr = b.BarProgressPtr
		update.TotalPtr = b.BarTotalPtr
	}
	update.InitSelf(update)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.doneErr <- fmt.Errorf("panic: %v", r)
			}
		}()
		if err := g.acquireSlot(g.ctx); err != nil {
			t.doneErr <- err
			return
		}
		defer g.releaseSlot()
		t.MarkStarted(time.Now())
		err := task(g.ctx, update)
		t.MarkFinished(time.Now())
		t.doneErr <- err
	}()

	l := b.Log
	if l == nil {
		l = g.log
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
	err := t.err

	msg := r.SuccessMsg
	if msg == "" {
		msg = *t.msgPtr.Load()
	}

	finalFields := t.builder.ResolveDynamicFields(*t.fieldsPtr.Load(), t.Duration(time.Now()))
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
	return r.task.err
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
	for _, t := range r.group.tasks {
		if t.err != nil {
			errs = append(errs, t.err)
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
