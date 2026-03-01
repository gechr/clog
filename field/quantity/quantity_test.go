package quantity_test

import (
	"testing"

	"github.com/gechr/clog/field/quantity"
	"github.com/stretchr/testify/assert"
)

func TestSetUnitsIgnoreCase(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := quantity.UnitsIgnoreCase()
			t.Cleanup(func() { quantity.SetUnitsIgnoreCase(original) })

			quantity.SetUnitsIgnoreCase(tt.enabled)
			assert.Equal(t, tt.enabled, quantity.UnitsIgnoreCase())
		})
	}
}

func TestUnitsIgnoreCaseDefaultTrue(t *testing.T) {
	original := quantity.UnitsIgnoreCase()
	t.Cleanup(func() { quantity.SetUnitsIgnoreCase(original) })

	// Restore to known default and verify.
	quantity.SetUnitsIgnoreCase(true)
	assert.True(t, quantity.UnitsIgnoreCase(), "default UnitsIgnoreCase should be true")
}

func TestSetUnitsIgnoreCaseToggle(t *testing.T) {
	original := quantity.UnitsIgnoreCase()
	t.Cleanup(func() { quantity.SetUnitsIgnoreCase(original) })

	quantity.SetUnitsIgnoreCase(false)
	assert.False(t, quantity.UnitsIgnoreCase(), "expected false after SetUnitsIgnoreCase(false)")

	quantity.SetUnitsIgnoreCase(true)
	assert.True(t, quantity.UnitsIgnoreCase(), "expected true after SetUnitsIgnoreCase(true)")
}
