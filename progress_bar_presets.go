package clog

import "github.com/charmbracelet/lipgloss"

// defaultCapStyle is the bold white style used for bar caps in all presets.
var defaultCapStyle = new(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")))

// Predefined bar styles for common visual appearances.
// Pass any of these to [AnimationBuilder.Style] to change the bar's look.
var (
	// BarBasic uses only ASCII characters for maximum terminal compatibility.
	//
	//	[=====>    ] 50%
	BarBasic = BarStyle{
		CapStyle:   defaultCapStyle,
		FilledChar: '=',
		EmptyChar:  ' ',
		HeadChar:   '>',
		LeftCap:    "[",
		RightCap:   "]",
		Separator:  " ",
		MinWidth:   barDefaultBarMinWidth,
		MaxWidth:   barDefaultMaxWidth,
	}

	// BarBlock uses solid block characters without sub-cell resolution.
	//
	//	│█████░░░░░│ 50%
	BarBlock = BarStyle{
		CapStyle:   defaultCapStyle,
		FilledChar: '█',
		EmptyChar:  '░',
		LeftCap:    "│",
		RightCap:   "│",
		Separator:  " ",
		MinWidth:   barDefaultBarMinWidth,
		MaxWidth:   barDefaultMaxWidth,
	}

	// BarDash uses a simple dash for filled cells and spaces for empty.
	//
	//	[-----     ] 50%
	BarDash = BarStyle{
		CapStyle:   defaultCapStyle,
		FilledChar: '-',
		EmptyChar:  ' ',
		LeftCap:    "[",
		RightCap:   "]",
		Separator:  " ",
		MinWidth:   barDefaultBarMinWidth,
		MaxWidth:   barDefaultMaxWidth,
	}

	// BarGradient uses block-element characters with 8x sub-cell resolution
	// for the smoothest possible progression.
	//
	//	│██████▍   │ 64%
	BarGradient = BarStyle{
		CapStyle:     defaultCapStyle,
		FilledChar:   '█',
		EmptyChar:    ' ',
		FillGradient: []rune{'▏', '▎', '▍', '▌', '▋', '▊', '▉'},
		LeftCap:      "│",
		RightCap:     "│",
		Separator:    " ",
		MinWidth:     barDefaultBarMinWidth,
		MaxWidth:     barDefaultMaxWidth,
	}

	// BarThin uses box-drawing characters with half-cell resolution for smooth
	// progress, inspired by Python's Rich library. This is the default style.
	//
	//	[━━━━━╸╺──────] 45%
	BarThin = BarStyle{
		CapStyle:   defaultCapStyle,
		FilledChar: '━',
		EmptyChar:  '─',
		HalfFilled: '╸',
		HalfEmpty:  '╺',
		LeftCap:    "[",
		RightCap:   "]",
		Separator:  " ",
		MinWidth:   barDefaultBarMinWidth,
		MaxWidth:   barDefaultMaxWidth,
	}

	// BarSmooth uses block characters with a half-block leading edge for
	// smoother progression than [BarBlock].
	//
	//	│████▌     │ 45%
	BarSmooth = BarStyle{
		CapStyle:   defaultCapStyle,
		FilledChar: '█',
		EmptyChar:  ' ',
		HalfFilled: '▌',
		LeftCap:    "│",
		RightCap:   "│",
		Separator:  " ",
		MinWidth:   barDefaultBarMinWidth,
		MaxWidth:   barDefaultMaxWidth,
	}
)
