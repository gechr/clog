package clog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigureColorOutputNever(t *testing.T) {
	origForced := colorsForced.Load()
	origDisabled := colorsDisabledFlag.Load()
	origEnabled := hyperlinksEnabled.Load()

	defer func() {
		colorsForced.Store(origForced)
		colorsDisabledFlag.Store(origDisabled)
		hyperlinksEnabled.Store(origEnabled)
	}()

	ConfigureColorOutput("never")

	assert.False(t, colorsForced.Load(), "expected colorsForced false")
	assert.True(t, colorsDisabledFlag.Load(), "expected colorsDisabledFlag true")
	assert.False(t, hyperlinksEnabled.Load(), "expected hyperlinks disabled")
}

func TestConfigureColorOutputAlways(t *testing.T) {
	origForced := colorsForced.Load()
	origDisabled := colorsDisabledFlag.Load()

	defer func() {
		colorsForced.Store(origForced)
		colorsDisabledFlag.Store(origDisabled)
	}()

	ConfigureColorOutput("always")

	assert.True(t, colorsForced.Load(), "expected colorsForced true")
	assert.False(t, colorsDisabledFlag.Load(), "expected colorsDisabledFlag false")
}

func TestConfigureColorOutputAuto(t *testing.T) {
	origForced := colorsForced.Load()
	origDisabled := colorsDisabledFlag.Load()

	defer func() {
		colorsForced.Store(origForced)
		colorsDisabledFlag.Store(origDisabled)
	}()

	// First set to non-default state.
	colorsForced.Store(true)
	colorsDisabledFlag.Store(true)

	ConfigureColorOutput("auto")

	assert.False(t, colorsForced.Load(), "expected colorsForced false")
	assert.False(t, colorsDisabledFlag.Load(), "expected colorsDisabledFlag false")
}

func TestColorsDisabledForced(t *testing.T) {
	origForced := colorsForced.Load()
	defer colorsForced.Store(origForced)

	colorsForced.Store(true)

	assert.False(t, ColorsDisabled(), "expected ColorsDisabled() false when forced")
}

func TestColorsDisabledFlag(t *testing.T) {
	origForced := colorsForced.Load()
	origDisabled := colorsDisabledFlag.Load()

	defer func() {
		colorsForced.Store(origForced)
		colorsDisabledFlag.Store(origDisabled)
	}()

	colorsForced.Store(false)
	colorsDisabledFlag.Store(true)

	assert.True(t, ColorsDisabled(), "expected ColorsDisabled() true when flag set")
}
