package clog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/fx"
	"github.com/gechr/clog/fx/spinner"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpinnerConstructor(t *testing.T) {
	b := Spinner("loading")

	assert.Equal(t, "loading", b.InitialMessage())
	assert.Equal(t, fx.AnimationNone, b.AnimationMode())
	assert.True(t, b.UsesAnimatedSymbol())
	assert.Equal(t, spinner.DefaultConfig().Interval, b.SpinnerStyle().Interval)
	assert.Empty(t, b.Fields)
}

func TestBoomerangFrames(t *testing.T) {
	assert.Equal(
		t,
		[]string{"a", "b", "c", "d", "c", "b"},
		spinner.BoomerangFrames([]string{"a", "b", "c", "d"}),
	)
}

func TestBoomerangFramesThree(t *testing.T) {
	assert.Equal(t, []string{"a", "b", "c", "b"}, spinner.BoomerangFrames([]string{"a", "b", "c"}))
}

func TestBoomerangFramesTwoUnchanged(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, spinner.BoomerangFrames([]string{"a", "b"}))
}

func TestBoomerangFramesOneUnchanged(t *testing.T) {
	assert.Equal(t, []string{"a"}, spinner.BoomerangFrames([]string{"a"}))
}

func TestBoomerangFramesEmpty(t *testing.T) {
	assert.Empty(t, spinner.BoomerangFrames(nil))
}

func TestBoomerangFramesDoesNotMutateInput(t *testing.T) {
	orig := []string{"a", "b", "c"}
	_ = spinner.BoomerangFrames(orig)

	assert.Equal(t, []string{"a", "b", "c"}, orig)
}

func TestSpinnerBuilderType(t *testing.T) {
	b := Spinner("test", spinner.WithConfig(spinner.Dot))

	assert.Equal(t, spinner.Dot.Interval, b.SpinnerStyle().Interval)
}

func TestSpinnerWithBoomerang(t *testing.T) {
	b := Spinner("test", spinner.WithBoomerang())

	assert.True(t, b.SpinnerStyle().Boomerang)
}

func TestSpinnerWithBoomerangAndStyle(t *testing.T) {
	b := Spinner("test", spinner.WithConfig(spinner.Dot), spinner.WithBoomerang())

	assert.Equal(t, spinner.Dot.Interval, b.SpinnerStyle().Interval)
	assert.True(t, b.SpinnerStyle().Boomerang)
}

func TestSpinnerWithReverse(t *testing.T) {
	// Default is already reversed (Moon reverse), so start from a non-reversed style.
	b := Spinner("test", spinner.WithConfig(spinner.Dot), spinner.WithReverse())

	assert.True(t, b.SpinnerStyle().Reverse)
}

func TestSpinnerWithInterval(t *testing.T) {
	custom := 50 * time.Millisecond
	b := Spinner("test", spinner.WithInterval(custom))

	assert.Equal(t, custom, b.SpinnerStyle().Interval)
}

func TestSpinnerWithIntervalZeroNoOp(t *testing.T) {
	b := Spinner("test", spinner.WithInterval(0))

	assert.Equal(t, spinner.DefaultConfig().Interval, b.SpinnerStyle().Interval)
}

func TestSpinnerWithIntervalNegativeNoOp(t *testing.T) {
	b := Spinner("test", spinner.WithInterval(-1))

	assert.Equal(t, spinner.DefaultConfig().Interval, b.SpinnerStyle().Interval)
}

func TestSpinnerBuilderStr(t *testing.T) {
	b := Spinner("test").Str("k", "v")

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "k", b.Fields[0].Key)
	assert.Equal(t, "v", b.Fields[0].Value)
}

func TestSpinnerBuilderStrs(t *testing.T) {
	b := Spinner("test").Strs("tags", []string{"a", "b"})

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "tags", b.Fields[0].Key)
}

func TestSpinnerBuilderInt(t *testing.T) {
	b := Spinner("test").Int("n", 42)

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "n", b.Fields[0].Key)
	assert.Equal(t, 42, b.Fields[0].Value)
}

func TestSpinnerBuilderUint64(t *testing.T) {
	b := Spinner("test").Uint64("size", 100)

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "size", b.Fields[0].Key)
	assert.Equal(t, uint64(100), b.Fields[0].Value)
}

func TestSpinnerBuilderUints64(t *testing.T) {
	b := Spinner("test").Uints64("sizes", []uint64{1, 2})

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "sizes", b.Fields[0].Key)
}

func TestSpinnerBuilderFloat64(t *testing.T) {
	b := Spinner("test").Float64("pi", 3.14)

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "pi", b.Fields[0].Key)
	assert.InDelta(t, 3.14, b.Fields[0].Value, 0)
}

func TestSpinnerBuilderBool(t *testing.T) {
	b := Spinner("test").Bool("ok", true)

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "ok", b.Fields[0].Key)
	assert.Equal(t, true, b.Fields[0].Value)
}

func TestSpinnerBuilderBools(t *testing.T) {
	b := Spinner("test").Bools("flags", []bool{true, false})

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "flags", b.Fields[0].Key)
	assert.Equal(t, []bool{true, false}, b.Fields[0].Value)
}

func TestSpinnerBuilderDur(t *testing.T) {
	b := Spinner("test").Duration("elapsed", time.Second)

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "elapsed", b.Fields[0].Key)
	assert.Equal(t, core.DurationField{Value: time.Second}, b.Fields[0].Value)
}

func TestSpinnerBuilderPath(t *testing.T) {
	b := Spinner("test").Path("dir", "/tmp")

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "dir", b.Fields[0].Key)
	assert.Equal(t, "/tmp", b.Fields[0].Value)
}

func TestSpinnerBuilderLine(t *testing.T) {
	b := Spinner("test").Line("file", "main.go", 5)

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "file", b.Fields[0].Key)
	assert.Equal(t, "main.go:5", b.Fields[0].Value)
}

func TestSpinnerBuilderFloats64(t *testing.T) {
	b := Spinner("test").Floats64("vals", []float64{1.1, 2.2})

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "vals", b.Fields[0].Key)
	assert.Equal(t, []float64{1.1, 2.2}, b.Fields[0].Value)
}

func TestSpinnerBuilderColumn(t *testing.T) {
	b := Spinner("test").Column("loc", "main.go", 5, 10)

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "loc", b.Fields[0].Key)
	assert.Equal(t, "main.go:5:10", b.Fields[0].Value)
}

func TestSpinnerBuilderColumnMinimum(t *testing.T) {
	b := Spinner("test").Column("loc", "main.go", 0, 0)

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "loc", b.Fields[0].Key)
	// Both line and column should be clamped to 1.
	assert.Equal(t, "main.go:1:1", b.Fields[0].Value)
}

func TestSpinnerBuilderLineMinimum(t *testing.T) {
	b := Spinner("test").Line("file", "main.go", 0)

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "file", b.Fields[0].Key)
	// line < 1 is clamped to 1.
	assert.Equal(t, "main.go:1", b.Fields[0].Value)
}

func TestSpinnerBuilderLink(t *testing.T) {
	b := Spinner("test").Link("docs", "https://example.com", "docs")

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "docs", b.Fields[0].Key)
	// In test env, colors are disabled so hyperlink returns plain text.
	assert.Equal(t, "docs", b.Fields[0].Value)
}

func TestSpinnerBuilderURL(t *testing.T) {
	b := Spinner("test").URL("link", "https://example.com")

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "link", b.Fields[0].Key)
	// In test env, colors are disabled so hyperlink returns plain text.
	assert.Equal(t, "https://example.com", b.Fields[0].Value)
}

func TestSpinnerBuilderAny(t *testing.T) {
	b := Spinner("test").Any("data", 123)

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "data", b.Fields[0].Key)
	assert.Equal(t, 123, b.Fields[0].Value)
}

func TestSpinnerBuilderChaining(t *testing.T) {
	b := Spinner("test").Str("a", "1").Int("b", 2).Bool("c", true)

	require.Len(t, b.Fields, 3)
}

func TestSpinnerBuilderParts(t *testing.T) {
	b := Spinner("test").Parts(PartSymbol, PartMessage)

	parts, ok := b.PartOrder()
	require.True(t, ok)
	assert.Equal(t, []Part{PartSymbol, PartMessage}, parts)
}

func TestSpinnerPartsThreadsToWaitResult(t *testing.T) {
	l, buf := newTestLogger()

	// Override: only show message on the spinner + completion.
	result := l.Spinner("loading").
		Parts(PartMessage).
		Wait(context.Background(), func(_ context.Context) error {
			return nil
		})

	// Non-TTY path prints an initial animation line, then completion writes another.
	// The initial line also uses the overridden parts (PartMessage only).
	buf.Reset()
	require.NoError(t, result.Msg("done"))
	assert.Equal(t, "done\n", buf.String())
}

func TestWaitResultPartsOverride(t *testing.T) {
	l, buf := newTestLogger()

	// Builder does NOT set parts, but WaitResult overrides.
	result := l.Spinner("loading").
		Wait(context.Background(), func(_ context.Context) error {
			return nil
		})

	buf.Reset()
	require.NoError(t, result.Parts(PartMessage).Msg("done"))
	assert.Equal(t, "done\n", buf.String())
}

func TestWaitResultPartsNilUsesLoggerDefault(t *testing.T) {
	l, buf := newTestLogger()
	l.SetParts(PartMessage)

	result := l.Spinner("loading").
		Wait(context.Background(), func(_ context.Context) error {
			return nil
		})

	// No Parts() call on builder or result - should use logger's PartMessage.
	buf.Reset()
	require.NoError(t, result.Msg("done"))
	assert.Equal(t, "done\n", buf.String())
}

func TestSpinnerWaitSuccess(t *testing.T) {
	// In test env, ColorsDisabled() == true, so runAnimation takes fast path.
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	result := Spinner("loading").
		Str("file", "test.go").
		Wait(context.Background(), func(_ context.Context) error {
			return nil
		})

	require.NoError(t, result.TaskErr)
	assert.Equal(t, "loading", result.SuccessMsg)
}

func TestSpinnerWaitError(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	testErr := errors.New("test error")
	result := Spinner("loading").Wait(context.Background(), func(_ context.Context) error {
		return testErr
	})

	require.ErrorIs(t, result.TaskErr, testErr)
}

func TestSpinnerProgressSuccess(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	result := Spinner("step 1").
		Str("file", "a.go").
		Progress(context.Background(), func(_ context.Context, update *fx.Update) error {
			update.Msg("step 2").Str("file", "b.go").Send()
			return nil
		})

	require.NoError(t, result.TaskErr)
}

func TestSpinnerProgressError(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	testErr := errors.New("fail")
	result := Spinner(
		"loading",
	).Progress(context.Background(), func(_ context.Context, _ *fx.Update) error {
		return testErr
	})

	require.ErrorIs(t, result.TaskErr, testErr)
}

func TestSpinnerProgressMsgOnly(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	result := Spinner(
		"step 1",
	).Progress(context.Background(), func(_ context.Context, update *fx.Update) error {
		// Update message without additional fields.
		update.Msg("step 2").Send()
		return nil
	})

	require.NoError(t, result.TaskErr)
	assert.Equal(t, "step 2", result.SuccessMsg)
}

func TestSpinnerProgressMsgf(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	result := Spinner(
		"step 1",
	).Progress(context.Background(), func(_ context.Context, update *fx.Update) error {
		update.Msgf("step %d of %d", 2, 3).Send()
		return nil
	})

	require.NoError(t, result.TaskErr)
	assert.Equal(t, "step 2 of 3", result.SuccessMsg)
}

// newTestWaitResult creates a WaitResult with initSelf called for test use.
func newTestWaitResult(msg string, err error) *fx.WaitResult {
	return fx.NewWaitResult(err, fxLogger{Default()}, nil, LevelInfo, msg)
}

func TestWaitResultStr(t *testing.T) {
	w := newTestWaitResult("test", nil)
	w.Str("k", "v")

	require.Len(t, w.Fields, 1)
	assert.Equal(t, "k", w.Fields[0].Key)
	assert.Equal(t, "v", w.Fields[0].Value)
}

func TestWaitResultInt(t *testing.T) {
	w := newTestWaitResult("test", nil)
	w.Int("n", 42)

	require.Len(t, w.Fields, 1)
	assert.Equal(t, "n", w.Fields[0].Key)
	assert.Equal(t, 42, w.Fields[0].Value)
}

func TestWaitResultAny(t *testing.T) {
	w := newTestWaitResult("test", nil)
	w.Any("data", true)

	require.Len(t, w.Fields, 1)
	assert.Equal(t, "data", w.Fields[0].Key)
	assert.Equal(t, true, w.Fields[0].Value)
}

func TestWaitResultSymbol(t *testing.T) {
	w := newTestWaitResult("test", nil)
	w.Symbol("done")

	require.NotNil(t, w.SymbolStr)
	assert.Equal(t, "done", *w.SymbolStr)
}

func TestWaitResultMsgSuccess(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	var got Entry

	Default().SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	w := newTestWaitResult("loading", nil)
	err := w.Msg("done")

	require.NoError(t, err)
	assert.Equal(t, LevelInfo, got.Level)
	assert.Equal(t, "done", got.Message)
}

func TestWaitResultMsgError(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	var got Entry

	Default().SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	testErr := errors.New("boom")
	w := newTestWaitResult("loading", testErr)

	err := w.Msg("done")

	require.ErrorIs(t, err, testErr)
	assert.Equal(t, LevelError, got.Level)
	assert.Equal(t, "boom", got.Message)

	// Error should appear only as the message, not also as an error= field.
	for _, f := range got.Fields {
		assert.NotEqual(
			t,
			ErrorKey,
			f.Key,
			"default error should not produce a duplicate error field",
		)
	}
}

func TestWaitResultErrSuccess(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	var got Entry

	Default().SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	w := newTestWaitResult("loading", nil)
	err := w.Err()

	require.NoError(t, err)
	assert.Equal(t, LevelInfo, got.Level)
	assert.Equal(t, "loading", got.Message)
}

func TestWaitResultErrError(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	var got Entry

	Default().SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	testErr := errors.New("boom")
	w := newTestWaitResult("loading", testErr)

	err := w.Err()

	require.ErrorIs(t, err, testErr)
	assert.Equal(t, LevelError, got.Level)

	// Error should appear only as the message, not also as an error= field.
	for _, f := range got.Fields {
		assert.NotEqual(
			t,
			ErrorKey,
			f.Key,
			"default error should not produce a duplicate error field",
		)
	}
}

func TestWaitResultOnSuccessLevel(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))
	Default().SetLevel(LevelWarn)

	var got Entry

	Default().SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	w := newTestWaitResult("loading", nil)
	err := w.OnSuccessLevel(LevelWarn).Send()

	require.NoError(t, err)
	assert.Equal(t, LevelWarn, got.Level)
	assert.Equal(t, "loading", got.Message)
}

func TestWaitResultOnSuccessMessage(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	var got Entry

	Default().SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	w := newTestWaitResult("loading", nil)
	err := w.OnSuccessMessage("finished").Send()

	require.NoError(t, err)
	assert.Equal(t, LevelInfo, got.Level)
	assert.Equal(t, "finished", got.Message)
}

func TestWaitResultOnErrorLevel(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))
	Default().SetExitFunc(func(_ int) {}) // prevent os.Exit

	var got Entry

	Default().SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	testErr := errors.New("boom")
	w := newTestWaitResult("loading", testErr)
	err := w.OnErrorLevel(LevelFatal).Send()

	require.ErrorIs(t, err, testErr)
	assert.Equal(t, LevelFatal, got.Level)
}

func TestWaitResultOnErrorMessage(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	var got Entry

	Default().SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	testErr := errors.New("boom")
	w := newTestWaitResult("loading", testErr)
	err := w.OnErrorMessage("custom failure").Send()

	require.ErrorIs(t, err, testErr)
	assert.Equal(t, LevelError, got.Level)
	assert.Equal(t, "custom failure", got.Message)

	// With a custom error message, the actual error should appear as an error= field.
	var found bool
	for _, f := range got.Fields {
		if f.Key == ErrorKey {
			assert.Equal(t, testErr, f.Value)
			found = true
		}
	}
	assert.True(
		t,
		found,
		"custom error message should include error= field with the original error",
	)
}

func TestWaitResultOnErrorMessageDefault(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	var got Entry

	Default().SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	testErr := errors.New("boom")
	w := newTestWaitResult("loading", testErr)
	err := w.Send()

	require.ErrorIs(t, err, testErr)
	assert.Equal(t, "boom", got.Message)

	// Without a custom error message, the error IS the message - no error= field.
	for _, f := range got.Fields {
		assert.NotEqual(
			t,
			ErrorKey,
			f.Key,
			"default error should not produce a duplicate error field",
		)
	}
}

func TestWaitResultSendSuccess(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	var got Entry

	Default().SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	w := newTestWaitResult("loading", nil)
	err := w.Send()

	require.NoError(t, err)
	assert.Equal(t, LevelInfo, got.Level)
	assert.Equal(t, "loading", got.Message)
}

func TestWaitResultSilent(t *testing.T) {
	testErr := errors.New("boom")
	w := newTestWaitResult("loading", testErr)

	require.ErrorIs(t, w.Silent(), testErr)
}

func TestWaitResultSilentNil(t *testing.T) {
	w := newTestWaitResult("loading", nil)

	require.NoError(t, w.Silent())
}

func TestWaitResultEventWithSymbol(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	var got Entry

	Default().SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	w := newTestWaitResult("test", nil)
	w.Symbol("done")

	_ = w.Msg("done")

	assert.Equal(t, "done", got.Symbol)
}

func TestWaitResultEventWithFields(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	var got Entry

	Default().SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	w := newTestWaitResult("test", nil)
	w.Str("a", "1").Int("b", 2)

	_ = w.Msg("done")

	require.Len(t, got.Fields, 2)
	assert.Equal(t, "a", got.Fields[0].Key)
	assert.Equal(t, "b", got.Fields[1].Key)
}

func TestWaitResultEventLevelFiltered(_ *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))
	Default().SetLevel(LevelFatal) // filter out everything

	w := newTestWaitResult("test", nil)
	// Should not panic even when event is nil (filtered out).
	_ = w.Msg("done")
}

func TestRunAnimationDoneCase(t *testing.T) {
	origDefault := Default()

	defer func() {
		SetDefault(origDefault)
	}()

	var buf bytes.Buffer

	SetDefault(New(NewOutput(&buf, ColorAlways)))
	Default().SetLevel(LevelInfo) // ensure not verbose

	// Use a very fast spinner so tick fires quickly.
	fastSpinner := spinner.Config{
		Frames:   []string{"A", "B"},
		Interval: time.Millisecond,
	}

	result := Spinner(
		"loading", spinner.WithConfig(fastSpinner),
	).Wait(context.Background(), func(_ context.Context) error {
		// Wait long enough for at least one spinner frame to render.
		time.Sleep(20 * time.Millisecond)
		return nil
	})

	require.NoError(t, result.TaskErr)

	got := buf.String()

	// Should have written cursor hide/show and at least one frame.
	assert.NotEmpty(t, got, "expected some output from spinner animation")
}

func TestRunAnimationContextCancel(t *testing.T) {
	origDefault := Default()

	defer func() {
		SetDefault(origDefault)
	}()

	var buf bytes.Buffer

	SetDefault(New(NewOutput(&buf, ColorAlways)))
	Default().SetLevel(LevelInfo)

	fastSpinner := spinner.Config{
		Frames:   []string{"A"},
		Interval: time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context shortly so ctx.Done() fires in the select loop
	// before the action completes.
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	result := Spinner(
		"loading",
		spinner.WithConfig(fastSpinner),
	).Wait(ctx, func(_ context.Context) error {
		// Block much longer than the cancel delay.
		time.Sleep(10 * time.Second)
		return nil
	})

	require.ErrorIs(t, result.TaskErr, context.Canceled)
}

func TestRunAnimationError(t *testing.T) {
	origDefault := Default()

	defer func() {
		SetDefault(origDefault)
	}()

	var buf bytes.Buffer

	SetDefault(New(NewOutput(&buf, ColorAlways)))
	Default().SetLevel(LevelInfo)

	fastSpinner := spinner.Config{
		Frames:   []string{"A"},
		Interval: time.Millisecond,
	}

	testErr := errors.New("action failed")
	result := Spinner(
		"loading", spinner.WithConfig(fastSpinner),
	).Wait(context.Background(), func(_ context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return testErr
	})

	require.ErrorIs(t, result.TaskErr, testErr)
}

func TestRunAnimationVerboseFastPath(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))
	Default().SetLevel(LevelDebug)

	// When IsVerbose() returns true, runAnimation should take fast path.
	result := Spinner("test").Wait(context.Background(), func(_ context.Context) error {
		return nil
	})

	require.NoError(t, result.TaskErr)
}

func TestRunAnimationNoColorWithTimestamp(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	var buf bytes.Buffer

	SetDefault(New(TestOutput(&buf)))
	Default().SetReportTimestamp(true)
	Default().SetTimeFormat("TIMESTAMP")

	result := Spinner("loading").Wait(context.Background(), func(_ context.Context) error {
		return nil
	})

	require.NoError(t, result.TaskErr)

	got := buf.String()
	// Should contain the timestamp and the message with hourglass emoji.
	assert.Equal(t, "TIMESTAMP INF ⏳ loading\n", got)
}

func TestRunAnimationWithTimestamp(t *testing.T) {
	origDefault := Default()

	defer func() {
		SetDefault(origDefault)
	}()

	var buf bytes.Buffer

	SetDefault(New(NewOutput(&buf, ColorAlways)))
	Default().SetReportTimestamp(true)

	fastSpinner := spinner.Config{
		Frames:   []string{"A"},
		Interval: time.Millisecond,
	}

	result := Spinner("loading", spinner.WithConfig(fastSpinner)).
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(20 * time.Millisecond)
			return nil
		})

	require.NoError(t, result.TaskErr)

	got := buf.String()
	// Animated output with timestamps should include time-like content.
	assert.NotEmpty(t, got)
}

func TestUpdateReuseAfterSend(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	result := Spinner("step 1").
		Progress(context.Background(), func(_ context.Context, update *fx.Update) error {
			// First send with a field.
			update.Msg("step 2").Str("k", "v1").Send()

			// After Send, fields should be reset. Add new fields and send again.
			update.Msg("step 3").Str("k", "v2").Int("n", 42).Send()
			return nil
		})

	require.NoError(t, result.TaskErr)
	assert.Equal(t, "step 3", result.SuccessMsg)
}

func TestUpdateErr(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	testErr := errors.New("progress error")

	result := Spinner("loading").
		Progress(context.Background(), func(_ context.Context, update *fx.Update) error {
			update.Err(testErr).Send()
			return nil
		})

	require.NoError(t, result.TaskErr)

	// The error field should have been stored.
	fields := result.Fields
	require.NotEmpty(t, fields)

	found := false
	for _, f := range fields {
		if f.Key == ErrorKey {
			assert.Equal(t, testErr, f.Value)
			found = true
		}
	}
	assert.True(t, found, "expected error field in result fields")
}

func TestUpdateErrNil(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	result := Spinner("loading").
		Progress(context.Background(), func(_ context.Context, update *fx.Update) error {
			update.Err(nil).Send()
			return nil
		})

	require.NoError(t, result.TaskErr)

	// No error field should have been added.
	for _, f := range result.Fields {
		assert.NotEqual(t, ErrorKey, f.Key, "nil error should not produce an error field")
	}
}

func TestUpdateStringer(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	result := Spinner("loading").
		Progress(context.Background(), func(_ context.Context, update *fx.Update) error {
			update.Stringer("item", &testStringer{s: "hello"}).Send()
			return nil
		})

	require.NoError(t, result.TaskErr)
	require.NotEmpty(t, result.Fields)
	assert.Equal(t, "item", result.Fields[0].Key)
	assert.Equal(t, "hello", result.Fields[0].Value)
}

func TestUpdateStringerTypedNil(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	result := Spinner("loading").
		Progress(context.Background(), func(_ context.Context, update *fx.Update) error {
			var s *testStringer // typed nil
			update.Stringer("item", s).Send()
			return nil
		})

	require.NoError(t, result.TaskErr)
	// Typed nil should be skipped - no fields added.
	for _, f := range result.Fields {
		assert.NotEqual(t, "item", f.Key, "typed nil stringer should not produce a field")
	}
}

func TestUpdateStringers(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))

	result := Spinner("loading").
		Progress(context.Background(), func(_ context.Context, update *fx.Update) error {
			var nilStringer *testStringer
			update.Stringers("items", []fmt.Stringer{
				&testStringer{s: "a"},
				nil,
				nilStringer,
				&testStringer{s: "d"},
			}).Send()
			return nil
		})

	require.NoError(t, result.TaskErr)
	require.NotEmpty(t, result.Fields)
	assert.Equal(t, "items", result.Fields[0].Key)
	assert.Equal(t, []string{"a", Nil, Nil, "d"}, result.Fields[0].Value)
}

func TestSpinnerSymbolStyleApplied(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	var buf bytes.Buffer
	SetDefault(New(NewOutput(&buf, ColorAlways)))
	Default().SetParts(PartSymbol, PartMessage)
	Default().SetLevelAlign(AlignNone)
	Default().SetStyles(&style.Config{
		Symbols: style.LevelMap{
			LevelInfo: new(lipgloss.NewStyle().Foreground(lipgloss.Color("2"))),
		},
	})

	_ = Spinner("loading", spinner.WithConfig(spinner.Config{
		Frames:   []string{"X"},
		Interval: time.Millisecond,
	})).Wait(context.Background(), func(_ context.Context) error {
		return nil
	}).Msg("done")

	got := buf.String()
	// The done line symbol (ℹ️) should be styled with green (color 2).
	assert.Equal(t, "⏳ loading\n\x1b[32mℹ️\x1b[m done\n", got)
}

func TestSetSpinnerDefaults(t *testing.T) {
	logger := NewWriter(io.Discard)
	logger.SetSpinnerDefaults(
		spinner.WithConfig(spinner.Dots),
		spinner.WithInterval(time.Millisecond),
	)

	b := logger.Spinner("test")
	assert.Equal(t, spinner.Dots.Frames, b.SpinnerStyle().Frames)
	assert.Equal(t, time.Millisecond, b.SpinnerStyle().Interval)

	// Per-spinner options still override the logger default.
	b = logger.Spinner("test", spinner.WithConfig(spinner.Dot))
	assert.Equal(t, spinner.Dot.Frames, b.SpinnerStyle().Frames)
}

func TestSpinnerWithGradient(t *testing.T) {
	stops := []style.ColorStop{
		{Position: 0, Color: style.InterpolateGradient(0, spinner.DefaultGradient())},
		{Position: 1, Color: style.InterpolateGradient(1, spinner.DefaultGradient())},
	}
	b := Spinner("test", spinner.WithGradient(stops...))

	assert.Equal(t, stops, b.SpinnerStyle().Gradient)
}

func TestSpinnerWithGradientDefaultStops(t *testing.T) {
	b := Spinner("test", spinner.WithGradient())

	assert.Equal(t, spinner.DefaultGradient(), b.SpinnerStyle().Gradient)
}

func TestSpinnerStyleClonesGradient(t *testing.T) {
	b := Spinner("test", spinner.WithGradient())

	got := b.SpinnerStyle().Gradient
	got[0].Position = 0.99
	assert.Equal(t, spinner.DefaultGradient(), b.SpinnerStyle().Gradient,
		"SpinnerStyle should return a defensive copy of the gradient")
}

func TestSetSpinnerDefaultsGradientInherited(t *testing.T) {
	logger := NewWriter(io.Discard)
	logger.SetSpinnerDefaults(
		spinner.WithGradient(),
		spinner.WithGradientTiming(spinner.GradientTimeBased),
		spinner.WithGradientSpeed(2),
	)

	b := logger.Spinner("test")
	got := b.SpinnerStyle()
	assert.Equal(t, spinner.DefaultGradient(), got.Gradient)
	assert.Equal(t, spinner.GradientTimeBased, got.GradientTiming)
	assert.InDelta(t, 2.0, got.GradientSpeed, 1e-9)
}
