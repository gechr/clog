package style

import (
	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/level"
	"github.com/gechr/clog/theme"
	"github.com/lucasb-eyer/go-colorful"
)

// Default returns the default color styles.
func Default() *Config {
	return &Config{
		Backtick: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("183")), // lavender
		),
		DividerLine:  new(lipgloss.NewStyle().Faint(true)),
		DividerTitle: new(lipgloss.NewStyle().Bold(true)),
		FieldDurationNumber: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // magenta
		),
		FieldDurationUnit: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // magenta
		),
		FieldError: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("1")), // red
		),
		HCL:  DefaultHCL(),
		JSON: DefaultJSON(),
		TOML: DefaultTOML(),
		YAML: DefaultYAML(),
		FieldNumber: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // magenta
		),
		FieldQuantityNumber: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // magenta
		),
		FieldQuantityUnit: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // magenta
		),
		FieldString: new(lipgloss.NewStyle()),
		FieldTime: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // magenta
		),
		KeyDefault: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("4")), // blue
		),
		Keys: make(Map),
		Levels: LevelMap{
			level.Trace: new(lipgloss.NewStyle().
				Bold(true).
				Faint(true).
				Foreground(lipgloss.Color("6"))), // dim cyan
			level.Debug: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("6"))), // cyan
			level.Info: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("2"))), // green
			level.Hint: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("4"))), // blue
			level.Dry: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("5"))), // magenta
			level.Warn: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("3"))), // yellow
			level.Error: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("1"))), // red
			level.Fatal: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("1"))), // red
		},
		DurationGradient:   DefaultElapsedGradient(),
		DurationThresholds: make(ThresholdMap),
		DurationUnits:      make(Map),
		Messages:           DefaultMessages(),
		ElapsedGradient:    DefaultElapsedGradient(),
		PercentGradient:    DefaultPercentGradient(),
		QuantityThresholds: make(ThresholdMap),
		QuantityUnits:      make(Map),
		Separator:          new(lipgloss.NewStyle().Faint(true)),
		Symbols:            make(LevelMap),
		Timestamp:          new(lipgloss.NewStyle().Faint(true)),
		Values:             DefaultValues(),
	}
}

// DefaultElapsedGradient returns the default green -> yellow -> red gradient
// used for [Config.ElapsedGradient], tuned for a dark background.
//
// Terminal-aware light/dark selection is applied by the [Logger]; see
// [ElapsedGradientFor] for the background-specific stops.
func DefaultElapsedGradient() []ColorStop {
	return ElapsedGradientFor(theme.BackgroundDark)
}

// DefaultPercentGradient returns the default red -> yellow -> green gradient
// used for [Config.PercentGradient], tuned for a dark background.
//
// Terminal-aware light/dark selection is applied by the [Logger]; see
// [PercentGradientFor] for the background-specific stops.
func DefaultPercentGradient() []ColorStop {
	return PercentGradientFor(theme.BackgroundDark)
}

// ElapsedGradientFor returns the green -> yellow -> red gradient used for
// [Config.ElapsedGradient] and [Config.DurationGradient], with stops chosen for
// readable contrast against bg.
//
// The dark stops are vivid primaries that pop on a dark background; the light
// stops are darkened, saturated variants because pure green and yellow are
// nearly invisible on a light background.
func ElapsedGradientFor(bg theme.Background) []ColorStop {
	start, middle, end := 0.0, 0.5, 1.0
	if bg == theme.BackgroundLight {
		return []ColorStop{
			{Position: start, Color: hex("#1a7f37")},  // green
			{Position: middle, Color: hex("#b8860b")}, // amber
			{Position: end, Color: hex("#cf222e")},    // red
		}
	}
	return []ColorStop{
		{Position: start, Color: colorful.Color{R: 0, G: 1, B: 0}},  // green
		{Position: middle, Color: colorful.Color{R: 1, G: 1, B: 0}}, // yellow
		{Position: end, Color: colorful.Color{R: 1, G: 0, B: 0}},    // red
	}
}

// PercentGradientFor returns the red -> yellow -> green gradient used for
// [Config.PercentGradient], with stops chosen for readable contrast against bg.
// It mirrors [ElapsedGradientFor] with the stop order reversed.
func PercentGradientFor(bg theme.Background) []ColorStop {
	start, middle, end := 0.0, 0.5, 1.0
	if bg == theme.BackgroundLight {
		return []ColorStop{
			{Position: start, Color: hex("#cf222e")},  // red
			{Position: middle, Color: hex("#b8860b")}, // amber
			{Position: end, Color: hex("#1a7f37")},    // green
		}
	}
	return []ColorStop{
		{Position: start, Color: colorful.Color{R: 1, G: 0, B: 0}},  // red
		{Position: middle, Color: colorful.Color{R: 1, G: 1, B: 0}}, // yellow
		{Position: end, Color: colorful.Color{R: 0, G: 1, B: 0}},    // green
	}
}

// hex parses a "#rrggbb" color, panicking on malformed input. Intended only for
// the compile-time constant gradient stops above.
func hex(s string) colorful.Color {
	c, err := colorful.Hex(s)
	if err != nil {
		panic("style: invalid gradient hex " + s + ": " + err.Error())
	}
	return c
}

// DefaultMessages returns the default per-level message styles (empty map;
// falls back to the global [Config.Message] style when entries are nil).
func DefaultMessages() LevelMap {
	return LevelMap{}
}

// DefaultValues returns sensible default styles for common value strings.
func DefaultValues() ValueMap {
	return ValueMap{
		true:    new(lipgloss.NewStyle().Foreground(lipgloss.Color("2"))), // green
		false:   new(lipgloss.NewStyle().Foreground(lipgloss.Color("1"))), // red
		nil:     new(lipgloss.NewStyle().Faint(true)),
		"<nil>": new(lipgloss.NewStyle().Faint(true)),
		"":      new(lipgloss.NewStyle().Faint(true)),
	}
}
