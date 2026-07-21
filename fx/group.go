package fx

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"charm.land/lipgloss/v2"
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

	liveMu     sync.Mutex
	liveRegion *core.LiveRegion
	liveSlot   uint64
	liveWake   chan struct{}
	suspended  bool

	clearOnCancel     bool
	fieldAlignment    FieldAlignment
	footer            *groupStatus
	header            *groupStatus
	hideDone          bool
	log               Logger
	maxLines          int
	maxHeightPercent  float64
	monotonic         bool
	overflowFunc      OverflowIndicatorFunc
	overflowIndicator bool
	overflowStyle     *lipgloss.Style
	parallelism       int
	renderDelay       time.Duration
	syncAnimations    bool
	tasks             []*groupTask
	transientFooter   bool
	transientHeader   bool

	sem chan struct{}
}

// NewGroup creates a new animation group bound to ctx and log. Configure it
// with [GroupOption] values; animations in a group share a common epoch
// unless [WithoutSyncAnimations] is given.
func NewGroup(ctx context.Context, log Logger, opts ...GroupOption) *Group {
	g := &Group{
		ctx: ctx, log: log, syncAnimations: true,
		liveWake: make(chan struct{}, 1),
	}
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

// groupTask holds per-animation state for the group render loop.
type groupTask struct {
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

// started reports whether the task has begun executing.
func (t *groupTask) started() bool {
	return !t.startTime().IsZero()
}

// duration returns the elapsed execution time, or zero while the task is
// queued. Once the task has finished the duration is frozen at the finish
// time, so done lines and final results report the task's actual runtime
// rather than time that keeps accruing while sibling tasks run.
func (t *groupTask) duration(now time.Time) time.Duration {
	start := t.startTime()
	if start.IsZero() {
		return 0
	}
	if finish := t.finishTime(); !finish.IsZero() && finish.Before(now) {
		now = finish
	}
	return now.Sub(start)
}

// markStarted records the actual task start time.
func (t *groupTask) markStarted(now time.Time) {
	if now.IsZero() {
		return
	}
	t.startedAt.Store(now.UnixNano())
}

// finishTime returns the actual finish time, or the zero value if unfinished.
func (t *groupTask) finishTime() time.Time {
	if finishedAt := t.finishedAt.Load(); finishedAt > 0 {
		return time.Unix(0, finishedAt)
	}
	return time.Time{}
}

// markFinished records the actual task finish time.
func (t *groupTask) markFinished(now time.Time) {
	if now.IsZero() {
		return
	}
	t.finishedAt.Store(now.UnixNano())
}

func (t *groupTask) startTime() time.Time {
	if startedAt := t.startedAt.Load(); startedAt > 0 {
		return time.Unix(0, startedAt)
	}
	return t.start
}

// Add registers an animation builder with the group and returns a
// [GroupEntry] for starting the task.
func (g *Group) Add(b *Builder) *GroupEntry {
	if b.log == nil {
		b.log = g.log
	}

	msgPtr, fieldsPtr, symbolPtr := newTaskPointers(b)

	levelPtr := &atomic.Int64{}
	levelPtr.Store(int64(level.Unset))

	gt := &groupTask{
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

// Suspend temporarily removes this group's block from its shared live region
// so another interactive owner can use the terminal. Other animation slots
// remain live, including slots registered after the suspension begins. By
// default it preserves the current cursor visibility; use [WithShowCursor] to
// show the cursor while the group is hidden. Suspension is remembered before
// the group's first frame, so a concurrently starting group cannot race past
// it. It is safe to call repeatedly.
func (g *Group) Suspend(opts ...SuspendOption) {
	if g == nil {
		return
	}
	cfg := suspendOptions{}
	for _, opt := range opts {
		opt(&cfg)
	}

	g.liveMu.Lock()
	g.suspended = true
	region := g.liveRegion
	id := g.liveSlot
	g.liveSlot = 0
	if id != 0 && region != nil {
		region.UnregisterWithCursor(id, cfg.showCursor)
	}
	g.liveMu.Unlock()
}

// Resume restores a group block previously hidden with [Group.Suspend]. The
// render loop is woken so the latest group frame is registered and painted
// immediately. It is safe to call repeatedly and is a no-op when the group is
// not suspended.
func (g *Group) Resume() {
	if g == nil {
		return
	}
	g.liveMu.Lock()
	if !g.suspended {
		g.liveMu.Unlock()
		return
	}
	g.suspended = false
	g.liveMu.Unlock()

	select {
	case g.liveWake <- struct{}{}:
	default:
	}
}

// registerLiveSlot registers the group's current block unless the group is
// suspended. The group lock spans registration so Suspend cannot miss a slot
// that is being created concurrently.
func (g *Group) registerLiveSlot(
	region *core.LiveRegion,
	render func(time.Time) string,
	tick time.Duration,
) {
	g.liveMu.Lock()
	defer g.liveMu.Unlock()
	if g.suspended {
		return
	}
	if g.liveSlot != 0 {
		return
	}
	g.liveRegion = region
	g.liveSlot = region.Register(render, tick)
}

func (g *Group) hasLiveSlot() bool {
	g.liveMu.Lock()
	defer g.liveMu.Unlock()
	return g.liveSlot != 0
}

func (g *Group) unregisterLiveSlot(region *core.LiveRegion) {
	g.liveMu.Lock()
	defer g.liveMu.Unlock()
	id := g.liveSlot
	g.liveSlot = 0
	if id != 0 {
		region.Unregister(id)
	}
	g.liveRegion = nil
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
	task  *groupTask
	group *Group
}

// Run starts a simple task (no progress updates) and returns a [TaskResult].
func (ge *GroupEntry) Run(task TaskFunc) *TaskResult {
	return ge.Progress(func(ctx context.Context, _ *Update) error {
		return task(ctx)
	})
}

// newUpdate builds the [Update] that drives this task's rendered line, wiring it
// to the task's atomic state so each Send is picked up by the render loop. It is
// shared by [GroupEntry.Progress] (clog runs the work) and [GroupEntry.Manual]
// (the caller runs the work).
func (t *groupTask) newUpdate() *Update {
	b := t.builder
	update := &Update{
		msgText:           b.message,
		msgPtr:            t.msgPtr,
		fieldsPtr:         t.fieldsPtr,
		base:              b.Fields,
		levelPtr:          t.levelPtr,
		symbolOverridePtr: &t.symbolOverride,
		symbolPtr:         t.symbolPtr,
		elapsed:           func() time.Duration { return t.duration(time.Now()) },
	}
	if b.mode == AnimationBar {
		update.progressPtr = b.barProgressPtr
		update.totalPtr = b.barTotalPtr
	}
	update.InitSelf(update)
	return update
}

// Manual registers the task for live rendering but does not run it: the caller
// drives the returned [Update] from its own goroutine and invokes finish exactly
// once when the work ends, passing any error (nil on success). Use it to render
// progress for work scheduled by your own executor - clog renders, you run.
//
// Unlike [GroupEntry.Progress], Manual spawns no goroutine and consumes no
// parallelism slot, since the caller owns concurrency; the group's parallelism
// option therefore does not apply. The task is marked started now, and finish
// marks it finished and reports its result to [Group.Wait], which must still be
// called (typically from another goroutine) to render and to await completion.
// Every task added to the group must eventually finish, or Wait blocks forever.
func (ge *GroupEntry) Manual() (*Update, func(err error)) {
	t := ge.task
	t.markStarted(time.Now())
	update := t.newUpdate()

	var once sync.Once
	finish := func(err error) {
		once.Do(func() {
			t.markFinished(time.Now())
			t.doneErr <- err
		})
	}
	return update, finish
}

// Progress starts a task with progress update capability and returns a [TaskResult].
func (ge *GroupEntry) Progress(task UpdateFunc) *TaskResult {
	t := ge.task
	b := t.builder
	g := ge.group

	update := t.newUpdate()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.markFinished(time.Now())
				t.doneErr <- fmt.Errorf("panic: %v", r)
			}
		}()
		if err := g.acquireSlot(g.ctx); err != nil {
			t.doneErr <- err
			return
		}
		defer g.releaseSlot()
		t.markStarted(time.Now())
		err := task(g.ctx, update)
		t.markFinished(time.Now())
		t.doneErr <- err
	}()

	l := b.log
	if l == nil {
		l = g.log
	}

	r := &TaskResult{
		resultBase: resultBase[TaskResult]{
			Log:          b.IndentedLogger(l),
			PartOverride: b.partOverrides,
			SuccessLevel: b.lvl,
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

	task *groupTask
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

	finalFields := t.builder.ResolveDoneFields(*t.fieldsPtr.Load(), t.duration(time.Now()))
	if len(r.Fields) > 0 {
		finalFields = core.MergeFields(finalFields, r.Fields)
	}

	sendResult(
		r.Log,
		finalFields,
		r.PartOverride,
		r.SymbolStr,
		r.MsgStyle,
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
		r.MsgStyle,
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
	msgStyle *lipgloss.Style,
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
		Level:    lvl,
		Fields:   fields,
		MsgStyle: msgStyle,
		Parts:    parts,
		Symbol:   symbol,
		Msg:      msg,
		Err:      errField,
	}
	l.Done(evt)
}
