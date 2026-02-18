package clog

import "time"

// fieldBuilder provides common field-appending methods for fluent builders.
// Embed it and call initSelf in the constructor to enable method chaining.
type fieldBuilder[T any] struct {
	fields []Field
	self   *T
}

// Any adds a field with an arbitrary value.
func (fb *fieldBuilder[T]) Any(key string, val any) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: val})
	return fb.self
}

// Anys adds a slice of arbitrary values. Individual elements are
// highlighted using reflection to determine their type.
func (fb *fieldBuilder[T]) Anys(key string, vals []any) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: vals})
	return fb.self
}

// Bool adds a bool field.
func (fb *fieldBuilder[T]) Bool(key string, val bool) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: val})
	return fb.self
}

// Bools adds a bool slice field.
func (fb *fieldBuilder[T]) Bools(key string, vals []bool) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: vals})
	return fb.self
}

// Duration adds a [time.Duration] field.
func (fb *fieldBuilder[T]) Duration(key string, val time.Duration) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: val})
	return fb.self
}

// Durations adds a [time.Duration] slice field.
func (fb *fieldBuilder[T]) Durations(key string, vals []time.Duration) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: vals})
	return fb.self
}

// Float64 adds a float64 field.
func (fb *fieldBuilder[T]) Float64(key string, val float64) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: val})
	return fb.self
}

// Floats64 adds a float64 slice field.
func (fb *fieldBuilder[T]) Floats64(key string, vals []float64) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: vals})
	return fb.self
}

func (fb *fieldBuilder[T]) initSelf(s *T) { fb.self = s }

// Int adds an int field.
func (fb *fieldBuilder[T]) Int(key string, val int) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: val})
	return fb.self
}

// Ints adds an int slice field.
func (fb *fieldBuilder[T]) Ints(key string, vals []int) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: vals})
	return fb.self
}

// Percent adds a percentage field (0–100) with gradient color styling.
// Values are clamped to the 0–100 range. The color is interpolated from
// the [Styles.PercentGradient] stops (default: red → yellow → green).
func (fb *fieldBuilder[T]) Percent(key string, val float64) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: percent(clampPercent(val))})
	return fb.self
}

// Quantities adds a quantity string slice field. Each element is styled
// with [Styles.FieldQuantityNumber] and [Styles.FieldQuantityUnit].
func (fb *fieldBuilder[T]) Quantities(key string, vals []string) *T {
	q := make([]quantity, len(vals))
	for i, v := range vals {
		q[i] = quantity(v)
	}
	fb.fields = append(fb.fields, Field{Key: key, Value: q})
	return fb.self
}

// Quantity adds a quantity string field where numeric and unit segments are
// styled independently (e.g. "5m", "5.1km", "100MB").
// The value is styled with [Styles.FieldQuantityNumber] and [Styles.FieldQuantityUnit].
func (fb *fieldBuilder[T]) Quantity(key, val string) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: quantity(val)})
	return fb.self
}

// Str adds a string field.
func (fb *fieldBuilder[T]) Str(key, val string) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: val})
	return fb.self
}

// Strs adds a string slice field.
func (fb *fieldBuilder[T]) Strs(key string, vals []string) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: vals})
	return fb.self
}

// Time adds a [time.Time] field.
func (fb *fieldBuilder[T]) Time(key string, val time.Time) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: val})
	return fb.self
}

// Uint64 adds a uint64 field.
func (fb *fieldBuilder[T]) Uint64(key string, val uint64) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: val})
	return fb.self
}

// Uints64 adds a uint64 slice field.
func (fb *fieldBuilder[T]) Uints64(key string, vals []uint64) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: vals})
	return fb.self
}
