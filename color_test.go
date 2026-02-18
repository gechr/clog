package clog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetGlobalColorModeNever(t *testing.T) {
	origForced := colorsForced.Load()
	origDisabled := colorsDisabledFlag.Load()
	origEnabled := hyperlinksEnabled.Load()

	defer func() {
		colorsForced.Store(origForced)
		colorsDisabledFlag.Store(origDisabled)
		hyperlinksEnabled.Store(origEnabled)
	}()

	SetGlobalColorMode(ColorNever)

	assert.False(t, colorsForced.Load(), "expected colorsForced false")
	assert.True(t, colorsDisabledFlag.Load(), "expected colorsDisabledFlag true")
	assert.False(t, hyperlinksEnabled.Load(), "expected hyperlinks disabled")
}

func TestSetGlobalColorModeAlways(t *testing.T) {
	origForced := colorsForced.Load()
	origDisabled := colorsDisabledFlag.Load()

	defer func() {
		colorsForced.Store(origForced)
		colorsDisabledFlag.Store(origDisabled)
	}()

	SetGlobalColorMode(ColorAlways)

	assert.True(t, colorsForced.Load(), "expected colorsForced true")
	assert.False(t, colorsDisabledFlag.Load(), "expected colorsDisabledFlag false")
}

func TestSetGlobalColorModeAuto(t *testing.T) {
	origForced := colorsForced.Load()
	origDisabled := colorsDisabledFlag.Load()

	defer func() {
		colorsForced.Store(origForced)
		colorsDisabledFlag.Store(origDisabled)
	}()

	// First set to non-default state.
	colorsForced.Store(true)
	colorsDisabledFlag.Store(true)

	SetGlobalColorMode(ColorAuto)

	assert.False(t, colorsForced.Load(), "expected colorsForced false")
	assert.False(t, colorsDisabledFlag.Load(), "expected colorsDisabledFlag false")
}

func TestColorModeMarshalText(t *testing.T) {
	for _, tt := range []struct {
		mode ColorMode
		want string
	}{
		{ColorAuto, "auto"},
		{ColorAlways, "always"},
		{ColorNever, "never"},
	} {
		got, err := tt.mode.MarshalText()
		require.NoError(t, err)
		assert.Equal(t, tt.want, string(got))
	}
}

func TestColorModeUnmarshalText(t *testing.T) {
	for _, tt := range []struct {
		text string
		want ColorMode
	}{
		{"auto", ColorAuto},
		{"always", ColorAlways},
		{"never", ColorNever},
	} {
		var got ColorMode
		require.NoError(t, got.UnmarshalText([]byte(tt.text)))
		assert.Equal(t, tt.want, got)
	}
}

func TestColorModeUnmarshalTextInvalid(t *testing.T) {
	var m ColorMode
	err := m.UnmarshalText([]byte("bogus"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown color mode")
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
