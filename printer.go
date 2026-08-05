package clog

import (
	"bytes"
	stdjson "encoding/json"
	"reflect"

	"github.com/gechr/clog/printer/hcl"
	"github.com/gechr/clog/printer/json"
	"github.com/gechr/clog/printer/toml"
	"github.com/gechr/clog/printer/yaml"
	"github.com/gechr/clog/style"
	goyaml "github.com/goccy/go-yaml"
	gotoml "github.com/pelletier/go-toml/v2"
)

// JSONPrintMode controls how the [Printer] formats its output.
type JSONPrintMode int

const (
	// JSONPretty pretty-prints output with normalized indentation.
	JSONPretty JSONPrintMode = iota
	// JSONFlat flattens output to a single line, matching the compact
	// format used by inline log fields.
	JSONFlat
	// JSONPreserve keeps original whitespace intact, only adding syntax highlighting.
	JSONPreserve
)

// defaultPrintIndent is the default indentation string for [JSONPretty].
const defaultPrintIndent = "  "

// Printer outputs styled data directly to the logger's output, without
// requiring a log level. Create one with [Logger.Print] or the package-level
// [Print] function.
//
//	clog.Print().JSON(data)
//	clog.Print().RawJSON([]byte(`{"a":1}`))
//
// A nil *Printer is not inert; [Logger.Print] never returns one, and on a nil
// [Logger] it prints to [io.Discard] instead.
type Printer struct {
	logger   *Logger
	modeJSON *JSONPrintMode // nil = use logger default
}

// Print returns a new [Printer] for writing styled output without a log level.
// The printer inherits the logger's print mode (see [Logger.SetJSONPrintMode]).
// Use [Printer.Mode] for per-call overrides.
func (l *Logger) Print() *Printer {
	return &Printer{logger: l.orDiscard()}
}

// Mode sets the print mode for this call, overriding the logger default.
func (p *Printer) Mode(mode JSONPrintMode) *Printer {
	p.modeJSON = &mode
	return p
}

// effectiveMode returns the effective print mode, falling back to the
// logger default. Must be called with l.mu held.
func (p *Printer) effectiveMode() JSONPrintMode {
	if p.modeJSON != nil {
		return *p.modeJSON
	}
	return p.logger.jsonPrintMode
}

// resolveIndent returns specific when set, falling back to the logger's print
// indent and then the default. Must be called with l.mu held.
func (p *Printer) resolveIndent(specific string) string {
	if specific != "" {
		return specific
	}
	if p.logger.printIndent != "" {
		return p.logger.printIndent
	}
	return defaultPrintIndent
}

// resolveJSONIndent returns the JSON indent string for the current mode.
// Empty string means flatten to a single line. Must be called with l.mu held.
func (p *Printer) resolveJSONIndent() string {
	if p.effectiveMode() != JSONPretty {
		return ""
	}
	return p.resolveIndent(p.logger.jsonIndent)
}

// resolveYAMLIndent returns the YAML indent string. Must be called with l.mu held.
func (p *Printer) resolveYAMLIndent() string {
	return p.resolveIndent(p.logger.yamlIndent)
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
// Only token colors are inherited from the logger's JSON configuration;
// field-specific settings (Mode, Spacing, OmitCommas) are not applied.
func (p *Printer) RawJSON(data []byte) {
	l := p.logger
	l.mu.Lock()
	defer l.mu.Unlock()

	indent := p.resolveJSONIndent()
	preserve := p.effectiveMode() == JSONPreserve

	styles := &style.JSON{Indent: indent, PreserveFormat: preserve}
	if !l.colorsDisabled() && l.styles.JSON != nil {
		copyPointerFields(styles, l.styles.JSON)
	}

	highlighted := json.Highlight(string(data), styles)
	l.runHooks(HookBeforeWrite)
	l.output.WriteLine(highlighted + nl)
	l.runHooks(HookAfterWrite)
}

// YAML marshals v to YAML and writes syntax-highlighted output.
// If marshalling fails, the error string is written instead.
func (p *Printer) YAML(v any) {
	l := p.logger
	l.mu.Lock()
	indent := p.resolveYAMLIndent()
	indentSeq := l.yamlIndentSequence == nil || *l.yamlIndentSequence
	l.mu.Unlock()

	var buf bytes.Buffer
	enc := goyaml.NewEncoder(&buf,
		goyaml.Indent(len(indent)),
		goyaml.IndentSequence(indentSeq),
	)
	if err := enc.Encode(v); err != nil {
		p.write(err.Error())
		return
	}
	p.RawYAML(buf.Bytes())
}

// rawHighlight locks the logger, highlights data with the logger's styles of
// type T (nil when colors are disabled), ensures a trailing newline, and
// writes the result between the write hooks. It is the shared body of
// [Printer.RawYAML], [Printer.RawTOML], and [Printer.RawHCL].
func rawHighlight[T any](
	p *Printer,
	data []byte,
	styleFor func(*style.Config) *T,
	highlight func(string, *T) string,
) {
	l := p.logger
	l.mu.Lock()
	defer l.mu.Unlock()

	var styles *T
	if !l.colorsDisabled() {
		styles = styleFor(l.styles)
	}

	highlighted := highlight(string(data), styles)
	if len(highlighted) == 0 || highlighted[len(highlighted)-1] != '\n' {
		highlighted += nl
	}
	l.runHooks(HookBeforeWrite)
	l.output.WriteLine(highlighted)
	l.runHooks(HookAfterWrite)
}

// RawYAML writes pre-serialized YAML bytes with syntax highlighting.
// Token colors are inherited from the logger's YAML configuration.
func (p *Printer) RawYAML(data []byte) {
	rawHighlight(p, data, func(c *style.Config) *style.YAML { return c.YAML }, yaml.Highlight)
}

// TOML marshals v to TOML and writes syntax-highlighted output.
// If marshalling fails, the error string is written instead.
func (p *Printer) TOML(v any) {
	data, err := gotoml.Marshal(v)
	if err != nil {
		p.write(err.Error())
		return
	}
	p.RawTOML(data)
}

// RawTOML writes pre-serialized TOML bytes with syntax highlighting.
// Token colors are inherited from the logger's TOML configuration.
func (p *Printer) RawTOML(data []byte) {
	rawHighlight(p, data, func(c *style.Config) *style.TOML { return c.TOML }, toml.Highlight)
}

// RawHCL writes pre-serialized HCL bytes with syntax highlighting.
// Token colors are inherited from the logger's HCL configuration.
func (p *Printer) RawHCL(data []byte) {
	rawHighlight(p, data, func(c *style.Config) *style.HCL { return c.HCL }, hcl.Highlight)
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
	l.output.WriteLine(s + nl)
	l.runHooks(HookAfterWrite)
}
