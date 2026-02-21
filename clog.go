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
	"context"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ErrorKey is the default field key used by [Event.Err] and [Context.Err].
const ErrorKey = "error"

const (
	// LevelTrace is the "trace" level string.
	LevelTrace = "trace"
	// LevelDebug is the "debug" level string.
	LevelDebug = "debug"
	// LevelInfo is the "info" level string.
	LevelInfo = "info"
	// LevelDry is the "dry" level string.
	LevelDry = "dry"
	// LevelWarn is the "warn" level string.
	LevelWarn = "warn"
	// LevelError is the "error" level string.
	LevelError = "error"
	// LevelFatal is the "fatal" level string.
	LevelFatal = "fatal"
)

// Nil is the string representation used for nil values (e.g. in [DefaultValueStyles]).
const Nil = "<nil>"

// Default is the default logger instance.
var Default = New(Stdout(ColorAuto))

// ctxKey is the private context key used by [Logger.WithContext] and [Ctx].
type ctxKey struct{}

// Default emoji prefixes for each level.
var defaultPrefixes = LevelMap{
	TraceLevel: "🔍",
	DebugLevel: "🐞",
	InfoLevel:  "ℹ️",
	DryLevel:   "🚧",
	WarnLevel:  "⚠️",
	ErrorLevel: "❌",
	FatalLevel: "💥",
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
//
// Level implements [encoding.TextMarshaler] and [encoding.TextUnmarshaler],
// so it works directly with [flag.TextVar] and most flag libraries.
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

// levelNames maps Level constants to their canonical lowercase names.
var levelNames = map[Level]string{
	TraceLevel: LevelTrace,
	DebugLevel: LevelDebug,
	InfoLevel:  LevelInfo,
	DryLevel:   LevelDry,
	WarnLevel:  LevelWarn,
	ErrorLevel: LevelError,
	FatalLevel: LevelFatal,
}

// String returns the short label for the level (e.g. "INF", "ERR").
func (l Level) String() string {
	if s, ok := levelLabels[l]; ok {
		return s
	}
	return fmt.Sprintf("LVL(%d)", int(l))
}

// MarshalText implements [encoding.TextMarshaler].
func (l Level) MarshalText() ([]byte, error) {
	if name, ok := levelNames[l]; ok {
		return []byte(name), nil
	}
	return nil, fmt.Errorf("unknown level: %d", int(l))
}

// UnmarshalText implements [encoding.TextUnmarshaler].
func (l *Level) UnmarshalText(text []byte) error {
	parsed, err := ParseLevel(string(text))
	if err != nil {
		return err
	}
	*l = parsed
	return nil
}

// ParseLevel maps a level name string to a [Level] value.
// It accepts the canonical names ("trace", "debug", "info", "dry", "warn",
// "error", "fatal") plus aliases ("warning" → Warn, "critical" → Fatal).
// Matching is case-insensitive.
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(s) {
	case LevelTrace:
		return TraceLevel, nil
	case LevelDebug:
		return DebugLevel, nil
	case LevelInfo:
		return InfoLevel, nil
	case LevelDry:
		return DryLevel, nil
	case LevelWarn, "warning":
		return WarnLevel, nil
	case LevelError:
		return ErrorLevel, nil
	case LevelFatal, "critical":
		return FatalLevel, nil
	default:
		return 0, fmt.Errorf("unknown level: %q", s)
	}
}

// LevelMap maps levels to strings (used for labels, prefixes, etc.).
type LevelMap map[Level]string

// Align controls how text is aligned within a fixed-width column.
type Align int

const (
	// AlignNone disables alignment padding.
	AlignNone Align = iota
	// AlignLeft left-aligns text (pads with trailing spaces).
	AlignLeft
	// AlignRight right-aligns text (pads with leading spaces).
	AlignRight
	// AlignCenter center-aligns text (pads with leading and trailing spaces).
	AlignCenter
)

// ColorMode controls how a [Logger] determines colour and hyperlink output.
//
// ColorMode implements [encoding.TextMarshaler] and [encoding.TextUnmarshaler],
// so it works directly with [flag.TextVar] and most flag libraries.
//
//go:generate go tool golang.org/x/tools/cmd/stringer -type=ColorMode -linecomment
type ColorMode int

const (
	// ColorAuto uses global detection (terminal, NO_COLOR, etc.). This is the default.
	ColorAuto ColorMode = iota // auto
	// ColorAlways forces colours and hyperlinks, even when output is not a TTY.
	ColorAlways // always
	// ColorNever disables colours and hyperlinks.
	ColorNever // never
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

	atomicLevel             atomic.Int32 // lock-free level check for newEvent() hot path
	elapsedFormatFunc       func(time.Duration) string
	elapsedMinimum          time.Duration
	elapsedPrecision        int
	elapsedRound            time.Duration
	exitFunc                func(int) // called by Fatal-level events; defaults to os.Exit
	fieldSort               Sort
	fieldStyleLevel         Level
	fieldTimeFormat         string
	fields                  []Field
	handler                 Handler
	labelWidth              int
	labels                  LevelMap
	labelsPadded            LevelMap
	level                   Level
	levelAlign              Align
	omitEmpty               bool
	omitZero                bool
	output                  *Output
	parts                   []Part
	percentFormatFunc       func(float64) string
	percentPrecision        int
	prefix                  *string // nil = use default emoji for level
	prefixes                LevelMap
	quantityUnitsIgnoreCase bool
	quoteClose              rune // 0 means same as quoteOpen (or default)
	quoteMode               QuoteMode
	quoteOpen               rune // 0 means default ('"' via strconv.Quote)
	reportTimestamp         bool
	separatorText           string
	styles                  *Styles
	timeFormat              string
	timeLocation            *time.Location
}

// New creates a new [Logger] that writes to the given [Output].
func New(output *Output) *Logger {
	l := &Logger{
		mu: &sync.Mutex{},

		elapsedMinimum:          time.Second,
		elapsedRound:            time.Second,
		exitFunc:                os.Exit,
		fieldStyleLevel:         InfoLevel,
		fieldTimeFormat:         time.RFC3339,
		labels:                  DefaultLabels(),
		level:                   InfoLevel,
		levelAlign:              AlignRight,
		output:                  output,
		parts:                   DefaultParts(),
		prefixes:                DefaultPrefixes(),
		quantityUnitsIgnoreCase: true,
		separatorText:           "=",
		styles:                  DefaultStyles(),
		timeFormat:              "15:04:05.000",
		timeLocation:            time.Local,
	}
	l.atomicLevel.Store(int32(InfoLevel))
	l.labelWidth = computeLabelWidth(l.labels)
	l.recomputePaddedLabels()
	return l
}

// NewWriter creates a new [Logger] that writes to w with [ColorAuto].
func NewWriter(w io.Writer) *Logger {
	return New(NewOutput(w, ColorAuto))
}

// SetElapsedFormatFunc sets a custom format function for Elapsed fields.
// When set to nil (the default), the built-in [formatElapsed] is used.
func (l *Logger) SetElapsedFormatFunc(fn func(time.Duration) string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.elapsedFormatFunc = fn
}

// SetElapsedMinimum sets the minimum duration for Elapsed fields to be displayed.
// Elapsed values below this threshold are hidden. Defaults to [time.Second].
// Set to 0 to show all values.
func (l *Logger) SetElapsedMinimum(d time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.elapsedMinimum = d
}

// SetElapsedPrecision sets the number of decimal places for Elapsed display.
// For example, 0 = "3s", 1 = "3.2s", 2 = "3.21s". Defaults to 0.
func (l *Logger) SetElapsedPrecision(precision int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.elapsedPrecision = precision
}

// SetElapsedRound sets the rounding granularity for Elapsed values.
// Defaults to [time.Second]. Set to 0 to disable rounding.
func (l *Logger) SetElapsedRound(d time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.elapsedRound = d
}

// SetExitFunc sets the function called by Fatal-level events.
// Defaults to [os.Exit]. This can be used in tests to intercept fatal exits.
// If fn is nil, the default [os.Exit] is used.
func (l *Logger) SetExitFunc(fn func(int)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if fn == nil {
		fn = os.Exit
	}
	l.exitFunc = fn
}

// SetFieldSort sets the sort order for fields in log output.
// Default [SortNone] preserves insertion order.
func (l *Logger) SetFieldSort(sort Sort) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fieldSort = sort
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

// SetLevel sets the minimum log level.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
	l.atomicLevel.Store(int32(level)) //nolint:gosec // Level values are small constants (0-6)
}

// SetLevelAlign sets the alignment mode for level labels.
func (l *Logger) SetLevelAlign(align Align) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.levelAlign = align
	l.recomputePaddedLabels()
}

// SetLabelWidth sets an explicit minimum width for level labels.
// If width is 0, the width is computed automatically from the current labels.
func (l *Logger) SetLabelWidth(width int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if width <= 0 {
		width = computeLabelWidth(l.labels)
	}
	l.labelWidth = width
	l.recomputePaddedLabels()
}

// SetLevelLabels sets the level labels used in log output.
// Pass a map from [Level] to label string (e.g. {WarnLevel: "WARN"}).
// Missing levels fall back to the defaults.
func (l *Logger) SetLevelLabels(labels LevelMap) {
	l.mu.Lock()
	defer l.mu.Unlock()
	merged := DefaultLabels()
	maps.Copy(merged, labels)
	l.labels = merged
	l.labelWidth = computeLabelWidth(merged)
	l.recomputePaddedLabels()
}

// SetOmitEmpty enables or disables omitting fields with empty values.
// Empty means nil, empty strings, and nil or empty slices/maps.
func (l *Logger) SetOmitEmpty(omit bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.omitEmpty = omit
}

// SetOmitZero enables or disables omitting fields with zero values.
// Zero means the zero value for any type (0, false, "", nil, etc.).
// This is a superset of [Logger.SetOmitEmpty].
func (l *Logger) SetOmitZero(omit bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.omitZero = omit
}

// SetOutput sets the output.
func (l *Logger) SetOutput(out *Output) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = out
}

// SetOutputWriter sets the output writer with [ColorAuto].
func (l *Logger) SetOutputWriter(w io.Writer) {
	l.SetOutput(NewOutput(w, ColorAuto))
}

// Output returns the logger's [Output].
func (l *Logger) Output() *Output {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.output
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

// SetPercentFormatFunc sets a custom format function for Percent fields.
// When set to nil (the default), the built-in format is used.
func (l *Logger) SetPercentFormatFunc(fn func(float64) string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.percentFormatFunc = fn
}

// SetPercentPrecision sets the number of decimal places for Percent display.
// For example, 0 = "75%", 1 = "75.0%". Defaults to 0.
func (l *Logger) SetPercentPrecision(precision int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.percentPrecision = precision
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

// SetQuantityUnitsIgnoreCase sets whether quantity unit matching is
// case-insensitive. Defaults to true.
func (l *Logger) SetQuantityUnitsIgnoreCase(ignoreCase bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.quantityUnitsIgnoreCase = ignoreCase
}

// SetQuoteChar sets the character used to quote field values that contain
// spaces or special characters. The default (zero value) uses Go-style
// double-quoted strings via [strconv.Quote]. Setting a non-zero rune wraps
// values with that character on both sides (e.g. '\").
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

// SetQuoteMode sets the quoting behaviour for field values.
// [QuoteAuto] (default) quotes only when needed; [QuoteAlways] always quotes
// string/error/default-kind values; [QuoteNever] never quotes.
func (l *Logger) SetQuoteMode(mode QuoteMode) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.quoteMode = mode
}

// SetReportTimestamp enables or disables timestamp reporting.
func (l *Logger) SetReportTimestamp(report bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reportTimestamp = report
}

// SetSeparatorText sets the separator between field keys and values.
// Defaults to "=".
func (l *Logger) SetSeparatorText(sep string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.separatorText = sep
}

// SetStyles sets the display styles. If styles is nil, [DefaultStyles] is used.
func (l *Logger) SetStyles(styles *Styles) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if styles == nil {
		styles = DefaultStyles()
	}
	l.styles = styles
}

// SetTimeFormat sets the timestamp format string.
func (l *Logger) SetTimeFormat(format string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.timeFormat = format
}

// SetTimeLocation sets the timezone for timestamps. Defaults to [time.Local].
// If loc is nil, [time.Local] is used.
func (l *Logger) SetTimeLocation(loc *time.Location) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if loc == nil {
		loc = time.Local
	}
	l.timeLocation = loc
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

// WithContext returns a copy of ctx with the logger stored as a value.
func (l *Logger) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
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

// colorsDisabled returns true if this logger should suppress colours.
func (l *Logger) colorsDisabled() bool {
	return l.output.ColorsDisabled()
}

// exit calls the logger's exit function (used by Fatal-level events).
func (l *Logger) exit(code int) {
	l.mu.Lock()
	fn := l.exitFunc
	l.mu.Unlock()

	fn(code)
}

// formatLabel returns the pre-computed padded level label.
func (l *Logger) formatLabel(level Level) string {
	if l.labelsPadded == nil {
		l.recomputePaddedLabels()
	}
	return l.labelsPadded[level]
}

// recomputePaddedLabels rebuilds the labelsPadded cache from the current
// labels, labelWidth, and levelAlign settings. Must be called with l.mu held.
func (l *Logger) recomputePaddedLabels() {
	m := make(LevelMap, len(l.labels))
	maxW := l.labelWidth
	for lvl, label := range l.labels {
		switch l.levelAlign {
		case AlignLeft:
			pad := maxW - len(label)
			if pad > 0 {
				m[lvl] = label + strings.Repeat(" ", pad)
			} else {
				m[lvl] = label
			}
		case AlignRight:
			pad := maxW - len(label)
			if pad > 0 {
				m[lvl] = strings.Repeat(" ", pad) + label
			} else {
				m[lvl] = label
			}
		case AlignCenter:
			m[lvl] = centerPad(label, maxW)
		case AlignNone:
			m[lvl] = label
		}
	}
	l.labelsPadded = m
}

// log writes a log entry using either the custom handler or the built-in pretty formatter.
func (l *Logger) log(e *Event, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Merge logger context fields with event fields.
	var allFields []Field
	needsFilter := l.omitZero || l.omitEmpty
	switch {
	case len(l.fields) == 0 && len(e.fields) == 0:
		// no fields
	case len(l.fields) == 0:
		if needsFilter {
			allFields = slices.Clone(e.fields)
		} else {
			allFields = e.fields
		}
	case len(e.fields) == 0:
		if needsFilter {
			allFields = slices.Clone(l.fields)
		} else {
			allFields = l.fields
		}
	default:
		allFields = slices.Concat(l.fields, e.fields)
	}

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

	var partsArr [8]string
	parts := partsArr[:0]

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
				elapsedFormatFunc:       l.elapsedFormatFunc,
				elapsedMinimum:          l.elapsedMinimum,
				elapsedPrecision:        l.elapsedPrecision,
				elapsedRound:            l.elapsedRound,
				fieldSort:               l.fieldSort,
				fieldStyleLevel:         l.fieldStyleLevel,
				level:                   e.level,
				noColor:                 noColor,
				percentFormatFunc:       l.percentFormatFunc,
				percentPrecision:        l.percentPrecision,
				quantityUnitsIgnoreCase: l.quantityUnitsIgnoreCase,
				quoteClose:              l.quoteClose,
				quoteMode:               l.quoteMode,
				quoteOpen:               l.quoteOpen,
				separatorText:           l.separatorText,
				styles:                  l.styles,
				timeFormat:              l.fieldTimeFormat,
			}), " ")
		}

		if s != "" {
			parts = append(parts, s)
		}
	}

	var lineBuf strings.Builder
	for i, p := range parts {
		if i > 0 {
			lineBuf.WriteByte(' ')
		}
		lineBuf.WriteString(p)
	}
	lineBuf.WriteByte('\n')
	_, _ = io.WriteString(l.output.Writer(), lineBuf.String())
}

// computeLabelWidth returns the length of the longest label in the map.
func computeLabelWidth(labels LevelMap) int {
	maxWidth := 0
	for _, lbl := range labels {
		if len(lbl) > maxWidth {
			maxWidth = len(lbl)
		}
	}
	return maxWidth
}

// newEvent creates a new [Event] for the given level.
// Returns nil if the level is below the logger's minimum (all Event methods
// are no-ops on nil).
func (l *Logger) newEvent(level Level) *Event {
	// Fast path: lock-free level check to skip disabled events without
	// acquiring the mutex.
	//nolint:gosec // Level values are small constants (0-6)
	if int32(level) < l.atomicLevel.Load() {
		return nil
	}
	return &Event{
		logger: l,
		level:  level,
	}
}

// resolvePrefix returns the appropriate prefix for a log entry, checking
// event override -> logger preset -> default for level.
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
	// Output is the output to use (defaults to [Stdout]([ColorAuto])).
	Output *Output
	// Styles allows customising the visual styles.
	Styles *Styles
}

// Configure sets up the [Default] logger with the given configuration.
// Call this once at application startup.
//
// Note: this respects the log level environment variable — it won't reset
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
		Default.SetLevel(DebugLevel)
		Default.SetReportTimestamp(true)
		return
	}

	// Respect the env var if set (custom prefix or CLOG_LOG_LEVEL).
	if getEnv(envLogLevel) != "" {
		return
	}

	Default.SetLevel(InfoLevel)
	Default.SetReportTimestamp(false)
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

// Level returns the current minimum log level.
func (l *Logger) Level() Level {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

// GetLevel returns the current log level of the [Default] logger.
func GetLevel() Level {
	return Default.Level()
}

// IsVerbose returns true if verbose/debug mode is enabled on the [Default] logger.
// Returns true for both [TraceLevel] and [DebugLevel].
func IsVerbose() bool {
	return GetLevel() <= DebugLevel
}

// Package-level convenience functions that use the [Default] logger.

// SetColorMode sets the colour mode by recreating the logger's [Output]
// with the given mode.
func (l *Logger) SetColorMode(mode ColorMode) {
	l.mu.Lock()
	defer l.mu.Unlock()
	w := l.output.Writer()
	l.output = NewOutput(w, mode)
}

// SetColorMode sets the colour mode on the [Default] logger by recreating
// its [Output] with the given mode.
func SetColorMode(mode ColorMode) {
	Default.SetColorMode(mode)
}

// SetElapsedFormatFunc sets the elapsed format function on the [Default] logger.
func SetElapsedFormatFunc(fn func(time.Duration) string) { Default.SetElapsedFormatFunc(fn) }

// SetElapsedMinimum sets the elapsed minimum threshold on the [Default] logger.
func SetElapsedMinimum(d time.Duration) { Default.SetElapsedMinimum(d) }

// SetElapsedPrecision sets the elapsed precision on the [Default] logger.
func SetElapsedPrecision(precision int) { Default.SetElapsedPrecision(precision) }

// SetElapsedRound sets the elapsed rounding granularity on the [Default] logger.
func SetElapsedRound(d time.Duration) { Default.SetElapsedRound(d) }

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

// SetLevel sets the minimum log level on the [Default] logger.
func SetLevel(level Level) { Default.SetLevel(level) }

// SetLevelAlign sets the level-label alignment on the [Default] logger.
func SetLevelAlign(align Align) { Default.SetLevelAlign(align) }

// SetLevelLabels sets the level labels on the [Default] logger.
func SetLevelLabels(labels LevelMap) { Default.SetLevelLabels(labels) }

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

// SetPercentFormatFunc sets the percent format function on the [Default] logger.
func SetPercentFormatFunc(fn func(float64) string) { Default.SetPercentFormatFunc(fn) }

// SetPercentPrecision sets the percent precision on the [Default] logger.
func SetPercentPrecision(precision int) { Default.SetPercentPrecision(precision) }

// SetPrefixes sets the level prefixes on the [Default] logger.
func SetPrefixes(prefixes LevelMap) { Default.SetPrefixes(prefixes) }

// SetQuantityUnitsIgnoreCase sets case-insensitive quantity unit matching on the [Default] logger.
func SetQuantityUnitsIgnoreCase(ignoreCase bool) { Default.SetQuantityUnitsIgnoreCase(ignoreCase) }

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

// SetStyles sets the display styles on the [Default] logger.
func SetStyles(styles *Styles) { Default.SetStyles(styles) }

// SetTimeFormat sets the timestamp format on the [Default] logger.
func SetTimeFormat(format string) { Default.SetTimeFormat(format) }

// SetTimeLocation sets the timestamp timezone on the [Default] logger.
func SetTimeLocation(loc *time.Location) { Default.SetTimeLocation(loc) }

// Ctx retrieves the logger from ctx. Returns [Default] if ctx is nil
// or contains no logger.
func Ctx(ctx context.Context) *Logger {
	if ctx == nil {
		return Default
	}
	if l, ok := ctx.Value(ctxKey{}).(*Logger); ok {
		return l
	}
	return Default
}

// WithContext stores the [Default] logger in ctx.
func WithContext(ctx context.Context) context.Context {
	return Default.WithContext(ctx)
}

// With returns a [Context] for building a sub-logger from the [Default] logger.
func With() *Context { return Default.With() }

// Dict returns a new detached [Event] for use as a nested dictionary field.
func Dict() *Event { return &Event{} }

// Trace returns a new trace-level [Event] from the [Default] logger.
func Trace() *Event { return Default.Trace() }

// Debug returns a new debug-level [Event] from the [Default] logger.
func Debug() *Event { return Default.Debug() }

// Info returns a new info-level [Event] from the [Default] logger.
func Info() *Event { return Default.Info() }

// Dry returns a new dry-level [Event] from the [Default] logger.
func Dry() *Event { return Default.Dry() }

// Warn returns a new warn-level [Event] from the [Default] logger.
func Warn() *Event { return Default.Warn() }

// Error returns a new error-level [Event] from the [Default] logger.
func Error() *Event { return Default.Error() }

// Fatal returns a new fatal-level [Event] from the [Default] logger.
func Fatal() *Event { return Default.Fatal() }

// centerPad centres s within width, padding with spaces.
func centerPad(s string, width int) string {
	pad := width - len(s)
	left := pad / 2 //nolint:mnd // half the padding goes left
	right := pad - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}
