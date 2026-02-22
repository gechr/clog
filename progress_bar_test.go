package clog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderBarThinDefault(t *testing.T) {
	style := DefaultBarStyle()
	style.Width = 10

	// 50%: 10 half-cells, even → trail char ╺
	assert.Equal(t, "[━━━━━╺────]", renderBar(5, 10, style, 0))
	// 0%: all empty
	assert.Equal(t, "[──────────]", renderBar(0, 10, style, 0))
	// 100%: all filled
	assert.Equal(t, "[━━━━━━━━━━]", renderBar(10, 10, style, 0))
	// 45%: 9 half-cells, odd → head char ╸
	assert.Equal(t, "[━━━━╸─────]", renderBar(9, 20, style, 0))
}

func TestRenderBarBlock(t *testing.T) {
	style := BarBlock
	style.Width = 10

	assert.Equal(t, "│█████░░░░░│", renderBar(5, 10, style, 0))
	assert.Equal(t, "│░░░░░░░░░░│", renderBar(0, 10, style, 0))
	assert.Equal(t, "│██████████│", renderBar(10, 10, style, 0))
}

func TestRenderBarSmooth(t *testing.T) {
	style := BarSmooth
	style.Width = 10

	// 45%: odd halves → ▌ head, no trail (HalfEmpty is 0)
	assert.Equal(t, "│████▌     │", renderBar(9, 20, style, 0))
	// 50%: even halves, no HalfEmpty → no trail
	assert.Equal(t, "│█████     │", renderBar(5, 10, style, 0))
	// 0%
	assert.Equal(t, "│          │", renderBar(0, 10, style, 0))
	// 100%
	assert.Equal(t, "│██████████│", renderBar(10, 10, style, 0))
}

func TestRenderBarBasic(t *testing.T) {
	style := BarBasic
	style.Width = 10

	// CharHead '>' at leading edge
	assert.Equal(t, "[====>     ]", renderBar(5, 10, style, 0))
	assert.Equal(t, "[          ]", renderBar(0, 10, style, 0))
	assert.Equal(t, "[==========]", renderBar(10, 10, style, 0))
}

func TestRenderBarGradient(t *testing.T) {
	style := BarGradient
	style.Width = 10

	// 0%: all empty
	assert.Equal(t, "│          │", renderBar(0, 100, style, 0))
	// 100%: all filled
	assert.Equal(t, "│██████████│", renderBar(100, 100, style, 0))
	// 50%: 5 full cells, no remainder
	assert.Equal(t, "│█████     │", renderBar(50, 100, style, 0))
	// 25%: 2 full cells + remainder 4 of 8 → '▌' (index 3)
	assert.Equal(t, "│██▌       │", renderBar(25, 100, style, 0))
	// 1/80 of 10 cells = 1 sub-unit → '▏' (index 0)
	assert.Equal(t, "│▏         │", renderBar(1, 80, style, 0))
}

func TestRenderBarGradientCustom(t *testing.T) {
	// 4x resolution gradient (3 chars + full = 4 sub-units per cell).
	style := BarStyle{
		CapLeft:      "[",
		CapRight:     "]",
		CharEmpty:    ' ',
		CharFill:     '█',
		GradientFill: []rune{'░', '▒', '▓'},
		Width:        8,
	}

	// 0%
	assert.Equal(t, "[        ]", renderBar(0, 100, style, 0))
	// 100%
	assert.Equal(t, "[████████]", renderBar(100, 100, style, 0))
	// 50%: 8*4*50/100 = 16 parts → 16/4 = 4 full, 0 remainder
	assert.Equal(t, "[████    ]", renderBar(50, 100, style, 0))
	// 1/32 of 8 cells = 1 sub-unit → '░' (index 0)
	assert.Equal(t, "[░       ]", renderBar(1, 32, style, 0))
	// 2/32 = 2 sub-units → '▒' (index 1)
	assert.Equal(t, "[▒       ]", renderBar(2, 32, style, 0))
	// 3/32 = 3 sub-units → '▓' (index 2)
	assert.Equal(t, "[▓       ]", renderBar(3, 32, style, 0))
	// 4/32 = 4 sub-units → 1 full cell
	assert.Equal(t, "[█       ]", renderBar(4, 32, style, 0))
}

func TestRenderBarGradientOverridesHalfFilled(t *testing.T) {
	// When both GradientFill and HalfFilled are set, GradientFill wins.
	style := BarStyle{
		CapLeft:      "[",
		CapRight:     "]",
		CharEmpty:    ' ',
		CharFill:     '█',
		GradientFill: []rune{'▏', '▎', '▍', '▌', '▋', '▊', '▉'},
		HalfFilled:   '▌',
		Width:        10,
	}

	// 1/80 = 1 sub-unit → should use gradient '▏', not HalfFilled '▌'
	assert.Equal(t, "[▏         ]", renderBar(1, 80, style, 0))
}

func TestRenderBarEdgeCases(t *testing.T) {
	style := DefaultBarStyle()
	style.Width = 10

	// total <= 0 treated as 1 (so 0/1 = 0%)
	assert.Equal(t, "[──────────]", renderBar(0, 0, style, 0))

	// clamp over 100%
	assert.Equal(t, "[━━━━━━━━━━]", renderBar(20, 10, style, 0))

	// clamp negative current
	assert.Equal(t, "[──────────]", renderBar(-5, 10, style, 0))
}

func TestRenderBarCustomChars(t *testing.T) {
	style := BarStyle{
		CapLeft:   "(",
		CapRight:  ")",
		CharEmpty: '-',
		CharFill:  '=',
		Width:     4,
	}

	assert.Equal(t, "(==--)", renderBar(2, 4, style, 0))
}

func TestRenderBarCharHead(t *testing.T) {
	// CharHead is only used when HalfFilled is 0.
	style := BarBlock
	style.Width = 10
	style.CharHead = '>'

	// at 50%: 5 filled, head takes one slot → 4 filled + head + 5 empty
	assert.Equal(t, "│████>░░░░░│", renderBar(5, 10, style, 0))

	// at 0%: no head when filled == 0
	assert.Equal(t, "│░░░░░░░░░░│", renderBar(0, 10, style, 0))

	// at 100%: no head when filled == innerWidth
	assert.Equal(t, "│██████████│", renderBar(10, 10, style, 0))
}

func TestRenderBarAutoWidth(t *testing.T) {
	style := DefaultBarStyle()
	// WidthMin=10, WidthMax=40; termWidth=80 → 80/4=20, clamped to [10,40] → 20
	result := renderBar(10, 20, style, 80)
	// 20 inner cells, 10/20 = 50%: 20 half-cells, even → trail char
	assert.Equal(t, "[━━━━━━━━━━╺─────────]", result)
}

func TestRenderBarAutoWidthClampMin(t *testing.T) {
	style := DefaultBarStyle()
	// termWidth=0 → fallback to WidthMin=10
	result := renderBar(5, 10, style, 0)
	assert.Equal(t, "[━━━━━╺────]", result)
}

func TestRenderBarNoCaps(t *testing.T) {
	style := DefaultBarStyle()
	style.Width = 10
	style.CapLeft = ""
	style.CapRight = ""

	assert.Equal(t, "━━━━━╺────", renderBar(5, 10, style, 0))
}

func TestBarPercent(t *testing.T) {
	assert.Equal(t, "0%", barPercent(0, 100, 0, false))
	assert.Equal(t, "50%", barPercent(50, 100, 0, false))
	assert.Equal(t, "100%", barPercent(100, 100, 0, false))
	assert.Equal(t, "0%", barPercent(0, 0, 0, false)) // total=0 edge case
	assert.Equal(t, "100%", barPercent(200, 100, 0, false))
}

func TestBarPercentPadded(t *testing.T) {
	assert.Equal(t, "  0%", barPercent(0, 100, 0, true))
	assert.Equal(t, "  1%", barPercent(1, 100, 0, true))
	assert.Equal(t, " 50%", barPercent(50, 100, 0, true))
	assert.Equal(t, "100%", barPercent(100, 100, 0, true)) // full width
	assert.Equal(t, "  0%", barPercent(0, 0, 0, true))
	assert.Equal(t, "100%", barPercent(200, 100, 0, true)) // clamped, full width
}

func TestBarPercentDigits(t *testing.T) {
	// Trailing zeros are stripped: "0.0%" → "0%", "50.0%" → "50%".
	assert.Equal(t, "0%", barPercent(0, 100, 1, false))
	assert.Equal(t, "50%", barPercent(50, 100, 1, false))
	assert.Equal(t, "100%", barPercent(100, 100, 1, false))
	assert.Equal(t, "33.3%", barPercent(1, 3, 1, false))
	assert.Equal(t, "33.33%", barPercent(1, 3, 2, false))
}

func TestBarPercentDigitsPadded(t *testing.T) {
	// Padded to unstripped "100.0%" width (6 chars).
	assert.Equal(t, "    0%", barPercent(0, 100, 1, true))
	assert.Equal(t, "    1%", barPercent(1, 100, 1, true))
	assert.Equal(t, "   50%", barPercent(50, 100, 1, true))
	assert.Equal(t, "  100%", barPercent(100, 100, 1, true))
	assert.Equal(t, " 33.3%", barPercent(1, 3, 1, true)) // full width
	assert.Equal(t, " 66.7%", barPercent(2, 3, 1, true)) // full width
}

func TestBarBuilderMode(t *testing.T) {
	b := Bar("test", 100)
	assert.Equal(t, animationBar, b.mode)
	require.NotNil(t, b.barProgressPtr)
	require.NotNil(t, b.barTotalPtr)
	assert.Equal(t, int64(100), b.barTotalPtr.Load())
	assert.Equal(t, int64(0), b.barProgressPtr.Load())
}

func TestBarBuilderTotalClamp(t *testing.T) {
	// total <= 0 clamped to 1
	b := Bar("test", 0)
	assert.Equal(t, int64(1), b.barTotalPtr.Load())

	b2 := Bar("test", -5)
	assert.Equal(t, int64(1), b2.barTotalPtr.Load())
}

func TestProgressUpdateSetProgress(t *testing.T) {
	var pAtom atomic.Int64
	var tAtom atomic.Int64
	tAtom.Store(100)

	u := &ProgressUpdate{
		progressPtr: &pAtom,
		totalPtr:    &tAtom,
	}
	u.initSelf(u)

	result := u.SetProgress(42)
	assert.Equal(t, u, result) // fluent return
	assert.Equal(t, int64(42), pAtom.Load())

	result = u.SetTotal(200)
	assert.Equal(t, u, result)
	assert.Equal(t, int64(200), tAtom.Load())
}

func TestProgressUpdateSetProgressClamp(t *testing.T) {
	var pAtom atomic.Int64
	var tAtom atomic.Int64
	tAtom.Store(100)

	u := &ProgressUpdate{progressPtr: &pAtom, totalPtr: &tAtom}
	u.initSelf(u)

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

func TestProgressUpdateSetProgressNilNoOp(t *testing.T) {
	// Non-bar ProgressUpdate has nil pointers — should be a no-op.
	u := &ProgressUpdate{}
	u.initSelf(u)

	assert.NotPanics(t, func() {
		u.SetProgress(50)
		u.SetTotal(100)
	})
}

func TestProgressUpdateSetTotalClamp(t *testing.T) {
	var pAtom atomic.Int64
	var tAtom atomic.Int64
	tAtom.Store(100)

	u := &ProgressUpdate{progressPtr: &pAtom, totalPtr: &tAtom}
	u.initSelf(u)

	u.SetTotal(0)
	assert.Equal(t, int64(1), tAtom.Load())

	u.SetTotal(-10)
	assert.Equal(t, int64(1), tAtom.Load())
}

func TestBarProgressSharedWithProgressUpdate(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()
	Default = NewWriter(io.Discard)

	var capturedProgress int64
	_ = Bar("Downloading", 100).
		After(10*time.Millisecond). // suppress animation display
		Progress(context.Background(), func(_ context.Context, p *ProgressUpdate) error {
			p.SetProgress(75)
			capturedProgress = p.progressPtr.Load()
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

func TestBarStyleMethod(t *testing.T) {
	custom := BarStyle{
		CapLeft:   "|",
		CapRight:  "|",
		CharEmpty: '-',
		CharFill:  '=',
		Width:     20,
	}
	b := Bar("test", 100).Style(custom)
	assert.Equal(t, custom, b.barStyle)
}

func TestBarDefaultStyle(t *testing.T) {
	s := DefaultBarStyle()
	// Function fields can't be compared with DeepEqual, so verify key structural fields.
	assert.Equal(t, BarThin.CharFill, s.CharFill)
	assert.Equal(t, BarThin.CharEmpty, s.CharEmpty)
	assert.Equal(t, BarThin.HalfFilled, s.HalfFilled)
	assert.Equal(t, BarThin.HalfEmpty, s.HalfEmpty)
	assert.Equal(t, BarThin.CapLeft, s.CapLeft)
	assert.Equal(t, BarThin.CapRight, s.CapRight)
	assert.Equal(t, BarThin.Separator, s.Separator)
	assert.Equal(t, BarThin.WidthMin, s.WidthMin)
	assert.Equal(t, BarThin.WidthMax, s.WidthMax)
	assert.NotNil(t, s.WidgetRight, "DefaultBarStyle should include a WidgetRight")
}

func TestBarPresets(t *testing.T) {
	// Verify all presets have sensible defaults.
	for name, style := range map[string]BarStyle{
		"BarThin":     BarThin,
		"BarBasic":    BarBasic,
		"BarBlock":    BarBlock,
		"BarDash":     BarDash,
		"BarGradient": BarGradient,
		"BarSmooth":   BarSmooth,
	} {
		assert.NotZero(t, style.CharFill, "%s: CharFill", name)
		assert.NotZero(t, style.WidthMin, "%s: WidthMin", name)
		assert.NotZero(t, style.WidthMax, "%s: WidthMax", name)
		assert.NotEmpty(t, style.Separator, "%s: Separator", name)
	}
}

func TestBarAlignZeroValue(t *testing.T) {
	// BarAlignRightPad must be the zero value so presets default to right-padded.
	assert.Equal(t, BarAlignRightPad, BarAlign(0))
	assert.Equal(t, BarAlignRightPad, BarStyle{}.Align)
	assert.Equal(t, BarAlignRightPad, DefaultBarStyle().Align)
}

func TestAlignBarLineInline(t *testing.T) {
	// BarAlignInline: alignBarLine returns msgParts unchanged (bar already in msg).
	got := alignBarLine(
		"INF ⏳ Downloading [====>     ] 50%",
		"[====>     ] 50%",
		" ",
		BarAlignInline,
		80,
	)
	assert.Equal(t, "INF ⏳ Downloading [====>     ] 50%", got)
}

func TestAlignBarLineRightPad(t *testing.T) {
	msg := "INF Downloading"  // 15 visible chars
	bar := "[====      ] 50%" // 16 visible chars
	tw := 50

	got := alignBarLine(msg, bar, " ", BarAlignRightPad, tw)
	// gap = 50 - 15 - 16 = 19 spaces
	expected := msg + strings.Repeat(" ", 19) + bar
	assert.Equal(t, expected, got)
	assert.Len(t, got, 50) // total width matches terminal
}

func TestAlignBarLineLeftPad(t *testing.T) {
	msg := "INF Downloading"  // 15 visible chars
	bar := "[====      ] 50%" // 16 visible chars
	tw := 50

	got := alignBarLine(msg, bar, " ", BarAlignLeftPad, tw)
	// gap = 50 - 16 - 15 = 19 spaces
	expected := bar + strings.Repeat(" ", 19) + msg
	assert.Equal(t, expected, got)
	assert.Len(t, got, 50)
}

func TestAlignBarLineRightPadNarrow(t *testing.T) {
	// When terminal is too narrow, fall back to separator.
	msg := "INF Downloading"
	bar := "[====      ] 50%"
	tw := 20 // narrower than msg+bar

	got := alignBarLine(msg, bar, " ", BarAlignRightPad, tw)
	assert.Equal(t, msg+" "+bar, got)
}

func TestAlignBarLineLeftPadNarrow(t *testing.T) {
	msg := "INF Downloading"
	bar := "[====      ] 50%"
	tw := 20

	got := alignBarLine(msg, bar, " ", BarAlignLeftPad, tw)
	assert.Equal(t, bar+" "+msg, got)
}

func TestAlignBarLineRight(t *testing.T) {
	// BarAlignRight: bar after message, no padding.
	msg := "INF Downloading"
	bar := "[====      ] 50%"

	got := alignBarLine(msg, bar, " ", BarAlignRight, 80)
	assert.Equal(t, msg+" "+bar, got)
}

func TestAlignBarLineLeft(t *testing.T) {
	// BarAlignLeft: bar before message, no padding.
	msg := "INF Downloading"
	bar := "[====      ] 50%"

	got := alignBarLine(msg, bar, " ", BarAlignLeft, 80)
	assert.Equal(t, bar+" "+msg, got)
}

func TestAlignBarLineCustomSeparator(t *testing.T) {
	// Narrow fallback uses the provided separator.
	msg := "INF Downloading"
	bar := "[====      ] 50%"

	got := alignBarLine(msg, bar, " | ", BarAlignRightPad, 10)
	assert.Equal(t, msg+" | "+bar, got)
}

func TestAlignBarLineExactFit(t *testing.T) {
	// gap == 0: no padding, fall back to separator.
	msg := "AB"  // 2 chars
	bar := "CDE" // 3 chars
	tw := 5      // exactly msg+bar, gap=0

	got := alignBarLine(msg, bar, " ", BarAlignRightPad, tw)
	assert.Equal(t, "AB CDE", got) // separator used, total > tw
}

func TestRenderBarProgressGradient(t *testing.T) {
	// Force TrueColor so lipgloss emits ANSI escapes in the test runner.
	r := lipgloss.DefaultRenderer()
	old := r.ColorProfile()
	r.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { r.SetColorProfile(old) })

	gradient := DefaultBarGradient()
	style := BarStyle{
		CapLeft:          "[",
		CapRight:         "]",
		CharEmpty:        ' ',
		CharFill:         '█',
		ProgressGradient: gradient,
		Width:            10,
	}

	// 0%: no filled cells, so no ANSI escape sequences
	result0 := renderBar(0, 100, style, 0)
	assert.Equal(t, "[          ]", result0)

	// 50%: filled cells present, gradient should produce ANSI colored output
	result50 := renderBar(50, 100, style, 0)
	assert.Contains(t, result50, "\x1b[", "50%% bar should contain ANSI escape sequences")
	assert.Contains(t, result50, "█", "50%% bar should contain filled characters")

	// 100%: all filled, gradient should produce ANSI colored output
	result100 := renderBar(100, 100, style, 0)
	assert.Contains(t, result100, "\x1b[", "100%% bar should contain ANSI escape sequences")

	// Verify the colors differ between 10% and 90% progress.
	// At low progress the gradient is red; at high progress it's green.
	result10 := renderBar(10, 100, style, 0)
	result90 := renderBar(90, 100, style, 0)
	assert.NotEqual(
		t,
		result10,
		result90,
		"different progress values should produce different colors",
	)
}

func TestRenderBarWithoutProgressGradient(t *testing.T) {
	// Verify that bars without ProgressGradient remain unchanged (no ANSI).
	style := BarStyle{
		CapLeft:   "[",
		CapRight:  "]",
		CharEmpty: ' ',
		CharFill:  '█',
		Width:     10,
	}

	result := renderBar(50, 100, style, 0)
	assert.Equal(t, "[█████     ]", result)
	assert.NotContains(
		t,
		result,
		"\x1b[",
		"bar without ProgressGradient should not contain ANSI escapes",
	)
}

func TestBarNonTTYStripsDynamicFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))
	logger.SetElapsedMinimum(0)

	_ = logger.Bar("downloading", 100).
		Str("file", "release.tar.gz").
		BarPercent("progress").
		Elapsed("elapsed").
		Progress(context.Background(), func(_ context.Context, p *ProgressUpdate) error {
			p.SetProgress(50).Send()
			return nil
		}).
		Silent()

	out := buf.String()
	assert.Contains(t, out, "file=release.tar.gz")
	assert.NotContains(t, out, "progress=")
	assert.NotContains(t, out, "elapsed=")
}

func TestDefaultBarGradient(t *testing.T) {
	gradient := DefaultBarGradient()
	assert.Equal(t, DefaultPercentGradient(), gradient)
	assert.Len(t, gradient, 3)
}

func TestWidgetPercent(t *testing.T) {
	w := WidgetPercent()
	assert.Equal(t, "  0%", w(BarState{Current: 0, Total: 100}))
	assert.Equal(t, "  1%", w(BarState{Current: 1, Total: 100}))
	assert.Equal(t, " 50%", w(BarState{Current: 50, Total: 100}))
	assert.Equal(t, "100%", w(BarState{Current: 100, Total: 100}))

	// WithDigits(1): trailing zeros stripped, padded to "100.0%" width (6).
	w1 := WidgetPercent(WithDigits(1))
	assert.Equal(t, "    0%", w1(BarState{Current: 0, Total: 100}))
	assert.Equal(t, " 33.3%", w1(BarState{Current: 1, Total: 3})) // full width
	assert.Equal(t, " 66.7%", w1(BarState{Current: 2, Total: 3})) // full width
	assert.Equal(t, "   50%", w1(BarState{Current: 50, Total: 100}))
	assert.Equal(t, "  100%", w1(BarState{Current: 100, Total: 100}))
}

func TestWidgetBytes(t *testing.T) {
	w := WidgetBytes() // default digits=3

	// MB range — significant digits with zero stripping, padded to max width.
	total := 100 * 1000 * 1000 // 100 MB
	assert.Equal(t, "    0 B / 100 MB", w(BarState{Current: 0, Total: total}))
	assert.Equal(t, "9.52 MB / 100 MB", w(BarState{Current: 9_524_000, Total: total})) // full width
	assert.Equal(t, "  50 MB / 100 MB", w(BarState{Current: 50_000_000, Total: total}))
	assert.Equal(t, "82.9 MB / 100 MB", w(BarState{Current: 82_854_982, Total: total}))
	assert.Equal(t, " 100 MB / 100 MB", w(BarState{Current: total, Total: total}))

	// GB range — padded to max width for "X.XX GB" (7 chars).
	totalGB := 2 * 1000 * 1000 * 1000 // 2 GB
	assert.Equal(t, "   1 GB / 2 GB", w(BarState{Current: 1_000_000_000, Total: totalGB}))
	assert.Equal(t, "1.52 GB / 2 GB", w(BarState{Current: 1_524_000_000, Total: totalGB}))
	assert.Equal(t, " 1.5 GB / 2 GB", w(BarState{Current: 1_500_000_000, Total: totalGB}))

	// Width stays constant for same total.
	r1 := w(BarState{Current: 1000, Total: totalGB})
	r2 := w(BarState{Current: 1_500_000_000, Total: totalGB})
	assert.Len(t, r2, len(r1), "output width should be constant for same total")

	// WithDigits(1) → coarser output.
	w1 := WidgetBytes(WithDigits(1))
	assert.Equal(t, "1 GB / 2 GB", w1(BarState{Current: 1_000_000_000, Total: totalGB}))

	// WithDigits(5) → high precision, padded to "X.XXXX GB" width (9 chars).
	w5 := WidgetBytes(WithDigits(5))
	assert.Equal(t, "      0 B / 2 GB", w5(BarState{Current: 0, Total: totalGB}))
	assert.Equal(t, "     1 GB / 2 GB", w5(BarState{Current: 1_000_000_000, Total: totalGB}))
	assert.Equal(t, "   1.5 GB / 2 GB", w5(BarState{Current: 1_500_000_000, Total: totalGB}))
	assert.Equal(t, "1.5241 GB / 2 GB", w5(BarState{Current: 1_524_120_000, Total: totalGB}))
	assert.Equal(t, "     2 GB / 2 GB", w5(BarState{Current: totalGB, Total: totalGB}))
}

func TestWidgetIBytes(t *testing.T) {
	w := WidgetIBytes() // default digits=3

	// MiB range — padded to max width for "X.XX MiB" (8 chars).
	total := 100 * 1024 * 1024 // 100 MiB
	assert.Equal(t, "     0 B / 100 MiB", w(BarState{Current: 0, Total: total}))
	assert.Equal(t, "79.5 MiB / 100 MiB", w(BarState{Current: 83_361_587, Total: total}))
	assert.Equal(t, "9.53 MiB / 100 MiB", w(BarState{Current: 9_991_946, Total: total}))
	assert.Equal(t, " 100 MiB / 100 MiB", w(BarState{Current: total, Total: total}))

	// GiB range — padded to max width for "X.XX GiB" (8 chars).
	totalGiB := 2 * 1024 * 1024 * 1024 // 2 GiB
	assert.Equal(t, "     0 B / 2 GiB", w(BarState{Current: 0, Total: totalGiB}))
	assert.Equal(t, "   1 GiB / 2 GiB", w(BarState{Current: 1024 * 1024 * 1024, Total: totalGiB}))
	assert.Equal(
		t,
		"1.52 GiB / 2 GiB",
		w(BarState{Current: 1024*1024*1024 + 536*1024*1024, Total: totalGiB}),
	)
	assert.Equal(
		t,
		" 1.5 GiB / 2 GiB",
		w(BarState{Current: 1024*1024*1024 + 512*1024*1024, Total: totalGiB}),
	)
}

func TestWidgetNone(t *testing.T) {
	assert.Empty(t, WidgetNone(BarState{Current: 50, Total: 100}))
	assert.Empty(t, WidgetNone(BarState{}))
}

func TestBarStyleWidgetRight(_ *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	custom := func(s BarState) string {
		return fmt.Sprintf("%d/%d", s.Current, s.Total)
	}

	_ = logger.Bar("testing", 100).
		Style(BarStyle{
			CapLeft:     "[",
			CapRight:    "]",
			CharEmpty:   '-',
			CharFill:    '=',
			Separator:   " ",
			Width:       10,
			WidgetRight: custom,
			WidthMax:    40,
			WidthMin:    10,
		}).
		Progress(context.Background(), func(_ context.Context, p *ProgressUpdate) error {
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

	custom := func(s BarState) string {
		return fmt.Sprintf("%d%%", s.Current*100/max(s.Total, 1))
	}

	_ = logger.Bar("testing", 100).
		Style(BarStyle{
			CapLeft:     "[",
			CapRight:    "]",
			CharEmpty:   '-',
			CharFill:    '=',
			Separator:   " ",
			Width:       10,
			WidgetLeft:  custom,
			WidgetRight: WidgetNone,
			WidthMax:    40,
			WidthMin:    10,
		}).
		Progress(context.Background(), func(_ context.Context, p *ProgressUpdate) error {
			p.SetProgress(50).Send()
			return nil
		}).
		Silent()
}

func TestBarPresetsHaveWidgetRight(t *testing.T) {
	for name, style := range map[string]BarStyle{
		"BarThin":     BarThin,
		"BarBasic":    BarBasic,
		"BarBlock":    BarBlock,
		"BarDash":     BarDash,
		"BarGradient": BarGradient,
		"BarSmooth":   BarSmooth,
	} {
		assert.NotNil(t, style.WidgetRight, "%s: WidgetRight should be set", name)
	}
}
