package fx

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/level"
)

// Group manages a set of concurrent animations rendered as a multi-line
// block. Create one with root clog's Group function, add animations with
// [Group.Add], then call [Group.Wait] to run the render loop.
type Group struct {
	Ctx context.Context //nolint:containedctx // Group shares a single ctx with all child goroutines
	Mu  sync.Mutex

	Log   Logger
	Tasks []*GroupTask
}

// GroupTask holds per-animation state for the group render loop.
// This is exported so the root clog rendering code can access it.
type GroupTask struct {
	Builder   *Builder
	DoneErr   chan error // buffered(1); goroutine sends result here
	Err       error      // populated by Wait() after DoneErr is drained
	FieldsPtr *atomic.Pointer[[]core.Field]
	MsgPtr    *atomic.Pointer[string]
	StartTime time.Time
	SymbolPtr *atomic.Pointer[string]
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
		sym = "⏳"
	}
	symbolPtr.Store(&sym)

	gt := &GroupTask{
		Builder:   b,
		DoneErr:   make(chan error, 1),
		FieldsPtr: fieldsPtr,
		MsgPtr:    msgPtr,
		StartTime: time.Now(),
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
		MsgText:   b.Message,
		MsgPtr:    t.MsgPtr,
		FieldsPtr: t.FieldsPtr,
		Base:      b.Fields,
		SymbolPtr: t.SymbolPtr,
	}
	if b.Mode == AnimationBar {
		update.ProgressPtr = b.BarProgressPtr
		update.TotalPtr = b.BarTotalPtr
	}
	update.InitSelf(update)

	go func() {
		t.DoneErr <- task(g.Ctx, update)
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

	finalFields := t.Builder.ResolveDynamicFields(*t.FieldsPtr.Load(), time.Since(t.StartTime))
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
