// Package clog provides structured CLI logging with terminal-aware colors,
// hyperlinks, and spinners.
//
// It uses a zerolog-style fluent API for building log entries:
//
//	clog.Info().Str("port", "8080").Msg("Server started")
//
// The default output is a pretty terminal formatter. A custom [Handler] can
// be set for alternative formats (e.g. JSON).
//
// # Nil receivers
//
// A nil [Logger] and a nil [Event] are inert: every method on them is safe to
// call, each returns a zero value or an equally inert object, and nothing is
// written. A logging call is therefore never the thing that takes a process
// down - a lazily-initialised package-level logger used before its
// initialisation runs drops its output instead of panicking. The missing
// initialisation is still a bug; it is just no longer a fatal one.
//
// The package never hands out a nil of either type: every constructor returns
// a live value, and the builders on a nil Logger fall back to an inert logger
// rather than propagating the nil (so [Logger.With], [Logger.Print],
// [Logger.Divider], [Logger.Spinner] and friends are safe to chain). The
// remaining exported pointer types - [Context], [Output], [Printer],
// [DividerBuilder] - are deliberately not nil-tolerant: a nil one can only
// come from a caller declaring it and never assigning it, and for Context it
// is not even possible, because it embeds a value-typed field builder whose
// promoted methods dereference the receiver before any guard could run (the
// same constraint that stops [Event] embedding one).
package clog

import (
	"context"
	"io"
	"maps"
	"os"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gechr/clog/fx/spinner"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/level"
	"github.com/gechr/clog/style"
	"github.com/gechr/clog/theme"
	xansi "github.com/gechr/x/ansi"
	xstrings "github.com/gechr/x/strings"
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
	LevelHint  = level.Hint
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
	LevelHintValue  = level.HintValue
	LevelDryValue   = level.DryValue
	LevelWarnValue  = level.WarnValue
	LevelErrorValue = level.ErrorValue
	LevelFatalValue = level.FatalValue
)

// nl is the newline terminator used throughout the package.
const nl = "\n"

// ErrorKey is the default field key used by [Event.Err] and [Context.Err].
const ErrorKey = core.ErrorKey

// Nil is the string representation used for nil values.
const Nil = core.Nil

var defaultLogger = func() *atomic.Pointer[Logger] {
	p := new(atomic.Pointer[Logger])
	p.Store(New(Stdout(ColorAuto)))
	return p
}()

// discardLogger backs every object a nil [Logger] hands out - printers,
// dividers, animation builders, sub-loggers - so they stay inert instead of
// carrying a nil logger to a panic one call away from the cause. It writes
// nowhere and is never mutated: nil-receiver mutators return before they can
// reach it.
var discardLogger = New(NewOutput(io.Discard, ColorNever))

// orDiscard returns l, or the shared inert logger when l is nil.
func (l *Logger) orDiscard() *Logger {
	if l == nil {
		return discardLogger
	}
	return l
}

// Default returns the default [Logger] used by the package-level functions.
func Default() *Logger { return defaultLogger.Load() }

// SetDefault sets the default [Logger] used by the package-level functions.
// It panics if logger is nil.
func SetDefault(logger *Logger) {
	if logger == nil {
		panic("clog: nil Logger")
	}

	defaultLogger.Store(logger)
}

// Indices into styleOverride / themedStyles, one per theme-derived style.
const (
	themedBacktick = iota
	themedJSON
	themedYAML
	themedTOML
	themedHCL
	themedGradient
	themedStyleCount
)

// styleOverride records which theme-derived styles the user supplied
// explicitly via [Logger.SetStyles]. Each set flag is excluded from
// background-driven theme resolution so it is never overwritten.
type styleOverride [themedStyleCount]bool

// themedStyles drives the theme-derived style machinery: isCustom reports
// whether an incoming [Logger.SetStyles] config deliberately overrides the
// style (see customStyle), and apply rebuilds it from a resolved theme.
// Adding a theme-derived style means adding an index above and a row here.
var themedStyles = [themedStyleCount]struct {
	isCustom func(*style.Config) bool
	apply    func(*style.Config, *theme.Theme)
}{
	themedBacktick: {
		isCustom: func(s *style.Config) bool {
			return customStyle(s.Backtick, style.BacktickFor(theme.BackgroundDark))
		},
		apply: func(s *style.Config, t *theme.Theme) { s.Backtick = style.BacktickFor(t.Background) },
	},
	themedJSON: {
		isCustom: func(s *style.Config) bool { return customStyle(s.JSON, style.DefaultJSON()) },
		apply:    func(s *style.Config, t *theme.Theme) { s.JSON = style.NewJSON(t) },
	},
	themedYAML: {
		isCustom: func(s *style.Config) bool { return customStyle(s.YAML, style.DefaultYAML()) },
		apply:    func(s *style.Config, t *theme.Theme) { s.YAML = style.NewYAML(t) },
	},
	themedTOML: {
		isCustom: func(s *style.Config) bool { return customStyle(s.TOML, style.DefaultTOML()) },
		apply:    func(s *style.Config, t *theme.Theme) { s.TOML = style.NewTOML(t) },
	},
	themedHCL: {
		isCustom: func(s *style.Config) bool { return customStyle(s.HCL, style.DefaultHCL()) },
		apply:    func(s *style.Config, t *theme.Theme) { s.HCL = style.NewHCL(t) },
	},
	themedGradient: {
		isCustom: customGradient,
		apply: func(s *style.Config, t *theme.Theme) {
			s.DurationGradient = style.ElapsedGradientFor(t.Background)
			s.ElapsedGradient = style.ElapsedGradientFor(t.Background)
			s.DeadlineGradient = style.ElapsedGradientFor(t.Background)
			s.PercentGradient = style.PercentGradientFor(t.Background)
		},
	},
}

// Logger is the main structured logger.
//
// A nil *Logger is inert, not fatal: every method is safe to call on one,
// mirroring the nil-[Event] contract. Events are dropped, accessors report
// their defaults, mutators are no-ops, prompts read nothing, and the builders
// ([Logger.Print], [Logger.Divider], [Logger.Spinner] and friends) render to
// [io.Discard]. See the package documentation for the contract in full.
type Logger struct {
	mu *sync.Mutex

	animationInterval  time.Duration
	atomicLevel        *atomic.Int32                 // lock-free level check for newEvent() hot path
	fieldFormats       *atomic.Pointer[FieldFormats] // immutable snapshot; nil = DefaultFieldFormats
	exitCode           int                           // default exit code for Fatal-level events; 0 means 1
	exitFunc           func(int)                     // called by Fatal-level events; defaults to os.Exit
	fields             []Field
	fieldSort          Sort
	fieldStyleLevel    Level
	fieldTimeFormat    string
	handler            Handler
	hooks              map[HookPoint][]func()
	indent             int          // number of indent levels for nested output
	indentPrefixes     []string     // per-depth decorations cycled after space indent
	indentPrefixSep    *string      // separator after indent prefix; nil = default " "
	indentWidth        int          // spaces per indent level (default 2)
	input              *inputSource // source for Input/Password; nil = os.Stdin, lazily wrapped
	labels             LabelMap
	labelsPadded       LabelMap
	labelWidth         int
	levelAlign         Align
	nonTTYLevel        Level // events below this level are suppressed on non-TTY writers
	omitEmpty          bool
	omitZero           bool
	jsonIndent         string        // JSON-specific indent; "" = use printIndent
	jsonPrintMode      JSONPrintMode // default JSON print mode
	output             *Output
	parts              []Part
	printIndent        string // indent string for Printer output; "" = use default ("  ")
	promptMarker       string // leading marker prepended to every prompt; "" = none
	quoteClose         rune   // 0 means same as quoteOpen (or default)
	quoteMode          Quote
	quoteOpen          rune // 0 means default ('"' via strconv.Quote)
	reportTimestamp    bool
	separatorText      string
	sliceClose         rune // 0 means default (']')
	sliceOpen          rune // 0 means default ('[')
	sliceSep           string
	smartQuoteChars    []QuotePair     // preference order; empty = defaults
	smartQuotes        bool            // enables content-adaptive quoting
	spinnerConfig      *spinner.Config // nil = use spinner.DefaultConfig()
	styles             *style.Config
	printThemePair     *theme.Pair   // light/dark source for auto-detection; nil = built-in default pair
	printThemeDirty    bool          // printer styles need (re)building from the detected background
	styleOverride      styleOverride // theme-derived styles the user supplied explicitly
	suppressEcho       bool          // suppress terminal echo while animations are live
	symbol             *string       // nil = use default emoji for level
	symbols            LabelMap
	timeFormat         string
	timeLocation       *time.Location
	tree               []TreePos // nil = no tree mode; one entry per tree level
	treeChars          TreeChars // box-drawing characters for tree indentation
	wrap               Wrap
	yamlIndent         string // YAML-specific indent; "" = use printIndent
	yamlIndentSequence *bool  // nil = true (indent sequences by default)
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
		atomicLevel:       &atomic.Int32{},
		exitFunc:          os.Exit,
		fieldFormats:      &atomic.Pointer[FieldFormats]{},
		fieldStyleLevel:   LevelInfo,
		indentWidth:       2, //nolint:mnd // default indent: 2 spaces per level
		fieldTimeFormat:   time.RFC3339,
		labels:            DefaultLabels(),
		levelAlign:        AlignRight,
		output:            output,
		parts:             DefaultParts(),
		symbols:           DefaultSymbols(),
		separatorText:     "=",
		sliceSep:          ", ",
		smartQuotes:       true,
		styles:            DefaultStyles(),
		printThemeDirty:   true,
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

// Hint returns a new [Event] at hint level, or nil if hint is disabled.
func (l *Logger) Hint() *Event { return l.newEvent(LevelHint) }

// Dry returns a new [Event] at dry level, or nil if dry is disabled.
func (l *Logger) Dry() *Event { return l.newEvent(LevelDry) }

// Warn returns a new [Event] at warn level, or nil if warn is disabled.
func (l *Logger) Warn() *Event { return l.newEvent(LevelWarn) }

// Error returns a new [Event] at error level, or nil if error is disabled.
func (l *Logger) Error() *Event { return l.newEvent(LevelError) }

// Fatal returns a new [Event] at fatal level.
func (l *Logger) Fatal() *Event { return l.newEvent(LevelFatal) }

// Level returns the current minimum log level. A nil [Logger] reports
// [LevelInfo] - the Level zero value, and the level a real logger starts at -
// but emits nothing at any level, which is what [Logger.LevelEnabled] reports.
func (l *Logger) Level() Level {
	if l == nil {
		return LevelInfo
	}
	return Level(l.atomicLevel.Load())
}

// LevelEnabled reports whether the logger handles records at the given level.
// A nil [Logger] handles no level, so it always reports false.
func (l *Logger) LevelEnabled(level Level) bool {
	if l == nil {
		return false
	}

	//nolint:gosec // Level values are small constants (-10 to 15)
	return int32(level) >= l.atomicLevel.Load()
}

// With returns a [Context] for building a sub-logger with preset fields.
//
//	logger := clog.With().Str("component", "auth").Logger()
//	logger.Info().Str("user", "john").Msg("Authenticated")
//
// A sub-logger of a nil [Logger] is inert too, so the whole chain stays safe.
func (l *Logger) With() *Context {
	l = l.orDiscard()

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
// A nil [Logger] is not stored - ctx is returned unchanged, so [Ctx] falls
// back to [Default] rather than handing callers an inert logger.
func (l *Logger) WithContext(ctx context.Context) context.Context {
	if l == nil {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, l)
}

// LogFields logs a message at the given level with the provided timestamp and fields.
// This is used by adapters (e.g. [sloghandler]) that build fields externally.
// The timestamp overrides the [time.Now] value but is only rendered when the
// logger reports timestamps ([Logger.SetReportTimestamp]) -- an adapter inherits
// the logger's timestamp visibility rather than forcing its own. A zero
// timestamp means the source record carries no time (slog semantics): none is
// rendered, even when reporting is enabled.
// Unlike direct Fatal() calls, LogFields does not trigger [os.Exit] for
// [LevelFatal] events -- adapters should not cause process termination.
func (l *Logger) LogFields(level Level, ts time.Time, msg string, fields []Field) {
	e := l.newEvent(level)
	if e == nil {
		return
	}
	e.timestamp = ts
	e.noTimestamp = ts.IsZero()
	e.fields = fields
	e.noExit = true
	e.Msg(msg)
}

// Log returns a new [Event] at the given level from the [Default] logger.
// Use this for custom levels registered with [RegisterLevel].
func Log(level Level) *Event { return Default().Log(level) }

// Trace returns a new trace-level [Event] from the [Default] logger.
func Trace() *Event { return Default().Trace() }

// Debug returns a new debug-level [Event] from the [Default] logger.
func Debug() *Event { return Default().Debug() }

// Info returns a new info-level [Event] from the [Default] logger.
func Info() *Event { return Default().Info() }

// Hint returns a new hint-level [Event] from the [Default] logger.
func Hint() *Event { return Default().Hint() }

// Dry returns a new dry-level [Event] from the [Default] logger.
func Dry() *Event { return Default().Dry() }

// Warn returns a new warn-level [Event] from the [Default] logger.
func Warn() *Event { return Default().Warn() }

// Error returns a new error-level [Event] from the [Default] logger.
func Error() *Event { return Default().Error() }

// Fatal returns a new fatal-level [Event] from the [Default] logger.
func Fatal() *Event { return Default().Fatal() }

// With returns a [Context] for building a sub-logger from the [Default] logger.
func With() *Context { return Default().With() }

// WithContext stores the [Default] logger in ctx.
func WithContext(ctx context.Context) context.Context {
	return Default().WithContext(ctx)
}

// Ctx retrieves the logger from ctx. Returns [Default] if ctx is nil
// or contains no logger.
func Ctx(ctx context.Context) *Logger {
	if ctx == nil {
		return Default()
	}
	if l, ok := ctx.Value(ctxKey{}).(*Logger); ok {
		return l
	}
	return Default()
}

// Dict returns a new detached [Event] for use as a nested dictionary field.
// The event uses the [Default] logger's output for hyperlink/color resolution.
func Dict() *Event { return Default().Dict() }

// Dict returns a new detached [Event] for use as a nested dictionary field.
// The event uses the logger's output for hyperlink/color resolution.
// A nil [Logger] returns a nil Event: every Event method is a no-op on nil,
// and [Event.Dict]/[Context.Dict] drop a nil dictionary.
func (l *Logger) Dict() *Event {
	if l == nil {
		return nil
	}
	return &Event{logger: l, dict: true}
}

// Divider returns a new [DividerBuilder] for rendering a horizontal rule
// using the [Default] logger.
func Divider() *DividerBuilder { return Default().Divider() }

// Print returns a new [Printer] for writing styled output from the [Default] logger.
func Print() *Printer { return Default().Print() }

// GetLevel returns the current log level of the [Default] logger.
func GetLevel() Level {
	return Default().Level()
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
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.animationInterval = d
}

// SetSuppressEchoDuringAnimations controls whether the terminal's local echo
// is disabled while animations (spinners, bars, pulse, shimmer) are live on
// the logger's output. While an animation block is on screen, characters
// typed by the user are echoed by the terminal into the block, corrupting
// its row accounting and leaving stale frames behind in the scrollback.
// When enabled, echo is turned off when the first animation starts and
// restored when the last one finishes (and while the live block is
// suspended). Only echo is affected: line editing, Ctrl-C, and job control
// keep working. The default is off.
//
// The setting sticks to the logger: it is re-applied whenever the output is
// replaced ([Logger.SetOutput], [Logger.SetOutputWriter],
// [Logger.SetColorMode]) and has no effect on non-TTY writers. A crash while
// animations are live can leave the terminal with echo disabled (the same
// exposure as the hidden cursor); `stty echo` or a shell prompt that resets
// terminal modes recovers it.
func (l *Logger) SetSuppressEchoDuringAnimations(suppress bool) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.suppressEcho = suppress
	l.output.SetSuppressEchoDuringAnimations(suppress)
}

// SetColorMode sets the color mode by recreating the logger's [Output]
// with the given mode.
func (l *Logger) SetColorMode(mode ColorMode) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	w := l.output.Writer()
	l.output = NewOutput(w, mode)
	l.output.SetSuppressEchoDuringAnimations(l.suppressEcho)
}

// SetExitFunc sets the function called by Fatal-level events.
// Defaults to [os.Exit]. This can be used in tests to intercept fatal exits.
// If fn is nil, the default [os.Exit] is used.
func (l *Logger) SetExitFunc(fn func(int)) {
	if l == nil {
		return
	}
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
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.exitCode = code
}

// SetFieldSort sets the sort order for fields in log output.
// Default [SortNone] preserves insertion order.
func (l *Logger) SetFieldSort(sort Sort) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fieldSort = sort
}

// SetFieldStyleLevel sets the minimum log level at which field values are
// styled (colored). Events below this level render fields as plain text.
// Defaults to [LevelInfo].
func (l *Logger) SetFieldStyleLevel(level Level) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fieldStyleLevel = level
}

// SetFieldTimeFormat sets the format string used for [time.Time] field values
// added via [Event.Time] and [Context.Time]. Defaults to [time.RFC3339].
func (l *Logger) SetFieldTimeFormat(format string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fieldTimeFormat = format
}

// SetHandler sets a custom log handler. When set, the handler receives all
// log entries instead of the built-in pretty formatter.
func (l *Logger) SetHandler(h Handler) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.handler = h
}

// AddHook registers a function to run at the given [HookPoint].
// Multiple hooks per point are supported; they run in registration order.
// Hooks are called under the logger's mutex.
func (l *Logger) AddHook(point HookPoint, fn func()) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.hooks == nil {
		l.hooks = make(map[HookPoint][]func())
	}
	l.hooks[point] = append(l.hooks[point], fn)
}

// ClearAllHooks removes all registered hooks at every [HookPoint].
func (l *Logger) ClearAllHooks() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hooks = nil
}

// ClearHooks removes all hooks registered at the given [HookPoint].
func (l *Logger) ClearHooks(point HookPoint) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hooks, point)
}

// SetIndent sets the indent depth (number of indent levels) on the logger.
func (l *Logger) SetIndent(levels int) {
	if l == nil {
		return
	}
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
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.indentPrefixes = prefixes
}

// SetIndentPrefixSeparator sets the separator appended after an indent prefix.
// Defaults to " " (a single space).
func (l *Logger) SetIndentPrefixSeparator(sep string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.indentPrefixSep = &sep
}

// SetIndentWidth sets the number of spaces per indent level.
// Defaults to 2.
func (l *Logger) SetIndentWidth(width int) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.indentWidth = width
}

// SetLabelWidth sets an explicit minimum width for level labels.
// If width is 0, the width is computed automatically from the current labels.
func (l *Logger) SetLabelWidth(width int) {
	if l == nil {
		return
	}
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
	if l == nil {
		return
	}
	l.atomicLevel.Store(int32(level)) //nolint:gosec // Level values are small constants (-10 to 15)
}

// SetNonTTYLevel sets the minimum log level for non-TTY writers (CI, piped
// output, etc.). Events below this level are suppressed when the logger's
// output is not connected to a terminal, including animation progress lines.
// Pass [UnsetLevel] to restore the default behaviour.
func (l *Logger) SetNonTTYLevel(level Level) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.nonTTYLevel = level
}

// SetLevelAlign sets the alignment mode for level labels.
func (l *Logger) SetLevelAlign(align Align) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.levelAlign = align
	l.recomputePaddedLabels()
}

// SetLabels sets the level labels used in log output.
// Pass a map from [Level] to label string (e.g. {LevelWarn: "WARN"}).
// Missing levels fall back to the defaults.
func (l *Logger) SetLabels(labels LabelMap) {
	if l == nil {
		return
	}
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
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.omitEmpty = omit
}

// SetOmitZero enables or disables omitting fields with zero values.
// Zero means the zero value for any type (0, false, "", nil, etc.).
// This is a superset of [Logger.SetOmitEmpty].
func (l *Logger) SetOmitZero(omit bool) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.omitZero = omit
}

// SetPrintIndent sets the indentation string used by [Printer] output in
// [PrintMultiline] mode. Defaults to two spaces ("  ").
func (l *Logger) SetPrintIndent(indent string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.printIndent = indent
}

// SetJSONIndent sets a JSON-specific indentation string, overriding [SetPrintIndent]
// for JSON output only. Pass "" to clear and fall back to [SetPrintIndent].
func (l *Logger) SetJSONIndent(indent string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.jsonIndent = indent
}

// SetJSONPrintMode sets the default [JSONPrintMode] for [Printer] JSON output.
// The default is [JSONPretty]. Per-call overrides are available
// via [Printer.Mode].
func (l *Logger) SetJSONPrintMode(mode JSONPrintMode) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.jsonPrintMode = mode
}

// SetYAMLIndent sets a YAML-specific indentation string, overriding [SetPrintIndent]
// for YAML output only. Pass "" to clear and fall back to [SetPrintIndent].
func (l *Logger) SetYAMLIndent(indent string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.yamlIndent = indent
}

// SetYAMLIndentSequence controls whether YAML sequences (arrays) are indented
// under their parent key. Defaults to true. Pass false for compact output:
//
//	# true (default)
//	tags:
//	  - a
//	  - b
//
//	# false
//	tags:
//	- a
//	- b
func (l *Logger) SetYAMLIndentSequence(indent bool) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.yamlIndentSequence = &indent
}

// SetOutput sets the output. Logger-held output state is pushed to the new
// output: hyperlink formats, so [Output.PathLink] and [Output.Hyperlink] keep
// honouring the logger's [FieldFormats], and the echo-suppression setting
// from [Logger.SetSuppressEchoDuringAnimations].
func (l *Logger) SetOutput(out *Output) {
	if l == nil {
		return
	}
	if f := l.fieldFormats.Load(); f != nil {
		out.setHyperlinks(f.hyperlinkConfig())
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = out
	l.output.SetSuppressEchoDuringAnimations(l.suppressEcho)
}

// SetOutputWriter sets the output writer with [ColorAuto].
func (l *Logger) SetOutputWriter(w io.Writer) {
	if l == nil {
		return
	}
	l.SetOutput(NewOutput(w, ColorAuto))
}

// Output returns the logger's [Output]. A nil [Logger] returns the inert
// output every other nil-receiver builder writes to, so callers can use it
// without a nil check of their own.
func (l *Logger) Output() *Output {
	l = l.orDiscard()

	l.mu.Lock()
	defer l.mu.Unlock()
	return l.output
}

// SetInput sets the reader used by [Logger.Input] and [Logger.Password].
// Defaults to [os.Stdin]. Primarily useful for tests. Pass nil to restore
// the default.
func (l *Logger) SetInput(r io.Reader) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if r == nil {
		l.input = nil
		return
	}
	l.input = newInputSource(r)
}

// SetPromptMarker sets a leading marker prepended to every prompt rendered by
// [Logger.Input], [Logger.Password], and their context variants. The marker
// is rendered with the [style.Config.Prompt] style, so it can carry its own
// color independent of the prompt message. Pass "" to disable.
func (l *Logger) SetPromptMarker(marker string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.promptMarker = marker
}

// SetParts sets the order in which parts appear in log output.
// Parts not included in the order are hidden. Parts can be reordered freely.
// Panics if no parts are provided.
func (l *Logger) SetParts(parts ...Part) {
	if l == nil {
		return
	}
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
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	merged := DefaultSymbols()
	maps.Copy(merged, symbols)

	l.symbols = merged
}

// SetQuoteChars sets the opening and closing characters used to quote field
// values that contain spaces or special characters (e.g. '[' and ']', or
// '«' and '»'). Pass the same rune twice for symmetric quoting. The default
// (zero values) uses Go-style double-quoted strings via [strconv.Quote].
// Setting a non-zero openChar overrides [Logger.SetSmartQuotes], even when
// smart quoting is enabled.
func (l *Logger) SetQuoteChars(openChar, closeChar rune) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.quoteOpen = openChar
	l.quoteClose = closeChar
}

// SetQuote sets the quoting behaviour for field values.
// [QuoteAuto] (default) quotes only when needed; [QuoteAlways] always quotes
// string/error/default-kind values; [QuoteNever] never quotes.
func (l *Logger) SetQuote(mode Quote) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.quoteMode = mode
}

// defaultSmartQuoteChars is the delimiter preference order used by smart
// quoting when no custom pairs are configured: double quotes, then single
// quotes, then backticks.
var defaultSmartQuoteChars = []QuotePair{{Open: '"'}, {Open: '\''}, {Open: '`'}}

// SetSmartQuotes enables or disables content-adaptive quoting. When enabled,
// each quoted value is wrapped in the first delimiter pair (see
// [Logger.SetSmartQuoteChars]) whose delimiters do not occur in the value, so
// no escaping is needed; it falls back to Go-style escaped quoting only when
// no pair fits (or the value contains backslashes or non-printable runes).
// [Logger.SetQuoteChars] takes precedence over smart quoting when set explicitly.
func (l *Logger) SetSmartQuotes(enabled bool) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.smartQuotes = enabled
}

// SetSmartQuoteChars sets the delimiter preference order used by smart quoting
// (see [Logger.SetSmartQuotes]). Each pair may use distinct open/close runes
// (e.g. {Open: '«', Close: '»'}); a zero Close mirrors Open. Passing no pairs
// restores the default order ('"', then '\”, then '`'). This configures the
// order only; it does not by itself enable smart quoting.
func (l *Logger) SetSmartQuoteChars(pairs ...QuotePair) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.smartQuoteChars = pairs
}

// smartQuotePairs returns the active smart-quote preference list: nil when
// smart quoting is disabled, when [Logger.SetQuoteChars] has been set
// explicitly (which takes precedence), the configured pairs when set, or
// [defaultSmartQuoteChars] otherwise. Callers must hold l.mu.
func (l *Logger) smartQuotePairs() []QuotePair {
	if !l.smartQuotes || l.quoteOpen != 0 {
		return nil
	}
	if len(l.smartQuoteChars) > 0 {
		return l.smartQuoteChars
	}
	return defaultSmartQuoteChars
}

// SetReportTimestamp enables or disables timestamp reporting.
func (l *Logger) SetReportTimestamp(report bool) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.reportTimestamp = report
}

// SetSeparatorText sets the separator between field keys and values.
// Defaults to "=".
func (l *Logger) SetSeparatorText(sep string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.separatorText = sep
}

// SetSliceBrackets sets the opening and closing characters for slice field
// values. Pass the same rune twice for symmetric brackets.
// Defaults to '[' and ']'.
func (l *Logger) SetSliceBrackets(openChar, closeChar rune) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sliceOpen = openChar
	l.sliceClose = closeChar
}

// SetSliceSeparator sets the separator between elements in slice field values.
// Defaults to ", ".
func (l *Logger) SetSliceSeparator(sep string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sliceSep = sep
}

// resolveSpinnerConfig returns the logger's spinner configuration, falling
// back to [spinner.DefaultConfig] if none has been set. The caller must not
// hold l.mu.
func (l *Logger) resolveSpinnerConfig() spinner.Config {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.spinnerConfig != nil {
		return *l.spinnerConfig
	}
	return spinner.DefaultConfig()
}

// SetSpinnerDefaults sets the default spinner configuration used by
// [Logger.Spinner], built by applying opts over [spinner.DefaultConfig].
func (l *Logger) SetSpinnerDefaults(opts ...spinner.Option) {
	if l == nil {
		return
	}
	cfg := spinner.Apply(opts...)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.spinnerConfig = &cfg
}

// SetTheme sets the light/dark theme pair used for printer styles
// (JSON, YAML, TOML, HCL). The side matching the detected terminal
// background is applied lazily on the next write. Pass nil to restore the
// built-in default pair. For a fixed theme regardless of background, use
// [theme.Single]. Per-token overrides via [SetStyles] still apply.
func (l *Logger) SetTheme(p *theme.Pair) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.printThemePair = p
	l.printThemeDirty = true
}

// resolvePrintThemeLocked lazily themes the printer styles from the output's
// terminal background, at most once. It is a no-op when colors are disabled,
// a theme was set explicitly, or it has already run. l.mu must be held.
func (l *Logger) resolvePrintThemeLocked() {
	if !l.printThemeDirty || l.colorsDisabled() {
		return
	}
	l.printThemeDirty = false

	pair := l.printThemePair
	if pair == nil {
		pair = theme.DefaultPair()
	}
	if pair.Light == pair.Dark {
		l.applyPrintThemeLocked(pair.Light)
		return
	}
	bg, ok := l.output.background()
	if !ok {
		bg = pair.Fallback
	}
	l.applyPrintThemeLocked(pair.ForBackground(bg))
}

// applyPrintThemeLocked rebuilds the theme-derived styles (printer styles and
// value gradients) from t. l.mu must be held. Any style the user supplied
// explicitly via [Logger.SetStyles] is left untouched, so overriding one (e.g.
// JSON) does not opt the rest out of background detection.
func (l *Logger) applyPrintThemeLocked(t *theme.Theme) {
	for i, ts := range themedStyles {
		if !l.styleOverride[i] {
			ts.apply(l.styles, t)
		}
	}
}

// SetStyles merges the given styles into the current style configuration.
// Non-nil pointer fields overwrite existing values; map fields are merged
// key-by-key. Pass nil to reset to [DefaultStyles].
func (l *Logger) SetStyles(styles *style.Config) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if styles == nil {
		l.styles = DefaultStyles()
		l.printThemePair = nil
		l.printThemeDirty = true
		l.styleOverride = styleOverride{}
		return
	}
	l.styles.Merge(styles)
	// A theme-derived style counts as an override only when it differs from the
	// background-adaptable default. Passing DefaultStyles() through unchanged
	// (or with unrelated fields tweaked) must still adapt to the terminal, and
	// each style opts out individually so the rest keep adapting.
	for i, ts := range themedStyles {
		if ts.isCustom(styles) {
			l.styleOverride[i] = true
		}
	}
}

// customStyle reports whether v is a deliberate override rather than the
// background-adaptable default def. A nil v (field left unset) or one matching
// the default is not an override, so it continues to track the terminal.
//
// Printer style structs hold *lipgloss.Style, which is neither ==-comparable
// nor exposes value equality, so a deep compare is the only option here; it
// runs on the cold SetStyles path, never during rendering.
func customStyle[T any](v, def *T) bool {
	return v != nil && !reflect.DeepEqual(v, def)
}

// customGradient reports whether any gradient in s diverges from its
// background-adaptable default. DefaultStyles() pre-populates these with the
// dark defaults, so only a value-level difference counts as a real override.
// ColorStop is all float64 fields, so slices.Equal compares them directly.
func customGradient(s *style.Config) bool {
	elapsedDefault := style.DefaultElapsedGradient()
	percentDefault := style.DefaultPercentGradient()
	switch {
	case s.DurationGradient != nil && !slices.Equal(s.DurationGradient, elapsedDefault):
		return true
	case s.ElapsedGradient != nil && !slices.Equal(s.ElapsedGradient, elapsedDefault):
		return true
	case s.DeadlineGradient != nil && !slices.Equal(s.DeadlineGradient, elapsedDefault):
		return true
	case s.PercentGradient != nil && !slices.Equal(s.PercentGradient, percentDefault):
		return true
	default:
		return false
	}
}

// SetTimeFormat sets the timestamp format string.
func (l *Logger) SetTimeFormat(format string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.timeFormat = format
}

// SetTimeLocation sets the timezone for timestamps. Defaults to [time.Local].
// If loc is nil, [time.Local] is used.
func (l *Logger) SetTimeLocation(loc *time.Location) {
	if l == nil {
		return
	}
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
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.treeChars = chars
}

// SetWrap sets the line wrapping behaviour. When set to [WrapHard] or
// [WrapSoft], log lines exceeding the terminal width are wrapped before
// writing. [WrapSoft] prefers breaking at word boundaries.
// Has no effect on non-TTY outputs.
func (l *Logger) SetWrap(wrap Wrap) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.wrap = wrap
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
	logger := Default()
	logger.mu.Lock()
	logger.labels[lvl] = cfg.Label
	logger.symbols[lvl] = cfg.Symbol
	logger.labelWidth = computeLabelWidth(logger.labels)
	logger.recomputePaddedLabels()
	if cfg.Style != nil {
		if logger.styles.Levels == nil {
			logger.styles.Levels = make(style.LevelMap)
		}
		logger.styles.Levels[lvl] = cfg.Style
	}
	logger.mu.Unlock()
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
	return l.indentationWith(0, nil)
}

// indentationWith returns the indent string for the current indent level
// deepened by extraDepth, with extraTree connectors appended to the logger's
// own.
func (l *Logger) indentationWith(extraDepth int, extraTree []TreePos) string {
	tree := l.tree
	if len(extraTree) > 0 {
		tree = append(append([]TreePos{}, l.tree...), extraTree...)
	}
	return computeIndent(
		l.indent+extraDepth,
		l.indentWidth,
		l.indentPrefixes,
		l.indentPrefixSep,
	) + computeTreeIndent(
		tree,
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
			m[lvl] = xstrings.PadRight(label, maxW)
		case AlignRight:
			m[lvl] = xstrings.PadLeft(label, maxW)
		case AlignCenter:
			m[lvl] = xstrings.PadCenter(label, maxW)
		case AlignNone:
			m[lvl] = label
		}
	}
	l.labelsPadded = m
}

// fieldOpts snapshots the logger's field-formatting configuration for a
// single render. Callers must hold l.mu.
func (l *Logger) fieldOpts(level Level, sort Sort, noColor bool) formatFieldsOpts {
	return formatFieldsOpts{
		fieldSort:       sort,
		fieldStyleLevel: l.fieldStyleLevel,
		formats:         l.loadFieldFormats(),
		level:           level,
		noColor:         noColor,
		quoteOpen:       l.quoteOpen,
		quoteClose:      l.quoteClose,
		quoteMode:       l.quoteMode,
		quoteSmart:      l.smartQuotePairs(),
		separatorText:   l.separatorText,
		sliceClose:      l.sliceClose,
		sliceOpen:       l.sliceOpen,
		sliceSep:        l.sliceSep,
		styles:          l.styles,
		timeFormat:      l.fieldTimeFormat,
	}
}

// eventTime resolves the event's timestamp - explicit override or now - in
// the logger's time location.
func (l *Logger) eventTime(e *Event) time.Time {
	if !e.timestamp.IsZero() {
		return e.timestamp.In(l.timeLocation)
	}
	return time.Now().In(l.timeLocation)
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

	// Resolve omit settings: event override -> logger default.
	omitZero := l.omitZero
	if e.omitZero != nil {
		omitZero = *e.omitZero
	}
	omitEmpty := l.omitEmpty
	if e.omitEmpty != nil {
		omitEmpty = *e.omitEmpty
	}
	fieldSort := l.fieldSort
	if e.sort != nil {
		fieldSort = *e.sort
	}

	// Merge logger context fields with event fields.
	var allFields []Field
	needsFilter := omitZero || omitEmpty
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

	allFields = applyOmit(allFields, omitZero, omitEmpty)

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
		if l.reportTimestamp && !e.noTimestamp {
			entry.Time = l.eventTime(e)
		}

		l.runHooks(HookBeforeWrite)
		l.handler.Log(entry)
		l.runHooks(HookAfterWrite)
		return
	}

	// Built-in pretty formatter.
	noColor := l.colorsDisabled()
	l.resolvePrintThemeLocked()

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
			if !l.reportTimestamp || e.noTimestamp {
				continue
			}

			s = styledTimestamp(l.eventTime(e).Format(l.timeFormat), l.styles, noColor)
		case PartLevel:
			s = styledLevel(l.formatLabel(e.level), e.level, l.styles, noColor)
		case PartSymbol:
			if symbol == "" {
				continue
			}

			s = styledSymbol(symbol, e.level, l.styles, noColor)
		case PartMessage:
			if msg == "" && l.indent == 0 && len(l.tree) == 0 {
				continue
			}

			if noColor {
				s = msg
			} else {
				base := l.styles.Message
				if ms := l.styles.Messages[e.level]; ms != nil {
					base = ms
				}
				if e.msgStyle != nil {
					base = e.msgStyle
				}
				s = l.styles.BacktickMode.Render(msg, base, l.styles.Backtick)
			}

			if l.indent > 0 || len(l.tree) > 0 {
				s = l.indentation() + s
			}
		case PartFields:
			s = strings.TrimLeft(
				formatFields(allFields, l.fieldOpts(e.level, fieldSort, noColor)),
				" ",
			)
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

	line := lineBuf.String()
	if l.wrap != WrapNone {
		if w := l.output.Width(); w > 0 {
			line = wrapLine(line, w, l.wrap)
		}
	}

	l.runHooks(HookBeforeWrite)
	l.output.WriteLine(line + nl)
	l.runHooks(HookAfterWrite)
}

// applyOmit filters fields per the omit settings; omitZero is a superset of
// omitEmpty. The slice is filtered in place - callers whose slice shares a
// backing array must clone first.
func applyOmit(fields []Field, omitZero, omitEmpty bool) []Field {
	switch {
	case omitZero:
		return slices.DeleteFunc(fields, func(f Field) bool {
			return isZeroValue(f.Value)
		})
	case omitEmpty:
		return slices.DeleteFunc(fields, func(f Field) bool {
			return isEmptyValue(f.Value)
		})
	default:
		return fields
	}
}

// runHooks executes all hooks registered at the given point. Must be called with l.mu held.
func (l *Logger) runHooks(point HookPoint) {
	for _, fn := range l.hooks[point] {
		fn()
	}
}

// newEvent creates a new [Event] for the given level.
// Returns nil if the level is below the logger's minimum (all Event methods
// are no-ops on nil), or if the logger itself is nil (see [Logger]).
func (l *Logger) newEvent(level Level) *Event {
	if l == nil {
		return nil
	}

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

// wrapLine wraps a single log line to fit within width columns.
// The wrapping is ANSI-aware, preserving escape sequences (colors, hyperlinks).
func wrapLine(line string, width int, mode Wrap) string {
	switch mode { //nolint:exhaustive // WrapNone handled by default
	case WrapHard:
		return xansi.WrapHard(line, width)
	case WrapSoft:
		return xansi.WrapSoft(line, width)
	default:
		return line
	}
}
