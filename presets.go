package clog

import (
	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/fx/spinner"
	"github.com/gechr/clog/style"
)

// Preset is a declarative bundle of logger configuration - part order,
// alignment, wrapping, labels, symbols, styles, and spinner defaults -
// applied in one call with [Logger.ApplyPreset]. Nil or empty fields leave
// the corresponding setting unchanged, so sparse presets layer over the
// logger's current configuration. Fields are listed in application order.
type Preset struct {
	// Parts sets the part order via [Logger.SetParts] [empty = leave unchanged].
	Parts []Part
	// LevelAlign sets label alignment via [Logger.SetLevelAlign] [nil = leave unchanged].
	LevelAlign *Align
	// Wrap sets line wrapping via [Logger.SetWrap] [nil = leave unchanged].
	Wrap *Wrap
	// Labels merges level labels over the defaults via [Logger.SetLabels] [nil = leave unchanged].
	Labels LabelMap
	// Symbols merges level symbols over the defaults via [Logger.SetSymbols] [nil = leave unchanged].
	Symbols LabelMap
	// Styles merges into the current styles via [Logger.SetStyles] [nil = leave unchanged].
	Styles *style.Config
	// SpinnerDefaults builds the default spinner configuration over
	// [spinner.DefaultConfig] via [Logger.SetSpinnerDefaults] [empty = leave unchanged].
	SpinnerDefaults []spinner.Option
}

// ApplyPreset applies p by calling the corresponding setters in a fixed
// order: parts, level alignment, wrap, labels, symbols, styles, spinner
// defaults. Each field keeps its setter's semantics: labels, symbols, and
// styles merge over the current configuration; parts and spinner defaults
// replace. Nil or empty fields are skipped, and applying the same preset
// twice is harmless. A nil p is a no-op.
//
// Application is not atomic - each setter locks separately - so apply
// presets at startup, before logging begins.
func (l *Logger) ApplyPreset(p *Preset) {
	if l == nil || p == nil {
		return
	}

	if len(p.Parts) > 0 {
		l.SetParts(p.Parts...)
	}
	if p.LevelAlign != nil {
		l.SetLevelAlign(*p.LevelAlign)
	}
	if p.Wrap != nil {
		l.SetWrap(*p.Wrap)
	}
	if p.Labels != nil {
		l.SetLabels(p.Labels)
	}
	if p.Symbols != nil {
		l.SetSymbols(p.Symbols)
	}
	if p.Styles != nil {
		l.SetStyles(p.Styles)
	}
	if len(p.SpinnerDefaults) > 0 {
		l.SetSpinnerDefaults(p.SpinnerDefaults...)
	}
}

// TersePreset returns the built-in terse preset: symbol-first lines with no
// level labels or timestamps, minimal single-character glyphs, ANSI colors,
// soft wrapping, and a bouncing-dots spinner. It suits compact CLI tools.
// Each call returns a fresh value, safe to modify before applying.
func TersePreset() *Preset {
	green := new(lipgloss.NewStyle().Foreground(lipgloss.Color("2")))
	yellow := new(lipgloss.NewStyle().Foreground(lipgloss.Color("3")))
	red := new(lipgloss.NewStyle().Foreground(lipgloss.Color("1")))
	return &Preset{
		Parts:      []Part{PartSymbol, PartMessage, PartFields},
		LevelAlign: new(AlignNone),
		Wrap:       new(WrapSoft),
		Symbols: LabelMap{
			LevelInfo:    "·",
			LevelSuccess: "✔︎",
			LevelNotice:  "›",
			LevelWarn:    "!",
			LevelError:   "✘",
			LevelFatal:   "✘",
			LevelDry:     "$",
		},
		Styles: &style.Config{
			Message: new(lipgloss.NewStyle().Bold(true)),
			Messages: style.LevelMap{
				LevelInfo:    green,
				LevelSuccess: green,
				LevelNotice:  yellow,
				LevelWarn:    yellow,
				LevelError:   red,
				LevelFatal: new(
					lipgloss.NewStyle(),
				), // deliberately plain: overrides the default fatal style
				LevelDry: yellow,
			},
			Symbols: style.LevelMap{
				LevelInfo:    yellow,
				LevelSuccess: green,
				LevelNotice:  yellow,
				LevelWarn:    yellow,
				LevelError:   red,
				LevelFatal:   red,
				LevelDry:     new(lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)),
			},
		},
		SpinnerDefaults: []spinner.Option{spinner.WithConfig(spinner.DotsBounce)},
	}
}
