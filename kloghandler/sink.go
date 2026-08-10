// Package kloghandler provides a [logr.LogSink] adapter for clog, plus helpers
// for routing k8s.io/klog/v2 through it.
//
// klog has no handler interface of its own: [klog.SetLogger] accepts a
// [logr.Logger], so the bridge is a logr sink. That makes this package usable
// by anything built on logr - controller-runtime, client-go, operator-sdk -
// and not just klog. Use [SetKlog] for the klog wiring, or [NewLogger] to hand
// a [logr.Logger] to any other consumer.
package kloghandler

import (
	"fmt"
	"runtime"
	"slices"
	"strconv"
	"time"

	"github.com/gechr/clog"
	"github.com/go-logr/logr"
)

const (
	// DefaultNameKey is the field key holding the dot-joined
	// [logr.Logger.WithName] segments.
	DefaultNameKey = "logger"

	// SourceKey is the field key holding file:line when
	// [Options.AddSource] is set.
	SourceKey = "source"

	// badKey is used for a value with no matching key, mirroring
	// [log/slog]'s handling of an odd key-value count.
	badKey = "!BADKEY"

	// derivedFields is the headroom a record needs beyond its preset and
	// call-site fields: the logger name, the error, and the source.
	derivedFields = 3
)

// Options configures a [Sink].
type Options struct {
	// LevelFor maps a logr V-level to a clog level.
	// Nil uses [VerbosityLevel].
	LevelFor func(verbosity int) clog.Level

	// Verbosity caps the logr V-level reported as enabled, mirroring klog's
	// -v flag. Nil defers to the logger's own level, which is the mapping in
	// [VerbosityLevel] read backwards.
	Verbosity *int

	// NameKey is the field key for [logr.Logger.WithName] segments.
	// Empty uses [DefaultNameKey]. Unnamed loggers emit no such field.
	NameKey string

	// AddSource adds source file:line information to each log entry.
	AddSource bool

	// KlogSeverity preserves the severity of klog's unstructured calls
	// (Info, Warning, Error, Fatal), which klog otherwise flattens to Info
	// before handing them to a logr logger. Only [SetKlog] reads this; see
	// its documentation for the trade-off it carries.
	KlogSeverity bool
}

// Sink implements [logr.LogSink] by routing records through a clog
// [clog.Logger].
type Sink struct {
	logger *clog.Logger

	fields    []clog.Field // preset fields from WithValues (immutable after creation)
	name      string       // dot-joined WithName segments (immutable after creation)
	callDepth int          // extra frames to skip, from WithCallDepth
	initDepth int          // frames logr itself adds, from Init
	opts      Options
}

var (
	_ logr.LogSink          = (*Sink)(nil)
	_ logr.CallDepthLogSink = (*Sink)(nil)
)

// New creates a [logr.LogSink] that routes records through the given clog
// [clog.Logger]. Most callers want [NewLogger] or [SetKlog] instead.
func New(logger *clog.Logger, opts *Options) logr.LogSink {
	return newSink(logger, opts)
}

// NewLogger creates a [logr.Logger] backed by the given clog [clog.Logger],
// ready to hand to any logr consumer.
//
//	ctrl.SetLogger(kloghandler.NewLogger(clog.Default(), nil))
func NewLogger(logger *clog.Logger, opts *Options) logr.Logger {
	return logr.New(newSink(logger, opts))
}

func newSink(logger *clog.Logger, opts *Options) *Sink {
	s := &Sink{logger: logger}
	if opts != nil {
		s.opts = *opts
	}
	return s
}

// VerbosityLevel maps a logr V-level to a clog level following logr's own
// convention: V(0) is informational, V(1) is debug, and V(2) and above is
// trace. Override it with [Options.LevelFor] to follow a different scheme -
// Kubernetes components, for instance, treat V(4) as debug and V(5) as trace.
func VerbosityLevel(verbosity int) clog.Level {
	switch {
	case verbosity <= 0:
		return clog.LevelInfo
	case verbosity == 1:
		return clog.LevelDebug
	default:
		return clog.LevelTrace
	}
}

// Init receives runtime information from logr. It is called once, before the
// sink is shared, so mutating the receiver here is safe.
func (s *Sink) Init(info logr.RuntimeInfo) {
	s.initDepth = info.CallDepth
}

// Enabled reports whether the sink handles records at the given V-level.
func (s *Sink) Enabled(level int) bool {
	if s.opts.Verbosity != nil && level > *s.opts.Verbosity {
		return false
	}
	return s.logger.LevelEnabled(s.levelFor(level))
}

// Info logs a non-error record at the given V-level.
func (s *Sink) Info(level int, msg string, keysAndValues ...any) {
	s.log(s.levelFor(level), nil, msg, keysAndValues)
}

// Error logs an error record. The error is attached as a [clog.ErrorKey]
// field; a nil error is omitted, which is how klog reports its own
// unstructured Error calls.
func (s *Sink) Error(err error, msg string, keysAndValues ...any) {
	s.log(clog.LevelError, err, msg, keysAndValues)
}

// WithValues returns a new [Sink] with the given key-value pairs preset.
func (s *Sink) WithValues(keysAndValues ...any) logr.LogSink {
	if len(keysAndValues) == 0 {
		return s
	}

	fields := make([]clog.Field, 0, len(s.fields)+len(keysAndValues)/2+1)
	fields = append(fields, s.fields...)
	appendKeysAndValues(&fields, keysAndValues)

	c := *s
	c.fields = fields
	return &c
}

// WithName returns a new [Sink] with name appended to the existing name,
// joined with a dot.
func (s *Sink) WithName(name string) logr.LogSink {
	if name == "" {
		return s
	}

	c := *s
	c.fields = slices.Clone(s.fields)
	if c.name == "" {
		c.name = name
	} else {
		c.name += "." + name
	}
	return &c
}

// WithCallDepth returns a new [Sink] that skips depth additional frames when
// resolving the source location, implementing [logr.CallDepthLogSink].
func (s *Sink) WithCallDepth(depth int) logr.LogSink {
	c := *s
	c.fields = slices.Clone(s.fields)
	c.callDepth += depth
	return &c
}

// levelFor maps a logr V-level to a clog level.
func (s *Sink) levelFor(verbosity int) clog.Level {
	if s.opts.LevelFor != nil {
		return s.opts.LevelFor(verbosity)
	}
	return VerbosityLevel(verbosity)
}

// log builds the field list and emits the record.
func (s *Sink) log(level clog.Level, err error, msg string, keysAndValues []any) {
	fields := make([]clog.Field, 0, len(s.fields)+len(keysAndValues)/2+derivedFields)
	fields = s.appendPreset(fields)

	if err != nil {
		fields = append(fields, clog.Field{Key: clog.ErrorKey, Value: err})
	}
	if s.opts.AddSource {
		if src, ok := caller(s.initDepth + s.callDepth); ok {
			fields = append(fields, clog.Field{Key: SourceKey, Value: src})
		}
	}

	appendKeysAndValues(&fields, keysAndValues)
	s.logger.LogFields(level, time.Now(), msg, fields)
}

// appendPreset appends the logger name and WithValues fields to dst.
func (s *Sink) appendPreset(dst []clog.Field) []clog.Field {
	if s.name != "" {
		key := s.opts.NameKey
		if key == "" {
			key = DefaultNameKey
		}
		dst = append(dst, clog.Field{Key: key, Value: s.name})
	}
	return append(dst, s.fields...)
}

// caller resolves the source location extra frames above [Sink.log]. The fixed
// offset covers runtime.Callers, this function, [Sink.log], the Info or Error
// method, and logr's own wrapper - the latter reported via [Sink.Init].
func caller(extra int) (string, bool) {
	const baseSkip = 4

	var pcs [1]uintptr
	if runtime.Callers(baseSkip+extra, pcs[:]) == 0 {
		return "", false
	}

	f, _ := runtime.CallersFrames(pcs[:]).Next()
	if f.File == "" {
		return "", false
	}
	return f.File + ":" + strconv.Itoa(f.Line), true
}

// appendKeysAndValues converts logr's variadic key-value pairs into fields.
// Non-string keys are formatted with [fmt.Sprint], and a trailing value with
// no key is recorded under [badKey].
func appendKeysAndValues(dst *[]clog.Field, keysAndValues []any) {
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 == len(keysAndValues) {
			*dst = append(*dst, clog.Field{Key: badKey, Value: keysAndValues[i]})
			return
		}

		key, ok := keysAndValues[i].(string)
		if !ok {
			key = fmt.Sprint(keysAndValues[i])
		}

		*dst = append(*dst, clog.Field{Key: key, Value: resolve(keysAndValues[i+1])})
	}
}

// resolve unwraps a [logr.Marshaler], logr's analogue of [log/slog.LogValuer].
// Only one level is unwrapped, matching the behaviour of other logr backends.
func resolve(v any) any {
	if m, ok := v.(logr.Marshaler); ok {
		return m.MarshalLog()
	}
	return v
}
