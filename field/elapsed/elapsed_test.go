package elapsed_test

import (
	"testing"
	"time"

	"github.com/gechr/clog/field/elapsed"
	"github.com/stretchr/testify/assert"
)

func TestSetFormatFunc(t *testing.T) {
	snap := elapsed.Save()
	t.Cleanup(func() { elapsed.Restore(snap) })

	custom := func(_ time.Duration) string { return "custom" }
	elapsed.SetFormatFunc(custom)
	got := elapsed.FormatFunc()
	assert.NotNil(t, got, "expected non-nil FormatFunc after setting custom function")
	assert.Equal(t, "custom", got(5*time.Second), "expected custom format function to be used")
}

func TestSetFormatFuncNil(t *testing.T) {
	snap := elapsed.Save()
	t.Cleanup(func() { elapsed.Restore(snap) })

	// Set a non-nil function first, then clear it.
	elapsed.SetFormatFunc(func(_ time.Duration) string { return "x" })
	elapsed.SetFormatFunc(nil)
	assert.Nil(t, elapsed.FormatFunc(), "expected nil FormatFunc after setting nil")
}

func TestFormatFuncDefaultNil(t *testing.T) {
	snap := elapsed.Save()
	t.Cleanup(func() { elapsed.Restore(snap) })

	elapsed.SetFormatFunc(nil)
	assert.Nil(t, elapsed.FormatFunc(), "expected nil FormatFunc when no custom function is set")
}

func TestSetMinimum(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
	}{
		{"zero", 0},
		{"half_second", 500 * time.Millisecond},
		{"two_seconds", 2 * time.Second},
		{"one_minute", time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snap := elapsed.Save()
			t.Cleanup(func() { elapsed.Restore(snap) })

			elapsed.SetMinimum(tt.d)
			assert.Equal(t, tt.d, elapsed.Minimum())
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
		{"three", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snap := elapsed.Save()
			t.Cleanup(func() { elapsed.Restore(snap) })

			elapsed.SetPrecision(tt.n)
			assert.Equal(t, tt.n, elapsed.Precision())
		})
	}
}

func TestSetRound(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
	}{
		{"zero", 0},
		{"hundred_ms", 100 * time.Millisecond},
		{"one_second", time.Second},
		{"one_minute", time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snap := elapsed.Save()
			t.Cleanup(func() { elapsed.Restore(snap) })

			elapsed.SetRound(tt.d)
			assert.Equal(t, tt.d, elapsed.Round())
		})
	}
}

func TestSetGradientMax(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
	}{
		{"zero_disables", 0},
		{"ten_seconds", 10 * time.Second},
		{"thirty_seconds", 30 * time.Second},
		{"one_minute", time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snap := elapsed.Save()
			t.Cleanup(func() { elapsed.Restore(snap) })

			elapsed.SetGradientMax(tt.d)
			assert.Equal(t, tt.d, elapsed.GradientMax())
		})
	}
}

func TestGradientMaxSaveRestore(t *testing.T) {
	snap := elapsed.Save()
	t.Cleanup(func() { elapsed.Restore(snap) })

	elapsed.SetGradientMax(30 * time.Second)
	saved := elapsed.Save()

	// Change after save.
	elapsed.SetGradientMax(time.Minute)
	assert.Equal(t, time.Minute, elapsed.GradientMax())

	// Restore brings back the saved value.
	elapsed.Restore(saved)
	assert.Equal(t, 30*time.Second, elapsed.GradientMax())
}

func TestDefaultValues(t *testing.T) {
	snap := elapsed.Save()
	t.Cleanup(func() { elapsed.Restore(snap) })

	// Restore defaults explicitly.
	elapsed.SetFormatFunc(nil)
	elapsed.SetGradientMax(0)
	elapsed.SetMinimum(time.Second)
	elapsed.SetPrecision(0)
	elapsed.SetRound(time.Second)

	assert.Nil(t, elapsed.FormatFunc(), "default FormatFunc should be nil")
	assert.Equal(t, time.Duration(0), elapsed.GradientMax(), "default GradientMax should be 0")
	assert.Equal(t, time.Second, elapsed.Minimum(), "default Minimum should be 1s")
	assert.Equal(t, 0, elapsed.Precision(), "default Precision should be 0")
	assert.Equal(t, time.Second, elapsed.Round(), "default Round should be 1s")
}
