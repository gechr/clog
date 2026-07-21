// Command fieldmethods generates the field-appending methods shared by
// clog's *Event (event_fields.go) and core's FieldBuilder[T]
// (internal/core/field_builder_methods.go) from the methods spec table below,
// plus the output-dependent hyperlink and Dict methods shared by *Event and
// *Context (context_fields.go) from the outputMethods table. The method sets
// stay in sync by construction; methods whose semantics genuinely diverge
// between receivers (Duration, Err, Fraction, Percent, When) remain
// hand-written in event.go and field_builder.go.
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"log"
	"os"
	"strings"
)

// method describes one shared field-appending method. Body templates use the
// placeholders $fields (the destination slice), $self (the chain return
// value), $core (the qualifier for core symbols: "core." in clog, "" in core
// itself), and $output (the logger output accessor). Docs use $Recv for
// receiver-qualified cross-references.
type method struct {
	name     string
	params   string
	doc      string // FieldBuilder doc; also the Event doc when eventDoc is empty
	eventDoc string // Event doc override (for docs that reference clog symbols)
	value    string // append-value expression for plain append bodies
	body     string // full body template; overrides value when set
}

var methods = []method{
	{
		name:   "Any",
		params: "key string, val any",
		doc:    `Any adds a field with an arbitrary value.`,
		value:  "val",
	},
	{
		name:   "Anys",
		params: "key string, vals []any",
		doc:    `Anys adds a slice of arbitrary values.`,
		eventDoc: `Anys adds a slice of arbitrary values. Individual elements are
highlighted using reflection to determine their type.`,
		value: "vals",
	},
	{
		name:   "Base64",
		params: "key string, val []byte",
		doc:    `Base64 adds a []byte field encoded as a base64 string.`,
		value:  "base64.StdEncoding.EncodeToString(val)",
	},
	{
		name:   "Bool",
		params: "key string, val bool",
		doc:    `Bool adds a bool field.`,
		value:  "val",
	},
	{
		name:   "Bools",
		params: "key string, vals []bool",
		doc:    `Bools adds a bool slice field.`,
		value:  "vals",
	},
	{
		name:   "Bytes",
		params: "key string, val []byte",
		doc: `Bytes adds a []byte field. If val is valid JSON it is stored as [RawJSON]
with syntax highlighting; otherwise it is stored as a plain string.`,
		body: `if json.Valid(val) {
	$fields = append($fields, Field{Key: key, Value: $coreRawJSON(val)})
} else {
	$fields = append($fields, Field{Key: key, Value: string(val)})
}
return $self`,
	},
	{
		name:   "Durations",
		params: "key string, vals []time.Duration",
		doc:    `Durations adds a [time.Duration] slice field.`,
		value:  "vals",
	},
	{
		name:   "AnErr",
		params: "key string, err error",
		doc:    `AnErr adds an error as a keyed field. No-op if err is nil.`,
		eventDoc: `AnErr adds an error as a keyed field. No-op if err is nil.
Unlike [Event.Err], this does not interact with [Event.Send] or [Event.Msg]
semantics - the error is simply added as a regular field with the given key.`,
		body: `if err == nil {
	return $self
}

$fields = append($fields, Field{Key: key, Value: err})
return $self`,
	},
	{
		name:   "Errs",
		params: "key string, vals []error",
		doc: `Errs adds an error slice field. Each error is converted to its message
string; nil errors are rendered as Nil ("<nil>").`,
		eventDoc: `Errs adds an error slice field. Each error is converted to its message
string; nil errors are rendered as [Nil] ("<nil>").`,
		value: "$coreErrSliceToStrings(vals)",
	},
	{
		name:   "Float32",
		params: "key string, val float32",
		doc:    `Float32 adds a float32 field.`,
		value:  "val",
	},
	{
		name:   "Float64",
		params: "key string, val float64",
		doc:    `Float64 adds a float64 field.`,
		value:  "val",
	},
	{
		name:   "Floats32",
		params: "key string, vals []float32",
		doc:    `Floats32 adds a float32 slice field.`,
		value:  "vals",
	},
	{
		name:   "Floats64",
		params: "key string, vals []float64",
		doc:    `Floats64 adds a float64 slice field.`,
		value:  "vals",
	},
	{
		name:   "Hex",
		params: "key string, val []byte",
		doc:    `Hex adds a []byte field encoded as a hex string.`,
		value:  "hex.EncodeToString(val)",
	},
	{
		name:   "Int",
		params: "key string, val int",
		doc:    `Int adds an int field.`,
		value:  "val",
	},
	{
		name:   "Int8",
		params: "key string, val int8",
		doc:    `Int8 adds an int8 field.`,
		value:  "val",
	},
	{
		name:   "Int16",
		params: "key string, val int16",
		doc:    `Int16 adds an int16 field.`,
		value:  "val",
	},
	{
		name:   "Int32",
		params: "key string, val int32",
		doc:    `Int32 adds an int32 field.`,
		value:  "val",
	},
	{
		name:   "Int64",
		params: "key string, val int64",
		doc:    `Int64 adds an int64 field.`,
		value:  "val",
	},
	{
		name:   "Ints",
		params: "key string, vals []int",
		doc:    `Ints adds an int slice field.`,
		value:  "vals",
	},
	{
		name:   "Ints8",
		params: "key string, vals []int8",
		doc:    `Ints8 adds an int8 slice field.`,
		value:  "vals",
	},
	{
		name:   "Ints16",
		params: "key string, vals []int16",
		doc:    `Ints16 adds an int16 slice field.`,
		value:  "vals",
	},
	{
		name:   "Ints32",
		params: "key string, vals []int32",
		doc:    `Ints32 adds an int32 slice field.`,
		value:  "vals",
	},
	{
		name:   "Ints64",
		params: "key string, vals []int64",
		doc:    `Ints64 adds an int64 slice field.`,
		value:  "vals",
	},
	{
		name:   "JSON",
		params: "key string, val any",
		doc: `JSON marshals val to JSON and adds it as a highlighted field.
On marshal error the field value is the error string.`,
		body: `b, err := json.Marshal(val)
if err != nil {
	$fields = append($fields, Field{Key: key, Value: err.Error()})
	return $self
}

$fields = append($fields, Field{Key: key, Value: $coreRawJSON(b)})
return $self`,
	},
	{
		name:   "Quantities",
		params: "key string, vals []string",
		doc:    `Quantities adds a quantity string slice field.`,
		eventDoc: `Quantities adds a quantity string slice field. Each element is styled
with the [style.Config.FieldQuantity] segment styles.`,
		body: `q := make([]$coreQuantityField, len(vals))
for i, v := range vals {
	q[i] = $coreQuantityField(v)
}
$fields = append($fields, Field{Key: key, Value: q})
return $self`,
	},
	{
		name:   "Quantity",
		params: "key, val string",
		doc: `Quantity adds a quantity string field where numeric and unit segments are
styled independently (e.g. "5m", "5.1km", "100MB").`,
		eventDoc: `Quantity adds a quantity string field where numeric and unit segments are
styled independently (e.g. "5m", "5.1km", "100MB").
The value is styled with the [style.Config.FieldQuantity] segment styles.`,
		value: "$coreQuantityField(val)",
	},
	{
		name:   "RawJSON",
		params: "key string, val []byte",
		doc: `RawJSON adds a field with pre-serialized JSON bytes, emitted verbatim
without quoting or escaping.`,
		eventDoc: `RawJSON adds a field with pre-serialized JSON bytes, emitted verbatim
without quoting or escaping. The bytes must be valid JSON.`,
		value: "$coreRawJSON(val)",
	},
	{
		name:   "Str",
		params: "key, val string",
		doc:    `Str adds a string field.`,
		value:  "val",
	},
	{
		name:   "Stringer",
		params: "key string, val fmt.Stringer",
		doc:    `Stringer adds a field by calling the value's String method. No-op if val is nil.`,
		body: `if $coreIsNilStringer(val) {
	return $self
}

$fields = append($fields, Field{Key: key, Value: val.String()})
return $self`,
	},
	{
		name:   "Stringers",
		params: "key string, vals []fmt.Stringer",
		doc:    `Stringers adds a field with a slice of [fmt.Stringer] values.`,
		body: `strs := make([]string, len(vals))
for i, v := range vals {
	if $coreIsNilStringer(v) {
		strs[i] = Nil
	} else {
		strs[i] = v.String()
	}
}
$fields = append($fields, Field{Key: key, Value: strs})
return $self`,
	},
	{
		name:   "Strs",
		params: "key string, vals []string",
		doc:    `Strs adds a string slice field.`,
		value:  "vals",
	},
	{
		name:   "Time",
		params: "key string, val time.Time",
		doc:    `Time adds a [time.Time] field.`,
		value:  "val",
	},
	{
		name:   "Times",
		params: "key string, vals []time.Time",
		doc:    `Times adds a [time.Time] slice field.`,
		value:  "vals",
	},
	{
		name:   "TimeDiff",
		params: "key string, t, start time.Time",
		doc: `TimeDiff adds the field key with the duration between t and start.
If t is not after start, the duration is zero.`,
		body: `var d time.Duration
if t.After(start) {
	d = t.Sub(start)
}
$fields = append($fields, Field{Key: key, Value: d})
return $self`,
	},
	{
		name:   "Uint",
		params: "key string, val uint",
		doc:    `Uint adds a uint field.`,
		value:  "val",
	},
	{
		name:   "Uint8",
		params: "key string, val uint8",
		doc:    `Uint8 adds a uint8 field.`,
		value:  "val",
	},
	{
		name:   "Uint16",
		params: "key string, val uint16",
		doc:    `Uint16 adds a uint16 field.`,
		value:  "val",
	},
	{
		name:   "Uint32",
		params: "key string, val uint32",
		doc:    `Uint32 adds a uint32 field.`,
		value:  "val",
	},
	{
		name:   "Uint64",
		params: "key string, val uint64",
		doc:    `Uint64 adds a uint64 field.`,
		value:  "val",
	},
	{
		name:   "Uints",
		params: "key string, vals []uint",
		doc:    `Uints adds a uint slice field.`,
		value:  "vals",
	},
	{
		name:   "Uints8",
		params: "key string, vals []uint8",
		doc:    `Uints8 adds a uint8 slice field.`,
		value:  "vals",
	},
	{
		name:   "Uints16",
		params: "key string, vals []uint16",
		doc:    `Uints16 adds a uint16 slice field.`,
		value:  "vals",
	},
	{
		name:   "Uints32",
		params: "key string, vals []uint32",
		doc:    `Uints32 adds a uint32 slice field.`,
		value:  "vals",
	},
	{
		name:   "Uints64",
		params: "key string, vals []uint64",
		doc:    `Uints64 adds a uint64 slice field.`,
		value:  "vals",
	},
}

// outputMethods are the hyperlink and Dict methods shared by *Event
// (event_fields.go) and *Context (context_fields.go). They need the logger's
// output and so cannot live on core's FieldBuilder.
var outputMethods = []method{
	{
		name:   "Column",
		params: "key, path string, line, column int",
		doc: `Column adds a file path field with a line and column number as a clickable terminal hyperlink.
Respects the logger's [ColorMode] setting.`,
		body: `if line < 1 {
	line = 1
}

if column < 1 {
	column = 1
}

$fields = append(
	$fields,
	Field{Key: key, Value: $output.pathLink(path, line, column)},
)
return $self`,
	},
	{
		name:   "Columns",
		params: "key string, items []Column",
		doc: `Columns adds a string slice field where each element is a path:line:column
hyperlink. Respects the logger's [ColorMode] setting.`,
		body: `output := $output
vals := make([]string, len(items))
for i, item := range items {
	line, column := item.Line, item.Column
	if line < 1 {
		line = 1
	}
	if column < 1 {
		column = 1
	}
	vals[i] = output.pathLink(item.Path, line, column)
}
$fields = append($fields, Field{Key: key, Value: vals})
return $self`,
	},
	{
		name:   "Dict",
		params: "key string, dict *Event",
		doc: `Dict adds a group of fields under a key prefix using dot notation.
Build the nested fields using [Dict] to create a field-only Event:

	logger := clog.With().Dict("db", clog.Dict().
	    Str("host", "localhost").
	    Int("port", 5432),
	).Logger()`,
		eventDoc: `Dict adds a group of fields under a key prefix using dot notation.
Build the nested fields using [Dict] to create a field-only Event:

	clog.Info().Dict("request", clog.Dict().
	    Str("method", "GET").
	    Int("status", 200),
	).Msg("handled")
	// Output: INF ℹ️ handled request.method=GET request.status=200`,
		body: `if dict == nil {
	return $self
}

for _, f := range dict.fields {
	$fields = append($fields, Field{Key: key + "." + f.Key, Value: f.Value})
}
return $self`,
	},
	{
		name:   "Line",
		params: "key, path string, line int",
		doc: `Line adds a file path field with a line number as a clickable terminal hyperlink.
Respects the logger's [ColorMode] setting. If line < 1, the line number is
omitted and the field is rendered as a plain path hyperlink (equivalent to
[$Recv.Path]).`,
		body: `if line < 1 {
	return $self.Path(key, path)
}

$fields = append(
	$fields,
	Field{Key: key, Value: $output.pathLink(path, line, 0)},
)
return $self`,
	},
	{
		name:   "Lines",
		params: "key string, items []Line",
		doc: `Lines adds a string slice field where each element is a path:line
hyperlink. If an item's Line < 1, that element is rendered as a plain path
hyperlink (equivalent to [$Recv.Path]). Respects the logger's [ColorMode]
setting.`,
		body: `output := $output
vals := make([]string, len(items))
for i, item := range items {
	vals[i] = output.pathLink(item.Path, item.Line, 0)
}
$fields = append($fields, Field{Key: key, Value: vals})
return $self`,
	},
	{
		name:   "Link",
		params: "key, url, text string",
		doc: `Link adds a field as a clickable terminal hyperlink with custom URL and display text.
Respects the logger's [ColorMode] setting.`,
		body: `$fields = append(
	$fields,
	Field{Key: key, Value: $output.hyperlink(url, text)},
)
return $self`,
	},
	{
		name:   "Links",
		params: "key string, links []Link",
		doc:    `Links adds a string slice field where each element is a hyperlink.`,
		body: `output := $output
vals := make([]string, len(links))
for i, l := range links {
	vals[i] = output.hyperlink(l.URL, l.Text)
}
$fields = append($fields, Field{Key: key, Value: vals})
return $self`,
	},
	{
		name:   "Path",
		params: "key, path string",
		doc: `Path adds a file path field as a clickable terminal hyperlink.
Respects the logger's [ColorMode] setting.`,
		body: `$fields = append(
	$fields,
	Field{Key: key, Value: $output.pathLink(path, 0, 0)},
)
return $self`,
	},
	{
		name:   "Paths",
		params: "key string, paths []string",
		doc: `Paths adds a string slice field where each element is a path hyperlink.
Respects the logger's [ColorMode] setting.`,
		body: `output := $output
vals := make([]string, len(paths))
for i, p := range paths {
	vals[i] = output.pathLink(p, 0, 0)
}
$fields = append($fields, Field{Key: key, Value: vals})
return $self`,
	},
	{
		name:   "PathText",
		params: "key, text, path string",
		doc: `PathText adds a file path field as a clickable terminal hyperlink whose
visible label is text rather than path. The link still targets path, so a
caller can show an abbreviated or home-contracted path (e.g. ~/bin/foo)
while linking to its full location. Respects the logger's [ColorMode]
setting.`,
		body: `$fields = append(
	$fields,
	Field{Key: key, Value: $output.pathLinkText(text, path, 0, 0)},
)
return $self`,
	},
	{
		name:   "URL",
		params: "key, url string",
		doc: `URL adds a field as a clickable terminal hyperlink where the URL is also the display text.
Respects the logger's [ColorMode] setting.`,
		body: `$fields = append(
	$fields,
	Field{Key: key, Value: $output.hyperlink(url, url)},
)
return $self`,
	},
	{
		name:   "URLs",
		params: "key string, urls []string",
		doc: `URLs adds a string slice field where each element is a hyperlink
with the URL as the display text.`,
		body: `output := $output
vals := make([]string, len(urls))
for i, u := range urls {
	vals[i] = output.hyperlink(u, u)
}
$fields = append($fields, Field{Key: key, Value: vals})
return $self`,
	},
}

// target describes one generated file: its package, receiver shape, the
// placeholder substitutions applied to body templates, and the method tables
// rendered into it.
type target struct {
	path     string
	header   string
	recv     string     // method receiver
	ret      string     // method return type
	fields   string     // $fields substitution
	self     string     // $self substitution
	core     string     // $core substitution
	output   string     // $output substitution (logger output accessor)
	recvName string     // $Recv doc substitution
	guarded  bool       // prepend the Event nil-receiver guard
	tables   [][]method // method tables rendered into this file, in order
}

var targets = []target{
	{
		path: "event_fields.go",
		header: `// Code generated by internal/gen/fieldmethods; DO NOT EDIT.

package clog

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gechr/clog/internal/core"
)
`,
		recv:     "e *Event",
		ret:      "*Event",
		fields:   "e.fields",
		self:     "e",
		core:     "core.",
		output:   "e.output()",
		recvName: "Event",
		guarded:  true,
		tables:   [][]method{methods, outputMethods},
	},
	{
		path: "internal/core/field_builder_methods.go",
		header: `// Code generated by internal/gen/fieldmethods; DO NOT EDIT.

package core

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)
`,
		recv:     "fb *FieldBuilder[T]",
		ret:      "*T",
		fields:   "fb.Fields",
		self:     "fb.Self",
		core:     "",
		recvName: "FieldBuilder",
		tables:   [][]method{methods},
	},
	{
		path: "context_fields.go",
		header: `// Code generated by internal/gen/fieldmethods; DO NOT EDIT.

package clog
`,
		recv:     "c *Context",
		ret:      "*Context",
		fields:   "c.Fields",
		self:     "c",
		output:   "c.logger.Output()",
		recvName: "Context",
		tables:   [][]method{outputMethods},
	},
}

func render(t target, m method) string {
	doc := m.doc
	if t.guarded && m.eventDoc != "" {
		doc = m.eventDoc
	}
	doc = strings.ReplaceAll(doc, "$Recv", t.recvName)

	body := m.body
	if body == "" {
		body = fmt.Sprintf(
			"$fields = append($fields, Field{Key: key, Value: %s})\nreturn $self",
			m.value,
		)
	}
	body = strings.NewReplacer(
		"$fields", t.fields,
		"$self", t.self,
		"$core", t.core,
		"$output", t.output,
	).Replace(body)

	var b strings.Builder
	b.WriteString("\n")
	for line := range strings.SplitSeq(doc, "\n") {
		switch {
		case line == "":
			b.WriteString("//\n")
		case strings.HasPrefix(line, "\t"):
			fmt.Fprintf(&b, "//%s\n", line)
		default:
			fmt.Fprintf(&b, "// %s\n", line)
		}
	}
	fmt.Fprintf(&b, "func (%s) %s(%s) %s {\n", t.recv, m.name, m.params, t.ret)
	if t.guarded {
		fmt.Fprintf(&b, "\tif %s == nil {\n\t\treturn %s\n\t}\n\n", t.self, t.self)
	}
	for line := range strings.SplitSeq(body, "\n") {
		if line == "" {
			b.WriteString("\n")
			continue
		}
		fmt.Fprintf(&b, "\t%s\n", line)
	}
	b.WriteString("}\n")
	return b.String()
}

func main() {
	for _, t := range targets {
		var b bytes.Buffer
		b.WriteString(t.header)
		for _, table := range t.tables {
			for _, m := range table {
				b.WriteString(render(t, m))
			}
		}

		formatted, err := format.Source(b.Bytes())
		if err != nil {
			log.Fatalf("format %s: %v", t.path, err)
		}
		if err := os.WriteFile(t.path, formatted, 0o600); err != nil {
			log.Fatalf("write %s: %v", t.path, err)
		}
	}
}
