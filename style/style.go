// Package style provides styling types, color stops, and defaults for clog.
package style

import (
	"reflect"

	"charm.land/lipgloss/v2"
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

// KeyValueMap maps a field key name to per-value styles for that key.
type KeyValueMap = map[string]KeyValue

// KeyValue holds per-value styles for a single field key, plus an optional
// Default for that key's values that have no explicit entry. Values are keyed
// by Go equality (e.g. bool true != string "true").
//
// When a field key has a KeyValue entry it fully governs that key's value
// styling: a value present in Values takes its style; otherwise Default is
// applied, or - when Default is nil - the value renders plain (unstyled). This
// is the highest-priority value styling tier, taking precedence over Keys,
// Values and the type styles.
type KeyValue struct {
	// Values maps this key's typed values to styles (Go equality).
	Values ValueMap
	// Default styles this key's values absent from Values [nil = plain text].
	Default *lipgloss.Style
}

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
	// Per-token styles for HCL syntax highlighting.
	// nil disables HCL highlighting; use [DefaultHCL] to enable.
	HCL *HCL
	// Per-token styles for JSON syntax highlighting.
	// nil disables JSON highlighting; use [DefaultJSON] to enable.
	JSON *JSON
	// Per-token styles for TOML syntax highlighting.
	// nil disables TOML highlighting; use [DefaultTOML] to enable.
	TOML *TOML
	// Per-token styles for YAML syntax highlighting.
	// nil disables YAML highlighting; use [DefaultYAML] to enable.
	YAML *YAML

	// Style for text inside `backticks`, in both the message and string field
	// values [nil = leave backticks intact]. A non-color writer leaves the
	// backticks as written. BacktickMode controls the delimiters.
	Backtick *lipgloss.Style
	// How the backtick delimiters around styled spans are handled:
	// [BacktickStrip] (the default) drops them, [BacktickKeep] renders them
	// as part of the styled span so the message keeps the visible width the
	// caller wrote (pre-aligned content stays aligned).
	BacktickMode BacktickMode
	// Style for divider line characters (see [clog.DividerBuilder]) [nil = plain text]
	DividerLine *lipgloss.Style
	// Style for divider title text [nil = plain text]
	DividerTitle *lipgloss.Style
	// Gradient stops for Duration fields (default: green -> yellow -> red).
	// Active only when FieldFormats.DurationGradientMax > 0; overrides FieldDurationNumber/FieldDurationUnit.
	DurationGradient []ColorStop
	// How duration gradient colors transition: [GradientFade] (smooth) or [GradientStep] (discrete).
	DurationGradientMode GradientMode
	// Duration unit -> thresholds (evaluated high->low).
	DurationThresholds ThresholdMap
	// Duration unit -> style override (e.g. "s" -> yellow).
	DurationUnits Map
	// Gradient stops for Elapsed fields (default: green -> yellow -> red).
	// Active only when FieldFormats.ElapsedGradientMax > 0; overrides FieldElapsedNumber/FieldElapsedUnit.
	ElapsedGradient []ColorStop
	// How elapsed gradient colors transition: [GradientFade] (smooth) or [GradientStep] (discrete).
	ElapsedGradientMode GradientMode
	// Style for the numeric segments of duration values (e.g. "1" in "1m30s") [nil = plain text]
	FieldDurationNumber *lipgloss.Style
	// Style for the unit segments of duration values (e.g. "m" in "1m30s") [nil = plain text]
	FieldDurationUnit *lipgloss.Style
	// Style for the numeric segments of elapsed-time values [nil = falls back to FieldDurationNumber]
	FieldElapsedNumber *lipgloss.Style
	// Style for the unit segments of elapsed-time values [nil = falls back to FieldDurationUnit]
	FieldElapsedUnit *lipgloss.Style
	// Style for error field values [nil = plain text]
	FieldError *lipgloss.Style
	// Style for the "/" separator in fraction values. nil = the fraction's
	// current (gradient or base) color with the faint attribute, so it reads
	// as a dimmed version of the numbers.
	FieldFractionSeparator *lipgloss.Style
	// Style for int/float field values [nil = plain text]
	FieldNumber *lipgloss.Style
	// Base style for Percent fields (foreground overridden by gradient). nil = gradient color only.
	FieldPercent *lipgloss.Style
	// Style for the numeric part of quantity values (e.g. "5" in "5km") [nil = plain text]
	FieldQuantityNumber *lipgloss.Style
	// Style for the unit part of quantity values (e.g. "km" in "5km") [nil = plain text]
	FieldQuantityUnit *lipgloss.Style
	// Style for the quote delimiters around quoted values [nil = same style as the value]
	FieldQuote *QuoteStyle
	// Style for string field values [nil = plain text]
	FieldString *lipgloss.Style
	// Style for time.Time field values [nil = plain text]
	FieldTime *lipgloss.Style
	// Style for field key names without a per-key override.
	KeyDefault *lipgloss.Style
	// Field key name -> value style (e.g. "path" -> blue).
	Keys Map
	// Field key name -> per-value styles (see [KeyValue]). Highest-priority
	// value styling tier: when a key has an entry it fully governs that key's
	// values, overriding Keys, Values and the type styles.
	KeyValues KeyValueMap
	// Level label style (e.g. "INF", "ERR").
	Levels LevelMap
	// Global message text style [nil = plain text]. Overridden by Messages[level] when set.
	Message *lipgloss.Style
	// Message text style per level. Takes precedence over Message when set.
	Messages LevelMap
	// Gradient stops for Percent fields (default: red -> yellow -> green).
	PercentGradient []ColorStop
	// Style for a prompt's leading marker, rendered via [Logger.PromptStyle]
	// [nil = plain text]. Independent of Message so a prompt's marker (e.g.
	// "❯") can carry its own color without affecting the prompt text.
	Prompt *lipgloss.Style
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

// QuoteStyle configures how the quote delimiters around a quoted value are
// styled. When Inherit is true, Style is overlaid onto the value's own
// resolved style (the delimiters keep the value's color but gain Style's own
// attributes, e.g. bold); when false, Style is used as-is, ignoring the value.
type QuoteStyle struct {
	Style   lipgloss.Style
	Inherit bool
}

// Resolve returns the concrete delimiter style for a value whose resolved
// style is base (nil when the value has no style). With Inherit set, Style's
// explicitly-set attributes win and any unset ones are taken from base;
// otherwise Style is returned unchanged.
func (q QuoteStyle) Resolve(base *lipgloss.Style) lipgloss.Style {
	if q.Inherit && base != nil {
		return q.Style.Inherit(*base)
	}
	return q.Style
}

// Merge applies non-zero fields from other into c. Pointer fields are
// overwritten when non-nil; map fields are merged key-by-key; slice fields
// are replaced when non-nil; scalar fields are overwritten when non-zero.
func (c *Config) Merge(other *Config) {
	if other == nil {
		return
	}

	for sf, sv := range reflect.ValueOf(other).Elem().Fields() {
		if sv.IsZero() {
			continue
		}
		dv := reflect.ValueOf(c).Elem().FieldByIndex(sf.Index)

		// Map fields: merge key-by-key rather than replace.
		if sv.Kind() == reflect.Map {
			if dv.IsNil() {
				dv.Set(reflect.MakeMap(sv.Type()))
			}
			for _, k := range sv.MapKeys() {
				dv.SetMapIndex(k, sv.MapIndex(k))
			}
			continue
		}

		dv.Set(sv)
	}
}
