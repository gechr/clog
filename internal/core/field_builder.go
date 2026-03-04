package core

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// Nil is the string representation used for nil values.
const Nil = "<nil>"

// FieldBuilder provides common field-appending methods for fluent builders.
// Embed it and call InitSelf in the constructor to enable method chaining.
type FieldBuilder[T any] struct {
	Fields []Field
	Self   *T
}

// InitSelf sets the self-pointer for method chaining.
func (fb *FieldBuilder[T]) InitSelf(s *T) { fb.Self = s }

// Any adds a field with an arbitrary value.
func (fb *FieldBuilder[T]) Any(key string, val any) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: val})
	return fb.Self
}

// Anys adds a slice of arbitrary values.
func (fb *FieldBuilder[T]) Anys(key string, vals []any) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: vals})
	return fb.Self
}

// Base64 adds a []byte field encoded as a base64 string.
func (fb *FieldBuilder[T]) Base64(key string, val []byte) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: base64.StdEncoding.EncodeToString(val)})
	return fb.Self
}

// Bool adds a bool field.
func (fb *FieldBuilder[T]) Bool(key string, val bool) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: val})
	return fb.Self
}

// Bools adds a bool slice field.
func (fb *FieldBuilder[T]) Bools(key string, vals []bool) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: vals})
	return fb.Self
}

// Bytes adds a []byte field. If val is valid JSON it is stored as [RawJSON]
// with syntax highlighting; otherwise it is stored as a plain string.
func (fb *FieldBuilder[T]) Bytes(key string, val []byte) *T {
	if json.Valid(val) {
		fb.Fields = append(fb.Fields, Field{Key: key, Value: RawJSON(val)})
	} else {
		fb.Fields = append(fb.Fields, Field{Key: key, Value: string(val)})
	}
	return fb.Self
}

// Duration adds a [time.Duration] field.
func (fb *FieldBuilder[T]) Duration(key string, val time.Duration) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: val})
	return fb.Self
}

// Durations adds a [time.Duration] slice field.
func (fb *FieldBuilder[T]) Durations(key string, vals []time.Duration) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: vals})
	return fb.Self
}

// Err adds an error field with key "error". No-op if err is nil.
func (fb *FieldBuilder[T]) Err(err error) *T {
	if err == nil {
		return fb.Self
	}
	fb.Fields = append(fb.Fields, Field{Key: ErrorKey, Value: err})
	return fb.Self
}

// Errs adds an error slice field. Each error is converted to its message
// string; nil errors are rendered as Nil ("<nil>").
func (fb *FieldBuilder[T]) Errs(key string, vals []error) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: ErrSliceToStrings(vals)})
	return fb.Self
}

// Float64 adds a float64 field.
func (fb *FieldBuilder[T]) Float64(key string, val float64) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: val})
	return fb.Self
}

// Floats64 adds a float64 slice field.
func (fb *FieldBuilder[T]) Floats64(key string, vals []float64) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: vals})
	return fb.Self
}

// Hex adds a []byte field encoded as a hex string.
func (fb *FieldBuilder[T]) Hex(key string, val []byte) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: hex.EncodeToString(val)})
	return fb.Self
}

// Int adds an int field.
func (fb *FieldBuilder[T]) Int(key string, val int) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: val})
	return fb.Self
}

// Int64 adds an int64 field.
func (fb *FieldBuilder[T]) Int64(key string, val int64) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: val})
	return fb.Self
}

// Ints adds an int slice field.
func (fb *FieldBuilder[T]) Ints(key string, vals []int) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: vals})
	return fb.Self
}

// Ints64 adds an int64 slice field.
func (fb *FieldBuilder[T]) Ints64(key string, vals []int64) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: vals})
	return fb.Self
}

// JSON marshals val to JSON and adds it as a highlighted field.
// On marshal error the field value is the error string.
func (fb *FieldBuilder[T]) JSON(key string, val any) *T {
	b, err := json.Marshal(val)
	if err != nil {
		fb.Fields = append(fb.Fields, Field{Key: key, Value: err.Error()})
		return fb.Self
	}
	fb.Fields = append(fb.Fields, Field{Key: key, Value: RawJSON(b)})
	return fb.Self
}

// Percent adds a percentage field with gradient color styling.
// The value is stored as-is; use [Event.Percent] for clamped input.
func (fb *FieldBuilder[T]) Percent(key string, val float64, opts ...func(*Percent)) *T {
	p := Percent{Value: val}
	for _, o := range opts {
		o(&p)
	}
	fb.Fields = append(fb.Fields, Field{Key: key, Value: p})
	return fb.Self
}

// Quantities adds a quantity string slice field.
func (fb *FieldBuilder[T]) Quantities(key string, vals []string) *T {
	q := make([]QuantityField, len(vals))
	for i, v := range vals {
		q[i] = QuantityField(v)
	}
	fb.Fields = append(fb.Fields, Field{Key: key, Value: q})
	return fb.Self
}

// Quantity adds a quantity string field where numeric and unit segments are
// styled independently (e.g. "5m", "5.1km", "100MB").
func (fb *FieldBuilder[T]) Quantity(key, val string) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: QuantityField(val)})
	return fb.Self
}

// RawJSON adds a field with pre-serialized JSON bytes, emitted verbatim
// without quoting or escaping.
func (fb *FieldBuilder[T]) RawJSON(key string, val []byte) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: RawJSON(val)})
	return fb.Self
}

// Str adds a string field.
func (fb *FieldBuilder[T]) Str(key, val string) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: val})
	return fb.Self
}

// Stringer adds a field by calling the value's String method. No-op if val is nil.
func (fb *FieldBuilder[T]) Stringer(key string, val fmt.Stringer) *T {
	if IsNilStringer(val) {
		return fb.Self
	}
	fb.Fields = append(fb.Fields, Field{Key: key, Value: val.String()})
	return fb.Self
}

// Stringers adds a field with a slice of [fmt.Stringer] values.
func (fb *FieldBuilder[T]) Stringers(key string, vals []fmt.Stringer) *T {
	strs := make([]string, len(vals))
	for i, v := range vals {
		if IsNilStringer(v) {
			strs[i] = Nil
		} else {
			strs[i] = v.String()
		}
	}
	fb.Fields = append(fb.Fields, Field{Key: key, Value: strs})
	return fb.Self
}

// Strs adds a string slice field.
func (fb *FieldBuilder[T]) Strs(key string, vals []string) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: vals})
	return fb.Self
}

// Time adds a [time.Time] field.
func (fb *FieldBuilder[T]) Time(key string, val time.Time) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: val})
	return fb.Self
}

// Times adds a [time.Time] slice field.
func (fb *FieldBuilder[T]) Times(key string, vals []time.Time) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: vals})
	return fb.Self
}

// Uint adds a uint field.
func (fb *FieldBuilder[T]) Uint(key string, val uint) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: val})
	return fb.Self
}

// Uint64 adds a uint64 field.
func (fb *FieldBuilder[T]) Uint64(key string, val uint64) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: val})
	return fb.Self
}

// Uints adds a uint slice field.
func (fb *FieldBuilder[T]) Uints(key string, vals []uint) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: vals})
	return fb.Self
}

// Uints64 adds a uint64 slice field.
func (fb *FieldBuilder[T]) Uints64(key string, vals []uint64) *T {
	fb.Fields = append(fb.Fields, Field{Key: key, Value: vals})
	return fb.Self
}

// When calls fn with the builder if condition is true.
func (fb *FieldBuilder[T]) When(condition bool, fn func(*T)) *T {
	if condition && fn != nil {
		fn(fb.Self)
	}
	return fb.Self
}

// --- Helper functions ---

// ErrSliceToStrings converts a slice of errors to a slice of strings.
// Nil errors are rendered as Nil ("<nil>").
func ErrSliceToStrings(errs []error) []string {
	strs := make([]string, len(errs))
	for i, e := range errs {
		if e == nil {
			strs[i] = Nil
		} else {
			strs[i] = e.Error()
		}
	}
	return strs
}

// MergeFields merges base fields with overrides, replacing existing keys.
// Keys in overrides replace matching keys in base while preserving order.
func MergeFields(base, overrides []Field) []Field {
	if len(overrides) == 0 {
		return base
	}

	overrideMap := make(map[string]any)
	for _, f := range overrides {
		overrideMap[f.Key] = f.Value
	}

	result := make([]Field, 0, len(base)+len(overrides))
	usedKeys := make(map[string]bool)

	for _, f := range base {
		if val, ok := overrideMap[f.Key]; ok {
			result = append(result, Field{Key: f.Key, Value: val})
			usedKeys[f.Key] = true
		} else {
			result = append(result, f)
		}
	}

	for _, f := range overrides {
		if !usedKeys[f.Key] {
			result = append(result, f)
		}
	}
	return result
}
