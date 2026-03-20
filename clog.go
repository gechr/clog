// Package clog provides structured CLI logging with terminal-aware colors,
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
	"io"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gechr/clog/fx/spinner"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/level"
	"github.com/gechr/clog/style"
)

// Level represents a log level.
//
// Level implements [encoding.TextMarshaler] and [encoding.TextUnmarshaler],
// so it works directly with [flag.TextVar] and most flag libraries.
type Level = level.Level

const (
	LevelTrace = level.Trace
	LevelDebug = level.Debug
	LevelInfo  = level.Info
	LevelDry   = level.Dry
	LevelWarn  = level.Warn
	LevelError = level.Error
	LevelFatal = level.Fatal

	// UnsetLevel is passed to [SetNonTTYLevel] to disable the non-TTY level
	// filter. Its value is intentionally below all real log levels so the
	// check e.level < nonTTYLevel is always false, meaning no restriction.
	UnsetLevel = level.Unset
)

const (
	LevelTraceValue = level.TraceValue
	LevelDebugValue = level.DebugValue
	LevelInfoValue  = level.InfoValue
	LevelDryValue   = level.DryValue
	LevelWarnValue  = level.WarnValue
	LevelErrorValue = level.ErrorValue
	LevelFatalValue = level.FatalValue
)

// ErrorKey is the default field key used by [Event.Err] and [Context.Err].
const ErrorKey = core.ErrorKey

// Nil is the string representation used for nil values.
const Nil = core.Nil

// Default is the default logger instance.
var Default = New(Stdout(ColorAuto))

// Logger is the main structured logger.
type Logger struct {
	mu *sync.Mutex

	animationInterval time.Duration
	atomicLevel       atomic.Int32 // lock-free level check for newEvent() hot path
	exitCode          int          // default exit code for Fatal-level events; 0 means 1
	exitFunc          func(int)    // called by Fatal-level events; defaults to os.Exit
	fields            []Field
	fieldSort         Sort
	fieldStyleLevel   Level
	fieldTimeFormat   string
	handler           Handler
	hooks             map[HookPoint][]func()
	indent            int      // number of indent levels for nested output
	indentPrefixes    []string // per-depth decorations cycled after space indent
	indentPrefixSep   *string  // separator after indent prefix; nil = default " "
	indentWidth       int      // spaces per indent level (default 2)
	labels            LabelMap
	labelsPadded      LabelMap
	labelWidth        int
	level             Level
	levelAlign        Align
	nonTTYLevel       Level // events below this level are suppressed on non-TTY writers
	omitEmpty         bool
	omitZero          bool
	output            *Output
	parts             []Part
	quoteClose        rune // 0 means same as quoteOpen (or default)
	quoteMode         QuoteMode
	quoteOpen         rune // 0 means default ('"' via strconv.Quote)
	reportTimestamp   bool
	separatorText     string
	sliceClose        rune // 0 means default (']')
	sliceOpen         rune // 0 means default ('[')
	sliceSep          string
	spinnerStyle      *spinner.Style // nil = use spinner.DefaultStyle()
	styles            *style.Config
	symbol            *string // nil = use default emoji for level
	symbols           LabelMap
	timeFormat        string
	timeLocation      *time.Location
	tree              []TreePos // nil = no tree mode; one entry per tree level
	treeChars         TreeChars // box-drawing characters for tree indentation
}

// New creates a new [Logger] that writes to the given [Output].
// If output is nil, it defaults to [Stdout] with [ColorAuto].
func New(output *Output) *Logger {
	if output == nil {
		output = Stdout(ColorAuto)
	}
	l := &Logger{
		mu: &sync.Mutex{},

		animationInterval: 67 * time.Millisecond, //nolint:mnd // ~15fps
		exitFunc:          os.Exit,
		fieldStyleLevel:   LevelInfo,
		indentWidth:       2, //nolint:mnd // default indent: 2 spaces per level
		fieldTimeFormat:   time.RFC3339,
		labels:            DefaultLabels(),
		level:             LevelInfo,
		levelAlign:        AlignRight,
		output:            output,
		parts:             DefaultParts(),
		symbols:           DefaultSymbols(),
		separatorText:     "=",
		sliceSep:          ", ",
		styles:            DefaultStyles(),
		timeFormat:        "15:04:05.000",
		timeLocation:      time.Local,
		treeChars:         DefaultTreeChars(),
	}
	l.atomicLevel.Store(int32(LevelInfo))
	l.nonTTYLevel = UnsetLevel

	l.labelWidth = computeLabelWidth(l.labels)
	l.recomputePaddedLabels()
	return l
}

// NewWriter creates a new [Logger] that writes to w with [ColorAuto].
func NewWriter(w io.Writer) *Logger {
	return New(NewOutput(w, ColorAuto))
}

// Log returns a new [Event] at the given level. Use this for custom levels
// registered with [RegisterLevel].
func (l *Logger) Log(level Level) *Event { return l.newEvent(level) }

// Trace returns a new [Event] at trace level, or nil if trace is disabled.
func (l *Logger) Trace() *Event { return l.newEvent(LevelTrace) }

// Debug returns a new [Event] at debug level, or nil if debug is disabled.
func (l *Logger) Debug() *Event { return l.newEvent(LevelDebug) }

// Info returns a new [Event] at info level, or nil if info is disabled.
func (l *Logger) Info() *Event { return l.newEvent(LevelInfo) }

// Dry returns a new [Event] at dry level, or nil if dry is disabled.
func (l *Logger) Dry() *Event { return l.newEvent(LevelDry) }

// Warn returns a new [Event] at warn level, or nil if warn is disabled.
func (l *Logger) Warn() *Event { return l.newEvent(LevelWarn) }

// Error returns a new [Event] at error level, or nil if error is disabled.
func (l *Logger) Error() *Event { return l.newEvent(LevelError) }

// Fatal returns a new [Event] at fatal level.
func (l *Logger) Fatal() *Event { return l.newEvent(LevelFatal) }

// Level returns the current minimum log level.
func (l *Logger) Level() Level {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

// LevelEnabled reports whether the logger handles records at the given level.
func (l *Logger) LevelEnabled(level Level) bool {
	//nolint:gosec // Level values are small constants (-10 to 15)
	return int32(level) >= l.atomicLevel.Load()
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
		indent: l.indent,
		logger: l,
		symbol: l.symbol,
		tree:   append([]TreePos{}, l.tree...),
	}
	c.Fields = fields
	c.InitSelf(c)
	return c
}

// WithContext returns a copy of ctx with the logger stored as a value.
func (l *Logger) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// LogFields logs a message at the given level with the provided timestamp and fields.
// This is used by adapters (e.g. [sloghandler]) that build fields externally.
// Unlike direct Fatal() calls, LogFields does not trigger [os.Exit] for
// [LevelFatal] events -- adapters should not cause process termination.
func (l *Logger) LogFields(level Level, ts time.Time, msg string, fields []Field) {
	e := l.newEvent(level)
	if e == nil {
		return
	}
	e.timestamp = ts
	e.fields = fields
	e.noExit = true
	e.Msg(msg)
}

// Log returns a new [Event] at the given level from the [Default] logger.
// Use this for custom levels registered with [RegisterLevel].
func Log(level Level) *Event { return Default.Log(level) }

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

// With returns a [Context] for building a sub-logger from the [Default] logger.
func With() *Context { return Default.With() }

// WithContext stores the [Default] logger in ctx.
func WithContext(ctx context.Context) context.Context {
	return Default.WithContext(ctx)
}

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

// Dict returns a new detached [Event] for use as a nested dictionary field.
// The event uses the [Default] logger's output for hyperlink/color resolution.
func Dict() *Event { return Default.Dict() }

// Dict returns a new detached [Event] for use as a nested dictionary field.
// The event uses the logger's output for hyperlink/color resolution.
func (l *Logger) Dict() *Event { return &Event{logger: l, dict: true} }

// Divider returns a new [DividerBuilder] for rendering a horizontal rule
// using the [Default] logger.
func Divider() *DividerBuilder { return Default.Divider() }

// GetLevel returns the current log level of the [Default] logger.
func GetLevel() Level {
	return Default.Level()
}

// IsVerbose returns true if verbose/debug mode is enabled on the [Default] logger.
// Returns true for both [LevelTrace] and [LevelDebug].
func IsVerbose() bool {
	return GetLevel() <= LevelDebug
}

// SetAnimationInterval sets a minimum refresh interval for all animations
// (spinners, bars, pulse, shimmer). Any animation whose built-in tick rate
// is faster than d will be clamped to d. The default is 67ms (~15fps).
// Zero means use built-in rates unchanged.
func (l *Logger) SetAnimationInterval(d time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.animationInterval = d
}

// SetColorMode sets the color mode by recreating the logger's [Output]
// with the given mode.
func (l *Logger) SetColorMode(mode ColorMode) {
	l.mu.Lock()
	defer l.mu.Unlock()
	w := l.output.Writer()
	l.output = NewOutput(w, mode)
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

// SetExitCode sets the default exit code for [LevelFatal] events.
// If code is 0, the default exit code (1) is used.
// This can be overridden per-event with [Event.ExitCode].
func (l *Logger) SetExitCode(code int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.exitCode = code
}

// SetFieldSort sets the sort order for fields in log output.
// Default [SortNone] preserves insertion order.
func (l *Logger) SetFieldSort(sort Sort) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fieldSort = sort
}

// SetFieldStyleLevel sets the minimum log level at which field values are
// styled (colored). Events below this level render fields as plain text.
// Defaults to [LevelInfo].
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

// AddHook registers a function to run at the given [HookPoint].
// Multiple hooks per point are supported; they run in registration order.
// Hooks are called under the logger's mutex.
func (l *Logger) AddHook(point HookPoint, fn func()) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.hooks == nil {
		l.hooks = make(map[HookPoint][]func())
	}
	l.hooks[point] = append(l.hooks[point], fn)
}

// ClearAllHooks removes all registered hooks at every [HookPoint].
func (l *Logger) ClearAllHooks() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hooks = nil
}

// ClearHooks removes all hooks registered at the given [HookPoint].
func (l *Logger) ClearHooks(point HookPoint) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hooks, point)
}

// SetIndent sets the indent depth (number of indent levels) on the logger.
func (l *Logger) SetIndent(levels int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.indent = levels
}

// SetIndentPrefixes sets per-depth decorations placed after the space-based
// indent. A trailing space is appended automatically. Prefixes are cycled:
// at depth N the prefix is prefixes[(N-1) % len(prefixes)].
// For example, with []string{"│"}, depth 2 produces "    │ " (4 spaces + "│" + space).
// Pass nil to clear.
func (l *Logger) SetIndentPrefixes(prefixes []string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.indentPrefixes = prefixes
}

// SetIndentPrefixSeparator sets the separator appended after an indent prefix.
// Defaults to " " (a single space).
func (l *Logger) SetIndentPrefixSeparator(sep string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.indentPrefixSep = &sep
}

// SetIndentWidth sets the number of spaces per indent level.
// Defaults to 2.
func (l *Logger) SetIndentWidth(width int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.indentWidth = width
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

// SetLevel sets the minimum log level.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
	l.atomicLevel.Store(int32(level)) //nolint:gosec // Level values are small constants (-10 to 15)
}

// SetNonTTYLevel sets the minimum log level for non-TTY writers (CI, piped
// output, etc.). Events below this level are suppressed when the logger's
// output is not connected to a terminal, including animation progress lines.
// Pass [UnsetLevel] to restore the default behaviour.
func (l *Logger) SetNonTTYLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.nonTTYLevel = level
}

// SetLevelAlign sets the alignment mode for level labels.
func (l *Logger) SetLevelAlign(align Align) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.levelAlign = align
	l.recomputePaddedLabels()
}

// SetLabels sets the level labels used in log output.
// Pass a map from [Level] to label string (e.g. {LevelWarn: "WARN"}).
// Missing levels fall back to the defaults.
func (l *Logger) SetLabels(labels LabelMap) {
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

// SetSymbols sets the emoji symbols used for each level.
// Pass a map from [Level] to symbol string. Missing levels fall back to the defaults.
func (l *Logger) SetSymbols(symbols LabelMap) {
	l.mu.Lock()
	defer l.mu.Unlock()
	merged := DefaultSymbols()
	maps.Copy(merged, symbols)

	l.symbols = merged
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

// SetSliceBracket sets the same opening and closing character for slice field values.
// Defaults to '[' and ']'.
func (l *Logger) SetSliceBracket(char rune) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sliceOpen = char
	l.sliceClose = char
}

// SetSliceBrackets sets separate opening and closing characters for slice field values.
// Defaults to '[' and ']'.
func (l *Logger) SetSliceBrackets(openChar, closeChar rune) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sliceOpen = openChar
	l.sliceClose = closeChar
}

// SetSliceSeparator sets the separator between elements in slice field values.
// Defaults to ", ".
func (l *Logger) SetSliceSeparator(sep string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sliceSep = sep
}

// SetSpinnerStyle sets the default spinner style used by [Logger.Spinner].
func (l *Logger) SetSpinnerStyle(s spinner.Style) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.spinnerStyle = new(s)
}

// SetStyles merges the given styles into the current style configuration.
// Non-nil pointer fields overwrite existing values; map fields are merged
// key-by-key. Pass nil to reset to [DefaultStyles].
func (l *Logger) SetStyles(styles *style.Config) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if styles == nil {
		l.styles = DefaultStyles()
		return
	}
	l.styles.Merge(styles)
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

// SetTreeChars sets the box-drawing characters used for tree indentation.
// See [DefaultTreeChars] for the defaults.
func (l *Logger) SetTreeChars(chars TreeChars) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.treeChars = chars
}

// LevelConfig configures a custom log level for use with [RegisterLevel].
type LevelConfig struct {
	Label  string // short display label (e.g. "SCS") [default: uppercase Name, max 3 chars]
	Name   string // canonical name for ParseLevel/MarshalText (e.g. "success") [required]
	Style  Style  // lipgloss style for the level label [default: nil]
	Symbol string // emoji symbol (e.g. "✅") [default: ""]
}

// RegisterLevel registers a custom log level with the given numeric value
// and configuration. The level value must not conflict with a built-in level.
//
// After registration the level works with [ParseLevel], [Level.MarshalText],
// [Level.String], and the [Default] logger's labels, symbols, and styles.
//
// RegisterLevel panics if cfg.Name is empty or level conflicts with a built-in level.
func RegisterLevel(lvl Level, cfg LevelConfig) {
	if cfg.Name == "" {
		panic("clog: RegisterLevel requires a non-empty Name")
	}

	// Default label: uppercase name, max 3 chars.
	if cfg.Label == "" {
		lbl := strings.ToUpper(cfg.Name)
		if len(lbl) > defaultMaxLabelLen {
			lbl = lbl[:defaultMaxLabelLen]
		}
		cfg.Label = lbl
	}

	// Register name and label in the level package (panics on built-in conflict).
	level.Register(lvl, cfg.Name, cfg.Label)

	customLevelsMu.Lock()
	customLevels[lvl] = cfg
	defaultSymbols[lvl] = cfg.Symbol
	customLevelsMu.Unlock()

	// Update the Default logger.
	Default.mu.Lock()
	Default.labels[lvl] = cfg.Label
	Default.symbols[lvl] = cfg.Symbol
	Default.labelWidth = computeLabelWidth(Default.labels)
	Default.recomputePaddedLabels()
	if cfg.Style != nil {
		if Default.styles.Levels == nil {
			Default.styles.Levels = make(style.LevelMap)
		}
		Default.styles.Levels[lvl] = cfg.Style
	}
	Default.mu.Unlock()
}

// ParseLevel maps a level name string to a [Level] value.
// It accepts the canonical names ("trace", "debug", "info", "dry", "warn",
// "error", "fatal") plus aliases ("warning" → Warn, "critical" → Fatal).
// Matching is case-insensitive.
func ParseLevel(s string) (Level, error) { return level.Parse(s) }

// Levels returns all registered levels (built-in and custom) in ascending
// severity order.
func Levels() []Level {
	all := level.All()
	levels := make([]Level, 0, len(all))
	for _, lvl := range all {
		levels = append(levels, lvl)
	}
	slices.Sort(levels)
	return levels
}

// customLevelsMu guards the custom levels registry and the maps it updates.
var customLevelsMu sync.RWMutex

// customLevels holds custom level registrations.
var customLevels = map[Level]LevelConfig{}

// ctxKey is the private context key used by [Logger.WithContext] and [Ctx].
type ctxKey struct{}

// colorsDisabled returns true if this logger should suppress colors.
func (l *Logger) colorsDisabled() bool {
	return l.output.ColorsDisabled()
}

// indentation returns the indent string for the current indent level,
// including any tree-drawing connectors.
func (l *Logger) indentation() string {
	return computeIndent(
		l.indent,
		l.indentWidth,
		l.indentPrefixes,
		l.indentPrefixSep,
	) + computeTreeIndent(
		l.tree,
		l.treeChars,
	)
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

// allPaddedLabels returns a copy of the padded labels map. Must be called with l.mu held.
func (l *Logger) allPaddedLabels() LabelMap {
	if l.labelsPadded == nil {
		l.recomputePaddedLabels()
	}
	return maps.Clone(l.labelsPadded)
}

// recomputePaddedLabels rebuilds the labelsPadded cache from the current
// labels, labelWidth, and levelAlign settings. Must be called with l.mu held.
func (l *Logger) recomputePaddedLabels() {
	m := make(LabelMap, len(l.labels))
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

	// Inherit the logger's default exit code if the event doesn't override it.
	if e.exitCode == 0 && l.exitCode != 0 {
		e.exitCode = l.exitCode
	}

	// Suppress events below the non-TTY level threshold on non-terminal writers.
	if l.nonTTYLevel != UnsetLevel && !l.output.IsTTY() && e.level < l.nonTTYLevel {
		return
	}

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

	symbol := l.resolveSymbol(e)

	// Delegate to custom handler if set.
	if l.handler != nil {
		entry := Entry{
			Level:   e.level,
			Symbol:  symbol,
			Indent:  l.indent,
			Message: msg,
			Fields:  allFields,
			Tree:    l.tree,
		}
		if !e.timestamp.IsZero() {
			entry.Time = e.timestamp.In(l.timeLocation)
		} else if l.reportTimestamp {
			entry.Time = time.Now().In(l.timeLocation)
		}

		l.runHooks(HookBeforeWrite)
		l.handler.Log(entry)
		l.runHooks(HookAfterWrite)
		return
	}

	// Built-in pretty formatter.
	noColor := l.colorsDisabled()

	// Resolve parts: event override -> logger default.
	partsOrder := l.parts
	if e.parts != nil {
		partsOrder = *e.parts
	}

	var partsArr [8]string
	parts := partsArr[:0]

	for _, p := range partsOrder {
		var s string

		switch p {
		case PartTimestamp:
			if e.timestamp.IsZero() && !l.reportTimestamp {
				continue
			}

			var now time.Time
			if !e.timestamp.IsZero() {
				now = e.timestamp.In(l.timeLocation)
			} else {
				now = time.Now().In(l.timeLocation)
			}
			ts := now.Format(l.timeFormat)
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
		case PartSymbol:
			if symbol == "" {
				continue
			}

			if style := l.styles.Symbols[e.level]; !noColor && style != nil {
				s = style.Render(symbol)
			} else {
				s = symbol
			}
		case PartMessage:
			if msg == "" && l.indent == 0 && len(l.tree) == 0 {
				continue
			}

			if style := l.styles.Messages[e.level]; !noColor && style != nil {
				s = style.Render(msg)
			} else if !noColor && l.styles.Message != nil {
				s = l.styles.Message.Render(msg)
			} else {
				s = msg
			}

			if l.indent > 0 || len(l.tree) > 0 {
				s = l.indentation() + s
			}
		case PartFields:
			s = strings.TrimLeft(formatFields(allFields, formatFieldsOpts{
				fieldSort:       l.fieldSort,
				fieldStyleLevel: l.fieldStyleLevel,
				level:           e.level,
				noColor:         noColor,
				quoteOpen:       l.quoteOpen,
				quoteClose:      l.quoteClose,
				quoteMode:       l.quoteMode,
				separatorText:   l.separatorText,
				sliceClose:      l.sliceClose,
				sliceOpen:       l.sliceOpen,
				sliceSep:        l.sliceSep,
				styles:          l.styles,
				timeFormat:      l.fieldTimeFormat,
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
	l.runHooks(HookBeforeWrite)
	writeString(l.output.Writer(), lineBuf.String())
	l.runHooks(HookAfterWrite)
}

// runHooks executes all hooks registered at the given point. Must be called with l.mu held.
func (l *Logger) runHooks(point HookPoint) {
	for _, fn := range l.hooks[point] {
		fn()
	}
}

// newEvent creates a new [Event] for the given level.
// Returns nil if the level is below the logger's minimum (all Event methods
// are no-ops on nil).
func (l *Logger) newEvent(level Level) *Event {
	// Fast path: lock-free level check to skip disabled events without
	// acquiring the mutex.
	//nolint:gosec // Level values are small constants (-10 to 15)
	if int32(level) < l.atomicLevel.Load() {
		return nil
	}
	return &Event{
		logger: l,
		level:  level,
	}
}

// resolveSymbol returns the appropriate symbol for a log entry, checking
// event override -> logger preset -> default for level.
func (l *Logger) resolveSymbol(e *Event) string {
	if e.symbol != nil {
		return *e.symbol
	}

	if l.symbol != nil {
		return *l.symbol
	}
	return l.symbols[e.level]
}

// computeIndent builds the indent string for the given depth.
// It is indentWidth*depth spaces followed by the cycled prefix from
// prefixes (if set) and the separator.
func computeIndent(depth, width int, prefixes []string, sep *string) string {
	if depth <= 0 {
		return ""
	}
	s := strings.Repeat(" ", depth*width)
	if len(prefixes) > 0 {
		psep := " "
		if sep != nil {
			psep = *sep
		}
		s += prefixes[(depth-1)%len(prefixes)] + psep
	}
	return s
}

// computeTreeIndent builds the tree-drawing prefix for the given tree
// positions. Each ancestor level renders a continuation line (│ or blank)
// and the deepest level renders its connector (├── or └──).
func computeTreeIndent(tree []TreePos, chars TreeChars) string {
	if len(tree) == 0 {
		return ""
	}
	var b strings.Builder
	for i, pos := range tree {
		if i == len(tree)-1 {
			// Deepest level - draw the connector.
			switch pos {
			case TreeFirst:
				b.WriteString(chars.First)
			case TreeMiddle:
				b.WriteString(chars.Middle)
			case TreeLast:
				b.WriteString(chars.Last)
			}
		} else {
			// Ancestor level - draw continuation or blank.
			if pos == TreeLast {
				b.WriteString(chars.Blank)
			} else {
				b.WriteString(chars.Continue)
			}
		}
	}
	return b.String()
}
