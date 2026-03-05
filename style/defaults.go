package style

import (
	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/level"
	"github.com/lucasb-eyer/go-colorful"
)

// Default returns the default color styles.
func Default() *Config {
	return &Config{
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
		FieldJSON: DefaultJSON(),
		FieldNumber: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // magenta
		),
		FieldQuantityNumber: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // magenta
		),
		FieldQuantityUnit: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // magenta
		),
		FieldString: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("15")), // white
		),
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
// used for [Config.ElapsedGradient].
func DefaultElapsedGradient() []ColorStop {
	start, middle, end := 0.0, 0.5, 1.0
	return []ColorStop{
		{
			Position: start,
			Color:    colorful.Color{R: 0, G: 1, B: 0}, // green
		},
		{
			Position: middle,
			Color:    colorful.Color{R: 1, G: 1, B: 0}, // yellow
		},
		{
			Position: end,
			Color:    colorful.Color{R: 1, G: 0, B: 0}, // red
		},
	}
}

// DefaultPercentGradient returns the default red -> yellow -> green gradient
// used for [Config.PercentGradient].
func DefaultPercentGradient() []ColorStop {
	start, middle, end := 0.0, 0.5, 1.0
	return []ColorStop{
		{
			Position: start,
			Color:    colorful.Color{R: 1, G: 0, B: 0}, // red
		},
		{
			Position: middle,
			Color:    colorful.Color{R: 1, G: 1, B: 0}, // yellow
		},
		{
			Position: end,
			Color:    colorful.Color{R: 0, G: 1, B: 0}, // green
		},
	}
}

// DefaultMessages returns the default per-level message styles (unstyled).
func DefaultMessages() LevelMap {
	return LevelMap{
		level.Trace: new(lipgloss.NewStyle()),
		level.Debug: new(lipgloss.NewStyle()),
		level.Info:  new(lipgloss.NewStyle()),
		level.Dry:   new(lipgloss.NewStyle()),
		level.Warn:  new(lipgloss.NewStyle()),
		level.Error: new(lipgloss.NewStyle()),
		level.Fatal: new(lipgloss.NewStyle()),
	}
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
