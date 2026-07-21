package core

import (
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

// Duration adds a [time.Duration] field. opts are applied to a
// [DurationField] wrapping val (see the duration package's Option type).
func (fb *FieldBuilder[T]) Duration(
	key string,
	val time.Duration,
	opts ...func(*DurationField),
) *T {
	f := DurationField{Value: val}
	for _, o := range opts {
		o(&f)
	}
	fb.Fields = append(fb.Fields, Field{Key: key, Value: f})
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

// Fraction adds a current/total field with gradient color styling.
// The color is interpolated based on current/total progress.
func (fb *FieldBuilder[T]) Fraction(key string, current, total int, opts ...func(*Fraction)) *T {
	f := Fraction{Current: current, Total: total}
	for _, o := range opts {
		o(&f)
	}
	fb.Fields = append(fb.Fields, Field{Key: key, Value: f})
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
