package clog

import (
	"context"
	"fmt"
	"io"
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
	assert.Equal(t, fx.AnimationBar, b.AnimationMode())
	current, total, ok := b.BarProgress()
	require.True(t, ok)
	assert.Equal(t, int64(100), total)
	assert.Equal(t, int64(0), current)
}

func TestBarSpinner(t *testing.T) {
	b := Bar("test", 100).Spinner()

	assert.Equal(t, fx.AnimationBar, b.AnimationMode())
	assert.True(t, b.UsesAnimatedSymbol())
}

func TestBarSpinnerWithOptions(t *testing.T) {
	b := Bar("test", 100).Spinner(spinner.WithConfig(spinner.Dots))

	assert.Equal(t, fx.AnimationBar, b.AnimationMode())
	assert.True(t, b.UsesAnimatedSymbol())
	assert.Equal(t, spinner.Dots.Interval, b.SpinnerStyle().Interval)
}

func TestBarBuilderTotalClamp(t *testing.T) {
	// total <= 0 clamped to 1
	b := Bar("test", 0)
	_, total, ok := b.BarProgress()
	require.True(t, ok)
	assert.Equal(t, int64(1), total)

	b2 := Bar("test", -5)
	_, total, ok = b2.BarProgress()
	require.True(t, ok)
	assert.Equal(t, int64(1), total)
}

func TestBarProgressSharedWithUpdate(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()
	SetDefault(NewWriter(io.Discard))

	var capturedProgress int
	_ = Bar("Downloading", 100).
		After(10*time.Millisecond). // suppress animation display
		Progress(context.Background(), func(_ context.Context, p *fx.Update) error {
			p.SetProgress(75)
			capturedProgress = p.Progress()
			return nil
		}).
		Silent()

	assert.Equal(t, 75, capturedProgress)
}

func TestBarWait(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()
	SetDefault(NewWriter(io.Discard))

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
	assert.Equal(t, custom, b.BarStyle())
}

func TestBarPendingModeOption(t *testing.T) {
	b := Bar("test", 100, bar.WithPendingMode(bar.PendingHide))
	assert.Equal(t, bar.PendingHide, b.BarStyle().PendingMode)
}

func TestBarUpdateIntervalOption(t *testing.T) {
	b := Bar("test", 100, bar.WithUpdateInterval(time.Second))
	assert.Equal(t, time.Second, b.BarStyle().UpdateInterval)

	b = Bar("test", 100, bar.WithUpdateInterval(-time.Second))
	assert.Zero(t, b.BarStyle().UpdateInterval)
}

func TestBarTruncationMarkerOption(t *testing.T) {
	b := Bar("test", 100, bar.WithTruncationMarker("..."))

	require.NotNil(t, b.BarStyle().TruncationMarker)
	assert.Equal(t, "...", *b.BarStyle().TruncationMarker)
	assert.Equal(t, "...", bar.ResolveTruncationMarker(b.BarStyle()))
}

func TestBarSmoothingModeOption(t *testing.T) {
	b := Bar("test", 100, bar.WithSmoothingMode(bar.SmoothNone))
	assert.Equal(t, bar.SmoothNone, b.BarStyle().Smoothing)

	b = Bar("test", 100)
	assert.Equal(t, bar.SmoothEase, b.BarStyle().Smoothing)
}

func TestBarNonTTYStripsDynamicFields(t *testing.T) {
	logger, buf := newTestLogger()
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
	logger, _ := newTestLogger()

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
	logger, _ := newTestLogger()

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

func TestAddTotalNilNoOp(t *testing.T) {
	u := &fx.Update{}
	u.InitSelf(u)

	assert.NotPanics(t, func() {
		u.AddTotal(50)
	})
}
