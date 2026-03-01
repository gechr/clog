package widget_test

import (
	"fmt"
	"testing"

	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/fx/bar/widget"
	"github.com/stretchr/testify/assert"
)

// TestWidgetPercent verifies basic percentage formatting and padding.
func TestWidgetPercent(t *testing.T) {
	tests := []struct {
		name    string
		current int
		total   int
		want    string
	}{
		{"zero", 0, 100, "  0%"},
		{"one", 1, 100, "  1%"},
		{"fifty", 50, 100, " 50%"},
		{"hundred", 100, 100, "100%"},
	}

	w := widget.Percent()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w(bar.State{Current: tt.current, Total: tt.total})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWidgetPercentWithDigits(t *testing.T) {
	// WithDigits(1): trailing zeros stripped, padded to "100.0%" width (6).
	w := widget.Percent(widget.WithDigits(1))
	tests := []struct {
		name    string
		current int
		total   int
		want    string
	}{
		{"zero", 0, 100, "    0%"},
		{"fifty", 50, 100, "   50%"},
		{"hundred", 100, 100, "  100%"},
		{"third", 1, 3, " 33.3%"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w(bar.State{Current: tt.current, Total: tt.total})
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestWidgetBytes verifies SI byte progress formatting.
func TestWidgetBytes(t *testing.T) {
	w := widget.Bytes()
	total := 100 * 1000 * 1000 // 100 MB

	tests := []struct {
		name    string
		current int
		want    string
	}{
		{"zero", 0, "    0 B / 100 MB"},
		{"fifty_mb", 50_000_000, "  50 MB / 100 MB"},
		{"total", total, " 100 MB / 100 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w(bar.State{Current: tt.current, Total: total})
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestWidgetIBytes verifies IEC byte progress formatting.
func TestWidgetIBytes(t *testing.T) {
	w := widget.IBytes()
	total := 100 * 1024 * 1024 // 100 MiB

	tests := []struct {
		name    string
		current int
		want    string
	}{
		{"zero", 0, "     0 B / 100 MiB"},
		{"total", total, " 100 MiB / 100 MiB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w(bar.State{Current: tt.current, Total: total})
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestWidgetRate verifies rate formatting without unit.
func TestWidgetRate(t *testing.T) {
	tests := []struct {
		rate float64
		want string
	}{
		{0, "0/s"},
		{150, "150/s"},
		{1500, "1.5k/s"},
		{2_000_000, "2M/s"},
		{0.5, "0.5/s"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("rate=%.1f", tt.rate), func(t *testing.T) {
			// Fresh widget per test to avoid high-water mark accumulation.
			w := widget.Rate()
			got := w(bar.State{Rate: tt.rate})
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestWidgetBytesRate verifies SI byte-rate formatting.
func TestWidgetBytesRate(t *testing.T) {
	tests := []struct {
		name string
		rate float64
		want string
	}{
		{"zero", 0, "0 B/s"},
		{"hundred_mb", 100_000_000, "100 MB/s"},
		{"one_point_five_gb", 1_500_000_000, "1.5 GB/s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := widget.BytesRate()
			got := w(bar.State{Rate: tt.rate})
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestWidgetIBytesRate verifies IEC byte-rate formatting.
func TestWidgetIBytesRate(t *testing.T) {
	tests := []struct {
		name string
		rate float64
		want string
	}{
		{"zero", 0, "0 B/s"},
		{"hundred_mib", float64(100 * 1024 * 1024), "100 MiB/s"},
		{"one_point_five_gib", 1.5 * 1024 * 1024 * 1024, "1.5 GiB/s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := widget.IBytesRate()
			got := w(bar.State{Rate: tt.rate})
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestFormatRate verifies the standalone FormatRate helper.
func TestFormatRate(t *testing.T) {
	tests := []struct {
		rate float64
		unit string
		want string
	}{
		{0, "", "0/s"},
		{150, "", "150/s"},
		{1500, "", "1.5k/s"},
		{2_000_000, "", "2M/s"},
		{0.5, "", "0.5/s"},
		{150, "ops", "150 ops/s"},
		{1500, "files", "1.5k files/s"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("rate=%.1f_unit=%q", tt.rate, tt.unit), func(t *testing.T) {
			got := widget.FormatRate(tt.rate, tt.unit)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestWidgetETA verifies ETA display logic.
func TestWidgetETA(t *testing.T) {
	t.Run("complete_returns_empty", func(t *testing.T) {
		w := widget.ETA()
		got := w(bar.State{Current: 100, Total: 100, Rate: 10})
		assert.Empty(t, got)
	})

	t.Run("zero_rate_returns_infinity", func(t *testing.T) {
		w := widget.ETA()
		got := w(bar.State{Current: 0, Total: 100, Rate: 0})
		assert.Equal(t, "ETA \u221e", got)
	})

	t.Run("ten_seconds_remaining", func(t *testing.T) {
		w := widget.ETA()
		// rate=10, remaining=100 -> 10s
		got := w(bar.State{Current: 0, Total: 100, Rate: 10})
		assert.Equal(t, "ETA 10s", got)
	})

	t.Run("minutes_and_seconds", func(t *testing.T) {
		w := widget.ETA()
		got := w(bar.State{Current: 100, Total: 1000, Rate: 10})
		assert.Equal(t, "ETA 1m30s", got)
	})
}

// TestWidgetNone verifies the None widget always returns "".
func TestWidgetNone(t *testing.T) {
	w := widget.None()
	assert.Empty(t, w(bar.State{}))
	assert.Empty(t, w(bar.State{Current: 50, Total: 100}))
}

// TestWidgetSeparator verifies the Separator widget returns its string.
func TestWidgetSeparator(t *testing.T) {
	tests := []struct {
		sep string
	}{
		{"|"},
		{"│"},
		{" / "},
	}

	for _, tt := range tests {
		t.Run(tt.sep, func(t *testing.T) {
			w := widget.Separator(tt.sep)
			assert.Equal(t, tt.sep, w(bar.State{}))
			assert.Equal(t, tt.sep, w(bar.State{Current: 50, Total: 100}))
		})
	}
}

// TestWidgets verifies Widgets combines widget outputs with spaces.
func TestWidgets(t *testing.T) {
	a := func(bar.State) string { return "AAA" }
	b := func(bar.State) string { return "BBB" }
	w := widget.Widgets(a, b)
	assert.Equal(t, "AAA BBB", w(bar.State{}))
}

// TestWidgetsSkipsEmpty verifies Widgets skips widgets that return "".
func TestWidgetsSkipsEmpty(t *testing.T) {
	a := func(bar.State) string { return "AAA" }
	empty := func(bar.State) string { return "" }
	b := func(bar.State) string { return "BBB" }
	w := widget.Widgets(a, empty, b)
	assert.Equal(t, "AAA BBB", w(bar.State{}))
}

// TestWidgetsWithPercentAndETA verifies combining Percent and ETA widgets.
func TestWidgetsWithPercentAndETA(t *testing.T) {
	w := widget.Widgets(widget.Percent(), widget.ETA())
	state := bar.State{Current: 50, Total: 100, Rate: 10}
	got := w(state)
	// Percent produces " 50%", ETA produces "ETA 5s".
	assert.Equal(t, " 50% ETA 5s", got)
}

// TestWidgetRateWithUnit verifies rate formatting with a unit label.
func TestWidgetRateWithUnit(t *testing.T) {
	tests := []struct {
		name string
		rate float64
		unit string
		want string
	}{
		{"zero", 0, "ops", "0 ops/s"},
		{"small", 150, "ops", "150 ops/s"},
		{"kilo", 1500, "ops", "1.5k ops/s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := widget.Rate(widget.WithUnit(tt.unit))
			got := w(bar.State{Rate: tt.rate})
			assert.Equal(t, tt.want, got)
		})
	}
}
