package clog

import (
	"context"
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

	// HeadChar '>' at leading edge
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
		FilledChar:   '█',
		EmptyChar:    ' ',
		FillGradient: []rune{'░', '▒', '▓'},
		LeftCap:      "[",
		RightCap:     "]",
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
	// When both FillGradient and HalfFilled are set, FillGradient wins.
	style := BarStyle{
		FilledChar:   '█',
		EmptyChar:    ' ',
		HalfFilled:   '▌',
		FillGradient: []rune{'▏', '▎', '▍', '▌', '▋', '▊', '▉'},
		LeftCap:      "[",
		RightCap:     "]",
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
		FilledChar: '=',
		EmptyChar:  '-',
		LeftCap:    "(",
		RightCap:   ")",
		Width:      4,
	}

	assert.Equal(t, "(==--)", renderBar(2, 4, style, 0))
}

func TestRenderBarHeadChar(t *testing.T) {
	// HeadChar is only used when HalfFilled is 0.
	style := BarBlock
	style.Width = 10
	style.HeadChar = '>'

	// at 50%: 5 filled, head takes one slot → 4 filled + head + 5 empty
	assert.Equal(t, "│████>░░░░░│", renderBar(5, 10, style, 0))

	// at 0%: no head when filled == 0
	assert.Equal(t, "│░░░░░░░░░░│", renderBar(0, 10, style, 0))

	// at 100%: no head when filled == innerWidth
	assert.Equal(t, "│██████████│", renderBar(10, 10, style, 0))
}

func TestRenderBarAutoWidth(t *testing.T) {
	style := DefaultBarStyle()
	// MinWidth=10, MaxWidth=40; termWidth=80 → 80/4=20, clamped to [10,40] → 20
	result := renderBar(10, 20, style, 80)
	// 20 inner cells, 10/20 = 50%: 20 half-cells, even → trail char
	assert.Equal(t, "[━━━━━━━━━━╺─────────]", result)
}

func TestRenderBarAutoWidthClampMin(t *testing.T) {
	style := DefaultBarStyle()
	// termWidth=0 → fallback to MinWidth=10
	result := renderBar(5, 10, style, 0)
	assert.Equal(t, "[━━━━━╺────]", result)
}

func TestRenderBarNoCaps(t *testing.T) {
	style := DefaultBarStyle()
	style.Width = 10
	style.LeftCap = ""
	style.RightCap = ""

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
	assert.Equal(t, " 50%", barPercent(50, 100, 0, true))
	assert.Equal(t, "100%", barPercent(100, 100, 0, true))
	assert.Equal(t, "  0%", barPercent(0, 0, 0, true))
	assert.Equal(t, "100%", barPercent(200, 100, 0, true))
}

func TestBarPercentPrecision(t *testing.T) {
	assert.Equal(t, "0.0%", barPercent(0, 100, 1, false))
	assert.Equal(t, "50.0%", barPercent(50, 100, 1, false))
	assert.Equal(t, "100.0%", barPercent(100, 100, 1, false))
	assert.Equal(t, "33.33%", barPercent(1, 3, 2, false))
}

func TestBarPercentPrecisionPadded(t *testing.T) {
	assert.Equal(t, "  0.0%", barPercent(0, 100, 1, true))
	assert.Equal(t, " 50.0%", barPercent(50, 100, 1, true))
	assert.Equal(t, "100.0%", barPercent(100, 100, 1, true))
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
		FilledChar: '=',
		EmptyChar:  '-',
		LeftCap:    "|",
		RightCap:   "|",
		Width:      20,
	}
	b := Bar("test", 100).Style(custom)
	assert.Equal(t, custom, b.barStyle)
}

func TestBarDefaultStyle(t *testing.T) {
	s := DefaultBarStyle()
	assert.Equal(t, BarThin, s)
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
		assert.NotZero(t, style.FilledChar, "%s: FilledChar", name)
		assert.NotZero(t, style.MinWidth, "%s: MinWidth", name)
		assert.NotZero(t, style.MaxWidth, "%s: MaxWidth", name)
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
		FilledChar:       '█',
		EmptyChar:        ' ',
		LeftCap:          "[",
		RightCap:         "]",
		Width:            10,
		ProgressGradient: gradient,
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
		FilledChar: '█',
		EmptyChar:  ' ',
		LeftCap:    "[",
		RightCap:   "]",
		Width:      10,
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

func TestDefaultBarGradient(t *testing.T) {
	gradient := DefaultBarGradient()
	assert.Equal(t, DefaultPercentGradient(), gradient)
	assert.Len(t, gradient, 3)
}
