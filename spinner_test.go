package clog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpinnerConstructor(t *testing.T) {
	b := Spinner("loading")

	assert.Equal(t, "loading", b.title)
	assert.Equal(t, DefaultSpinner.FPS, b.spinner.FPS)
	assert.Empty(t, b.fields)
}

func TestSpinnerBuilderType(t *testing.T) {
	b := Spinner("test").Type(spinner.Dot)

	assert.Equal(t, spinner.Dot.FPS, b.spinner.FPS)
}

func TestSpinnerBuilderStr(t *testing.T) {
	b := Spinner("test").Str("k", "v")

	require.Len(t, b.fields, 1)
	assert.Equal(t, "k", b.fields[0].Key)
	assert.Equal(t, "v", b.fields[0].Value)
}

func TestSpinnerBuilderStrs(t *testing.T) {
	b := Spinner("test").Strs("tags", []string{"a", "b"})

	require.Len(t, b.fields, 1)
	assert.Equal(t, "tags", b.fields[0].Key)
}

func TestSpinnerBuilderInt(t *testing.T) {
	b := Spinner("test").Int("n", 42)

	require.Len(t, b.fields, 1)
	assert.Equal(t, "n", b.fields[0].Key)
	assert.Equal(t, 42, b.fields[0].Value)
}

func TestSpinnerBuilderUint64(t *testing.T) {
	b := Spinner("test").Uint64("size", 100)

	require.Len(t, b.fields, 1)
	assert.Equal(t, "size", b.fields[0].Key)
	assert.Equal(t, uint64(100), b.fields[0].Value)
}

func TestSpinnerBuilderUints64(t *testing.T) {
	b := Spinner("test").Uints64("sizes", []uint64{1, 2})

	require.Len(t, b.fields, 1)
	assert.Equal(t, "sizes", b.fields[0].Key)
}

func TestSpinnerBuilderFloat64(t *testing.T) {
	b := Spinner("test").Float64("pi", 3.14)

	require.Len(t, b.fields, 1)
	assert.Equal(t, "pi", b.fields[0].Key)
	assert.InDelta(t, 3.14, b.fields[0].Value, 0)
}

func TestSpinnerBuilderBool(t *testing.T) {
	b := Spinner("test").Bool("ok", true)

	require.Len(t, b.fields, 1)
	assert.Equal(t, "ok", b.fields[0].Key)
	assert.Equal(t, true, b.fields[0].Value)
}

func TestSpinnerBuilderBools(t *testing.T) {
	b := Spinner("test").Bools("flags", []bool{true, false})

	require.Len(t, b.fields, 1)
	assert.Equal(t, "flags", b.fields[0].Key)
	assert.Equal(t, []bool{true, false}, b.fields[0].Value)
}

func TestSpinnerBuilderDur(t *testing.T) {
	b := Spinner("test").Duration("elapsed", time.Second)

	require.Len(t, b.fields, 1)
	assert.Equal(t, "elapsed", b.fields[0].Key)
	assert.Equal(t, time.Second, b.fields[0].Value)
}

func TestSpinnerBuilderPath(t *testing.T) {
	b := Spinner("test").Path("dir", "/tmp")

	require.Len(t, b.fields, 1)
	assert.Equal(t, "dir", b.fields[0].Key)
	assert.Equal(t, "/tmp", b.fields[0].Value)
}

func TestSpinnerBuilderLine(t *testing.T) {
	b := Spinner("test").Line("file", "main.go", 5)

	require.Len(t, b.fields, 1)
	assert.Equal(t, "file", b.fields[0].Key)
	assert.Equal(t, "main.go:5", b.fields[0].Value)
}

func TestSpinnerBuilderFloats64(t *testing.T) {
	b := Spinner("test").Floats64("vals", []float64{1.1, 2.2})

	require.Len(t, b.fields, 1)
	assert.Equal(t, "vals", b.fields[0].Key)
	assert.Equal(t, []float64{1.1, 2.2}, b.fields[0].Value)
}

func TestSpinnerBuilderColumn(t *testing.T) {
	b := Spinner("test").Column("loc", "main.go", 5, 10)

	require.Len(t, b.fields, 1)
	assert.Equal(t, "loc", b.fields[0].Key)
	assert.Equal(t, "main.go:5:10", b.fields[0].Value)
}

func TestSpinnerBuilderColumnMinimum(t *testing.T) {
	b := Spinner("test").Column("loc", "main.go", 0, 0)

	require.Len(t, b.fields, 1)
	assert.Equal(t, "loc", b.fields[0].Key)
	// Both line and column should be clamped to 1.
	assert.Equal(t, "main.go:1:1", b.fields[0].Value)
}

func TestSpinnerBuilderLineMinimum(t *testing.T) {
	b := Spinner("test").Line("file", "main.go", 0)

	require.Len(t, b.fields, 1)
	assert.Equal(t, "file", b.fields[0].Key)
	// line < 1 is clamped to 1.
	assert.Equal(t, "main.go:1", b.fields[0].Value)
}

func TestSpinnerBuilderLink(t *testing.T) {
	b := Spinner("test").Link("docs", "https://example.com", "docs")

	require.Len(t, b.fields, 1)
	assert.Equal(t, "docs", b.fields[0].Key)
	// In test env, colors are disabled so hyperlink returns plain text.
	assert.Equal(t, "docs", b.fields[0].Value)
}

func TestSpinnerBuilderURL(t *testing.T) {
	b := Spinner("test").URL("link", "https://example.com")

	require.Len(t, b.fields, 1)
	assert.Equal(t, "link", b.fields[0].Key)
	// In test env, colors are disabled so hyperlink returns plain text.
	assert.Equal(t, "https://example.com", b.fields[0].Value)
}

func TestSpinnerBuilderAny(t *testing.T) {
	b := Spinner("test").Any("data", 123)

	require.Len(t, b.fields, 1)
	assert.Equal(t, "data", b.fields[0].Key)
	assert.Equal(t, 123, b.fields[0].Value)
}

func TestSpinnerBuilderChaining(t *testing.T) {
	b := Spinner("test").Str("a", "1").Int("b", 2).Bool("c", true)

	require.Len(t, b.fields, 3)
}

func TestSpinnerWaitSuccess(t *testing.T) {
	// In test env, ColorsDisabled() == true, so runAnimation takes fast path.
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	result := Spinner("loading").
		Str("file", "test.go").
		Wait(context.Background(), func(_ context.Context) error {
			return nil
		})

	require.NoError(t, result.err)
	assert.Equal(t, "loading", result.successMsg)
}

func TestSpinnerWaitError(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	testErr := errors.New("test error")
	result := Spinner("loading").Wait(context.Background(), func(_ context.Context) error {
		return testErr
	})

	require.ErrorIs(t, result.err, testErr)
}

func TestSpinnerProgressSuccess(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	result := Spinner("step 1").
		Str("file", "a.go").
		Progress(context.Background(), func(_ context.Context, update *ProgressUpdate) error {
			update.Title("step 2").Str("file", "b.go").Send()
			return nil
		})

	require.NoError(t, result.err)
}

func TestSpinnerProgressError(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	testErr := errors.New("fail")
	result := Spinner(
		"loading",
	).Progress(context.Background(), func(_ context.Context, _ *ProgressUpdate) error {
		return testErr
	})

	require.ErrorIs(t, result.err, testErr)
}

func TestSpinnerProgressTitleOnly(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	result := Spinner(
		"step 1",
	).Progress(context.Background(), func(_ context.Context, update *ProgressUpdate) error {
		// Update title without additional fields.
		update.Title("step 2").Send()
		return nil
	})

	require.NoError(t, result.err)
	assert.Equal(t, "step 2", result.successMsg)
}

// newTestWaitResult creates a WaitResult with initSelf called for test use.
func newTestWaitResult(title string, err error) *WaitResult {
	w := &WaitResult{
		err:          err,
		successLevel: InfoLevel,
		successMsg:   title,
		errorLevel:   ErrorLevel,
	}
	w.initSelf(w)
	return w
}

func TestWaitResultStr(t *testing.T) {
	w := newTestWaitResult("test", nil)
	w.Str("k", "v")

	require.Len(t, w.fields, 1)
	assert.Equal(t, "k", w.fields[0].Key)
	assert.Equal(t, "v", w.fields[0].Value)
}

func TestWaitResultInt(t *testing.T) {
	w := newTestWaitResult("test", nil)
	w.Int("n", 42)

	require.Len(t, w.fields, 1)
	assert.Equal(t, "n", w.fields[0].Key)
	assert.Equal(t, 42, w.fields[0].Value)
}

func TestWaitResultAny(t *testing.T) {
	w := newTestWaitResult("test", nil)
	w.Any("data", true)

	require.Len(t, w.fields, 1)
	assert.Equal(t, "data", w.fields[0].Key)
	assert.Equal(t, true, w.fields[0].Value)
}

func TestWaitResultPrefix(t *testing.T) {
	w := newTestWaitResult("test", nil)
	w.Prefix("done")

	require.NotNil(t, w.prefix)
	assert.Equal(t, "done", *w.prefix)
}

func TestWaitResultMsgSuccess(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	var got Entry

	Default.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	w := newTestWaitResult("loading", nil)
	err := w.Msg("done")

	require.NoError(t, err)
	assert.Equal(t, InfoLevel, got.Level)
	assert.Equal(t, "done", got.Message)
}

func TestWaitResultMsgError(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	var got Entry

	Default.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	testErr := errors.New("boom")
	w := newTestWaitResult("loading", testErr)

	err := w.Msg("done")

	require.ErrorIs(t, err, testErr)
	assert.Equal(t, ErrorLevel, got.Level)
	assert.Equal(t, "boom", got.Message)
}

func TestWaitResultErrSuccess(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	var got Entry

	Default.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	w := newTestWaitResult("loading", nil)
	err := w.Err()

	require.NoError(t, err)
	assert.Equal(t, InfoLevel, got.Level)
	assert.Equal(t, "loading", got.Message)
}

func TestWaitResultErrError(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	var got Entry

	Default.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	testErr := errors.New("boom")
	w := newTestWaitResult("loading", testErr)

	err := w.Err()

	require.ErrorIs(t, err, testErr)
	assert.Equal(t, ErrorLevel, got.Level)
}

func TestWaitResultOnSuccessLevel(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	Default.SetLevel(WarnLevel)

	var got Entry

	Default.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	w := newTestWaitResult("loading", nil)
	err := w.OnSuccessLevel(WarnLevel).Send()

	require.NoError(t, err)
	assert.Equal(t, WarnLevel, got.Level)
	assert.Equal(t, "loading", got.Message)
}

func TestWaitResultOnSuccessMessage(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	var got Entry

	Default.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	w := newTestWaitResult("loading", nil)
	err := w.OnSuccessMessage("finished").Send()

	require.NoError(t, err)
	assert.Equal(t, InfoLevel, got.Level)
	assert.Equal(t, "finished", got.Message)
}

func TestWaitResultOnErrorLevel(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	Default.SetExitFunc(func(_ int) {}) // prevent os.Exit

	var got Entry

	Default.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	testErr := errors.New("boom")
	w := newTestWaitResult("loading", testErr)
	err := w.OnErrorLevel(FatalLevel).Send()

	require.ErrorIs(t, err, testErr)
	assert.Equal(t, FatalLevel, got.Level)
}

func TestWaitResultOnErrorMessage(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	var got Entry

	Default.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	testErr := errors.New("boom")
	w := newTestWaitResult("loading", testErr)
	err := w.OnErrorMessage("custom failure").Send()

	require.ErrorIs(t, err, testErr)
	assert.Equal(t, ErrorLevel, got.Level)
	assert.Equal(t, "custom failure", got.Message)
}

func TestWaitResultOnErrorMessageDefault(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	var got Entry

	Default.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	testErr := errors.New("boom")
	w := newTestWaitResult("loading", testErr)
	err := w.Send()

	require.ErrorIs(t, err, testErr)
	assert.Equal(t, "boom", got.Message)
}

func TestWaitResultSendSuccess(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	var got Entry

	Default.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	w := newTestWaitResult("loading", nil)
	err := w.Send()

	require.NoError(t, err)
	assert.Equal(t, InfoLevel, got.Level)
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

func TestWaitResultEventWithPrefix(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	var got Entry

	Default.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	w := newTestWaitResult("test", nil)
	w.Prefix("done")

	_ = w.Msg("done")

	assert.Equal(t, "done", got.Prefix)
}

func TestWaitResultEventWithFields(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	var got Entry

	Default.SetHandler(HandlerFunc(func(e Entry) {
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
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	Default.SetLevel(FatalLevel) // filter out everything

	w := newTestWaitResult("test", nil)
	// Should not panic even when event is nil (filtered out).
	_ = w.Msg("done")
}

func TestRunSpinnerAnimationDoneCase(t *testing.T) {
	origDefault := Default

	defer func() {
		Default = origDefault
	}()

	var buf bytes.Buffer

	Default = New(NewOutput(&buf, ColorAlways))
	Default.SetLevel(InfoLevel) // ensure not verbose

	// Use a very fast spinner so tick fires quickly.
	fastSpinner := spinner.Spinner{
		Frames: []string{"A", "B"},
		FPS:    time.Millisecond,
	}

	result := Spinner(
		"loading",
	).Type(fastSpinner).
		Wait(context.Background(), func(_ context.Context) error {
			// Wait long enough for at least one spinner frame to render.
			time.Sleep(20 * time.Millisecond)
			return nil
		})

	require.NoError(t, result.err)

	got := buf.String()

	// Should have written cursor hide/show and at least one frame.
	assert.NotEmpty(t, got, "expected some output from spinner animation")
}

func TestRunSpinnerAnimationContextCancel(t *testing.T) {
	origDefault := Default

	defer func() {
		Default = origDefault
	}()

	var buf bytes.Buffer

	Default = New(NewOutput(&buf, ColorAlways))
	Default.SetLevel(InfoLevel)

	fastSpinner := spinner.Spinner{
		Frames: []string{"A"},
		FPS:    time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context shortly so ctx.Done() fires in the select loop
	// before the action completes.
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	result := Spinner("loading").Type(fastSpinner).Wait(ctx, func(_ context.Context) error {
		// Block much longer than the cancel delay.
		time.Sleep(10 * time.Second)
		return nil
	})

	require.ErrorIs(t, result.err, context.Canceled)
}

func TestRunSpinnerAnimationError(t *testing.T) {
	origDefault := Default

	defer func() {
		Default = origDefault
	}()

	var buf bytes.Buffer

	Default = New(NewOutput(&buf, ColorAlways))
	Default.SetLevel(InfoLevel)

	fastSpinner := spinner.Spinner{
		Frames: []string{"A"},
		FPS:    time.Millisecond,
	}

	testErr := errors.New("action failed")
	result := Spinner(
		"loading",
	).Type(fastSpinner).
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(10 * time.Millisecond)
			return testErr
		})

	require.ErrorIs(t, result.err, testErr)
}

func TestRunSpinnerVerboseFastPath(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	Default.SetLevel(DebugLevel)

	// When IsVerbose() returns true, runAnimation should take fast path.
	result := Spinner("test").Wait(context.Background(), func(_ context.Context) error {
		return nil
	})

	require.NoError(t, result.err)
}

func TestRunSpinnerNoColorWithTimestamp(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	var buf bytes.Buffer

	Default = New(TestOutput(&buf))
	Default.SetReportTimestamp(true)

	result := Spinner("loading").Wait(context.Background(), func(_ context.Context) error {
		return nil
	})

	require.NoError(t, result.err)

	got := buf.String()
	// Should contain the timestamp and the title with hourglass emoji.
	assert.Contains(t, got, "⏳")
	assert.Contains(t, got, "loading")
}

func TestRunSpinnerAnimationWithTimestamp(t *testing.T) {
	origDefault := Default

	defer func() {
		Default = origDefault
	}()

	var buf bytes.Buffer

	Default = New(NewOutput(&buf, ColorAlways))
	Default.SetReportTimestamp(true)

	fastSpinner := spinner.Spinner{
		Frames: []string{"A"},
		FPS:    time.Millisecond,
	}

	result := Spinner("loading").Type(fastSpinner).
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(20 * time.Millisecond)
			return nil
		})

	require.NoError(t, result.err)

	got := buf.String()
	// Animated output with timestamps should include time-like content.
	assert.NotEmpty(t, got)
}

func TestProgressUpdateReuseAfterSend(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	result := Spinner("step 1").
		Progress(context.Background(), func(_ context.Context, update *ProgressUpdate) error {
			// First send with a field.
			update.Title("step 2").Str("k", "v1").Send()

			// After Send, fields should be reset. Add new fields and send again.
			update.Title("step 3").Str("k", "v2").Int("n", 42).Send()
			return nil
		})

	require.NoError(t, result.err)
	assert.Equal(t, "step 3", result.successMsg)
}

func TestProgressUpdateErr(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	testErr := errors.New("progress error")

	result := Spinner("loading").
		Progress(context.Background(), func(_ context.Context, update *ProgressUpdate) error {
			update.Err(testErr).Send()
			return nil
		})

	require.NoError(t, result.err)

	// The error field should have been stored.
	fields := result.fields
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

func TestProgressUpdateErrNil(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	result := Spinner("loading").
		Progress(context.Background(), func(_ context.Context, update *ProgressUpdate) error {
			update.Err(nil).Send()
			return nil
		})

	require.NoError(t, result.err)

	// No error field should have been added.
	for _, f := range result.fields {
		assert.NotEqual(t, ErrorKey, f.Key, "nil error should not produce an error field")
	}
}

func TestProgressUpdateStringer(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	result := Spinner("loading").
		Progress(context.Background(), func(_ context.Context, update *ProgressUpdate) error {
			update.Stringer("item", &testStringer{s: "hello"}).Send()
			return nil
		})

	require.NoError(t, result.err)
	require.NotEmpty(t, result.fields)
	assert.Equal(t, "item", result.fields[0].Key)
	assert.Equal(t, "hello", result.fields[0].Value)
}

func TestProgressUpdateStringerTypedNil(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	result := Spinner("loading").
		Progress(context.Background(), func(_ context.Context, update *ProgressUpdate) error {
			var s *testStringer // typed nil
			update.Stringer("item", s).Send()
			return nil
		})

	require.NoError(t, result.err)
	// Typed nil should be skipped — no fields added.
	for _, f := range result.fields {
		assert.NotEqual(t, "item", f.Key, "typed nil stringer should not produce a field")
	}
}

func TestProgressUpdateStringers(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	result := Spinner("loading").
		Progress(context.Background(), func(_ context.Context, update *ProgressUpdate) error {
			var nilStringer *testStringer
			update.Stringers("items", []fmt.Stringer{
				&testStringer{s: "a"},
				nil,
				nilStringer,
				&testStringer{s: "d"},
			}).Send()
			return nil
		})

	require.NoError(t, result.err)
	require.NotEmpty(t, result.fields)
	assert.Equal(t, "items", result.fields[0].Key)
	assert.Equal(t, []string{"a", Nil, Nil, "d"}, result.fields[0].Value)
}
