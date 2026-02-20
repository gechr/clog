package clog

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

const (
	// pulseSpeed is the number of full oscillation cycles per second.
	pulseSpeed = 0.5

	// pulseTickRate is the repaint interval when pulse is active (~30fps).
	pulseTickRate = 33 * time.Millisecond
)

// DefaultPulseGradient returns a two-stop gradient for pulse effects:
// a muted blue-gray fading to a bright cyan.
func DefaultPulseGradient() []ColorStop {
	dimR, dimG, dimB := 0.4, 0.4, 0.6
	litR, litG, litB := 0.6, 0.9, 1.0
	return []ColorStop{
		{Position: 0, Color: colorful.Color{R: dimR, G: dimG, B: dimB}},
		{Position: 1, Color: colorful.Color{R: litR, G: litG, B: litB}},
	}
}

// pulseText renders all characters of text with a single gradient-interpolated
// foreground color, creating an animated pulse when called with advancing
// phase values. Spaces are passed through unstyled.
func pulseText(text string, phase float64, stops []ColorStop) string {
	if len(text) == 0 {
		return text
	}

	c := interpolateGradient(phase, stops)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(c.Clamped().Hex()))

	// Split text into runs of spaces and non-spaces, rendering only non-space
	// runs through the style. This reduces style.Render calls from ~N to a
	// small number of runs.
	var buf strings.Builder
	runStart := 0
	runes := []rune(text)
	isSpace := runes[0] == ' '

	for i := 1; i <= len(runes); i++ {
		atEnd := i == len(runes)
		curIsSpace := !atEnd && runes[i] == ' '

		if atEnd || curIsSpace != isSpace {
			run := string(runes[runStart:i])
			if isSpace {
				buf.WriteString(run)
			} else {
				buf.WriteString(style.Render(run))
			}
			runStart = i
			isSpace = curIsSpace
		}
	}

	return buf.String()
}
