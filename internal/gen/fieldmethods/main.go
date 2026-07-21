// Command fieldmethods generates the field-appending methods shared by
// clog's *Event (event_fields.go) and core's FieldBuilder[T]
// (internal/core/field_builder_methods.go) from the single spec table below.
// The two method sets stay in sync by construction; methods whose semantics
// genuinely diverge between the two receivers (Duration, Err, Fraction,
// Percent, When) remain hand-written in event.go and field_builder.go.
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
// value), and $core (the qualifier for core symbols: "core." in clog, ""
// in core itself).
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

// target describes one generated file: its package, receiver shape, and the
// placeholder substitutions applied to body templates.
type target struct {
	path    string
	header  string
	recv    string // method receiver
	ret     string // method return type
	fields  string // $fields substitution
	self    string // $self substitution
	core    string // $core substitution
	guarded bool   // prepend the Event nil-receiver guard
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
		recv:    "e *Event",
		ret:     "*Event",
		fields:  "e.fields",
		self:    "e",
		core:    "core.",
		guarded: true,
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
		recv:   "fb *FieldBuilder[T]",
		ret:    "*T",
		fields: "fb.Fields",
		self:   "fb.Self",
		core:   "",
	},
}

func render(t target, m method) string {
	doc := m.doc
	if t.guarded && m.eventDoc != "" {
		doc = m.eventDoc
	}

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
	).Replace(body)

	var b strings.Builder
	b.WriteString("\n")
	for line := range strings.SplitSeq(doc, "\n") {
		fmt.Fprintf(&b, "// %s\n", line)
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
		for _, m := range methods {
			b.WriteString(render(t, m))
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
