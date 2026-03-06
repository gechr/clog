package fx

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/gechr/clog/field/percent"
	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/fx/shimmer"
	"github.com/gechr/clog/fx/spinner"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/internal/gradient"
)

// Builder configures an animation before execution.
// Create one with [NewBuilder] or the root clog convenience constructors
// (Spinner, Pulse, Shimmer, Bar).
type Builder struct {
	core.FieldBuilder[Builder]

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
	PulseStops     []gradient.ColorStop
	ShimmerDir     shimmer.Direction
	ShimmerStops   []gradient.ColorStop
	Speed          float64
	SpinnerStyle   spinner.Style
	SuppressNonTTY bool           // when true, no output is produced on non-TTY writers
	SymbolIcon     string         // icon shown during animation; defaults to "⏳" for pulse/shimmer/bar
	TreePos        []core.TreePos // additional tree levels applied to the animation
}

// BuilderConfig provides the initial configuration for a [Builder].
type BuilderConfig struct {
	BarProgress  *atomic.Int64
	BarStyle     bar.Style
	BarTotal     *atomic.Int64
	Level        core.Level
	Logger       Logger
	Message      string
	Mode         Animation
	PulseStops   []gradient.ColorStop
	ShimmerDir   shimmer.Direction
	ShimmerStops []gradient.ColorStop
	Speed        float64
	SpinnerStyle spinner.Style
	SymbolIcon   string
}

// NewBuilder creates a new Builder from the given configuration.
func NewBuilder(cfg BuilderConfig) *Builder {
	b := &Builder{
		BarProgressPtr: cfg.BarProgress,
		BarStyle:       cfg.BarStyle,
		BarTotalPtr:    cfg.BarTotal,
		Level:          cfg.Level,
		Log:            cfg.Logger,
		Mode:           cfg.Mode,
		Message:        cfg.Message,
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
	m := percent.Maximum()
	pct := float64(cur) / float64(max(tot, 1)) * m
	return core.Percent{Value: min(pct, m)}
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
	var msgPtr atomic.Pointer[string]
	var fieldsPtr atomic.Pointer[[]core.Field]

	msgPtr.Store(&b.Message)
	fieldsPtr.Store(&b.Fields)

	update := &Update{
		MsgText:   b.Message,
		MsgPtr:    &msgPtr,
		FieldsPtr: &fieldsPtr,
		Base:      b.Fields,
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
	err := l.RunAnimation(ctx, AnimationConfig{
		Builder:   b,
		Task:      wrapped,
		MsgPtr:    &msgPtr,
		FieldsPtr: &fieldsPtr,
		StartTime: startTime,
	})

	msg := *msgPtr.Load()
	w := NewWaitResult(err, b.IndentedLogger(l), b.PartOverrides, b.Level, msg)
	w.Fields = b.ResolveDynamicFields(*fieldsPtr.Load(), time.Since(startTime))
	return w
}
