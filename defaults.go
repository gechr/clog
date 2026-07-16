package clog

import (
	"io"
	"maps"
	"time"

	"github.com/gechr/clog/fx/spinner"
	"github.com/gechr/clog/level"
	"github.com/gechr/clog/style"
	"github.com/gechr/clog/theme"
)

// DefaultStyles returns the default color styles.
func DefaultStyles() *style.Config {
	return style.Default()
}

// Config holds configuration options for the [Default] logger.
type Config struct {
	// Output is the output to use (defaults to [Stdout]([ColorAuto])).
	Output *Output
	// Styles allows customising the visual styles.
	Styles *style.Config
	// Verbose enables debug level logging and timestamps.
	Verbose bool
}

// Configure sets up the [Default] logger with the given configuration.
// Call this once at application startup.
//
// Note: this respects the log level environment variable - it won't reset
// the level if CLOG_LOG_LEVEL (or a custom prefix equivalent) was set and
// cfg.Verbose is false.
func Configure(cfg *Config) {
	if cfg == nil {
		return
	}

	if cfg.Output != nil {
		Default.SetOutput(cfg.Output)
	}

	if cfg.Styles != nil {
		Default.SetStyles(cfg.Styles)
	}

	SetVerbose(cfg.Verbose)
}

// SetVerbose enables or disables verbose mode on the [Default] logger.
// When verbose is true, it always enables debug logging. When false, it
// respects the log level environment variable if set.
func SetVerbose(verbose bool) {
	if verbose {
		Default.SetLevel(LevelDebug)
		Default.SetReportTimestamp(true)
		return
	}

	// Respect the env var if set (custom prefix or CLOG_LOG_LEVEL).
	if getEnv(envLogLevel) != "" {
		return
	}

	Default.SetLevel(LevelInfo)
	Default.SetReportTimestamp(false)
}

// Package-level convenience functions that use the [Default] logger.

// SetAnimationInterval sets the minimum animation refresh interval on the [Default] logger.
func SetAnimationInterval(d time.Duration) { Default.SetAnimationInterval(d) }

// SetSuppressEchoDuringAnimations controls terminal echo suppression while
// animations are live on the [Default] logger.
func SetSuppressEchoDuringAnimations(suppress bool) {
	Default.SetSuppressEchoDuringAnimations(suppress)
}

// SetColorMode sets the color mode on the [Default] logger by recreating
// its [Output] with the given mode.
func SetColorMode(mode ColorMode) {
	Default.SetColorMode(mode)
}

// SetExitCode sets the default fatal exit code on the [Default] logger.
func SetExitCode(code int) { Default.SetExitCode(code) }

// SetExitFunc sets the fatal-exit function on the [Default] logger.
func SetExitFunc(fn func(int)) { Default.SetExitFunc(fn) }

// SetFieldFormats replaces the field-format configuration on the [Default] logger.
func SetFieldFormats(f FieldFormats) { Default.SetFieldFormats(f) }

// SetDurationFormat sets a custom duration formatter on the [Default] logger.
func SetDurationFormat(format func(time.Duration) string) { Default.SetDurationFormat(format) }

// SetDurationGradientMax sets the duration gradient maximum on the [Default] logger.
func SetDurationGradientMax(maximum time.Duration) { Default.SetDurationGradientMax(maximum) }

// SetDurationMinimum sets the minimum duration shown on the [Default] logger.
func SetDurationMinimum(minimum time.Duration) { Default.SetDurationMinimum(minimum) }

// SetDurationPrecision sets the duration display precision on the [Default] logger.
func SetDurationPrecision(precision int) { Default.SetDurationPrecision(precision) }

// SetDurationRound sets the duration rounding granularity on the [Default] logger.
func SetDurationRound(round time.Duration) { Default.SetDurationRound(round) }

// SetDurationScale sets the duration rounding/precision scale on the [Default] logger.
func SetDurationScale(scale TimeScale) { Default.SetDurationScale(scale) }

// SetElapsedFormat sets a custom elapsed formatter on the [Default] logger.
func SetElapsedFormat(format func(time.Duration) string) { Default.SetElapsedFormat(format) }

// SetElapsedGradientMax sets the elapsed gradient maximum on the [Default] logger.
func SetElapsedGradientMax(maximum time.Duration) { Default.SetElapsedGradientMax(maximum) }

// SetElapsedMinimum sets the minimum elapsed duration shown on the [Default] logger.
func SetElapsedMinimum(minimum time.Duration) { Default.SetElapsedMinimum(minimum) }

// SetElapsedPrecision sets the elapsed display precision on the [Default] logger.
func SetElapsedPrecision(precision int) { Default.SetElapsedPrecision(precision) }

// SetElapsedRound sets the elapsed rounding granularity on the [Default] logger.
func SetElapsedRound(round time.Duration) { Default.SetElapsedRound(round) }

// SetElapsedScale sets the elapsed and deadline rounding/precision scale on the [Default] logger.
func SetElapsedScale(scale TimeScale) { Default.SetElapsedScale(scale) }

// SetTimeScale sets the shared rounding/precision scale for all time fields on the [Default] logger.
func SetTimeScale(scale TimeScale) { Default.SetTimeScale(scale) }

// SetTimeGradientMax sets both the duration and elapsed gradient maxima on the [Default] logger.
func SetTimeGradientMax(maximum time.Duration) { Default.SetTimeGradientMax(maximum) }

// SetHyperlinkEnabled enables or disables hyperlink rendering on the [Default] logger.
func SetHyperlinkEnabled(enabled bool) { Default.SetHyperlinkEnabled(enabled) }

// SetHyperlinkColumnFormat sets the file+line+column hyperlink format on the [Default] logger.
func SetHyperlinkColumnFormat(format string) { Default.SetHyperlinkColumnFormat(format) }

// SetHyperlinkDirFormat sets the directory hyperlink format on the [Default] logger.
func SetHyperlinkDirFormat(format string) { Default.SetHyperlinkDirFormat(format) }

// SetHyperlinkFileFormat sets the file-only hyperlink format on the [Default] logger.
func SetHyperlinkFileFormat(format string) { Default.SetHyperlinkFileFormat(format) }

// SetHyperlinkLineFormat sets the file+line hyperlink format on the [Default] logger.
func SetHyperlinkLineFormat(format string) { Default.SetHyperlinkLineFormat(format) }

// SetHyperlinkPathFormat sets the generic path hyperlink format on the [Default] logger.
func SetHyperlinkPathFormat(format string) { Default.SetHyperlinkPathFormat(format) }

// SetPercentFormat sets a custom percent formatter on the [Default] logger.
func SetPercentFormat(format func(float64) string) { Default.SetPercentFormat(format) }

// SetPercentMaximum sets the percent input maximum on the [Default] logger.
func SetPercentMaximum(maximum float64) { Default.SetPercentMaximum(maximum) }

// SetPercentPrecision sets the percent display precision on the [Default] logger.
func SetPercentPrecision(precision int) { Default.SetPercentPrecision(precision) }

// SetPercentReverseGradient flips the percent gradient direction on the [Default] logger.
func SetPercentReverseGradient(reverse bool) { Default.SetPercentReverseGradient(reverse) }

// SetQuantityUnitsIgnoreCase toggles case-insensitive quantity units on the [Default] logger.
func SetQuantityUnitsIgnoreCase(ignore bool) { Default.SetQuantityUnitsIgnoreCase(ignore) }

// SetFieldSort sets the field sort order on the [Default] logger.
func SetFieldSort(sort Sort) { Default.SetFieldSort(sort) }

// SetFieldStyleLevel sets the minimum level for styled fields on the [Default] logger.
func SetFieldStyleLevel(level Level) { Default.SetFieldStyleLevel(level) }

// SetFieldTimeFormat sets the time format for time fields on the [Default] logger.
func SetFieldTimeFormat(format string) { Default.SetFieldTimeFormat(format) }

// SetHandler sets the log handler on the [Default] logger.
func SetHandler(h Handler) { Default.SetHandler(h) }

// AddHook registers a hook on the [Default] logger at the given [HookPoint].
func AddHook(point HookPoint, fn func()) { Default.AddHook(point, fn) }

// ClearAllHooks removes all registered hooks on the [Default] logger.
func ClearAllHooks() { Default.ClearAllHooks() }

// ClearHooks removes all hooks at the given [HookPoint] on the [Default] logger.
func ClearHooks(point HookPoint) { Default.ClearHooks(point) }

// SetIndent sets the indent depth on the [Default] logger.
func SetIndent(levels int) { Default.SetIndent(levels) }

// SetIndentPrefixes sets per-depth indent prefixes on the [Default] logger.
func SetIndentPrefixes(prefixes []string) { Default.SetIndentPrefixes(prefixes) }

// SetIndentPrefixSeparator sets the indent prefix separator on the [Default] logger.
func SetIndentPrefixSeparator(sep string) { Default.SetIndentPrefixSeparator(sep) }

// SetIndentWidth sets the indent width on the [Default] logger.
func SetIndentWidth(width int) { Default.SetIndentWidth(width) }

// SetLabelWidth sets an explicit minimum level-label width on the [Default] logger.
func SetLabelWidth(width int) { Default.SetLabelWidth(width) }

// SetLevel sets the minimum log level on the [Default] logger.
func SetLevel(level Level) { Default.SetLevel(level) }

// SetNonTTYLevel sets the minimum log level for non-TTY writers on the [Default] logger.
// Pass [UnsetLevel] to restore the default behaviour.
func SetNonTTYLevel(level Level) { Default.SetNonTTYLevel(level) }

// SetLevelAlign sets the level-label alignment on the [Default] logger.
func SetLevelAlign(align Align) { Default.SetLevelAlign(align) }

// SetLabels sets the level labels on the [Default] logger.
func SetLabels(labels LabelMap) { Default.SetLabels(labels) }

// SetNumberFormat sets the numeric format for integer and fraction fields on the [Default] logger.
func SetNumberFormat(format NumberFormat) { Default.SetNumberFormat(format) }

// SetFractionFormat overrides the numeric format for fraction fields on the [Default] logger.
func SetFractionFormat(format NumberFormat) { Default.SetFractionFormat(format) }

// SetNumberGroupSeparator sets the digit-group separator on the [Default] logger.
func SetNumberGroupSeparator(sep string) { Default.SetNumberGroupSeparator(sep) }

// SetNumberCompactMinimum sets the minimum magnitude for compact abbreviation on the [Default] logger.
func SetNumberCompactMinimum(minimum int64) { Default.SetNumberCompactMinimum(minimum) }

// SetNumberCompactFallback sets how compact mode renders sub-minimum values on the [Default] logger.
func SetNumberCompactFallback(format NumberFormat) { Default.SetNumberCompactFallback(format) }

// SetOmitEmpty enables or disables omitting empty fields on the [Default] logger.
func SetOmitEmpty(omit bool) { Default.SetOmitEmpty(omit) }

// SetOmitZero enables or disables omitting zero-value fields on the [Default] logger.
func SetOmitZero(omit bool) { Default.SetOmitZero(omit) }

// SetInput sets the reader used by [Input] and [Password] on the [Default] logger.
func SetInput(r io.Reader) { Default.SetInput(r) }

// SetPromptMarker sets the leading prompt marker on the [Default] logger; see
// [Logger.SetPromptMarker].
func SetPromptMarker(marker string) { Default.SetPromptMarker(marker) }

// SetOutput sets the output on the [Default] logger.
func SetOutput(out *Output) { Default.SetOutput(out) }

// SetOutputWriter sets the output writer on the [Default] logger with [ColorAuto].
func SetOutputWriter(w io.Writer) { Default.SetOutputWriter(w) }

// SetParts sets the log-line part order on the [Default] logger.
func SetParts(order ...Part) { Default.SetParts(order...) }

// SetTheme sets the printer theme pair on the [Default] logger.
func SetTheme(p *theme.Pair) { Default.SetTheme(p) }

// SetPrintIndent sets the printer indentation string on the [Default] logger.
func SetPrintIndent(indent string) { Default.SetPrintIndent(indent) }

// SetJSONIndent sets a JSON-specific indent on the [Default] logger.
func SetJSONIndent(indent string) { Default.SetJSONIndent(indent) }

// SetJSONPrintMode sets the default [JSONPrintMode] on the [Default] logger.
func SetJSONPrintMode(mode JSONPrintMode) { Default.SetJSONPrintMode(mode) }

// SetYAMLIndent sets a YAML-specific indent on the [Default] logger.
func SetYAMLIndent(indent string) { Default.SetYAMLIndent(indent) }

// SetYAMLIndentSequence controls YAML sequence indentation on the [Default] logger.
func SetYAMLIndentSequence(indent bool) { Default.SetYAMLIndentSequence(indent) }

// SetSymbols sets the level symbols on the [Default] logger.
func SetSymbols(symbols LabelMap) { Default.SetSymbols(symbols) }

// SetQuoteChars sets the opening and closing quote characters on the [Default] logger.
func SetQuoteChars(openChar, closeChar rune) { Default.SetQuoteChars(openChar, closeChar) }

// SetQuote sets the quoting behaviour on the [Default] logger.
func SetQuote(mode Quote) { Default.SetQuote(mode) }

// SetSmartQuotes enables or disables content-adaptive quoting on the [Default] logger.
func SetSmartQuotes(enabled bool) { Default.SetSmartQuotes(enabled) }

// SetSmartQuoteChars sets the smart-quote delimiter preference order on the [Default] logger.
func SetSmartQuoteChars(pairs ...QuotePair) { Default.SetSmartQuoteChars(pairs...) }

// SetReportTimestamp enables or disables timestamps on the [Default] logger.
func SetReportTimestamp(report bool) { Default.SetReportTimestamp(report) }

// SetSeparatorText sets the key/value separator on the [Default] logger.
func SetSeparatorText(sep string) { Default.SetSeparatorText(sep) }

// SetSliceBrackets sets separate slice open/close bracket characters on the [Default] logger.
func SetSliceBrackets(openChar, closeChar rune) { Default.SetSliceBrackets(openChar, closeChar) }

// SetSliceSeparator sets the slice element separator on the [Default] logger.
func SetSliceSeparator(sep string) { Default.SetSliceSeparator(sep) }

// SetSpinnerDefaults sets the default spinner configuration on the [Default] logger.
func SetSpinnerDefaults(opts ...spinner.Option) { Default.SetSpinnerDefaults(opts...) }

// SetStyles sets the display styles on the [Default] logger.
func SetStyles(styles *style.Config) { Default.SetStyles(styles) }

// SetTimeFormat sets the timestamp format on the [Default] logger.
func SetTimeFormat(format string) { Default.SetTimeFormat(format) }

// SetTimeLocation sets the timestamp timezone on the [Default] logger.
func SetTimeLocation(loc *time.Location) { Default.SetTimeLocation(loc) }

// SetTreeChars sets the tree-drawing characters on the [Default] logger.
func SetTreeChars(chars TreeChars) { Default.SetTreeChars(chars) }

// SetWrap sets the line wrapping behaviour on the [Default] logger.
func SetWrap(wrap Wrap) { Default.SetWrap(wrap) }

// DefaultLabels returns a copy of the default level labels.
func DefaultLabels() LabelMap {
	m := level.Labels()
	result := make(LabelMap, len(m))
	maps.Copy(result, m)
	return result
}

// DefaultParts returns the default ordering of log line parts:
// timestamp, level, symbol, message, fields.
func DefaultParts() []Part {
	return []Part{PartTimestamp, PartLevel, PartSymbol, PartMessage, PartFields}
}

// Default emoji symbols for each level.
var defaultSymbols = LabelMap{
	LevelTrace: "🔍",
	LevelDebug: "🐞",
	LevelInfo:  "ℹ️",
	LevelHint:  "💡",
	LevelDry:   "🚧",
	LevelWarn:  "⚠️",
	LevelError: "❌",
	LevelFatal: "💥",
}

// defaultMaxLabelLen is the maximum length of an auto-generated level label.
const defaultMaxLabelLen = 3

// DefaultSymbols returns a copy of the default emoji symbols for each level.
func DefaultSymbols() LabelMap {
	return maps.Clone(defaultSymbols)
}

// DefaultTreeChars returns the default box-drawing characters for tree
// indentation.
func DefaultTreeChars() TreeChars {
	return TreeChars{
		First:    "├── ",
		Middle:   "├── ",
		Last:     "└── ",
		Continue: "│   ",
		Blank:    "    ",
	}
}
