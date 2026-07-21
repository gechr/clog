package fx

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/gechr/clog/field/deadline"
	"github.com/gechr/clog/field/elapsed"
	"github.com/gechr/clog/fx/spinner"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/level"
	"github.com/stretchr/testify/assert"
)

func TestUpdateSetProgress(t *testing.T) {
	var pAtom atomic.Int64
	var tAtom atomic.Int64
	tAtom.Store(100)

	u := &Update{
		progressPtr: &pAtom,
		totalPtr:    &tAtom,
	}
	u.InitSelf(u)

	result := u.SetProgress(42)
	assert.Equal(t, u, result) // fluent return
	assert.Equal(t, int64(42), pAtom.Load())
	assert.Equal(t, 42, u.Progress())

	result = u.SetTotal(200)
	assert.Equal(t, u, result)
	assert.Equal(t, int64(200), tAtom.Load())
}

func TestUpdateSetProgressClamp(t *testing.T) {
	var pAtom atomic.Int64
	var tAtom atomic.Int64
	tAtom.Store(100)

	u := &Update{progressPtr: &pAtom, totalPtr: &tAtom}
	u.InitSelf(u)

	// Clamp above total
	u.SetProgress(150)
	assert.Equal(t, int64(100), pAtom.Load())

	// Clamp below zero
	u.SetProgress(-10)
	assert.Equal(t, int64(0), pAtom.Load())

	// Normal value passes through
	u.SetProgress(50)
	assert.Equal(t, int64(50), pAtom.Load())
}

func TestUpdateSetProgressNilNoOp(t *testing.T) {
	// Non-bar Update has nil pointers - should be a no-op.
	u := &Update{}
	u.InitSelf(u)

	assert.NotPanics(t, func() {
		u.SetProgress(50)
		u.SetTotal(100)
	})
	assert.Equal(t, 0, u.Progress())
}

func TestUpdateSetTotalClamp(t *testing.T) {
	var pAtom atomic.Int64
	var tAtom atomic.Int64
	tAtom.Store(100)

	u := &Update{progressPtr: &pAtom, totalPtr: &tAtom}
	u.InitSelf(u)

	u.SetTotal(0)
	assert.Equal(t, int64(1), tAtom.Load())

	u.SetTotal(-10)
	assert.Equal(t, int64(1), tAtom.Load())
}

func TestUpdateSetSymbol(t *testing.T) {
	var sym atomic.Pointer[string]
	initial := "⏳"
	sym.Store(&initial)

	u := &Update{symbolPtr: &sym}
	u.InitSelf(u)

	result := u.SetSymbol("📦")
	assert.Equal(t, u, result) // fluent return
	assert.Equal(t, "📦", *sym.Load())
}

func TestUpdateSetLevel(t *testing.T) {
	var lvl atomic.Int64
	lvl.Store(int64(level.Info))

	u := &Update{levelPtr: &lvl}
	u.InitSelf(u)

	result := u.SetLevel(level.Error)
	assert.Equal(t, u, result) // fluent return
	assert.Equal(t, int64(level.Error), lvl.Load())
}

func TestUpdateSetLevelNilNoOp(t *testing.T) {
	u := &Update{}
	u.InitSelf(u)

	assert.NotPanics(t, func() {
		u.SetLevel(level.Error)
	})
}

func TestUpdateSetSymbolNilNoOp(t *testing.T) {
	u := &Update{}
	u.InitSelf(u)

	assert.NotPanics(t, func() {
		u.SetSymbol("📦")
	})
}

// TestNewUpdateSetSymbolStopsSpinner guards the standalone Progress wiring:
// newUpdate links the task's symbol override, so SetSymbol replaces an
// animated spinner with the static symbol.
func TestNewUpdateSetSymbolStopsSpinner(t *testing.T) {
	b := NewBuilder(BuilderConfig{
		AnimatedSymbol: true,
		Level:          level.Info,
		SpinnerConfig: spinner.Config{
			Frames:   []string{"a", "b"},
			Interval: time.Millisecond,
		},
	})
	sym := "⏳"
	symbolPtr := &atomic.Pointer[string]{}
	symbolPtr.Store(&sym)
	task := &groupTask{builder: b, start: time.Now(), symbolPtr: symbolPtr}
	gt := &renderTask{groupTask: task}
	gt.cfg.StyleSymbol = func(s string, _ core.Level) string { return s }

	u := task.newUpdate()
	assert.Equal(t, "a", resolveSymbol(gt, task.start))

	u.SetSymbol("✓")
	assert.Equal(t, "✓", resolveSymbol(gt, task.start))
}

func TestAddTotal(t *testing.T) {
	var tAtom atomic.Int64
	tAtom.Store(100)

	u := &Update{totalPtr: &tAtom}
	u.InitSelf(u)

	// Add positive delta
	result := u.AddTotal(50)
	assert.Equal(t, u, result) // fluent return
	assert.Equal(t, int64(150), tAtom.Load())

	// Add negative delta
	u.AddTotal(-30)
	assert.Equal(t, int64(120), tAtom.Load())

	// Clamp to minimum 1
	u.AddTotal(-200)
	assert.Equal(t, int64(1), tAtom.Load())
}

func TestAddTotalNilNoOp(t *testing.T) {
	u := &Update{}
	u.InitSelf(u)

	assert.NotPanics(t, func() {
		u.AddTotal(50)
	})
}

func TestUpdateMessage(t *testing.T) {
	var msgAtom atomic.Pointer[string]
	var fieldsAtom atomic.Pointer[[]core.Field]
	initial := "starting"
	msgAtom.Store(&initial)

	u := &Update{msgPtr: &msgAtom, fieldsPtr: &fieldsAtom}
	u.InitSelf(u)

	assert.Equal(t, "starting", u.Message())

	u.Msg("step 1").Send()
	assert.Equal(t, "step 1", u.Message())
}

func TestUpdateDeadlineAnchorsAtCall(t *testing.T) {
	var msgAtom atomic.Pointer[string]
	var fieldsAtom atomic.Pointer[[]core.Field]
	initial := "starting"
	msgAtom.Store(&initial)

	u := &Update{
		msgPtr:    &msgAtom,
		fieldsPtr: &fieldsAtom,
		elapsed:   func() time.Duration { return 5 * time.Second },
	}
	u.InitSelf(u)

	result := u.Deadline("wait", 10*time.Second)
	assert.Equal(t, u, result) // fluent return
	u.Send()

	// The task is 5s in when the deadline is attached, so the anchor folds
	// that into From: Remaining = From - taskElapsed counts 10s from now.
	assert.Equal(t, []core.Field{
		{
			Key: "wait",
			Value: core.DeadlineField{
				Remaining:  10 * time.Second,
				From:       15 * time.Second,
				OmitOnDone: true,
			},
		},
	}, *fieldsAtom.Load())

	// A later Send without the field clears the countdown - the deadline is
	// scoped to the phase that sent it.
	u.Msg("next phase").Send()
	assert.Empty(t, *fieldsAtom.Load())
}

func TestUpdateDeadlineNilElapsed(t *testing.T) {
	var msgAtom atomic.Pointer[string]
	var fieldsAtom atomic.Pointer[[]core.Field]
	initial := "starting"
	msgAtom.Store(&initial)

	u := &Update{msgPtr: &msgAtom, fieldsPtr: &fieldsAtom}
	u.InitSelf(u)

	assert.NotPanics(t, func() {
		u.Deadline("wait", 10*time.Second).Send()
	})
	assert.Equal(t, []core.Field{
		{
			Key: "wait",
			Value: core.DeadlineField{
				Remaining:  10 * time.Second,
				From:       10 * time.Second,
				OmitOnDone: true,
			},
		},
	}, *fieldsAtom.Load())
}

func TestUpdateDeadlineOptions(t *testing.T) {
	var msgAtom atomic.Pointer[string]
	var fieldsAtom atomic.Pointer[[]core.Field]
	initial := "starting"
	msgAtom.Store(&initial)

	u := &Update{
		msgPtr:    &msgAtom,
		fieldsPtr: &fieldsAtom,
		elapsed:   func() time.Duration { return 0 },
	}
	u.InitSelf(u)

	u.Deadline("wait", 10*time.Second, deadline.WithOmitOnDone(false)).Send()
	assert.Equal(t, []core.Field{
		{
			Key: "wait",
			Value: core.DeadlineField{
				Remaining: 10 * time.Second,
				From:      10 * time.Second,
			},
		},
	}, *fieldsAtom.Load())
}

func TestUpdateElapsedAnchorsAtCall(t *testing.T) {
	var msgAtom atomic.Pointer[string]
	var fieldsAtom atomic.Pointer[[]core.Field]
	initial := "starting"
	msgAtom.Store(&initial)

	u := &Update{
		msgPtr:    &msgAtom,
		fieldsPtr: &fieldsAtom,
		elapsed:   func() time.Duration { return 5 * time.Second },
	}
	u.InitSelf(u)

	result := u.Elapsed("waited", 2*time.Second)
	assert.Equal(t, u, result) // fluent return
	u.Send()

	// The task is 5s in when the stopwatch is attached, backdated 2s, so the
	// anchor lands at 3s: Value = taskElapsed - Start counts from 2s now.
	assert.Equal(t, []core.Field{
		{
			Key: "waited",
			Value: core.ElapsedField{
				Value:      2 * time.Second,
				Start:      3 * time.Second,
				OmitOnDone: true,
			},
		},
	}, *fieldsAtom.Load())

	// A later Send without the field clears the stopwatch - the timer is
	// scoped to the phase that sent it.
	u.Msg("next phase").Send()
	assert.Empty(t, *fieldsAtom.Load())
}

func TestUpdateElapsedNilElapsed(t *testing.T) {
	var msgAtom atomic.Pointer[string]
	var fieldsAtom atomic.Pointer[[]core.Field]
	initial := "starting"
	msgAtom.Store(&initial)

	u := &Update{msgPtr: &msgAtom, fieldsPtr: &fieldsAtom}
	u.InitSelf(u)

	assert.NotPanics(t, func() {
		u.Elapsed("waited", 2*time.Second).Send()
	})
	assert.Equal(t, []core.Field{
		{
			Key: "waited",
			Value: core.ElapsedField{
				Value:      2 * time.Second,
				Start:      -2 * time.Second,
				OmitOnDone: true,
			},
		},
	}, *fieldsAtom.Load())
}

func TestUpdateElapsedOptions(t *testing.T) {
	var msgAtom atomic.Pointer[string]
	var fieldsAtom atomic.Pointer[[]core.Field]
	initial := "starting"
	msgAtom.Store(&initial)

	u := &Update{
		msgPtr:    &msgAtom,
		fieldsPtr: &fieldsAtom,
		elapsed:   func() time.Duration { return 0 },
	}
	u.InitSelf(u)

	u.Elapsed("waited", 0, elapsed.WithOmitOnDone(false)).Send()
	assert.Equal(t, []core.Field{
		{
			Key:   "waited",
			Value: core.ElapsedField{},
		},
	}, *fieldsAtom.Load())
}

func TestUpdateClear(t *testing.T) {
	var msgAtom atomic.Pointer[string]
	var fieldsAtom atomic.Pointer[[]core.Field]
	var levelAtom atomic.Int64
	initial := "working"
	msgAtom.Store(&initial)
	levelAtom.Store(int64(level.Unset))

	u := &Update{
		msgPtr:    &msgAtom,
		fieldsPtr: &fieldsAtom,
		levelPtr:  &levelAtom,
		base:      []core.Field{{Key: "app", Value: "one"}},
	}
	u.InitSelf(u)
	u.Msg("step").Str("k", "v").SetLevel(level.Warn).Send()

	u.Clear()

	assert.Empty(t, *msgAtom.Load(), "the message must be emptied")
	assert.Empty(t, *fieldsAtom.Load(), "every field must go, including the builder's base fields")
	assert.Equal(t, int64(level.Unset), levelAtom.Load(), "the level override must reset")

	// A later Send starts from scratch - Clear must not poison reuse.
	u.Msg("again").Send()
	assert.Equal(t, "again", *msgAtom.Load())
}
