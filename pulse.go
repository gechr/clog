package clog

import (
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

const (
	// pulseSpeed is the number of full oscillation cycles per second.
	pulseSpeed = 0.5

	// pulseTickRate is the repaint interval when pulse is active (~30fps).
	pulseTickRate = 33 * time.Millisecond
)

// DefaultPulseGradient returns a three-stop gradient for pulse effects:
// pastel light blue fading through light green to white.
func DefaultPulseGradient() []ColorStop {
	lbR, lbG := 0.75, 0.9  // light blue (B = 1.0)
	lgR, lgB := 0.82, 0.88 // light green (G = 1.0)
	mid := 0.5
	lightBlue := colorful.Color{R: lbR, G: lbG, B: 1.0}
	lightGreen := colorful.Color{R: lgR, G: 1.0, B: lgB}
	white := colorful.Color{R: 1.0, G: 1.0, B: 1.0}
	return []ColorStop{
		{Position: 0, Color: lightBlue},
		{Position: mid, Color: lightGreen},
		{Position: 1, Color: white},
	}
}

// pulseCache holds the last-used hex color and its corresponding lipgloss
// style, allowing [pulseTextCached] to skip style creation when the color
// hasn't changed between frames.
type pulseCache struct {
	hex   string
	style lipgloss.Style
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
	return applyPulseStyle(text, style)
}

// pulseTextCached is like [pulseText] but reuses the cached style when the
// interpolated hex color matches the previous call. Pass a persistent
// *pulseCache across frames to avoid style allocations when the color is
// stable between ticks.
func pulseTextCached(text string, phase float64, stops []ColorStop, cache *pulseCache) string {
	if len(text) == 0 {
		return text
	}
	c := interpolateGradient(phase, stops)
	hex := c.Clamped().Hex()
	if hex != cache.hex {
		cache.style = lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
		cache.hex = hex
	}
	return applyPulseStyle(text, cache.style)
}

// applyPulseStyle renders text with the given style, passing spaces through
// unstyled. Non-space runs are batched into a single style.Render call to
// minimise allocations.
func applyPulseStyle(text string, style lipgloss.Style) string {
	// Split text into runs of spaces and non-spaces, rendering only non-space
	// runs through the style. This reduces style.Render calls from ~N to a
	// small number of runs.
	var buf strings.Builder
	runStart := 0
	isSpace := false
	first := true

	for i, r := range text {
		curIsSpace := unicode.IsSpace(r)
		if first {
			isSpace = curIsSpace
			first = false
			continue
		}
		if curIsSpace != isSpace {
			run := text[runStart:i]
			if isSpace {
				buf.WriteString(run)
			} else {
				buf.WriteString(style.Render(run))
			}
			runStart = i
			isSpace = curIsSpace
		}
	}
	// Flush final run.
	if run := text[runStart:]; isSpace {
		buf.WriteString(run)
	} else {
		buf.WriteString(style.Render(run))
	}

	return buf.String()
}
