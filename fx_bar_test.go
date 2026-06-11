package clog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gechr/clog/fx"
	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/fx/bar/widget"
	"github.com/gechr/clog/fx/spinner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBarBuilderMode(t *testing.T) {
	b := Bar("test", 100)
	assert.Equal(t, fx.AnimationBar, b.Mode)
	require.NotNil(t, b.BarProgressPtr)
	require.NotNil(t, b.BarTotalPtr)
	assert.Equal(t, int64(100), b.BarTotalPtr.Load())
	assert.Equal(t, int64(0), b.BarProgressPtr.Load())
}

func TestBarSpinner(t *testing.T) {
	b := Bar("test", 100).Spinner()

	assert.Equal(t, fx.AnimationBar, b.Mode)
	assert.True(t, b.AnimatedSymbol)
}

func TestBarSpinnerWithOptions(t *testing.T) {
	b := Bar("test", 100).Spinner(spinner.WithConfig(spinner.Dots))

	assert.Equal(t, fx.AnimationBar, b.Mode)
	assert.True(t, b.AnimatedSymbol)
	assert.Equal(t, spinner.Dots.Interval, b.SpinnerConfig.Interval)
}

func TestBarBuilderTotalClamp(t *testing.T) {
	// total <= 0 clamped to 1
	b := Bar("test", 0)
	assert.Equal(t, int64(1), b.BarTotalPtr.Load())

	b2 := Bar("test", -5)
	assert.Equal(t, int64(1), b2.BarTotalPtr.Load())
}

func TestUpdateSetProgress(t *testing.T) {
	var pAtom atomic.Int64
	var tAtom atomic.Int64
	tAtom.Store(100)

	u := &fx.Update{
		ProgressPtr: &pAtom,
		TotalPtr:    &tAtom,
	}
	u.InitSelf(u)

	result := u.SetProgress(42)
	assert.Equal(t, u, result) // fluent return
	assert.Equal(t, int64(42), pAtom.Load())

	result = u.SetTotal(200)
	assert.Equal(t, u, result)
	assert.Equal(t, int64(200), tAtom.Load())
}

func TestUpdateSetProgressClamp(t *testing.T) {
	var pAtom atomic.Int64
	var tAtom atomic.Int64
	tAtom.Store(100)

	u := &fx.Update{ProgressPtr: &pAtom, TotalPtr: &tAtom}
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
	u := &fx.Update{}
	u.InitSelf(u)

	assert.NotPanics(t, func() {
		u.SetProgress(50)
		u.SetTotal(100)
	})
}

func TestUpdateSetTotalClamp(t *testing.T) {
	var pAtom atomic.Int64
	var tAtom atomic.Int64
	tAtom.Store(100)

	u := &fx.Update{ProgressPtr: &pAtom, TotalPtr: &tAtom}
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

	u := &fx.Update{SymbolPtr: &sym}
	u.InitSelf(u)

	result := u.SetSymbol("📦")
	assert.Equal(t, u, result) // fluent return
	assert.Equal(t, "📦", *sym.Load())
}

func TestUpdateSetLevel(t *testing.T) {
	var lvl atomic.Int64
	lvl.Store(int64(LevelInfo))

	u := &fx.Update{LevelPtr: &lvl}
	u.InitSelf(u)

	result := u.SetLevel(LevelError)
	assert.Equal(t, u, result) // fluent return
	assert.Equal(t, int64(LevelError), lvl.Load())
}

func TestUpdateSetLevelNilNoOp(t *testing.T) {
	u := &fx.Update{}
	u.InitSelf(u)

	assert.NotPanics(t, func() {
		u.SetLevel(LevelError)
	})
}

func TestUpdateSetSymbolNilNoOp(t *testing.T) {
	u := &fx.Update{}
	u.InitSelf(u)

	assert.NotPanics(t, func() {
		u.SetSymbol("📦")
	})
}

func TestBarProgressSharedWithUpdate(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()
	Default = NewWriter(io.Discard)

	var capturedProgress int64
	_ = Bar("Downloading", 100).
		After(10*time.Millisecond). // suppress animation display
		Progress(context.Background(), func(_ context.Context, p *fx.Update) error {
			p.SetProgress(75)
			capturedProgress = p.ProgressPtr.Load()
			return nil
		}).
		Silent()

	assert.Equal(t, int64(75), capturedProgress)
}

func TestBarWait(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()
	Default = NewWriter(io.Discard)

	err := Bar("test", 10).
		Wait(context.Background(), func(_ context.Context) error {
			return nil
		}).
		Silent()

	require.NoError(t, err)
}

func TestBarConfigOption(t *testing.T) {
	custom := bar.Config{
		CapLeft:     "|",
		CapRight:    "|",
		CharEmpty:   '-',
		CharFill:    '=',
		PendingMode: bar.PendingHide,
		Width:       20,
	}
	b := Bar("test", 100, bar.WithConfig(custom))
	assert.Equal(t, custom, b.BarConfig)
}

func TestBarPendingModeOption(t *testing.T) {
	b := Bar("test", 100, bar.WithPendingMode(bar.PendingHide))
	assert.Equal(t, bar.PendingHide, b.BarConfig.PendingMode)
}

func TestBarUpdateIntervalOption(t *testing.T) {
	b := Bar("test", 100, bar.WithUpdateInterval(time.Second))
	assert.Equal(t, time.Second, b.BarConfig.UpdateInterval)

	b = Bar("test", 100, bar.WithUpdateInterval(-time.Second))
	assert.Zero(t, b.BarConfig.UpdateInterval)
}

func TestBarSmoothingModeOption(t *testing.T) {
	b := Bar("test", 100, bar.WithSmoothingMode(bar.SmoothNone))
	assert.Equal(t, bar.SmoothNone, b.BarConfig.Smoothing)

	b = Bar("test", 100)
	assert.Equal(t, bar.SmoothEase, b.BarConfig.Smoothing)
}

func TestBarNonTTYStripsDynamicFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))
	formats := DefaultFieldFormats()
	formats.ElapsedMinimum = 0
	logger.SetFieldFormats(formats)

	_ = logger.Bar("downloading", 100).
		Str("file", "release.tar.gz").
		BarPercent("progress").
		Elapsed("elapsed").
		Progress(context.Background(), func(_ context.Context, p *fx.Update) error {
			p.SetProgress(50).Send()
			return nil
		}).
		Silent()

	out := buf.String()
	assert.Equal(t, "INF ⏳ downloading file=release.tar.gz\n", out)
}

func TestBarConfigWidgetRight(_ *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	custom := func(s bar.State) string {
		return fmt.Sprintf("%d/%d", s.Current, s.Total)
	}

	_ = logger.Bar("testing", 100, bar.WithConfig(bar.Config{
		CapLeft:     "[",
		CapRight:    "]",
		CharEmpty:   '-',
		CharFill:    '=',
		Separator:   " ",
		Width:       10,
		WidgetRight: custom,
		WidthMax:    40,
		WidthMin:    10,
	})).
		Progress(context.Background(), func(_ context.Context, p *fx.Update) error {
			p.SetProgress(50).Send()
			return nil
		}).
		Silent()

	// Non-TTY: just verifying it doesn't panic and compiles.
	// The widget is rendered in TTY mode only.
}

func TestBarConfigWidgetLeft(_ *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	custom := func(s bar.State) string {
		return fmt.Sprintf("%d%%", s.Current*100/max(s.Total, 1))
	}

	_ = logger.Bar("testing", 100, bar.WithConfig(bar.Config{
		CapLeft:     "[",
		CapRight:    "]",
		CharEmpty:   '-',
		CharFill:    '=',
		Separator:   " ",
		Width:       10,
		WidgetLeft:  custom,
		WidgetRight: widget.None(),
		WidthMax:    40,
		WidthMin:    10,
	})).
		Progress(context.Background(), func(_ context.Context, p *fx.Update) error {
			p.SetProgress(50).Send()
			return nil
		}).
		Silent()
}

func TestAddTotal(t *testing.T) {
	var tAtom atomic.Int64
	tAtom.Store(100)

	u := &fx.Update{TotalPtr: &tAtom}
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
	u := &fx.Update{}
	u.InitSelf(u)

	assert.NotPanics(t, func() {
		u.AddTotal(50)
	})
}
