package fx

import (
	"context"
	"slices"
	"sync/atomic"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/field/deadline"
	"github.com/gechr/clog/field/elapsed"
	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/fx/shimmer"
	"github.com/gechr/clog/fx/spinner"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/internal/gradient"
	"github.com/gechr/clog/level"
	xstrings "github.com/gechr/x/strings"
)

// DefaultSymbol is the default icon shown during pulse, shimmer, and bar animations.
const DefaultSymbol = "⏳"

// Builder configures an animation before execution.
// Create one with [NewBuilder] or the root clog convenience constructors
// (Spinner, Pulse, Shimmer, Bar).
type Builder struct {
	core.FieldBuilder[Builder]

	animatedSymbol   bool               // when true, cycles spinnerConfig.Frames as the symbol instead of a static icon
	clearOnCancel    bool               // when true, erase the animation line on context cancellation
	barPercentKey    string             // when set, a formatted percent field is injected each tick
	barProgressPtr   *atomic.Int64      // bar mode: current progress; nil for non-bar modes
	barConfig        bar.Config         // bar mode: visual style
	barTotalPtr      *atomic.Int64      // bar mode: total progress; nil for non-bar modes
	deadlineKey      string             // when set, a formatted countdown field is injected each tick
	deadlineOverride core.DeadlineField // countdown start (From) and per-field overrides set via Deadline's options
	delayDur         time.Duration      // when set, suppresses animation until this duration elapses
	elapsedKey       string             // when set, a formatted elapsed-time field is injected each tick
	elapsedOverride  core.ElapsedField  // per-field gradient overrides set via Elapsed's options
	indentDepth      int                // additional indent depth applied to the animation
	log              Logger             // the logger interface; nil uses Default
	lvl              core.Level         // log level used during animation rendering (default: LevelInfo)
	message          string
	mode             Animation
	msgStyle         *lipgloss.Style // per-builder message text style override; nil = use level style
	partOverrides    *[]core.Part    // nil = use logger's parts
	percentMax       float64         // percent input-range maximum stamped from the logger's FieldFormats; 0 = 1.0
	pulseStops       []gradient.ColorStop
	shimmerDir       shimmer.Direction
	shimmerStops     []gradient.ColorStop
	speed            float64
	spinnerConfig    spinner.Config
	suppressNonTTY   bool           // when true, no output is produced on non-TTY writers
	symbolIcon       string         // icon shown during animation; defaults to DefaultSymbol for pulse/shimmer/bar
	treePos          []core.TreePos // additional tree levels applied to the animation
}

// BuilderConfig provides the initial configuration for a [Builder].
type BuilderConfig struct {
	AnimatedSymbol bool
	BarProgress    *atomic.Int64
	BarConfig      bar.Config
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
	SpinnerConfig  spinner.Config
	SymbolIcon     string
}

// NewBuilder creates a new Builder from the given configuration.
func NewBuilder(cfg BuilderConfig) *Builder {
	b := &Builder{
		animatedSymbol: cfg.AnimatedSymbol,
		barProgressPtr: cfg.BarProgress,
		barConfig:      cfg.BarConfig,
		barTotalPtr:    cfg.BarTotal,
		log:            cfg.Logger,
		lvl:            cfg.Level,
		message:        cfg.Message,
		mode:           cfg.Mode,
		percentMax:     cfg.PercentMax,
		pulseStops:     cfg.PulseStops,
		shimmerDir:     cfg.ShimmerDir,
		shimmerStops:   cfg.ShimmerStops,
		speed:          cfg.Speed,
		spinnerConfig:  cfg.SpinnerConfig,
		symbolIcon:     cfg.SymbolIcon,
	}
	b.InitSelf(b)
	return b
}

// UsesAnimatedSymbol reports whether the symbol slot uses spinner frames.
func (b *Builder) UsesAnimatedSymbol() bool { return b.animatedSymbol }

// BarStyle returns the bar style configuration snapshot.
func (b *Builder) BarStyle() bar.Config {
	cfg := b.barConfig
	cfg.GradientFill = slices.Clone(cfg.GradientFill)
	cfg.ProgressGradient = slices.Clone(cfg.ProgressGradient)
	return cfg
}

// BarProgress returns the current and total bar values.
func (b *Builder) BarProgress() (int64, int64, bool) {
	if b.barProgressPtr == nil || b.barTotalPtr == nil {
		return 0, 0, false
	}
	return b.barProgressPtr.Load(), b.barTotalPtr.Load(), true
}

// DeadlineFieldKey returns the dynamic countdown field key, if enabled.
func (b *Builder) DeadlineFieldKey() string { return b.deadlineKey }

// Delay returns the delay before the animation becomes visible.
func (b *Builder) Delay() time.Duration { return b.delayDur }

// ElapsedFieldKey returns the dynamic elapsed-time field key, if enabled.
func (b *Builder) ElapsedFieldKey() string { return b.elapsedKey }

// IndentLevel returns the additional indent depth applied to the animation.
func (b *Builder) IndentLevel() int { return b.indentDepth }

// LogLevel returns the log level used during animation rendering.
func (b *Builder) LogLevel() core.Level { return b.lvl }

// InitialMessage returns the initial animation message.
func (b *Builder) InitialMessage() string { return b.message }

// AnimationMode returns the animation mode.
func (b *Builder) AnimationMode() Animation { return b.mode }

// PartOrder returns the animation-specific part order override.
func (b *Builder) PartOrder() ([]core.Part, bool) {
	if b.partOverrides == nil {
		return nil, false
	}
	return slices.Clone(*b.partOverrides), true
}

// PulseGradient returns the pulse gradient stops.
func (b *Builder) PulseGradient() []gradient.ColorStop { return slices.Clone(b.pulseStops) }

// ShimmerDirection returns the shimmer animation direction.
func (b *Builder) ShimmerDirection() shimmer.Direction { return b.shimmerDir }

// ShimmerGradient returns the shimmer gradient stops.
func (b *Builder) ShimmerGradient() []gradient.ColorStop { return slices.Clone(b.shimmerStops) }

// AnimationSpeed returns the animation speed multiplier.
func (b *Builder) AnimationSpeed() float64 { return b.speed }

// SpinnerStyle returns the spinner configuration snapshot.
func (b *Builder) SpinnerStyle() spinner.Config {
	cfg := b.spinnerConfig
	cfg.Frames = slices.Clone(cfg.Frames)
	return cfg
}

// SuppressesNonTTY reports whether animation output is suppressed on non-TTY writers.
func (b *Builder) SuppressesNonTTY() bool { return b.suppressNonTTY }

// SymbolOverride returns the static animation icon override, if set.
func (b *Builder) SymbolOverride() string { return b.symbolIcon }

// MessageStyleOverride returns the per-builder message text style override, or
// nil to use the level style.
func (b *Builder) MessageStyleOverride() *lipgloss.Style { return b.msgStyle }

// TreePositions returns the additional tree levels applied to the animation.
func (b *Builder) TreePositions() []core.TreePos { return slices.Clone(b.treePos) }

// ResolveLogger returns the builder's logger, falling back to the given default.
func (b *Builder) ResolveLogger(def Logger) Logger {
	if b.log != nil {
		return b.log
	}
	return def
}

// IndentedLogger returns the builder's logger with any builder-level
// indent depth and tree levels applied. Used for completion logging so the
// result message has the same indentation as the animation.
func (b *Builder) IndentedLogger(def Logger) Logger {
	l := b.ResolveLogger(def)
	if b.indentDepth > 0 || len(b.treePos) > 0 {
		l = l.WithIndent(b.indentDepth, b.treePos)
	}
	return l
}

// Dedent removes one indent level from the animation output, down to a
// minimum of zero.
func (b *Builder) Dedent() *Builder {
	if b.indentDepth > 0 {
		b.indentDepth--
	}
	return b
}

// Depth adds multiple indent levels to the animation output.
func (b *Builder) Depth(n int) *Builder {
	b.indentDepth += n
	return b
}

// Indent adds one indent level to the animation output.
func (b *Builder) Indent() *Builder {
	b.indentDepth++
	return b
}

// Tree adds one tree-nesting level with the given position.
func (b *Builder) Tree(pos core.TreePos) *Builder {
	b.treePos = append(b.treePos, pos)
	return b
}

// NonTTYSilent controls whether animation output is suppressed entirely on
// non-TTY writers (CI, piped output). When silent is true, no output is
// produced. When false (default), a static line is printed.
func (b *Builder) NonTTYSilent(silent bool) *Builder {
	b.suppressNonTTY = silent
	return b
}

// After sets a delay before the animation becomes visible.
func (b *Builder) After(d time.Duration) *Builder {
	b.delayDur = d
	return b
}

// Parts overrides the log-line part order for this animation and its
// completion message.
func (b *Builder) Parts(parts ...core.Part) *Builder {
	b.partOverrides = new(parts)
	return b
}

// Symbol sets the icon displayed beside the message during animation.
func (b *Builder) Symbol(symbol string) *Builder {
	b.symbolIcon = symbol
	return b
}

// MessageStyle overrides the message text style for both the live animation and
// its completion line, taking precedence over the global and per-level styles
// without mutating them. An empty [lipgloss.NewStyle] renders the message plain.
func (b *Builder) MessageStyle(s *lipgloss.Style) *Builder {
	b.msgStyle = s
	return b
}

// Spinner enables an animated spinning symbol. The symbol slot cycles
// through spinner frames independently of the main animation mode.
// Options override the builder's existing spinner configuration. With no options
// the current style (set by the constructor or logger default) is used.
func (b *Builder) Spinner(opts ...spinner.Option) *Builder {
	b.animatedSymbol = true
	for _, o := range opts {
		o(&b.spinnerConfig)
	}
	return b
}

// Elapsed enables an auto-updating elapsed-time field. Use options from the
// [elapsed] package (e.g. [elapsed.WithGradientMax]) to override the
// logger's gradient settings for this builder's elapsed field only.
func (b *Builder) Elapsed(key string, opts ...elapsed.Option) *Builder {
	b.elapsedKey = key
	elapsed.Apply(&b.elapsedOverride, opts...)
	b.Fields = append(b.Fields, core.Field{Key: key, Value: b.elapsedOverride})
	return b
}

// Deadline enables an auto-updating countdown field that displays the time
// remaining until from has elapsed, clamped at 0. Coloring runs against the
// logger's elapsed gradient by consumed time, so a fresh deadline starts at
// the gradient's first stop (green) and an expiring one ends at the last
// (red). Use options from the [deadline] package (e.g. [deadline.WithGradient])
// to override the gradient settings for this builder's deadline field only.
func (b *Builder) Deadline(key string, from time.Duration, opts ...deadline.Option) *Builder {
	b.deadlineKey = key
	b.deadlineOverride.From = from
	b.deadlineOverride.Remaining = from
	deadline.Apply(&b.deadlineOverride, opts...)
	b.Fields = append(b.Fields, core.Field{Key: key, Value: b.deadlineOverride})
	return b
}

// BarPercent enables an auto-updating percentage field for [AnimationBar].
func (b *Builder) BarPercent(key string) *Builder {
	b.barPercentKey = key
	b.Fields = append(b.Fields, core.Field{Key: key, Value: core.Percent{}})
	return b
}

// BarPercentValue returns the current progress as a [core.Percent].
func (b *Builder) BarPercentValue() core.Percent {
	cur := int(b.barProgressPtr.Load())
	tot := int(b.barTotalPtr.Load())
	m := b.percentMaximum()
	pct := float64(cur) / float64(max(tot, 1)) * m
	return core.Percent{Value: min(pct, m)}
}

// percentMaximum returns the stamped percent input-range maximum,
// defaulting to 1.0 when unset.
func (b *Builder) percentMaximum() float64 {
	if b.percentMax > 0 {
		return b.percentMax
	}
	return 1
}

// StripDynamicFields returns fields with animation-only dynamic fields removed.
func (b *Builder) StripDynamicFields(fields []core.Field) []core.Field {
	if xstrings.AllEmpty(b.deadlineKey, b.elapsedKey, b.barPercentKey) {
		return fields
	}
	out := make([]core.Field, 0, len(fields))
	for _, f := range fields {
		if f.Key == b.deadlineKey || f.Key == b.elapsedKey || f.Key == b.barPercentKey {
			continue
		}
		out = append(out, f)
	}
	return out
}

// ResolveDynamicFields clones fields and injects elapsed/deadline/percent values.
func (b *Builder) ResolveDynamicFields(fields []core.Field, dur time.Duration) []core.Field {
	if xstrings.AllEmpty(b.deadlineKey, b.elapsedKey, b.barPercentKey) {
		return fields
	}

	out := make([]core.Field, len(fields))
	copy(out, fields)
	for i := range out {
		switch out[i].Key {
		case b.deadlineKey:
			f := b.deadlineOverride
			f.Remaining = max(f.From-dur, 0)
			out[i].Value = f
		case b.elapsedKey:
			f := b.elapsedOverride
			f.Value = dur
			out[i].Value = f
		case b.barPercentKey:
			out[i].Value = b.BarPercentValue()
		}
	}
	if b.elapsedOverride.Trailing && b.elapsedKey != "" {
		out = moveFieldLast(out, b.elapsedKey)
	}
	if b.deadlineOverride.Trailing && b.deadlineKey != "" {
		out = moveFieldLast(out, b.deadlineKey)
	}
	return out
}

// moveFieldLast returns out with the first field matching key moved to the
// end, preserving the order of the rest. out must be a fresh copy - the shift
// mutates it in place.
func moveFieldLast(out []core.Field, key string) []core.Field {
	idx := -1
	for i := range out {
		if out[i].Key == key {
			idx = i
			break
		}
	}
	if idx < 0 {
		return out
	}
	f := out[idx]
	out = append(out[:idx], out[idx+1:]...)
	return append(out, f)
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

	msgPtr.Store(&b.message)
	fieldsPtr.Store(&b.Fields)
	sym := b.symbolIcon
	if sym == "" {
		sym = DefaultSymbol
	}
	levelPtr.Store(int64(level.Unset))
	symbolPtr.Store(&sym)

	update := &Update{
		msgText:   b.message,
		msgPtr:    &msgPtr,
		fieldsPtr: &fieldsPtr,
		base:      b.Fields,
		levelPtr:  &levelPtr,
		symbolPtr: &symbolPtr,
	}
	if b.mode == AnimationBar {
		update.progressPtr = b.barProgressPtr
		update.totalPtr = b.barTotalPtr
	}
	update.InitSelf(update)

	wrapped := func(ctx context.Context) error {
		return task(ctx, update)
	}

	startTime := time.Now()
	l := b.log
	err := runAnimation(ctx, b, wrapped, &msgPtr, &fieldsPtr, &levelPtr, &symbolPtr, startTime)

	msg := *msgPtr.Load()
	w := NewWaitResult(err, b.IndentedLogger(l), b.partOverrides, b.lvl, msg)
	w.MsgStyle = b.msgStyle
	w.Fields = b.ResolveDynamicFields(*fieldsPtr.Load(), time.Since(startTime))
	return w
}
