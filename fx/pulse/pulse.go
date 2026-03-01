// Package pulse provides pulse animation rendering for clog.
package pulse

import (
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/gechr/clog/internal/gradient"
	"github.com/lucasb-eyer/go-colorful"
)

const (
	// Speed is the number of full oscillation cycles per second.
	Speed = 0.5

	// TickRate is the repaint interval when pulse is active (~30fps).
	TickRate = 33 * time.Millisecond
)

// Style holds resolved pulse configuration.
type Style struct {
	Gradient []gradient.ColorStop
	Speed    float64
}

// Cache holds the last-used hex color and its corresponding lipgloss
// style, allowing [TextCached] to skip style creation when the color
// hasn't changed between frames.
type Cache struct {
	Hex   string
	Style lipgloss.Style
}

// DefaultStyle returns the default pulse configuration.
func DefaultStyle() Style {
	return Style{
		Gradient: DefaultGradient(),
		Speed:    Speed,
	}
}

// DefaultGradient returns a three-stop gradient for pulse effects:
// pastel light blue fading through light green to white.
func DefaultGradient() []gradient.ColorStop {
	lbR, lbG := 0.75, 0.9  // light blue (B = 1.0)
	lgR, lgB := 0.82, 0.88 // light green (G = 1.0)
	mid := 0.5
	lightBlue := colorful.Color{R: lbR, G: lbG, B: 1.0}
	lightGreen := colorful.Color{R: lgR, G: 1.0, B: lgB}
	white := colorful.Color{R: 1.0, G: 1.0, B: 1.0}
	return []gradient.ColorStop{
		{Position: 0, Color: lightBlue},
		{Position: mid, Color: lightGreen},
		{Position: 1, Color: white},
	}
}

// Text renders all characters of text with a single gradient-interpolated
// foreground color, creating an animated pulse when called with advancing
// phase values. Spaces are passed through unstyled.
func Text(text string, phase float64, stops []gradient.ColorStop) string {
	if len(text) == 0 {
		return text
	}
	c := gradient.Interpolate(phase, stops)
	s := lipgloss.NewStyle().Foreground(lipgloss.Color(c.Clamped().Hex()))
	return applyStyle(text, s)
}

// TextCached is like [Text] but reuses the cached style when the
// interpolated hex color matches the previous call. Pass a persistent
// *Cache across frames to avoid style allocations when the color is
// stable between ticks. When r is non-nil, styles are bound to that renderer
// instead of the global default; pass nil to use [lipgloss.NewStyle].
func TextCached(
	text string,
	phase float64,
	stops []gradient.ColorStop,
	cache *Cache,
	r *lipgloss.Renderer,
) string {
	if len(text) == 0 {
		return text
	}
	c := gradient.Interpolate(phase, stops)
	hex := c.Clamped().Hex()
	if hex != cache.Hex {
		if r != nil {
			cache.Style = r.NewStyle().Foreground(lipgloss.Color(hex))
		} else {
			cache.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
		}
		cache.Hex = hex
	}
	return applyStyle(text, cache.Style)
}

// applyStyle renders text with the given style, passing spaces through
// unstyled. Non-space runs are batched into a single style.Render call to
// minimise allocations.
func applyStyle(text string, s lipgloss.Style) string {
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
				buf.WriteString(s.Render(run))
			}
			runStart = i
			isSpace = curIsSpace
		}
	}
	// Flush final run.
	if run := text[runStart:]; isSpace {
		buf.WriteString(run)
	} else {
		buf.WriteString(s.Render(run))
	}

	return buf.String()
}
