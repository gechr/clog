package bar

import "charm.land/lipgloss/v2"

// defaultCapStyle is the bold white style used for bar caps in all presets.
var defaultCapStyle = new(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")))

// Predefined bar styles for common visual appearances.
// Pass any of these to [WithStyle] to change the bar's look.
var (
	// Basic uses only ASCII characters for maximum terminal compatibility.
	//
	//	[=====>    ] 50%
	Basic = Style{
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
	Block = Style{
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
	Dash = Style{
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
	Gradient = Style{
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
	// progress, inspired by Python's Rich library. This is the default style.
	//
	//	[━━━━━╸╺──────] 45%
	Thin = Style{
		CapLeft:    "[",
		CapRight:   "]",
		CapStyle:   defaultCapStyle,
		CharEmpty:  '─',
		CharFill:   '━',
		HalfEmpty:  '╺',
		HalfFilled: '╸',
		Separator:  " ",
		WidthMax:   DefaultWidthMax,
		WidthMin:   DefaultWidthMin,
	}

	// Braille uses braille dot-fill characters with 8x sub-cell resolution,
	// inspired by Docker Compose's progress display.
	//
	//	[⣿⣿⣿⣿⣿⣦⠀⠀⠀⠀] 50%
	Braille = Style{
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

	// Smooth uses block characters with a half-block leading edge for
	// smoother progression than [Block].
	//
	//	│████▌     │ 45%
	Smooth = Style{
		CapLeft:    "│",
		CapRight:   "│",
		CapStyle:   defaultCapStyle,
		CharEmpty:  ' ',
		CharFill:   '█',
		HalfFilled: '▌',
		Separator:  " ",
		WidthMax:   DefaultWidthMax,
		WidthMin:   DefaultWidthMin,
	}
)
