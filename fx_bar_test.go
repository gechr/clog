package clog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gechr/clog/field/elapsed"
	"github.com/gechr/clog/fx"
	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/fx/bar/widget"
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

func TestBarStyleOption(t *testing.T) {
	custom := bar.Style{
		CapLeft:   "|",
		CapRight:  "|",
		CharEmpty: '-',
		CharFill:  '=',
		Width:     20,
	}
	b := Bar("test", 100, bar.WithStyle(custom))
	assert.Equal(t, custom, b.BarStyle)
}

func TestBarNonTTYStripsDynamicFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))
	elapsed.SetMinimum(0)
	t.Cleanup(func() { elapsed.SetMinimum(time.Second) })

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

func TestBarStyleWidgetRight(_ *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	custom := func(s bar.State) string {
		return fmt.Sprintf("%d/%d", s.Current, s.Total)
	}

	_ = logger.Bar("testing", 100, bar.WithStyle(bar.Style{
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

func TestBarStyleWidgetLeft(_ *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	custom := func(s bar.State) string {
		return fmt.Sprintf("%d%%", s.Current*100/max(s.Total, 1))
	}

	_ = logger.Bar("testing", 100, bar.WithStyle(bar.Style{
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
