package theme

import (
	"fmt"

	"github.com/gechr/x/terminal"
)

// Background describes the terminal background a theme is designed for.
type Background int

const (
	BackgroundUnspecified Background = iota
	BackgroundLight
	BackgroundDark
)

func (b Background) String() string {
	switch b {
	case BackgroundUnspecified:
		return "unspecified"
	case BackgroundLight:
		return "light"
	case BackgroundDark:
		return "dark"
	default:
		return "unspecified"
	}
}

func (b Background) valid() bool {
	return b == BackgroundLight || b == BackgroundDark
}

// DetectBackground queries the controlling terminal for its background color.
func DetectBackground() (Background, bool) {
	dark, ok := terminal.IsDark()
	if !ok {
		return BackgroundDark, false
	}
	if dark {
		return BackgroundDark, true
	}
	return BackgroundLight, true
}

// Pair holds the light and dark themes an application supports.
type Pair struct {
	Light    *Theme
	Dark     *Theme
	Fallback Background
}

// Auto selects from clog's built-in themes using the terminal background.
func Auto() *Theme {
	return DefaultPair().Auto()
}

// DefaultPair returns clog's built-in light/dark theme pair.
func DefaultPair(opts ...PairOption) *Pair {
	return MustPair(Light(), Dark(), opts...)
}

// PairOption configures a theme Pair.
type PairOption func(*Pair)

// WithFallback sets the background used when terminal detection is unavailable.
func WithFallback(bg Background) PairOption {
	return func(p *Pair) {
		p.Fallback = bg
	}
}

// NewPair creates a theme pair with one light theme and one dark theme.
func NewPair(light, dark *Theme, opts ...PairOption) (*Pair, error) {
	p := &Pair{
		Light:    light,
		Dark:     dark,
		Fallback: BackgroundDark,
	}
	for _, opt := range opts {
		opt(p)
	}
	if err := p.validate(); err != nil {
		return nil, err
	}
	return p, nil
}

// MustPair creates a theme pair and panics if it is invalid.
func MustPair(light, dark *Theme, opts ...PairOption) *Pair {
	p, err := NewPair(light, dark, opts...)
	if err != nil {
		panic(err)
	}
	return p
}

// Single returns a Pair that renders t on both light and dark backgrounds,
// bypassing the light/dark validation that [NewPair] performs. Background
// detection still runs but has no visible effect.
func Single(t *Theme) *Pair {
	return &Pair{Light: t, Dark: t, Fallback: BackgroundDark}
}

// Auto selects from the pair using the terminal background, falling back to
// the pair's Fallback when detection is unavailable.
func (p *Pair) Auto() *Theme {
	bg, ok := DetectBackground()
	if !ok {
		bg = p.Fallback
	}
	return p.ForBackground(bg)
}

// ForBackground returns the theme matching bg.
func (p *Pair) ForBackground(bg Background) *Theme {
	if bg == BackgroundLight {
		return p.Light
	}
	return p.Dark
}

func (p *Pair) validate() error {
	if p == nil {
		return fmt.Errorf("theme pair is nil")
	}
	if p.Light == nil {
		return fmt.Errorf("light theme is required")
	}
	if p.Dark == nil {
		return fmt.Errorf("dark theme is required")
	}
	if p.Light.Background != BackgroundLight {
		return fmt.Errorf(
			"light theme must declare background %q, got %q",
			BackgroundLight,
			p.Light.Background,
		)
	}
	if p.Dark.Background != BackgroundDark {
		return fmt.Errorf(
			"dark theme must declare background %q, got %q",
			BackgroundDark,
			p.Dark.Background,
		)
	}
	if !p.Fallback.valid() {
		return fmt.Errorf(
			"fallback background must be %q or %q, got %q",
			BackgroundLight,
			BackgroundDark,
			p.Fallback,
		)
	}
	return nil
}
