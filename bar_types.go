package clog

// Predefined bar styles for common visual appearances.
// Pass any of these to [AnimationBuilder.Style] to change the bar's look.
var (
	// BarASCII uses only ASCII characters for maximum terminal compatibility.
	//
	//	[=====>    ] 50%
	BarASCII = BarStyle{
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
	//	[█████░░░░░] 50%
	BarBlock = BarStyle{
		FilledChar: '█',
		EmptyChar:  '░',
		LeftCap:    "[",
		RightCap:   "]",
		Separator:  " ",
		MinWidth:   barDefaultBarMinWidth,
		MaxWidth:   barDefaultMaxWidth,
	}

	// BarGradient uses block-element characters with 8x sub-cell resolution
	// for the smoothest possible progression.
	//
	//	[██████▍   ] 64%
	BarGradient = BarStyle{
		FilledChar:   '█',
		EmptyChar:    ' ',
		FillGradient: []rune{'▏', '▎', '▍', '▌', '▋', '▊', '▉'},
		LeftCap:      "[",
		RightCap:     "]",
		Separator:    " ",
		MinWidth:     barDefaultBarMinWidth,
		MaxWidth:     barDefaultMaxWidth,
	}

	// BarThin uses box-drawing characters with half-cell resolution for smooth
	// progress, inspired by Python's Rich library. This is the default style.
	//
	//	[━━━━━╸╺──────] 45%
	BarThin = BarStyle{
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
	//	[████▌     ] 45%
	BarSmooth = BarStyle{
		FilledChar: '█',
		EmptyChar:  ' ',
		HalfFilled: '▌',
		LeftCap:    "[",
		RightCap:   "]",
		Separator:  " ",
		MinWidth:   barDefaultBarMinWidth,
		MaxWidth:   barDefaultMaxWidth,
	}
)
