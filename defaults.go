package clog

import (
	"io"
	"maps"
	"time"

	"github.com/gechr/clog/fx/spinner"
	"github.com/gechr/clog/level"
	"github.com/gechr/clog/style"
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

// SetColorMode sets the color mode on the [Default] logger by recreating
// its [Output] with the given mode.
func SetColorMode(mode ColorMode) {
	Default.SetColorMode(mode)
}

// SetExitCode sets the default fatal exit code on the [Default] logger.
func SetExitCode(code int) { Default.SetExitCode(code) }

// SetExitFunc sets the fatal-exit function on the [Default] logger.
func SetExitFunc(fn func(int)) { Default.SetExitFunc(fn) }

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

// SetLevel sets the minimum log level on the [Default] logger.
func SetLevel(level Level) { Default.SetLevel(level) }

// SetNonTTYLevel sets the minimum log level for non-TTY writers on the [Default] logger.
// Pass [UnsetLevel] to restore the default behaviour.
func SetNonTTYLevel(level Level) { Default.SetNonTTYLevel(level) }

// SetLevelAlign sets the level-label alignment on the [Default] logger.
func SetLevelAlign(align Align) { Default.SetLevelAlign(align) }

// SetLabels sets the level labels on the [Default] logger.
func SetLabels(labels LabelMap) { Default.SetLabels(labels) }

// SetOmitEmpty enables or disables omitting empty fields on the [Default] logger.
func SetOmitEmpty(omit bool) { Default.SetOmitEmpty(omit) }

// SetOmitZero enables or disables omitting zero-value fields on the [Default] logger.
func SetOmitZero(omit bool) { Default.SetOmitZero(omit) }

// SetOutput sets the output on the [Default] logger.
func SetOutput(out *Output) { Default.SetOutput(out) }

// SetOutputWriter sets the output writer on the [Default] logger with [ColorAuto].
func SetOutputWriter(w io.Writer) { Default.SetOutputWriter(w) }

// SetParts sets the log-line part order on the [Default] logger.
func SetParts(order ...Part) { Default.SetParts(order...) }

// SetSymbols sets the level symbols on the [Default] logger.
func SetSymbols(symbols LabelMap) { Default.SetSymbols(symbols) }

// SetQuoteChar sets the quote character on the [Default] logger.
func SetQuoteChar(char rune) { Default.SetQuoteChar(char) }

// SetQuoteChars sets the opening and closing quote characters on the [Default] logger.
func SetQuoteChars(openChar, closeChar rune) { Default.SetQuoteChars(openChar, closeChar) }

// SetQuoteMode sets the quoting behaviour on the [Default] logger.
func SetQuoteMode(mode QuoteMode) { Default.SetQuoteMode(mode) }

// SetReportTimestamp enables or disables timestamps on the [Default] logger.
func SetReportTimestamp(report bool) { Default.SetReportTimestamp(report) }

// SetSeparatorText sets the key/value separator on the [Default] logger.
func SetSeparatorText(sep string) { Default.SetSeparatorText(sep) }

// SetSliceBracket sets the same slice open/close bracket character on the [Default] logger.
func SetSliceBracket(char rune) { Default.SetSliceBracket(char) }

// SetSliceBrackets sets separate slice open/close bracket characters on the [Default] logger.
func SetSliceBrackets(openChar, closeChar rune) { Default.SetSliceBrackets(openChar, closeChar) }

// SetSliceSeparator sets the slice element separator on the [Default] logger.
func SetSliceSeparator(sep string) { Default.SetSliceSeparator(sep) }

// SetSpinnerStyle sets the default spinner style on the [Default] logger.
func SetSpinnerStyle(s spinner.Style) { Default.SetSpinnerStyle(s) }

// SetStyles sets the display styles on the [Default] logger.
func SetStyles(styles *style.Config) { Default.SetStyles(styles) }

// SetTimeFormat sets the timestamp format on the [Default] logger.
func SetTimeFormat(format string) { Default.SetTimeFormat(format) }

// SetTimeLocation sets the timestamp timezone on the [Default] logger.
func SetTimeLocation(loc *time.Location) { Default.SetTimeLocation(loc) }

// SetTreeChars sets the tree-drawing characters on the [Default] logger.
func SetTreeChars(chars TreeChars) { Default.SetTreeChars(chars) }

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
