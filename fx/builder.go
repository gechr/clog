package fx

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/fx/shimmer"
	"github.com/gechr/clog/fx/spinner"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/internal/gradient"
	"github.com/gechr/clog/level"
)

// DefaultSymbol is the default icon shown during pulse, shimmer, and bar animations.
const DefaultSymbol = "⏳"

// Builder configures an animation before execution.
// Create one with [NewBuilder] or the root clog convenience constructors
// (Spinner, Pulse, Shimmer, Bar).
type Builder struct {
	core.FieldBuilder[Builder]

	AnimatedSymbol bool          // when true, cycles SpinnerStyle.Frames as the symbol instead of a static icon
	ClearOnCancel  bool          // when true, erase the animation line on context cancellation
	BarPercentKey  string        // when set, a formatted percent field is injected each tick
	BarProgressPtr *atomic.Int64 // bar mode: current progress; nil for non-bar modes
	BarStyle       bar.Style     // bar mode: visual style
	BarTotalPtr    *atomic.Int64 // bar mode: total progress; nil for non-bar modes
	DelayDur       time.Duration // when set, suppresses animation until this duration elapses
	ElapsedKey     string        // when set, a formatted elapsed-time field is injected each tick
	IndentDepth    int           // additional indent depth applied to the animation
	Log            Logger        // the logger interface; nil uses Default
	Level          core.Level    // log level used during animation rendering (default: LevelInfo)
	Message        string
	Mode           Animation
	PartOverrides  *[]core.Part // nil = use logger's parts
	PercentMax     float64      // percent input-range maximum stamped from the logger's FieldFormats; 0 = 1.0
	PulseStops     []gradient.ColorStop
	ShimmerDir     shimmer.Direction
	ShimmerStops   []gradient.ColorStop
	Speed          float64
	SpinnerStyle   spinner.Style
	SuppressNonTTY bool           // when true, no output is produced on non-TTY writers
	SymbolIcon     string         // icon shown during animation; defaults to DefaultSymbol for pulse/shimmer/bar
	TreePos        []core.TreePos // additional tree levels applied to the animation
}

// BuilderConfig provides the initial configuration for a [Builder].
type BuilderConfig struct {
	AnimatedSymbol bool
	BarProgress    *atomic.Int64
	BarStyle       bar.Style
	BarTotal       *atomic.Int64
	Level          core.Level
	Logger         Logger
	Message        string
	Mode           Animation
	PercentMax     float64
	PulseStops     []gradient.ColorStop
	ShimmerDir     shimmer.Direction
	ShimmerStops   []gradient.ColorStop
	Speed          float64
	SpinnerStyle   spinner.Style
	SymbolIcon     string
}

// NewBuilder creates a new Builder from the given configuration.
func NewBuilder(cfg BuilderConfig) *Builder {
	b := &Builder{
		AnimatedSymbol: cfg.AnimatedSymbol,
		BarProgressPtr: cfg.BarProgress,
		BarStyle:       cfg.BarStyle,
		BarTotalPtr:    cfg.BarTotal,
		Level:          cfg.Level,
		Log:            cfg.Logger,
		Mode:           cfg.Mode,
		Message:        cfg.Message,
		PercentMax:     cfg.PercentMax,
		SymbolIcon:     cfg.SymbolIcon,
		PulseStops:     cfg.PulseStops,
		ShimmerDir:     cfg.ShimmerDir,
		ShimmerStops:   cfg.ShimmerStops,
		Speed:          cfg.Speed,
		SpinnerStyle:   cfg.SpinnerStyle,
	}
	b.InitSelf(b)
	return b
}

// ResolveLogger returns the builder's logger, falling back to the given default.
func (b *Builder) ResolveLogger(def Logger) Logger {
	if b.Log != nil {
		return b.Log
	}
	return def
}

// IndentedLogger returns the builder's logger with any builder-level
// indent depth and tree levels applied. Used for completion logging so the
// result message has the same indentation as the animation.
func (b *Builder) IndentedLogger(def Logger) Logger {
	l := b.ResolveLogger(def)
	if b.IndentDepth > 0 || len(b.TreePos) > 0 {
		l = l.WithIndent(b.IndentDepth, b.TreePos)
	}
	return l
}

// Dedent removes one indent level from the animation output, down to a
// minimum of zero.
func (b *Builder) Dedent() *Builder {
	if b.IndentDepth > 0 {
		b.IndentDepth--
	}
	return b
}

// Depth adds multiple indent levels to the animation output.
func (b *Builder) Depth(n int) *Builder {
	b.IndentDepth += n
	return b
}

// Indent adds one indent level to the animation output.
func (b *Builder) Indent() *Builder {
	b.IndentDepth++
	return b
}

// Tree adds one tree-nesting level with the given position.
func (b *Builder) Tree(pos core.TreePos) *Builder {
	b.TreePos = append(b.TreePos, pos)
	return b
}

// NonTTYSilent controls whether animation output is suppressed entirely on
// non-TTY writers (CI, piped output). When silent is true, no output is
// produced. When false (default), a static line is printed.
func (b *Builder) NonTTYSilent(silent bool) *Builder {
	b.SuppressNonTTY = silent
	return b
}

// After sets a delay before the animation becomes visible.
func (b *Builder) After(d time.Duration) *Builder {
	b.DelayDur = d
	return b
}

// Parts overrides the log-line part order for this animation and its
// completion message.
func (b *Builder) Parts(parts ...core.Part) *Builder {
	b.PartOverrides = new(parts)
	return b
}

// Symbol sets the icon displayed beside the message during animation.
func (b *Builder) Symbol(symbol string) *Builder {
	b.SymbolIcon = symbol
	return b
}

// Spinner enables an animated spinning symbol. The symbol slot cycles
// through [SpinnerStyle] frames independently of the main animation mode.
// Options override the builder's existing [SpinnerStyle]. With no options
// the current style (set by the constructor or logger default) is used.
func (b *Builder) Spinner(opts ...spinner.Option) *Builder {
	b.AnimatedSymbol = true
	for _, o := range opts {
		o(&b.SpinnerStyle)
	}
	return b
}

// Elapsed enables an auto-updating elapsed-time field.
func (b *Builder) Elapsed(key string) *Builder {
	b.ElapsedKey = key
	b.Fields = append(b.Fields, core.Field{Key: key, Value: core.ElapsedField(0)})
	return b
}

// BarPercent enables an auto-updating percentage field for [AnimationBar].
func (b *Builder) BarPercent(key string) *Builder {
	b.BarPercentKey = key
	b.Fields = append(b.Fields, core.Field{Key: key, Value: core.Percent{}})
	return b
}

// BarPercentValue returns the current progress as a [core.Percent].
func (b *Builder) BarPercentValue() core.Percent {
	cur := int(b.BarProgressPtr.Load())
	tot := int(b.BarTotalPtr.Load())
	m := b.percentMaximum()
	pct := float64(cur) / float64(max(tot, 1)) * m
	return core.Percent{Value: min(pct, m)}
}

// percentMaximum returns the stamped percent input-range maximum,
// defaulting to 1.0 when unset.
func (b *Builder) percentMaximum() float64 {
	if b.PercentMax > 0 {
		return b.PercentMax
	}
	return 1
}

// StripDynamicFields returns fields with animation-only dynamic fields removed.
func (b *Builder) StripDynamicFields(fields []core.Field) []core.Field {
	if b.ElapsedKey == "" && b.BarPercentKey == "" {
		return fields
	}
	out := make([]core.Field, 0, len(fields))
	for _, f := range fields {
		if f.Key == b.ElapsedKey || f.Key == b.BarPercentKey {
			continue
		}
		out = append(out, f)
	}
	return out
}

// ResolveDynamicFields clones fields and injects elapsed/percent values.
func (b *Builder) ResolveDynamicFields(fields []core.Field, dur time.Duration) []core.Field {
	if b.ElapsedKey == "" && b.BarPercentKey == "" {
		return fields
	}

	out := make([]core.Field, len(fields))
	copy(out, fields)
	for i := range out {
		switch out[i].Key {
		case b.ElapsedKey:
			out[i].Value = core.ElapsedField(dur)
		case b.BarPercentKey:
			out[i].Value = b.BarPercentValue()
		}
	}
	return out
}

// Path adds a file path field as a clickable terminal hyperlink.
func (b *Builder) Path(key, path string) *Builder {
	output := b.ResolveLogger(nil).Output()
	b.Fields = append(b.Fields, core.Field{Key: key, Value: output.PathLink(path, 0, 0)})
	return b
}

// Line adds a file path field with a line number as a clickable terminal hyperlink.
func (b *Builder) Line(key, path string, line int) *Builder {
	output := b.ResolveLogger(nil).Output()
	if line < 1 {
		line = 1
	}
	b.Fields = append(b.Fields, core.Field{Key: key, Value: output.PathLink(path, line, 0)})
	return b
}

// Column adds a file path field with a line and column number as a clickable terminal hyperlink.
func (b *Builder) Column(key, path string, line, column int) *Builder {
	output := b.ResolveLogger(nil).Output()
	if line < 1 {
		line = 1
	}
	if column < 1 {
		column = 1
	}
	b.Fields = append(b.Fields, core.Field{Key: key, Value: output.PathLink(path, line, column)})
	return b
}

// URL adds a field as a clickable terminal hyperlink.
func (b *Builder) URL(key, url string) *Builder {
	output := b.ResolveLogger(nil).Output()
	b.Fields = append(b.Fields, core.Field{Key: key, Value: output.Hyperlink(url, url)})
	return b
}

// Link adds a field as a clickable terminal hyperlink with custom display text.
func (b *Builder) Link(key, url, text string) *Builder {
	output := b.ResolveLogger(nil).Output()
	b.Fields = append(b.Fields, core.Field{Key: key, Value: output.Hyperlink(url, text)})
	return b
}

// Wait executes the task with the animation and returns a [WaitResult] for chaining.
func (b *Builder) Wait(ctx context.Context, task TaskFunc) *WaitResult {
	return b.Progress(ctx, func(ctx context.Context, _ *Update) error {
		return task(ctx)
	})
}

// Progress executes the task with the animation whose message and fields
// can be updated via the [Update] builder.
func (b *Builder) Progress(ctx context.Context, task UpdateFunc) *WaitResult {
	var fieldsPtr atomic.Pointer[[]core.Field]
	var levelPtr atomic.Int64
	var msgPtr atomic.Pointer[string]
	var symbolPtr atomic.Pointer[string]

	msgPtr.Store(&b.Message)
	fieldsPtr.Store(&b.Fields)
	sym := b.SymbolIcon
	if sym == "" {
		sym = DefaultSymbol
	}
	levelPtr.Store(int64(level.Unset))
	symbolPtr.Store(&sym)

	update := &Update{
		MsgText:   b.Message,
		MsgPtr:    &msgPtr,
		FieldsPtr: &fieldsPtr,
		Base:      b.Fields,
		LevelPtr:  &levelPtr,
		SymbolPtr: &symbolPtr,
	}
	if b.Mode == AnimationBar {
		update.ProgressPtr = b.BarProgressPtr
		update.TotalPtr = b.BarTotalPtr
	}
	update.InitSelf(update)

	wrapped := func(ctx context.Context) error {
		return task(ctx, update)
	}

	startTime := time.Now()
	l := b.Log
	err := runAnimation(ctx, b, wrapped, &msgPtr, &fieldsPtr, &levelPtr, &symbolPtr, startTime)

	msg := *msgPtr.Load()
	w := NewWaitResult(err, b.IndentedLogger(l), b.PartOverrides, b.Level, msg)
	w.Fields = b.ResolveDynamicFields(*fieldsPtr.Load(), time.Since(startTime))
	return w
}
