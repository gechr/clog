// Package clog provides structured CLI logging with terminal-aware colours,
// hyperlinks, and spinners.
//
// It uses a zerolog-style fluent API for building log entries:
//
//	clog.Info().Str("port", "8080").Msg("Server started")
//
// The default output is a pretty terminal formatter. A custom [Handler] can
// be set for alternative formats (e.g. JSON).
package clog

import (
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultEnvLogLevel is the default environment variable checked for the log level.
	DefaultEnvLogLevel = "CLOG_LEVEL"

	// DefaultEnvSeparator is the environment variable checked for the key-value separator.
	DefaultEnvSeparator = "CLOG_SEPARATOR"
)

// ErrorKey is the default field key used by [Event.Err] and [Context.Err].
const ErrorKey = "error"

const (
	// LevelTrace is the "trace" level string for [SetLevelFromEnv].
	LevelTrace = "trace"
	// LevelDebug is the "debug" level string for [SetLevelFromEnv].
	LevelDebug = "debug"
	// LevelInfo is the "info" level string.
	LevelInfo = "info"
	// LevelDry is the "dry" level string.
	LevelDry = "dry"
	// LevelWarn is the "warn" level string.
	LevelWarn = "warn"
	// LevelWarning is the "warning" level string (alias for warn).
	LevelWarning = "warning"
	// LevelError is the "error" level string.
	LevelError = "error"
	// LevelFatal is the "fatal" level string.
	LevelFatal = "fatal"
)

const nilStr = "<nil>"

// Default is the default logger instance.
var Default = New(os.Stdout)

// Default emoji prefixes for each level.
var defaultPrefixes = LevelMap{
	TraceLevel: "🔬",
	DebugLevel: "🔍",
	InfoLevel:  "ℹ️",
	DryLevel:   "🚧",
	WarnLevel:  "⚠️",
	ErrorLevel: "❌",
	FatalLevel: "‼️",
}

// levelLabels are the short text labels for each level.
var levelLabels = LevelMap{
	TraceLevel: "TRC",
	DebugLevel: "DBG",
	InfoLevel:  "INF",
	DryLevel:   "DRY",
	WarnLevel:  "WRN",
	ErrorLevel: "ERR",
	FatalLevel: "FTL",
}

// Level represents a log level.
type Level int

const (
	TraceLevel Level = iota
	DebugLevel
	InfoLevel
	DryLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

// String returns the short label for the level (e.g. "INF", "ERR").
func (l Level) String() string {
	if s, ok := levelLabels[l]; ok {
		return s
	}

	return fmt.Sprintf("LVL(%d)", int(l))
}

// LevelMap maps levels to strings (used for labels, prefixes, etc.).
type LevelMap map[Level]string

// LevelAlign controls how level labels are aligned when they have different widths.
type LevelAlign int

const (
	// AlignNone disables alignment padding.
	AlignNone LevelAlign = iota
	// AlignLeft left-aligns labels (pads with trailing spaces).
	AlignLeft
	// AlignRight right-aligns labels (pads with leading spaces). This is the default.
	AlignRight
	// AlignCenter center-aligns labels (pads with leading and trailing spaces).
	AlignCenter
)

// ColorMode controls how a [Logger] determines colour and hyperlink output.
type ColorMode int

const (
	// ColorAuto uses global detection (terminal, NO_COLOR, etc.). This is the default.
	ColorAuto ColorMode = iota
	// ColorAlways forces colours and hyperlinks, even when output is not a TTY.
	ColorAlways
	// ColorNever disables colours and hyperlinks.
	ColorNever
)

// QuoteMode controls how field values are quoted in log output.
type QuoteMode int

const (
	// QuoteAuto quotes values only when they contain spaces, unprintable
	// characters, or embedded quotes. This is the default.
	QuoteAuto QuoteMode = iota
	// QuoteAlways quotes all string, error, and default-kind values.
	QuoteAlways
	// QuoteNever disables quoting entirely.
	QuoteNever
)

// Part identifies a component of a formatted log line.
type Part int

const (
	// PartTimestamp is the timestamp component.
	PartTimestamp Part = iota
	// PartLevel is the level label component.
	PartLevel
	// PartPrefix is the emoji prefix component.
	PartPrefix
	// PartMessage is the log message component.
	PartMessage
	// PartFields is the structured fields component.
	PartFields
)

// Logger is the main structured logger.
type Logger struct {
	mu *sync.Mutex

	colorMode       ColorMode
	exitFunc        func(int) // called by Fatal-level events; defaults to os.Exit
	fieldTimeFormat string
	fields          []Field
	fieldStyleLevel Level
	handler         Handler
	labels          LevelMap
	level           Level
	levelAlign      LevelAlign
	omitEmpty       bool
	omitZero        bool
	quoteMode       QuoteMode
	out             io.Writer
	quoteOpen       rune // 0 means default ('"' via strconv.Quote)
	quoteClose      rune // 0 means same as quoteOpen (or default)
	parts           []Part
	prefix          *string // nil = use default emoji for level
	prefixes        LevelMap
	reportTimestamp bool
	styles          *Styles
	timeFormat      string
	timeLocation    *time.Location
}

// New creates a new [Logger] that writes to out.
func New(out io.Writer) *Logger {
	return &Logger{
		mu: &sync.Mutex{},

		exitFunc:        os.Exit,
		fieldTimeFormat: time.RFC3339,
		fieldStyleLevel: InfoLevel,
		labels:          DefaultLabels(),
		level:           InfoLevel,
		levelAlign:      AlignRight,
		out:             out,
		parts:           DefaultParts(),
		prefixes:        DefaultPrefixes(),
		styles:          DefaultStyles(),
		timeFormat:      "15:04:05.000",
		timeLocation:    time.Local,
	}
}

// SetOutput sets the output writer.
func (l *Logger) SetOutput(out io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.out = out
}

// SetLevel sets the minimum log level.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.level = level
}

// SetStyles sets the display styles.
func (l *Logger) SetStyles(styles *Styles) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.styles = styles
}

// SetReportTimestamp enables or disables timestamp reporting.
func (l *Logger) SetReportTimestamp(report bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.reportTimestamp = report
}

// SetTimeFormat sets the timestamp format string.
func (l *Logger) SetTimeFormat(format string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.timeFormat = format
}

// SetTimeLocation sets the timezone for timestamps. Defaults to [time.Local].
func (l *Logger) SetTimeLocation(loc *time.Location) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.timeLocation = loc
}

// SetFieldStyleLevel sets the minimum log level at which field values are
// styled (coloured). Events below this level render fields as plain text.
// Defaults to [InfoLevel].
func (l *Logger) SetFieldStyleLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.fieldStyleLevel = level
}

// SetFieldTimeFormat sets the format string used for [time.Time] field values
// added via [Event.Time] and [Context.Time]. Defaults to [time.RFC3339].
func (l *Logger) SetFieldTimeFormat(format string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.fieldTimeFormat = format
}

// SetHandler sets a custom log handler. When set, the handler receives all
// log entries instead of the built-in pretty formatter.
func (l *Logger) SetHandler(h Handler) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.handler = h
}

// SetLabels sets the level labels used in log output.
// Pass a map from [Level] to label string (e.g. {WarnLevel: "WARN"}).
// Missing levels fall back to the defaults.
func (l *Logger) SetLabels(labels LevelMap) {
	l.mu.Lock()
	defer l.mu.Unlock()

	merged := DefaultLabels()
	maps.Copy(merged, labels)

	l.labels = merged
}

// SetPrefixes sets the emoji prefixes used for each level.
// Pass a map from [Level] to prefix string. Missing levels fall back to the defaults.
func (l *Logger) SetPrefixes(prefixes LevelMap) {
	l.mu.Lock()
	defer l.mu.Unlock()

	merged := DefaultPrefixes()
	maps.Copy(merged, prefixes)

	l.prefixes = merged
}

// SetExitFunc sets the function called by Fatal-level events.
// Defaults to [os.Exit]. This can be used in tests to intercept fatal exits.
func (l *Logger) SetExitFunc(fn func(int)) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.exitFunc = fn
}

// SetOmitEmpty enables or disables omitting fields with empty values.
// Empty means nil, empty strings, and nil or empty slices/maps.
func (l *Logger) SetOmitEmpty(omit bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.omitEmpty = omit
}

// SetQuoteMode sets the quoting behaviour for field values.
// [QuoteAuto] (default) quotes only when needed; [QuoteAlways] always quotes
// string/error/default-kind values; [QuoteNever] never quotes.
func (l *Logger) SetQuoteMode(mode QuoteMode) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.quoteMode = mode
}

// SetOmitQuotes disables quoting of field values that contain spaces or
// special characters. By default, such values are wrapped in quotes for
// parseable output.
//
// Deprecated: Use [Logger.SetQuoteMode] with [QuoteNever] or [QuoteAuto] instead.
func (l *Logger) SetOmitQuotes(omit bool) {
	if omit {
		l.SetQuoteMode(QuoteNever)
	} else {
		l.SetQuoteMode(QuoteAuto)
	}
}

// SetOmitZero enables or disables omitting fields with zero values.
// Zero means the zero value for any type (0, false, "", nil, etc.).
// This is a superset of [Logger.SetOmitEmpty].
func (l *Logger) SetOmitZero(omit bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.omitZero = omit
}

// SetQuoteChar sets the character used to quote field values that contain
// spaces or special characters. The default (zero value) uses Go-style
// double-quoted strings via [strconv.Quote]. Setting a non-zero rune wraps
// values with that character on both sides (e.g. '\”).
//
// For asymmetric quotes (e.g. '[' and ']'), use [Logger.SetQuoteChars].
func (l *Logger) SetQuoteChar(char rune) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.quoteOpen = char
	l.quoteClose = char
}

// SetQuoteChars sets separate opening and closing characters for quoting
// field values (e.g. '[' and ']', or '«' and '»').
func (l *Logger) SetQuoteChars(openChar, closeChar rune) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.quoteOpen = openChar
	l.quoteClose = closeChar
}

// SetColorMode sets the colour mode for this logger.
// [ColorAuto] (default) uses global detection; [ColorAlways] forces colours
// and hyperlinks; [ColorNever] disables them.
func (l *Logger) SetColorMode(mode ColorMode) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.colorMode = mode
}

// SetLevelAlign sets the alignment mode for level labels.
func (l *Logger) SetLevelAlign(align LevelAlign) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.levelAlign = align
}

// SetParts sets the order in which parts appear in log output.
// Parts not included in the order are hidden. Parts can be reordered freely.
// Panics if no parts are provided.
func (l *Logger) SetParts(parts ...Part) {
	if len(parts) == 0 {
		panic("clog: SetParts requires at least one part")
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.parts = parts
}

// With returns a [Context] for building a sub-logger with preset fields.
//
//	logger := clog.With().Str("component", "auth").Logger()
//	logger.Info().Str("user", "john").Msg("Authenticated")
func (l *Logger) With() *Context {
	l.mu.Lock()
	defer l.mu.Unlock()

	fields := make([]Field, len(l.fields))
	copy(fields, l.fields)

	c := &Context{
		logger: l,
		prefix: l.prefix,
	}
	c.fields = fields
	c.initSelf(c)

	return c
}

// Trace returns a new [Event] at trace level, or nil if trace is disabled.
func (l *Logger) Trace() *Event { return l.newEvent(TraceLevel) }

// Debug returns a new [Event] at debug level, or nil if debug is disabled.
func (l *Logger) Debug() *Event { return l.newEvent(DebugLevel) }

// Info returns a new [Event] at info level, or nil if info is disabled.
func (l *Logger) Info() *Event { return l.newEvent(InfoLevel) }

// Dry returns a new [Event] at dry level, or nil if dry is disabled.
func (l *Logger) Dry() *Event { return l.newEvent(DryLevel) }

// Warn returns a new [Event] at warn level, or nil if warn is disabled.
func (l *Logger) Warn() *Event { return l.newEvent(WarnLevel) }

// Error returns a new [Event] at error level, or nil if error is disabled.
func (l *Logger) Error() *Event { return l.newEvent(ErrorLevel) }

// Fatal returns a new [Event] at fatal level.
func (l *Logger) Fatal() *Event { return l.newEvent(FatalLevel) }

// newEvent creates a new [Event] for the given level.
// Returns nil if the level is below the logger's minimum (all Event methods
// are no-ops on nil).
func (l *Logger) newEvent(level Level) *Event {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level {
		return nil
	}

	return &Event{
		logger: l,
		level:  level,
	}
}

// log writes a log entry using either the custom handler or the built-in pretty formatter.
func (l *Logger) log(e *Event, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Merge logger context fields with event fields.
	allFields := slices.Concat(l.fields, e.fields)

	if l.omitZero {
		allFields = slices.DeleteFunc(allFields, func(f Field) bool {
			return isZeroValue(f.Value)
		})
	} else if l.omitEmpty {
		allFields = slices.DeleteFunc(allFields, func(f Field) bool {
			return isEmptyValue(f.Value)
		})
	}

	prefix := l.resolvePrefix(e)

	// Delegate to custom handler if set.
	if l.handler != nil {
		entry := Entry{
			Level:   e.level,
			Message: msg,
			Prefix:  prefix,
			Fields:  allFields,
		}
		if l.reportTimestamp {
			entry.Time = time.Now().In(l.timeLocation)
		}

		l.handler.Log(entry)

		return
	}

	// Built-in pretty formatter.
	noColor := l.colorsDisabled()

	parts := make([]string, 0, len(l.parts))

	for _, p := range l.parts {
		var s string

		switch p {
		case PartTimestamp:
			if !l.reportTimestamp {
				continue
			}

			ts := time.Now().In(l.timeLocation).Format(l.timeFormat)
			if noColor || l.styles.Timestamp == nil {
				s = ts
			} else {
				s = l.styles.Timestamp.Render(ts)
			}
		case PartLevel:
			label := l.formatLabel(e.level)
			if style := l.styles.Levels[e.level]; !noColor && style != nil {
				s = style.Render(label)
			} else {
				s = label
			}
		case PartPrefix:
			if prefix == "" {
				continue
			}

			s = prefix
		case PartMessage:
			if msg == "" {
				continue
			}

			if style := l.styles.Messages[e.level]; !noColor && style != nil {
				s = style.Render(msg)
			} else {
				s = msg
			}
		case PartFields:
			s = strings.TrimLeft(formatFields(allFields, formatFieldsOpts{
				fieldStyleLevel: l.fieldStyleLevel,
				level:           e.level,
				noColor:         noColor,
				quoteClose:      l.quoteClose,
				quoteMode:       l.quoteMode,
				quoteOpen:       l.quoteOpen,
				styles:          l.styles,
				timeFormat:      l.fieldTimeFormat,
			}), " ")
		}

		if s != "" {
			parts = append(parts, s)
		}
	}

	_, _ = io.WriteString(l.out, strings.Join(parts, " ")+"\n")
}

// exit calls the logger's exit function (used by Fatal-level events).
func (l *Logger) exit(code int) {
	l.mu.Lock()
	fn := l.exitFunc
	l.mu.Unlock()

	fn(code)
}

// colorsDisabled returns true if this logger should suppress colours.
func (l *Logger) colorsDisabled() bool {
	switch l.colorMode {
	case ColorAlways:
		return false
	case ColorNever:
		return true
	case ColorAuto:
		return ColorsDisabled()
	}

	return ColorsDisabled()
}

// formatLabel returns the level label, padded according to the logger's alignment setting.
func (l *Logger) formatLabel(level Level) string {
	label := l.labels[level]

	maxW := l.maxLabelWidth()

	switch l.levelAlign {
	case AlignNone:
		return label
	case AlignLeft:
		return fmt.Sprintf("%-*s", maxW, label)
	case AlignRight:
		return fmt.Sprintf("%*s", maxW, label)
	case AlignCenter:
		return centerPad(label, maxW)
	}

	return label
}

// maxLabelWidth returns the length of the longest configured label.
func (l *Logger) maxLabelWidth() int {
	maxWidth := 0
	for _, lbl := range l.labels {
		if len(lbl) > maxWidth {
			maxWidth = len(lbl)
		}
	}

	return maxWidth
}

// resolvePrefix returns the appropriate prefix for a log entry, checking
// event override → logger preset → default for level.
func (l *Logger) resolvePrefix(e *Event) string {
	if e.prefix != nil {
		return *e.prefix
	}

	if l.prefix != nil {
		return *l.prefix
	}

	return l.prefixes[e.level]
}

// Config holds configuration options for the [Default] logger.
type Config struct {
	// Verbose enables debug level logging and timestamps.
	Verbose bool
	// Output is the writer to use (defaults to [os.Stdout]).
	Output io.Writer
	// Styles allows customising the visual styles.
	Styles *Styles
}

// Configure sets up the [Default] logger with the given configuration.
// Call this once at application startup.
//
// Note: this respects the CLOG_LEVEL environment variable — it won't reset
// the level if CLOG_LEVEL was set and cfg.Verbose is false.
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

	ConfigureVerbose(cfg.Verbose)
}

// ConfigureVerbose enables or disables verbose mode on the [Default] logger.
// When verbose is true, it always enables debug logging. When false, it
// respects the CLOG_LEVEL environment variable if set.
func ConfigureVerbose(verbose bool) {
	if verbose {
		Default.SetLevel(DebugLevel)
		Default.SetReportTimestamp(true)
	} else if os.Getenv(DefaultEnvLogLevel) == "" {
		Default.SetLevel(InfoLevel)
		Default.SetReportTimestamp(false)
	}
}

// SetLevelFromEnv reads the named environment variable and sets the level
// on the [Default] logger. Recognised values: trace, debug, info, dry, warn,
// warning, error, fatal.
func SetLevelFromEnv(envVar string) {
	level := os.Getenv(envVar)
	if level == "" {
		return
	}

	switch strings.ToLower(level) {
	case LevelTrace:
		Default.SetLevel(TraceLevel)
		Default.SetReportTimestamp(true)
	case LevelDebug:
		Default.SetLevel(DebugLevel)
		Default.SetReportTimestamp(true)
	case LevelInfo:
		Default.SetLevel(InfoLevel)
	case LevelDry:
		Default.SetLevel(DryLevel)
	case LevelWarn, LevelWarning:
		Default.SetLevel(WarnLevel)
	case LevelError:
		Default.SetLevel(ErrorLevel)
	case LevelFatal:
		Default.SetLevel(FatalLevel)
	default:
		fmt.Fprintf(os.Stderr, "clog: unrecognised log level %q in %s\n", level, envVar)
	}
}

// SetSeparatorFromEnv reads the named environment variable and sets the
// key-value separator on the [Default] logger's styles.
func SetSeparatorFromEnv(envVar string) {
	sep := os.Getenv(envVar)
	if sep == "" {
		return
	}

	Default.mu.Lock()
	defer Default.mu.Unlock()

	Default.styles.SeparatorText = sep
}

// DefaultLabels returns a copy of the default level labels.
func DefaultLabels() LevelMap {
	return maps.Clone(levelLabels)
}

// DefaultParts returns the default ordering of log line parts:
// timestamp, level, prefix, message, fields.
func DefaultParts() []Part {
	return []Part{PartTimestamp, PartLevel, PartPrefix, PartMessage, PartFields}
}

// DefaultPrefixes returns a copy of the default emoji prefixes for each level.
func DefaultPrefixes() LevelMap {
	return maps.Clone(defaultPrefixes)
}

// GetLevel returns the current log level of the [Default] logger.
func GetLevel() Level {
	Default.mu.Lock()
	defer Default.mu.Unlock()

	return Default.level
}

// IsVerbose returns true if verbose/debug mode is enabled on the [Default] logger.
// Returns true for both [TraceLevel] and [DebugLevel].
func IsVerbose() bool {
	return GetLevel() <= DebugLevel
}

// Package-level convenience functions that use the [Default] logger.

func SetColorMode(mode ColorMode)            { Default.SetColorMode(mode) }
func SetExitFunc(fn func(int))               { Default.SetExitFunc(fn) }
func SetHandler(h Handler)                   { Default.SetHandler(h) }
func SetLevel(level Level)                   { Default.SetLevel(level) }
func SetLevelAlign(align LevelAlign)         { Default.SetLevelAlign(align) }
func SetLevelLabels(labels LevelMap)         { Default.SetLabels(labels) }
func SetOmitEmpty(omit bool)                 { Default.SetOmitEmpty(omit) }
func SetOmitQuotes(omit bool)                { Default.SetOmitQuotes(omit) }
func SetOmitZero(omit bool)                  { Default.SetOmitZero(omit) }
func SetOutput(out io.Writer)                { Default.SetOutput(out) }
func SetQuoteChar(char rune)                 { Default.SetQuoteChar(char) }
func SetQuoteChars(openChar, closeChar rune) { Default.SetQuoteChars(openChar, closeChar) }
func SetQuoteMode(mode QuoteMode)            { Default.SetQuoteMode(mode) }
func SetParts(order ...Part)                 { Default.SetParts(order...) }
func SetPrefixes(prefixes LevelMap)          { Default.SetPrefixes(prefixes) }
func SetReportTimestamp(report bool)         { Default.SetReportTimestamp(report) }
func SetStyles(styles *Styles)               { Default.SetStyles(styles) }
func SetFieldStyleLevel(level Level)         { Default.SetFieldStyleLevel(level) }
func SetFieldTimeFormat(format string)       { Default.SetFieldTimeFormat(format) }
func SetTimeFormat(format string)            { Default.SetTimeFormat(format) }
func SetTimeLocation(loc *time.Location)     { Default.SetTimeLocation(loc) }

func With() *Context { return Default.With() }
func Dict() *Event   { return &Event{} }

func Trace() *Event { return Default.Trace() }
func Debug() *Event { return Default.Debug() }
func Info() *Event  { return Default.Info() }
func Dry() *Event   { return Default.Dry() }
func Warn() *Event  { return Default.Warn() }
func Error() *Event { return Default.Error() }
func Fatal() *Event { return Default.Fatal() }

// centerPad centres s within width, padding with spaces.
func centerPad(s string, width int) string {
	pad := width - len(s)
	left := pad / 2 //nolint:mnd // half the padding goes left
	right := pad - left

	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

func init() {
	SetLevelFromEnv(DefaultEnvLogLevel)
	SetSeparatorFromEnv(DefaultEnvSeparator)
}
