// Package sloghandler provides a [slog.Handler] adapter for clog.
package sloghandler

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"slices"

	"github.com/gechr/clog"
)

// Options configures a [Handler].
type Options struct {
	// AddSource adds source file:line information to each log entry.
	AddSource bool
	// Level overrides the minimum log level. If nil, the logger's level is used.
	Level slog.Leveler
}

// Handler implements [slog.Handler] by routing records through a clog [clog.Logger].
type Handler struct {
	logger *clog.Logger

	attrs       []clog.Field // preset fields from WithAttrs (immutable after creation)
	groupPrefix string       // dot-terminated key prefix from WithGroup (immutable after creation)
	opts        Options
}

// New creates a [slog.Handler] that routes [slog.Record] entries
// through the given clog [clog.Logger].
func New(logger *clog.Logger, opts *Options) slog.Handler {
	h := &Handler{logger: logger}
	if opts != nil {
		h.opts = *opts
	}
	return h
}

// Enabled reports whether the handler handles records at the given level.
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	if h.opts.Level != nil {
		return level >= h.opts.Level.Level()
	}
	return h.logger.LevelEnabled(slogLevelToClog(level))
}

// Handle converts a [slog.Record] into a clog event and logs it.
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	level := slogLevelToClog(r.Level)

	// Build the field list.
	var fields []clog.Field
	if len(h.attrs) > 0 {
		fields = slices.Clone(h.attrs)
	}

	// Add source if configured.
	if h.opts.AddSource && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		fields = append(fields, clog.Field{
			Key:   slog.SourceKey,
			Value: fmt.Sprintf("%s:%d", f.File, f.Line),
		})
	}

	// Convert record attrs.
	prefix := h.groupPrefix
	r.Attrs(func(a slog.Attr) bool {
		h.appendAttr(&fields, prefix, a)
		return true
	})

	h.logger.LogFields(level, r.Time, r.Message, fields)
	return nil
}

// WithAttrs returns a new [Handler] with the given attrs preset.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	fields := slices.Clone(h.attrs)
	for _, a := range attrs {
		h.appendAttr(&fields, h.groupPrefix, a)
	}

	return &Handler{
		logger:      h.logger,
		attrs:       fields,
		groupPrefix: h.groupPrefix,
		opts:        h.opts,
	}
}

// WithGroup returns a new [Handler] with the given group name appended.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return &Handler{
		logger:      h.logger,
		attrs:       slices.Clone(h.attrs),
		groupPrefix: h.groupPrefix + name + ".",
		opts:        h.opts,
	}
}

// appendAttr converts a slog.Attr and appends the resulting field(s) to dst.
// Empty attrs are dropped per slog convention.
func (h *Handler) appendAttr(dst *[]clog.Field, prefix string, a slog.Attr) {
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
	*dst = append(*dst, clog.Field{Key: key, Value: slogValueToAny(a.Value)})
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

// slogLevelToClog maps a [slog.Level] to a clog [clog.Level].
func slogLevelToClog(l slog.Level) clog.Level {
	switch {
	case l < slog.LevelDebug:
		return clog.LevelTrace
	case l < slog.LevelInfo:
		return clog.LevelDebug
	case l < slog.LevelWarn:
		return clog.LevelInfo
	case l < slog.LevelError:
		return clog.LevelWarn
	case l == slog.LevelError:
		return clog.LevelError
	default:
		return clog.LevelFatal
	}
}
