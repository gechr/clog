package clog

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"slices"
	"strings"
)

// SlogOptions configures a [SlogHandler].
type SlogOptions struct {
	// AddSource adds source file:line information to each log entry.
	AddSource bool
	// Level overrides the minimum log level. If nil, the logger's level is used.
	Level slog.Leveler
}

// SlogHandler implements [slog.Handler] by routing records through a clog [Logger].
type SlogHandler struct {
	logger *Logger
	attrs  []Field  // preset fields from WithAttrs (immutable after creation)
	groups []string // group prefix stack from WithGroup (immutable after creation)
	opts   SlogOptions
}

// NewSlogHandler creates a [slog.Handler] that routes [slog.Record] entries
// through the given clog [Logger].
func NewSlogHandler(logger *Logger, opts *SlogOptions) slog.Handler {
	h := &SlogHandler{logger: logger}
	if opts != nil {
		h.opts = *opts
	}
	return h
}

// Enabled reports whether the handler handles records at the given level.
func (h *SlogHandler) Enabled(_ context.Context, level slog.Level) bool {
	if h.opts.Level != nil {
		return level >= h.opts.Level.Level()
	}
	//nolint:gosec // Level values are small constants (0-6)
	return int32(slogLevelToClog(level)) >= h.logger.atomicLevel.Load()
}

// Handle converts a [slog.Record] into a clog [Event] and logs it.
func (h *SlogHandler) Handle(_ context.Context, r slog.Record) error {
	level := slogLevelToClog(r.Level)

	e := &Event{
		logger:    h.logger,
		level:     level,
		timestamp: r.Time,
	}

	// Add preset attrs first (handler attrs before record attrs).
	if len(h.attrs) > 0 {
		e.fields = slices.Clone(h.attrs)
	}

	// Add source if configured.
	if h.opts.AddSource && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		e.fields = append(e.fields, Field{
			Key:   slog.SourceKey,
			Value: fmt.Sprintf("%s:%d", f.File, f.Line),
		})
	}

	// Convert record attrs.
	prefix := h.groupPrefix()
	r.Attrs(func(a slog.Attr) bool {
		h.appendAttr(&e.fields, prefix, a)
		return true
	})

	h.logger.log(e, r.Message)
	return nil
}

// WithAttrs returns a new [SlogHandler] with the given attrs preset.
func (h *SlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	fields := slices.Clone(h.attrs)
	prefix := h.groupPrefix()
	for _, a := range attrs {
		h.appendAttr(&fields, prefix, a)
	}

	return &SlogHandler{
		logger: h.logger,
		attrs:  fields,
		groups: h.groups,
		opts:   h.opts,
	}
}

// WithGroup returns a new [SlogHandler] with the given group name appended.
func (h *SlogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return &SlogHandler{
		logger: h.logger,
		attrs:  slices.Clone(h.attrs),
		groups: append(slices.Clone(h.groups), name),
		opts:   h.opts,
	}
}

// groupPrefix returns the dot-joined group prefix, or "" if no groups are set.
func (h *SlogHandler) groupPrefix() string {
	if len(h.groups) == 0 {
		return ""
	}
	return strings.Join(h.groups, ".") + "."
}

// appendAttr converts a slog.Attr and appends the resulting field(s) to dst.
// Empty attrs are dropped per slog convention.
func (h *SlogHandler) appendAttr(dst *[]Field, prefix string, a slog.Attr) {
	a.Value = a.Value.Resolve()

	// Drop empty attrs per slog convention.
	if a.Equal(slog.Attr{}) {
		return
	}

	if a.Value.Kind() == slog.KindGroup {
		groupAttrs := a.Value.Group()
		// Inline group: no key prefix added.
		groupPrefix := prefix
		if a.Key != "" {
			groupPrefix = prefix + a.Key + "."
		}
		for _, ga := range groupAttrs {
			h.appendAttr(dst, groupPrefix, ga)
		}
		return
	}

	key := prefix + a.Key
	*dst = append(*dst, Field{Key: key, Value: slogValueToAny(a.Value)})
}

// slogValueToAny converts a resolved slog.Value to a Go value suitable for clog fields.
func slogValueToAny(v slog.Value) any {
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindInt64:
		return v.Int64()
	case slog.KindUint64:
		return v.Uint64()
	case slog.KindFloat64:
		return v.Float64()
	case slog.KindBool:
		return v.Bool()
	case slog.KindDuration:
		return v.Duration()
	case slog.KindTime:
		return v.Time()
	case slog.KindAny:
		return v.Any()
	case slog.KindGroup:
		// Groups are handled in appendAttr before reaching here.
		return v.Any()
	case slog.KindLogValuer:
		// LogValuer should be resolved before reaching here.
		return v.Resolve().Any()
	default:
		return v.Any()
	}
}

// slogLevelToClog maps a [slog.Level] to a clog [Level].
func slogLevelToClog(l slog.Level) Level {
	switch {
	case l < slog.LevelDebug:
		return TraceLevel
	case l < slog.LevelInfo:
		return DebugLevel
	case l < slog.LevelWarn:
		return InfoLevel
	case l < slog.LevelError:
		return WarnLevel
	case l == slog.LevelError:
		return ErrorLevel
	default:
		return FatalLevel
	}
}
