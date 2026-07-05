package core

import (
	"fmt"
	"reflect"

	xmath "github.com/gechr/x/math"
)

// ClampPercent restricts val to the [0, r] range.
// NaN and negative infinity clamp to 0; positive infinity clamps to r.
func ClampPercent(val, r float64) float64 {
	return xmath.Clamp(val, 0, r)
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
