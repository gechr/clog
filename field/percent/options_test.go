package percent_test

import (
	"testing"

	"github.com/gechr/clog/field/percent"
	"github.com/stretchr/testify/assert"
)

func TestSetFormatFunc(t *testing.T) {
	snap := percent.Save()
	t.Cleanup(func() { percent.Restore(snap) })

	custom := func(_ float64) string { return "custom" }
	percent.SetFormatFunc(custom)
	got := percent.FormatFunc()
	assert.NotNil(t, got, "expected non-nil FormatFunc after setting custom function")
	assert.Equal(t, "custom", got(75.0), "expected custom format function to be used")
}

func TestSetFormatFuncNil(t *testing.T) {
	snap := percent.Save()
	t.Cleanup(func() { percent.Restore(snap) })

	percent.SetFormatFunc(func(_ float64) string { return "x" })
	percent.SetFormatFunc(nil)
	assert.Nil(t, percent.FormatFunc(), "expected nil FormatFunc after setting nil")
}

func TestFormatFuncDefaultNil(t *testing.T) {
	snap := percent.Save()
	t.Cleanup(func() { percent.Restore(snap) })

	percent.SetFormatFunc(nil)
	assert.Nil(t, percent.FormatFunc(), "expected nil FormatFunc when unset")
}

func TestSetReverseGradient(t *testing.T) {
	tests := []struct {
		name string
		rev  bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snap := percent.Save()
			t.Cleanup(func() { percent.Restore(snap) })

			percent.SetReverseGradient(tt.rev)
			assert.Equal(t, tt.rev, percent.ReverseGradient())
		})
	}
}

func TestSetPrecision(t *testing.T) {
	tests := []struct {
		name string
		n    int
	}{
		{"zero", 0},
		{"one", 1},
		{"two", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snap := percent.Save()
			t.Cleanup(func() { percent.Restore(snap) })

			percent.SetPrecision(tt.n)
			assert.Equal(t, tt.n, percent.Precision())
		})
	}
}

func TestWithReverseGradient(t *testing.T) {
	// WithReverseGradient sets Reverse=true on a percent.Percent.
	p := &percent.Percent{Value: 50, Reverse: false}
	opt := percent.WithReverseGradient()
	percent.Apply(p, []percent.Option{opt})
	assert.True(t, p.Reverse, "expected Reverse=true after applying WithReverseGradient")
}

func TestWithReverseGradientNoMutationWithoutApply(t *testing.T) {
	// Calling WithReverseGradient() creates the option but applying zero options
	// leaves Reverse unchanged.
	p := &percent.Percent{Value: 50, Reverse: false}
	percent.Apply(p, []percent.Option{})
	assert.False(t, p.Reverse, "expected Reverse=false when no options applied")
}

func TestApplyMultipleOptions(t *testing.T) {
	snap := percent.Save()
	t.Cleanup(func() { percent.Restore(snap) })

	// Apply multiple options in sequence.
	p := &percent.Percent{Value: 75}
	opt1 := percent.WithReverseGradient()
	percent.Apply(p, []percent.Option{opt1})
	assert.True(t, p.Reverse)
}
