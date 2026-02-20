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
	runes := []rune(text)
	if len(runes) == 0 {
		return text
	}

	c := interpolateGradient(phase, stops)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(c.Clamped().Hex()))

	var buf strings.Builder
	for _, r := range runes {
		if r == ' ' {
			buf.WriteRune(r)
			continue
		}

		buf.WriteString(style.Render(string(r)))
	}
	return buf.String()
}
