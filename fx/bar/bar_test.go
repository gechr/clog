package bar_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/fx/bar/widget"
	"github.com/gechr/clog/style"
	"github.com/stretchr/testify/assert"
)

func TestRenderBarThinDefault(t *testing.T) {
	s := bar.DefaultStyle()
	s.Width = 10
	s.StyleFill = nil
	s.StyleEmpty = nil
	s.ProgressGradient = nil

	s.HalfFilled = '╸'
	s.HalfEmpty = '╺'

	// 50%: 10 half-cells, even -> trail char
	assert.Equal(t, "━━━━━╺━━━━", bar.Render(5, 10, s, 0))
	// 0%: all empty
	assert.Equal(t, "━━━━━━━━━━", bar.Render(0, 10, s, 0))
	// 100%: all filled
	assert.Equal(t, "━━━━━━━━━━", bar.Render(10, 10, s, 0))
	// 45%: 9 half-cells, odd -> head char
	assert.Equal(t, "━━━━╸━━━━━", bar.Render(9, 20, s, 0))
}

func TestRenderBarBlock(t *testing.T) {
	s := bar.Block
	s.Width = 10
	s.CapStyle = nil

	assert.Equal(t, "│█████░░░░░│", bar.Render(5, 10, s, 0))
	assert.Equal(t, "│░░░░░░░░░░│", bar.Render(0, 10, s, 0))
	assert.Equal(t, "│██████████│", bar.Render(10, 10, s, 0))
}

func TestRenderBarSmooth(t *testing.T) {
	s := bar.Smooth
	s.Width = 10
	s.StyleEmpty = nil

	assert.Equal(t, "██████████", bar.Render(5, 10, s, 0))
	assert.Equal(t, "██████████", bar.Render(0, 10, s, 0))
	assert.Equal(t, "██████████", bar.Render(10, 10, s, 0))
}

func TestRenderBarBasic(t *testing.T) {
	s := bar.Basic
	s.Width = 10
	s.CapStyle = nil

	// CharHead '>' at leading edge
	assert.Equal(t, "[====>     ]", bar.Render(5, 10, s, 0))
	assert.Equal(t, "[          ]", bar.Render(0, 10, s, 0))
	assert.Equal(t, "[==========]", bar.Render(10, 10, s, 0))
}

func TestRenderBarGradient(t *testing.T) {
	s := bar.Gradient
	s.Width = 10
	s.CapStyle = nil

	// 0%: all empty
	assert.Equal(t, "│          │", bar.Render(0, 100, s, 0))
	// 100%: all filled
	assert.Equal(t, "│██████████│", bar.Render(100, 100, s, 0))
	// 50%: 5 full cells, no remainder
	assert.Equal(t, "│█████     │", bar.Render(50, 100, s, 0))
	// 25%: 2 full cells + remainder 4 of 8 -> half (index 3)
	assert.Equal(t, "│██▌       │", bar.Render(25, 100, s, 0))
	// 1/80 of 10 cells = 1 sub-unit -> first gradient char (index 0)
	assert.Equal(t, "│▏         │", bar.Render(1, 80, s, 0))
}

func TestRenderBarGradientCustom(t *testing.T) {
	// 4x resolution gradient (3 chars + full = 4 sub-units per cell).
	s := bar.Style{
		CapLeft:      "[",
		CapRight:     "]",
		CharEmpty:    ' ',
		CharFill:     '█',
		GradientFill: []rune{'░', '▒', '▓'},
		Width:        8,
	}

	// 0%
	assert.Equal(t, "[        ]", bar.Render(0, 100, s, 0))
	// 100%
	assert.Equal(t, "[████████]", bar.Render(100, 100, s, 0))
	// 50%: 8*4*50/100 = 16 parts -> 16/4 = 4 full, 0 remainder
	assert.Equal(t, "[████    ]", bar.Render(50, 100, s, 0))
	// 1/32 of 8 cells = 1 sub-unit -> first char (index 0)
	assert.Equal(t, "[░       ]", bar.Render(1, 32, s, 0))
	// 2/32 = 2 sub-units -> second char (index 1)
	assert.Equal(t, "[▒       ]", bar.Render(2, 32, s, 0))
	// 3/32 = 3 sub-units -> third char (index 2)
	assert.Equal(t, "[▓       ]", bar.Render(3, 32, s, 0))
	// 4/32 = 4 sub-units -> 1 full cell
	assert.Equal(t, "[█       ]", bar.Render(4, 32, s, 0))
}

func TestRenderBarGradientOverridesHalfFilled(t *testing.T) {
	// When both GradientFill and HalfFilled are set, GradientFill wins.
	s := bar.Style{
		CapLeft:      "[",
		CapRight:     "]",
		CharEmpty:    ' ',
		CharFill:     '█',
		GradientFill: []rune{'▏', '▎', '▍', '▌', '▋', '▊', '▉'},
		HalfFilled:   '▌',
		Width:        10,
	}

	// 1/80 = 1 sub-unit -> should use gradient first char, not HalfFilled
	assert.Equal(t, "[▏         ]", bar.Render(1, 80, s, 0))
}

func TestRenderBarEdgeCases(t *testing.T) {
	s := bar.DefaultStyle()
	s.Width = 10
	s.StyleFill = nil
	s.StyleEmpty = nil
	s.ProgressGradient = nil

	// total <= 0 treated as 1 (so 0/1 = 0%)
	assert.Equal(t, "━━━━━━━━━━", bar.Render(0, 0, s, 0))

	// clamp over 100%
	assert.Equal(t, "━━━━━━━━━━", bar.Render(20, 10, s, 0))

	// clamp negative current
	assert.Equal(t, "━━━━━━━━━━", bar.Render(-5, 10, s, 0))
}

func TestRenderBarCustomChars(t *testing.T) {
	s := bar.Style{
		CapLeft:   "(",
		CapRight:  ")",
		CharEmpty: '-',
		CharFill:  '=',
		Width:     4,
	}

	assert.Equal(t, "(==--)", bar.Render(2, 4, s, 0))
}

func TestRenderBarCharHead(t *testing.T) {
	// CharHead is only used when HalfFilled is 0.
	s := bar.Block
	s.Width = 10
	s.CapStyle = nil
	s.CharHead = '>'

	// at 50%: 5 filled, head takes one slot -> 4 filled + head + 5 empty
	assert.Equal(t, "│████>░░░░░│", bar.Render(5, 10, s, 0))

	// at 0%: no head when filled == 0
	assert.Equal(t, "│░░░░░░░░░░│", bar.Render(0, 10, s, 0))

	// at 100%: no head when filled == innerWidth
	assert.Equal(t, "│██████████│", bar.Render(10, 10, s, 0))
}

func TestRenderBarAutoWidth(t *testing.T) {
	s := bar.DefaultStyle()
	s.StyleFill = nil
	s.StyleEmpty = nil
	s.ProgressGradient = nil
	s.HalfFilled = '╸'
	s.HalfEmpty = '╺'
	// WidthMin=10, WidthMax=40; termWidth=80 -> 80/4=20, clamped to [10,40] -> 20
	result := bar.Render(10, 20, s, 80)
	// 20 inner cells, 10/20 = 50%: 20 half-cells, even -> trail char
	assert.Equal(t, "━━━━━━━━━━╺━━━━━━━━━", result)
}

func TestRenderBarAutoWidthClampMin(t *testing.T) {
	s := bar.DefaultStyle()
	s.StyleFill = nil
	s.StyleEmpty = nil
	s.ProgressGradient = nil
	s.HalfFilled = '╸'
	s.HalfEmpty = '╺'
	// termWidth=0 -> fallback to WidthMin=10
	result := bar.Render(5, 10, s, 0)
	assert.Equal(t, "━━━━━╺━━━━", result)
}

func TestRenderBarWithCaps(t *testing.T) {
	s := bar.DefaultStyle()
	s.Width = 10
	s.StyleFill = nil
	s.StyleEmpty = nil
	s.ProgressGradient = nil
	s.HalfFilled = '╸'
	s.HalfEmpty = '╺'
	s.CapLeft = "["
	s.CapRight = "]"

	assert.Equal(t, "[━━━━━╺━━━━]", bar.Render(5, 10, s, 0))
}

func TestBarPercent(t *testing.T) {
	assert.Equal(t, "0%", bar.FormatPercent(0, 100, 0, false))
	assert.Equal(t, "50%", bar.FormatPercent(50, 100, 0, false))
	assert.Equal(t, "100%", bar.FormatPercent(100, 100, 0, false))
	assert.Equal(t, "0%", bar.FormatPercent(0, 0, 0, false)) // total=0 edge case
	assert.Equal(t, "100%", bar.FormatPercent(200, 100, 0, false))
}

func TestBarPercentPadded(t *testing.T) {
	assert.Equal(t, "  0%", bar.FormatPercent(0, 100, 0, true))
	assert.Equal(t, "  1%", bar.FormatPercent(1, 100, 0, true))
	assert.Equal(t, " 50%", bar.FormatPercent(50, 100, 0, true))
	assert.Equal(t, "100%", bar.FormatPercent(100, 100, 0, true)) // full width
	assert.Equal(t, "  0%", bar.FormatPercent(0, 0, 0, true))
	assert.Equal(t, "100%", bar.FormatPercent(200, 100, 0, true)) // clamped, full width
}

func TestBarPercentDigits(t *testing.T) {
	// Trailing zeros are stripped: "0.0%" -> "0%", "50.0%" -> "50%".
	assert.Equal(t, "0%", bar.FormatPercent(0, 100, 1, false))
	assert.Equal(t, "50%", bar.FormatPercent(50, 100, 1, false))
	assert.Equal(t, "100%", bar.FormatPercent(100, 100, 1, false))
	assert.Equal(t, "33.3%", bar.FormatPercent(1, 3, 1, false))
	assert.Equal(t, "33.33%", bar.FormatPercent(1, 3, 2, false))
}

func TestBarPercentDigitsPadded(t *testing.T) {
	// Padded to unstripped "100.0%" width (6 chars).
	assert.Equal(t, "    0%", bar.FormatPercent(0, 100, 1, true))
	assert.Equal(t, "    1%", bar.FormatPercent(1, 100, 1, true))
	assert.Equal(t, "   50%", bar.FormatPercent(50, 100, 1, true))
	assert.Equal(t, "  100%", bar.FormatPercent(100, 100, 1, true))
	assert.Equal(t, " 33.3%", bar.FormatPercent(1, 3, 1, true)) // full width
	assert.Equal(t, " 66.7%", bar.FormatPercent(2, 3, 1, true)) // full width
}

func TestBarDefaultStyle(t *testing.T) {
	s := bar.DefaultStyle()
	// Function fields can't be compared with DeepEqual, so verify key structural fields.
	assert.Equal(t, bar.Thin.CharFill, s.CharFill)
	assert.Equal(t, bar.Thin.CharEmpty, s.CharEmpty)
	assert.Equal(t, bar.Thin.HalfFilled, s.HalfFilled)
	assert.Equal(t, bar.Thin.HalfEmpty, s.HalfEmpty)
	assert.Equal(t, bar.Thin.CapLeft, s.CapLeft)
	assert.Equal(t, bar.Thin.CapRight, s.CapRight)
	assert.Equal(t, bar.Thin.Separator, s.Separator)
	assert.Equal(t, bar.Thin.WidthMin, s.WidthMin)
	assert.Equal(t, bar.Thin.WidthMax, s.WidthMax)
	// Presets no longer set WidgetRight; the bar renderer falls back to
	// padded percent when both widgets are nil.
	assert.Nil(
		t,
		s.WidgetRight,
		"bar.DefaultStyle should leave WidgetRight nil (fallback to percent)",
	)
}

func TestBarPresets(t *testing.T) {
	// Verify all presets have sensible defaults.
	for name, s := range map[string]bar.Style{
		"bar.Thin":     bar.Thin,
		"bar.Basic":    bar.Basic,
		"bar.Block":    bar.Block,
		"bar.Dash":     bar.Dash,
		"bar.Gradient": bar.Gradient,
		"bar.Smooth":   bar.Smooth,
	} {
		assert.NotZero(t, s.CharFill, "%s: CharFill", name)
		assert.NotZero(t, s.WidthMin, "%s: WidthMin", name)
		assert.NotZero(t, s.WidthMax, "%s: WidthMax", name)
		assert.NotEmpty(t, s.Separator, "%s: Separator", name)
	}
}

func TestBarPlacementZeroValue(t *testing.T) {
	// bar.PlaceRightPad must be the zero value so presets default to right-padded.
	assert.Equal(t, bar.PlaceRightPad, bar.Placement(0))
	assert.Equal(t, bar.PlaceRightPad, bar.Style{}.Placement)
	assert.Equal(t, bar.PlaceRightPad, bar.DefaultStyle().Placement)
}

func TestPendingModeZeroValue(t *testing.T) {
	assert.Equal(t, bar.PendingShow, bar.PendingMode(0))
	assert.Equal(t, bar.PendingShow, bar.Style{}.PendingMode)
}

func TestUpdateIntervalZeroValue(t *testing.T) {
	assert.Zero(t, bar.Style{}.UpdateInterval)
}

func TestSmoothingModeZeroValue(t *testing.T) {
	assert.Equal(t, bar.SmoothEase, bar.SmoothingMode(0))
	assert.Equal(t, bar.SmoothEase, bar.Style{}.Smoothing)
}

func TestShowPending(t *testing.T) {
	assert.True(t, bar.ShowPending(bar.Style{}, 0))
	assert.True(t, bar.ShowPending(bar.Style{PendingMode: bar.PendingHide}, 1))
	assert.False(t, bar.ShowPending(bar.Style{PendingMode: bar.PendingHide}, 0))
}

func TestFormatLineInline(t *testing.T) {
	// bar.PlaceInline: bar.FormatLine returns msgParts unchanged (bar already in msg).
	got := bar.FormatLine(
		"INF ⏳ Downloading [====>     ] 50%",
		"[====>     ] 50%",
		" ",
		bar.PlaceInline,
		80,
	)
	assert.Equal(t, "INF ⏳ Downloading [====>     ] 50%", got)
}

func TestFormatLineRightPad(t *testing.T) {
	msg := "INF Downloading" // 15 visible chars
	b := "[====      ] 50%"  // 16 visible chars
	tw := 50

	got := bar.FormatLine(msg, b, " ", bar.PlaceRightPad, tw)
	// gap = 50 - 15 - 16 = 19 spaces
	expected := msg + strings.Repeat(" ", 19) + b
	assert.Equal(t, expected, got)
	assert.Len(t, got, 50) // total width matches terminal
}

func TestFormatLineLeftPad(t *testing.T) {
	msg := "INF Downloading" // 15 visible chars
	b := "[====      ] 50%"  // 16 visible chars
	tw := 50

	got := bar.FormatLine(msg, b, " ", bar.PlaceLeftPad, tw)
	// gap = 50 - 16 - 15 = 19 spaces
	expected := b + strings.Repeat(" ", 19) + msg
	assert.Equal(t, expected, got)
	assert.Len(t, got, 50)
}

func TestFormatLineRightPadNarrow(t *testing.T) {
	// When terminal is too narrow, fall back to separator.
	msg := "INF Downloading"
	b := "[====      ] 50%"
	tw := 20 // narrower than msg+bar

	got := bar.FormatLine(msg, b, " ", bar.PlaceRightPad, tw)
	assert.Equal(t, msg+" "+b, got)
}

func TestFormatLineLeftPadNarrow(t *testing.T) {
	msg := "INF Downloading"
	b := "[====      ] 50%"
	tw := 20

	got := bar.FormatLine(msg, b, " ", bar.PlaceLeftPad, tw)
	assert.Equal(t, b+" "+msg, got)
}

func TestFormatLineRight(t *testing.T) {
	// bar.PlaceRight: bar after message, no padding.
	msg := "INF Downloading"
	b := "[====      ] 50%"

	got := bar.FormatLine(msg, b, " ", bar.PlaceRight, 80)
	assert.Equal(t, msg+" "+b, got)
}

func TestFormatLineLeft(t *testing.T) {
	// bar.PlaceLeft: bar before message, no padding.
	msg := "INF Downloading"
	b := "[====      ] 50%"

	got := bar.FormatLine(msg, b, " ", bar.PlaceLeft, 80)
	assert.Equal(t, b+" "+msg, got)
}

func TestFormatLineAlignedStandalone(t *testing.T) {
	// bar.PlaceAligned: falls back to bar.PlaceRight for standalone bars.
	msg := "INF Downloading"
	b := "[====      ] 50%"

	got := bar.FormatLine(msg, b, " ", bar.PlaceAligned, 80)
	assert.Equal(t, msg+" "+b, got)
}

func TestFormatLineCustomSeparator(t *testing.T) {
	// Narrow fallback uses the provided separator.
	msg := "INF Downloading"
	b := "[====      ] 50%"

	got := bar.FormatLine(msg, b, " | ", bar.PlaceRightPad, 10)
	assert.Equal(t, msg+" | "+b, got)
}

func TestFormatLineExactFit(t *testing.T) {
	// gap == 0: no padding, fall back to separator.
	msg := "AB" // 2 chars
	b := "CDE"  // 3 chars
	tw := 5     // exactly msg+bar, gap=0

	got := bar.FormatLine(msg, b, " ", bar.PlaceRightPad, tw)
	assert.Equal(t, "AB CDE", got) // separator used, total > tw
}

func TestRenderBarProgressGradient(t *testing.T) {
	gradient := bar.DefaultGradient()
	s := bar.Style{
		CapLeft:          "[",
		CapRight:         "]",
		CharEmpty:        ' ',
		CharFill:         '█',
		ProgressGradient: gradient,
		Width:            10,
	}

	// 0%: no filled cells, so no ANSI escape sequences
	result0 := bar.Render(0, 100, s, 0)
	assert.Equal(t, "[          ]", result0)

	// 50%: filled cells present, gradient should produce ANSI colored output
	result50 := bar.Render(50, 100, s, 0)
	assert.Equal(t, "[\x1b[38;2;255;255;0m█████\x1b[m     ]", result50)

	// 100%: all filled, gradient should produce ANSI colored output
	result100 := bar.Render(100, 100, s, 0)
	assert.Equal(t, "[\x1b[38;2;0;255;0m██████████\x1b[m]", result100)

	// Verify the colors differ between 10% and 90% progress.
	// At low progress the gradient is red; at high progress it's green.
	result10 := bar.Render(10, 100, s, 0)
	result90 := bar.Render(90, 100, s, 0)
	assert.NotEqual(
		t,
		result10,
		result90,
		"different progress values should produce different colors",
	)
}

func TestRenderBarWithoutProgressGradient(t *testing.T) {
	// Verify that bars without ProgressGradient remain unchanged (no ANSI).
	s := bar.Style{
		CapLeft:   "[",
		CapRight:  "]",
		CharEmpty: ' ',
		CharFill:  '█',
		Width:     10,
	}

	result := bar.Render(50, 100, s, 0)
	assert.Equal(t, "[█████     ]", result)
	assert.NotContains(
		t,
		result,
		"\x1b[",
		"bar without ProgressGradient should not contain ANSI escapes",
	)
}

func TestDefaultBarGradient(t *testing.T) {
	gradient := bar.DefaultGradient()
	assert.Equal(t, style.DefaultPercentGradient(), gradient)
	assert.Len(t, gradient, 3)
}

func TestWidgetPercent(t *testing.T) {
	w := widget.Percent()
	assert.Equal(t, "  0%", w(bar.State{Current: 0, Total: 100}))
	assert.Equal(t, "  1%", w(bar.State{Current: 1, Total: 100}))
	assert.Equal(t, " 50%", w(bar.State{Current: 50, Total: 100}))
	assert.Equal(t, "100%", w(bar.State{Current: 100, Total: 100}))

	// WithDigits(1): trailing zeros stripped, padded to "100.0%" width (6).
	w1 := widget.Percent(widget.WithDigits(1))
	assert.Equal(t, "    0%", w1(bar.State{Current: 0, Total: 100}))
	assert.Equal(t, " 33.3%", w1(bar.State{Current: 1, Total: 3})) // full width
	assert.Equal(t, " 66.7%", w1(bar.State{Current: 2, Total: 3})) // full width
	assert.Equal(t, "   50%", w1(bar.State{Current: 50, Total: 100}))
	assert.Equal(t, "  100%", w1(bar.State{Current: 100, Total: 100}))
}

func TestWidgetBytes(t *testing.T) {
	w := widget.Bytes() // default digits=3

	// MB range - significant digits with zero stripping, padded to max width.
	total := 100 * 1000 * 1000 // 100 MB
	assert.Equal(t, "    0 B / 100 MB", w(bar.State{Current: 0, Total: total}))
	assert.Equal(
		t,
		"9.52 MB / 100 MB",
		w(bar.State{Current: 9_524_000, Total: total}),
	) // full width
	assert.Equal(t, "  50 MB / 100 MB", w(bar.State{Current: 50_000_000, Total: total}))
	assert.Equal(t, "82.9 MB / 100 MB", w(bar.State{Current: 82_854_982, Total: total}))
	assert.Equal(t, " 100 MB / 100 MB", w(bar.State{Current: total, Total: total}))

	// GB range - padded to max width for "X.XX GB" (7 chars).
	totalGB := 2 * 1000 * 1000 * 1000 // 2 GB
	assert.Equal(t, "   1 GB / 2 GB", w(bar.State{Current: 1_000_000_000, Total: totalGB}))
	assert.Equal(t, "1.52 GB / 2 GB", w(bar.State{Current: 1_524_000_000, Total: totalGB}))
	assert.Equal(t, " 1.5 GB / 2 GB", w(bar.State{Current: 1_500_000_000, Total: totalGB}))

	// Width stays constant for same total.
	r1 := w(bar.State{Current: 1000, Total: totalGB})
	r2 := w(bar.State{Current: 1_500_000_000, Total: totalGB})
	assert.Len(t, r2, len(r1), "output width should be constant for same total")

	// WithDigits(1) -> coarser output.
	w1 := widget.Bytes(widget.WithDigits(1))
	assert.Equal(t, "1 GB / 2 GB", w1(bar.State{Current: 1_000_000_000, Total: totalGB}))

	// WithDigits(5) -> high precision, padded to "X.XXXX GB" width (9 chars).
	w5 := widget.Bytes(widget.WithDigits(5))
	assert.Equal(t, "      0 B / 2 GB", w5(bar.State{Current: 0, Total: totalGB}))
	assert.Equal(t, "     1 GB / 2 GB", w5(bar.State{Current: 1_000_000_000, Total: totalGB}))
	assert.Equal(t, "   1.5 GB / 2 GB", w5(bar.State{Current: 1_500_000_000, Total: totalGB}))
	assert.Equal(t, "1.5241 GB / 2 GB", w5(bar.State{Current: 1_524_120_000, Total: totalGB}))
	assert.Equal(t, "     2 GB / 2 GB", w5(bar.State{Current: totalGB, Total: totalGB}))
}

func TestWidgetIBytes(t *testing.T) {
	w := widget.IBytes() // default digits=3

	// MiB range - padded to max width for "X.XX MiB" (8 chars).
	total := 100 * 1024 * 1024 // 100 MiB
	assert.Equal(t, "     0 B / 100 MiB", w(bar.State{Current: 0, Total: total}))
	assert.Equal(t, "79.5 MiB / 100 MiB", w(bar.State{Current: 83_361_587, Total: total}))
	assert.Equal(t, "9.53 MiB / 100 MiB", w(bar.State{Current: 9_991_946, Total: total}))
	assert.Equal(t, " 100 MiB / 100 MiB", w(bar.State{Current: total, Total: total}))

	// GiB range - padded to max width for "X.XX GiB" (8 chars).
	totalGiB := 2 * 1024 * 1024 * 1024 // 2 GiB
	assert.Equal(t, "     0 B / 2 GiB", w(bar.State{Current: 0, Total: totalGiB}))
	assert.Equal(t, "   1 GiB / 2 GiB", w(bar.State{Current: 1024 * 1024 * 1024, Total: totalGiB}))
	assert.Equal(
		t,
		"1.52 GiB / 2 GiB",
		w(bar.State{Current: 1024*1024*1024 + 536*1024*1024, Total: totalGiB}),
	)
	assert.Equal(
		t,
		" 1.5 GiB / 2 GiB",
		w(bar.State{Current: 1024*1024*1024 + 512*1024*1024, Total: totalGiB}),
	)
}

func TestWidgetNone(t *testing.T) {
	assert.Empty(t, widget.None()(bar.State{Current: 50, Total: 100}))
	assert.Empty(t, widget.None()(bar.State{}))
}

func TestBarPresetsLeaveWidgetRightNil(t *testing.T) {
	// Presets leave WidgetRight nil; the bar renderer falls back to padded
	// percent when both widgets are nil and no BarPercent field is set.
	for name, s := range map[string]bar.Style{
		"bar.Thin":     bar.Thin,
		"bar.Basic":    bar.Basic,
		"bar.Block":    bar.Block,
		"bar.Dash":     bar.Dash,
		"bar.Gradient": bar.Gradient,
		"bar.Smooth":   bar.Smooth,
	} {
		assert.Nil(t, s.WidgetRight, "%s: WidgetRight should be nil (fallback percent)", name)
	}
}

func TestWidgetETA(t *testing.T) {
	t.Run("zero_rate", func(t *testing.T) {
		w := widget.ETA()
		assert.Equal(t, "ETA \u221e", w(bar.State{Current: 0, Total: 100, Rate: 0}))
	})

	t.Run("complete", func(t *testing.T) {
		w := widget.ETA()
		assert.Empty(t, w(bar.State{Current: 100, Total: 100, Rate: 10}))
	})

	t.Run("seconds", func(t *testing.T) {
		w := widget.ETA()
		assert.Equal(t, "ETA 5s", w(bar.State{Current: 50, Total: 100, Rate: 10}))
	})

	t.Run("minutes", func(t *testing.T) {
		w := widget.ETA()
		assert.Equal(t, "ETA 1m30s", w(bar.State{Current: 100, Total: 1000, Rate: 10}))
	})

	t.Run("hours", func(t *testing.T) {
		w := widget.ETA()
		assert.Equal(t, "ETA 1h", w(bar.State{Current: 0, Total: 36000, Rate: 10}))
	})

	t.Run("padding_stable_as_eta_shrinks", func(t *testing.T) {
		w := widget.ETA()
		// First call sets the high-water mark width.
		first := w(bar.State{Current: 100, Total: 1000, Rate: 10}) // "ETA 1m30s" -> 9 chars
		// Later call with shorter ETA pads to same width.
		second := w(bar.State{Current: 950, Total: 1000, Rate: 10}) // "ETA 5s" padded
		assert.Len(t, second, len(first), "width should stay stable as ETA shrinks")
		assert.Equal(t, "ETA 1m30s", first)
		assert.Equal(t, "   ETA 5s", second)
	})

	t.Run("zero_rate_pads_to_high_water_mark", func(t *testing.T) {
		w := widget.ETA()
		first := w(bar.State{Current: 100, Total: 1000, Rate: 10}) // "ETA 1m30s"
		noRate := w(bar.State{Current: 100, Total: 1000, Rate: 0}) // "ETA inf" padded
		assert.Len(t, noRate, len(first))
	})
}

func TestWidgetRate(t *testing.T) {
	// Fresh widget per case to avoid high-water mark accumulation.
	assert.Equal(t, "0/s", widget.Rate()(bar.State{Rate: 0}))
	assert.Equal(t, "1/s", widget.Rate()(bar.State{Rate: 1}))
	assert.Equal(t, "150/s", widget.Rate()(bar.State{Rate: 150}))
	assert.Equal(t, "1.5k/s", widget.Rate()(bar.State{Rate: 1500}))
	assert.Equal(t, "2M/s", widget.Rate()(bar.State{Rate: 2_000_000}))
	assert.Equal(t, "0.5/s", widget.Rate()(bar.State{Rate: 0.5}))

	t.Run("padding_stable", func(t *testing.T) {
		w := widget.Rate()
		wide := w(bar.State{Rate: 1500})  // "1.5k/s" -> 6 chars
		narrow := w(bar.State{Rate: 150}) // "150/s" padded to 6
		assert.Len(t, narrow, len(wide))
		assert.Equal(t, "1.5k/s", wide)
		assert.Equal(t, " 150/s", narrow)
	})
}

func TestWidgetRateWithUnit(t *testing.T) {
	assert.Equal(t, "0 ops/s", widget.Rate(widget.WithUnit("ops"))(bar.State{Rate: 0}))
	assert.Equal(t, "150 ops/s", widget.Rate(widget.WithUnit("ops"))(bar.State{Rate: 150}))
	assert.Equal(t, "1.5k ops/s", widget.Rate(widget.WithUnit("ops"))(bar.State{Rate: 1500}))
}

func TestWidgetBytesRate(t *testing.T) {
	assert.Equal(t, "0 B/s", widget.BytesRate()(bar.State{Rate: 0}))
	assert.Equal(t, "100 MB/s", widget.BytesRate()(bar.State{Rate: 100_000_000}))
	assert.Equal(t, "1.5 GB/s", widget.BytesRate()(bar.State{Rate: 1_500_000_000}))

	t.Run("padding_stable", func(t *testing.T) {
		w := widget.BytesRate()
		wide := w(bar.State{Rate: 58_300_000})   // "58.3 MB/s" -> 9 chars
		narrow := w(bar.State{Rate: 59_000_000}) // "59 MB/s" padded to 9
		assert.Len(t, narrow, len(wide))
	})
}

func TestWidgetIBytesRate(t *testing.T) {
	assert.Equal(t, "0 B/s", widget.IBytesRate()(bar.State{Rate: 0}))
	assert.Equal(t, "100 MiB/s", widget.IBytesRate()(bar.State{Rate: 100 * 1024 * 1024}))
	assert.Equal(t, "1.5 GiB/s", widget.IBytesRate()(bar.State{Rate: 1.5 * 1024 * 1024 * 1024}))

	t.Run("padding_stable", func(t *testing.T) {
		w := widget.IBytesRate()
		wide := w(bar.State{Rate: 55.6 * 1024 * 1024}) // "55.6 MiB/s" -> 10 chars
		narrow := w(bar.State{Rate: 56 * 1024 * 1024}) // "56 MiB/s" padded to 10
		assert.Len(t, narrow, len(wide))
	})
}

func TestWidgets(t *testing.T) {
	a := func(bar.State) string { return "AAA" }
	b := func(bar.State) string { return "BBB" }
	w := widget.Widgets(a, b)
	assert.Equal(t, "AAA BBB", w(bar.State{}))
}

func TestWidgetsSkipsEmpty(t *testing.T) {
	a := func(bar.State) string { return "AAA" }
	empty := func(bar.State) string { return "" }
	b := func(bar.State) string { return "BBB" }
	w := widget.Widgets(a, empty, b)
	assert.Equal(t, "AAA BBB", w(bar.State{}))
}

func TestWidgetsSingle(t *testing.T) {
	a := func(bar.State) string { return "AAA" }
	w := widget.Widgets(a)
	assert.Equal(t, "AAA", w(bar.State{}))
}

func TestWidgetsEmpty(t *testing.T) {
	w := widget.Widgets()
	assert.Empty(t, w(bar.State{}))
}

func TestWidgetSeparator(t *testing.T) {
	w := widget.Separator("│")
	assert.Equal(t, "│", w(bar.State{}))
	assert.Equal(t, "│", w(bar.State{Current: 50, Total: 100}))
}

func TestWidgetSeparatorWithStyle(t *testing.T) {
	st := new(lipgloss.NewStyle().Faint(true))
	w := widget.Separator("│", widget.WithStyle(st))
	assert.Equal(t, st.Render("│"), w(bar.State{}))
	assert.Equal(t, st.Render("│"), w(bar.State{Current: 50, Total: 100}))
}

// TestWithStylePaddingIsPlain verifies that WithStyle styles the content string
// only - leading alignment spaces must be plain so background colors don't bleed.
func TestWithStylePaddingIsPlain(t *testing.T) {
	st := new(lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("1")))

	t.Run("WidgetPercent", func(t *testing.T) {
		w := widget.Percent(widget.WithStyle(st))
		// At 0%, padded to "100%" width (4 chars): leading "  " should be plain spaces.
		result := w(bar.State{Current: 0, Total: 100})
		assert.Equal(t, "  "+st.Render("0%"), result)
	})

	t.Run("WidgetETA", func(t *testing.T) {
		w := widget.ETA(widget.WithStyle(st))
		wide := w(
			bar.State{Current: 0, Total: 100, Rate: 1},
		) // "ETA 1m40s" - sets high-water mark
		narrow := w(bar.State{Current: 90, Total: 100, Rate: 1}) // "ETA 10s" - padded
		// The wide value has no leading spaces (it IS the max width).
		assert.Equal(t, st.Render("ETA 1m40s"), wide)
		// The narrow value should have plain leading spaces, then styled content.
		assert.Equal(t, "  "+st.Render("ETA 10s"), narrow)
	})

	t.Run("WidgetRate", func(t *testing.T) {
		w := widget.Rate(widget.WithStyle(st))
		wide := w(bar.State{Rate: 1500})  // "1.5k/s" - sets high-water mark
		narrow := w(bar.State{Rate: 150}) // "150/s" padded
		assert.Equal(t, st.Render("1.5k/s"), wide)
		assert.Equal(t, " "+st.Render("150/s"), narrow)
	})

	t.Run("WidgetBytes", func(t *testing.T) {
		w := widget.Bytes(widget.WithStyle(st))
		total := 100 * 1000 * 1000
		// First call sets width; "82.9 MB" is the widest current format.
		_ = w(bar.State{Current: 82_854_982, Total: total})
		// "50 MB" is shorter -> leading spaces, then styled "50 MB / 100 MB".
		result := w(bar.State{Current: 50_000_000, Total: total})
		assert.Equal(t, "  "+st.Render("50 MB / 100 MB"), result)
	})
}

func TestWidgetsWithSeparator(t *testing.T) {
	w := widget.Widgets(widget.ETA(), widget.Separator("│"), widget.Rate())
	result := w(bar.State{Current: 50, Total: 100, Rate: 10})
	assert.Equal(t, "ETA 5s \u2502 10/s", result)
}

func TestBarStateRate(t *testing.T) {
	// Rate is set to 0 when elapsed or current is 0.
	state := bar.State{Current: 0, Total: 100, Elapsed: time.Second}
	assert.InDelta(t, 0, state.Rate, 0.001)

	// Non-zero current and elapsed -> rate populated externally.
	state = bar.State{Current: 50, Total: 100, Elapsed: 5 * time.Second, Rate: 10}
	assert.InDelta(t, 10, state.Rate, 0.001)
}

func TestFormatRate(t *testing.T) {
	tests := []struct {
		rate float64
		unit string
		want string
	}{
		{0, "", "0/s"},
		{1, "", "1/s"},
		{150, "", "150/s"},
		{1500, "", "1.5k/s"},
		{2000, "", "2k/s"},
		{1_500_000, "", "1.5M/s"},
		{0.5, "", "0.5/s"},
		{150, "ops", "150 ops/s"},
		{1500, "files", "1.5k files/s"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("rate=%.1f_unit=%s", tt.rate, tt.unit), func(t *testing.T) {
			assert.Equal(t, tt.want, widget.FormatRate(tt.rate, tt.unit))
		})
	}
}
