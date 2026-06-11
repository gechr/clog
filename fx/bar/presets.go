package bar

import (
	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/style"
)

// defaultCapStyle is the bold white style used for bar caps in all presets.
var defaultCapStyle = new(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")))

// thinEmptyStyle is the dark grey style used for Thin empty cells,
// matching Python Rich's bar.back color (grey23 / ANSI 237).
var thinEmptyStyle = new(lipgloss.NewStyle().Foreground(lipgloss.Color("237")))

// Predefined bar styles for common visual appearances.
// Pass any of these to [WithConfig] to change the bar's look.
var (
	// Basic uses only ASCII characters for maximum terminal compatibility.
	//
	//	[=====>    ] 50%
	Basic = Config{
		CapLeft:   "[",
		CapRight:  "]",
		CapStyle:  defaultCapStyle,
		CharEmpty: ' ',
		CharFill:  '=',
		CharHead:  '>',
		Separator: " ",
		WidthMax:  DefaultWidthMax,
		WidthMin:  DefaultWidthMin,
	}

	// Block uses solid block characters without sub-cell resolution.
	//
	//	│█████░░░░░│ 50%
	Block = Config{
		CapLeft:   "│",
		CapRight:  "│",
		CapStyle:  defaultCapStyle,
		CharEmpty: '░',
		CharFill:  '█',
		Separator: " ",
		WidthMax:  DefaultWidthMax,
		WidthMin:  DefaultWidthMin,
	}

	// Dash uses a simple dash for filled cells and spaces for empty.
	//
	//	[-----     ] 50%
	Dash = Config{
		CapLeft:   "[",
		CapRight:  "]",
		CapStyle:  defaultCapStyle,
		CharEmpty: ' ',
		CharFill:  '-',
		Separator: " ",
		WidthMax:  DefaultWidthMax,
		WidthMin:  DefaultWidthMin,
	}

	// Gradient uses block-element characters with 8x sub-cell resolution
	// for the smoothest possible progression.
	//
	//	│██████▍   │ 64%
	Gradient = Config{
		CapLeft:      "│",
		CapRight:     "│",
		CapStyle:     defaultCapStyle,
		CharEmpty:    ' ',
		CharFill:     '█',
		GradientFill: []rune{'▏', '▎', '▍', '▌', '▋', '▊', '▉'},
		Separator:    " ",
		WidthMax:     DefaultWidthMax,
		WidthMin:     DefaultWidthMin,
	}

	// Thin uses box-drawing characters with half-cell resolution for smooth
	// progress, inspired by Python's Rich library. Filled and empty cells
	// use the same character differentiated by color. The fill shifts through
	// a red → yellow → green gradient as progress advances.
	// This is the default style.
	//
	//	━━━━━━━━━╸╺━━━━━━━━━━━━━ 45%
	Thin = Config{
		CharEmpty:        '━',
		CharFill:         '━',
		ProgressGradient: style.DefaultPercentGradient(),
		Separator:        " ",
		StyleEmpty:       thinEmptyStyle,
		WidthMax:         DefaultWidthMax,
		WidthMin:         DefaultWidthMin,
	}

	// Braille uses braille dot-fill characters with 8x sub-cell resolution,
	// inspired by Docker Compose's progress display.
	//
	//	[⣿⣿⣿⣿⣿⣦⠀⠀⠀⠀] 50%
	Braille = Config{
		CapLeft:      "[",
		CapRight:     "]",
		CapStyle:     defaultCapStyle,
		CharEmpty:    '⠀',
		CharFill:     '⣿',
		GradientFill: []rune{'⡀', '⣀', '⣄', '⣤', '⣦', '⣶', '⣷'},
		Separator:    " ",
		WidthMax:     DefaultWidthMax,
		WidthMin:     DefaultWidthMin,
	}

	// Smooth uses solid block characters with no caps. Empty cells use
	// dim grey blocks for a continuous appearance.
	//
	//	██████████████████████ 45%
	Smooth = Config{
		CharEmpty:  '█',
		CharFill:   '█',
		StyleEmpty: thinEmptyStyle,
		Separator:  " ",
		WidthMax:   DefaultWidthMax,
		WidthMin:   DefaultWidthMin,
	}
)
