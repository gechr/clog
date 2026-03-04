package core

import (
	"fmt"
	"math"
	"reflect"
)

// ClampPercent restricts val to the [0, scale] range.
// NaN and negative infinity clamp to 0; positive infinity clamps to scale.
func ClampPercent(val, scale float64) float64 {
	if math.IsNaN(val) || math.IsInf(val, -1) {
		return 0
	}
	if math.IsInf(val, 1) {
		return scale
	}
	return max(0, min(scale, val))
}

// Clamp01 restricts val to the 0–1 range.
func Clamp01(val float64) float64 {
	return max(0, min(1, val))
}

// IsNilStringer reports whether val is nil, either as an untyped nil interface
// or as a typed nil whose underlying kind supports IsNil.
func IsNilStringer(val fmt.Stringer) bool {
	if val == nil {
		return true
	}

	rv := reflect.ValueOf(val)
	//nolint:exhaustive // only nilable kinds need checking
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return rv.IsNil()
	default:
		return false
	}
}
