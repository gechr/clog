package clog

import (
	stdjson "encoding/json"
	"reflect"

	"github.com/gechr/clog/field/json"
	"github.com/gechr/clog/style"
)

// PrintMode controls how the [Printer] formats its output.
type PrintMode int

const (
	// PrintMultiline pretty-prints output with indentation.
	PrintMultiline PrintMode = iota
	// PrintInline flattens output to a single line, matching the compact
	// format used by inline log fields.
	PrintInline
)

// defaultPrintIndent is the default indentation string for [PrintMultiline].
const defaultPrintIndent = "  "

// Printer outputs styled data directly to the logger's output, without
// requiring a log level. Create one with [Logger.Print] or the package-level
// [Print] function.
//
//	clog.Print().JSON(data)
//	clog.Print().RawJSON([]byte(`{"a":1}`))
type Printer struct {
	logger *Logger
	mode   *PrintMode // nil = use logger default
}

// Print returns a new [Printer] for writing styled output without a log level.
// The printer inherits the logger's print mode (see [Logger.SetPrintMode]).
// Use [Printer.Mode] for per-call overrides.
func (l *Logger) Print() *Printer {
	return &Printer{logger: l}
}

// Mode sets the print mode for this call, overriding the logger default.
func (p *Printer) Mode(mode PrintMode) *Printer {
	p.mode = &mode
	return p
}

// effectiveMode returns the effective print mode, falling back to the
// logger default. Must be called with l.mu held.
func (p *Printer) effectiveMode() PrintMode {
	if p.mode != nil {
		return *p.mode
	}
	return p.logger.printMode
}

// resolveIndent returns the indent string for the current mode.
// Empty string means flatten to a single line. Must be called with l.mu held.
func (p *Printer) resolveIndent() string {
	if p.effectiveMode() != PrintMultiline {
		return ""
	}
	if p.logger.printIndent != "" {
		return p.logger.printIndent
	}
	return defaultPrintIndent
}

// JSON marshals v to JSON and writes syntax-highlighted output.
// If marshalling fails, the error string is written instead.
func (p *Printer) JSON(v any) {
	data, err := stdjson.Marshal(v)
	if err != nil {
		p.write(err.Error())
		return
	}
	p.RawJSON(data)
}

// RawJSON writes pre-serialized JSON bytes with syntax highlighting.
// Only token colors are inherited from the logger's FieldJSON configuration;
// field-specific settings (Mode, Spacing, OmitCommas) are not applied.
func (p *Printer) RawJSON(data []byte) {
	l := p.logger
	l.mu.Lock()
	defer l.mu.Unlock()

	indent := p.resolveIndent()

	styles := &style.JSON{Indent: indent}
	if !l.colorsDisabled() && l.styles.FieldJSON != nil {
		copyPointerFields(styles, l.styles.FieldJSON)
	}

	highlighted := json.Highlight(string(data), styles)
	l.runHooks(HookBeforeWrite)
	writeString(l.output.Writer(), highlighted+"\n")
	l.runHooks(HookAfterWrite)
}

// copyPointerFields copies all pointer fields from src to dst, leaving
// value-type fields in dst unchanged. Both must be pointers to the same
// struct type.
func copyPointerFields[T any](dst, src *T) {
	dv := reflect.ValueOf(dst).Elem()
	sv := reflect.ValueOf(src).Elem()
	for sf, fv := range sv.Fields() {
		if sf.Type.Kind() == reflect.Pointer {
			dv.FieldByIndex(sf.Index).Set(fv)
		}
	}
}

// write outputs a plain string. Must NOT be called with l.mu held.
func (p *Printer) write(s string) {
	l := p.logger
	l.mu.Lock()
	defer l.mu.Unlock()
	l.runHooks(HookBeforeWrite)
	writeString(l.output.Writer(), s+"\n")
	l.runHooks(HookAfterWrite)
}
