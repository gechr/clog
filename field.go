package clog

import "time"

// fieldBuilder provides common field-appending methods for fluent builders.
// Embed it and call initSelf in the constructor to enable method chaining.
type fieldBuilder[T any] struct {
	fields []Field
	self   *T
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

// Dur adds a [time.Duration] field.
func (fb *fieldBuilder[T]) Dur(key string, val time.Duration) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: val})
	return fb.self
}

// Time adds a [time.Time] field.
func (fb *fieldBuilder[T]) Time(key string, val time.Time) *T {
	fb.fields = append(fb.fields, Field{Key: key, Value: val})
	return fb.self
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

func (fb *fieldBuilder[T]) initSelf(s *T) { fb.self = s }
