// Package style provides styling types, color stops, and defaults for clog.
package style

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/gechr/clog/internal/gradient"
	"github.com/gechr/clog/level"
)

// ColorStop defines a color at a specific position along a gradient.
// Position is in the range 0.0-1.0.
type ColorStop = gradient.ColorStop

// ThresholdStyle holds optional style overrides for the number and unit
// segments of a quantity or duration value. nil fields keep the default style.
type ThresholdStyle struct {
	Number *lipgloss.Style // Override for the number segment (nil = keep default).
	Unit   *lipgloss.Style // Override for the unit segment (nil = keep default).
}

// Threshold defines a style override when a quantity's numeric value
// meets or exceeds the given threshold. Thresholds are evaluated in descending
// order - the first match wins.
type Threshold struct {
	Value float64        // Minimum numeric value (inclusive) to trigger this style.
	Style ThresholdStyle // Style overrides for number and unit segments.
}

// Map maps string keys to lipgloss styles (e.g. field key names or unit strings).
type Map = map[string]*lipgloss.Style

// Thresholds is a list of [Threshold] entries, evaluated high -> low (first match wins).
type Thresholds = []Threshold

// ThresholdMap maps unit strings to their thresholds (evaluated high -> low).
type ThresholdMap = map[string]Thresholds

// LevelMap maps log levels to lipgloss styles.
type LevelMap = map[level.Level]*lipgloss.Style

// ValueMap maps typed values to lipgloss styles. Keys use Go equality
// (e.g. bool true != string "true").
type ValueMap = map[any]*lipgloss.Style

// GradientMode controls how gradient colors transition between stops.
type GradientMode int

const (
	// GradientFade smoothly interpolates between color stops.
	GradientFade GradientMode = iota
	// GradientStep uses discrete color jumps at stop boundaries.
	GradientStep
)

// Config holds lipgloss styles for the logger's pretty output.
// Pointer fields can be set to nil to disable that style entirely.
type Config struct {
	Renderer *lipgloss.Renderer
	// Style for divider line characters (see [clog.DividerBuilder]) [nil = plain text]
	DividerLine *lipgloss.Style
	// Style for divider title text [nil = plain text]
	DividerTitle *lipgloss.Style
	// Duration unit -> thresholds (evaluated high->low).
	DurationThresholds ThresholdMap
	// Duration unit -> style override (e.g. "s" -> yellow).
	DurationUnits Map
	// Style for the numeric segments of duration values (e.g. "1" in "1m30s") [nil = plain text]
	FieldDurationNumber *lipgloss.Style
	// Style for the unit segments of duration values (e.g. "m" in "1m30s") [nil = plain text]
	FieldDurationUnit *lipgloss.Style
	// Style for the numeric segments of elapsed-time values [nil = falls back to FieldDurationNumber]
	FieldElapsedNumber *lipgloss.Style
	// Style for the unit segments of elapsed-time values [nil = falls back to FieldDurationUnit]
	FieldElapsedUnit *lipgloss.Style
	// Gradient stops for Elapsed fields (default: green -> yellow -> red).
	// Active only when [elapsed.GradientMax] > 0; overrides FieldElapsedNumber/FieldElapsedUnit.
	ElapsedGradient []ColorStop
	// How elapsed gradient colors transition: [GradientFade] (smooth) or [GradientStep] (discrete).
	ElapsedGradientMode GradientMode
	// Style for error field values [nil = plain text]
	FieldError *lipgloss.Style
	// Per-token styles for JSON syntax highlighting.
	// nil disables JSON highlighting; use [DefaultJSON] to enable.
	FieldJSON *JSON
	// Style for int/float field values [nil = plain text]
	FieldNumber *lipgloss.Style
	// Base style for Percent fields (foreground overridden by gradient). nil = gradient color only.
	FieldPercent *lipgloss.Style
	// Style for the numeric part of quantity values (e.g. "5" in "5km") [nil = plain text]
	FieldQuantityNumber *lipgloss.Style
	// Style for the unit part of quantity values (e.g. "km" in "5km") [nil = plain text]
	FieldQuantityUnit *lipgloss.Style
	// Style for string field values [nil = plain text]
	FieldString *lipgloss.Style
	// Style for time.Time field values [nil = plain text]
	FieldTime *lipgloss.Style
	// Style for field key names without a per-key override.
	KeyDefault *lipgloss.Style
	// Field key name -> value style (e.g. "path" -> blue).
	Keys Map
	// Level label style (e.g. "INF", "ERR").
	Levels LevelMap
	// Message text style per level.
	Messages LevelMap
	// Gradient stops for Percent fields (default: red -> yellow -> green).
	PercentGradient []ColorStop
	// Quantity unit -> thresholds (evaluated high->low).
	QuantityThresholds ThresholdMap
	// Unit string -> style override (e.g. "km" -> green).
	QuantityUnits Map
	// Style for key/value separator.
	Separator *lipgloss.Style
	// Symbol text style per level (e.g. make "warning" bold yellow).
	// nil entries render the symbol unstyled.
	Symbols LevelMap
	// Style for the timestamp prefix.
	Timestamp *lipgloss.Style
	// Values maps typed values to styles. Keys use Go equality.
	// Allows differentiating between e.g. `true` (bool) and "true" (string).
	Values ValueMap
}

// WithRenderer rebinds all styles to the given renderer. This ensures styles
// render correctly when the logger's output differs from os.Stdout (e.g.
// logging to stderr while stdout is piped). It mutates and returns the
// receiver for fluent chaining.
func (s *Config) WithRenderer(r *lipgloss.Renderer) *Config {
	s.Renderer = r

	// Simple style fields.
	s.DividerLine = rebind(r, s.DividerLine)
	s.DividerTitle = rebind(r, s.DividerTitle)
	s.FieldDurationNumber = rebind(r, s.FieldDurationNumber)
	s.FieldDurationUnit = rebind(r, s.FieldDurationUnit)
	s.FieldElapsedNumber = rebind(r, s.FieldElapsedNumber)
	s.FieldElapsedUnit = rebind(r, s.FieldElapsedUnit)
	s.FieldError = rebind(r, s.FieldError)
	s.FieldNumber = rebind(r, s.FieldNumber)
	s.FieldPercent = rebind(r, s.FieldPercent)
	s.FieldQuantityNumber = rebind(r, s.FieldQuantityNumber)
	s.FieldQuantityUnit = rebind(r, s.FieldQuantityUnit)
	s.FieldString = rebind(r, s.FieldString)
	s.FieldTime = rebind(r, s.FieldTime)
	s.KeyDefault = rebind(r, s.KeyDefault)
	s.Separator = rebind(r, s.Separator)
	s.Timestamp = rebind(r, s.Timestamp)

	// Map fields.
	rebindStyleMap(r, s.Keys)
	rebindStyleMap(r, s.DurationUnits)
	rebindStyleMap(r, s.QuantityUnits)

	// LevelMap fields.
	rebindStyleMap(r, s.Levels)
	rebindStyleMap(r, s.Messages)
	rebindStyleMap(r, s.Symbols)

	// ValueMap.
	for k, v := range s.Values {
		s.Values[k] = rebind(r, v)
	}

	// ThresholdMap fields.
	for unit, thresholds := range s.DurationThresholds {
		for i := range thresholds {
			thresholds[i].Style.Number = rebind(r, thresholds[i].Style.Number)
			thresholds[i].Style.Unit = rebind(r, thresholds[i].Style.Unit)
		}
		s.DurationThresholds[unit] = thresholds
	}
	for unit, thresholds := range s.QuantityThresholds {
		for i := range thresholds {
			thresholds[i].Style.Number = rebind(r, thresholds[i].Style.Number)
			thresholds[i].Style.Unit = rebind(r, thresholds[i].Style.Unit)
		}
		s.QuantityThresholds[unit] = thresholds
	}

	// Delegate to JSON.
	if s.FieldJSON != nil {
		s.FieldJSON.WithRenderer(r)
	}

	return s
}

// rebindStyleMap rebinds all style values in a map to the given renderer.
func rebindStyleMap[K comparable](r *lipgloss.Renderer, m map[K]*lipgloss.Style) {
	for k, s := range m {
		m[k] = rebind(r, s)
	}
}

// rebind rebinds a lipgloss style to the given renderer. Returns nil for nil styles.
func rebind(r *lipgloss.Renderer, s *lipgloss.Style) *lipgloss.Style {
	if s == nil {
		return nil
	}
	ns := s.Renderer(r)
	return &ns
}
