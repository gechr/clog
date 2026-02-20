package clog

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestColorModeUnmarshalTextCaseInsensitive(t *testing.T) {
	for _, tt := range []struct {
		text string
		want ColorMode
	}{
		{"AUTO", ColorAuto},
		{"Auto", ColorAuto},
		{"auto", ColorAuto},
		{"ALWAYS", ColorAlways},
		{"Always", ColorAlways},
		{"always", ColorAlways},
		{"NEVER", ColorNever},
		{"Never", ColorNever},
		{"never", ColorNever},
	} {
		t.Run(tt.text, func(t *testing.T) {
			var got ColorMode
			require.NoError(t, got.UnmarshalText([]byte(tt.text)))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestColorModeUnmarshalTextInvalid(t *testing.T) {
	var m ColorMode
	err := m.UnmarshalText([]byte("bogus"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown color mode")
}

func TestColorsDisabledAlways(t *testing.T) {
	l := New(NewOutput(io.Discard, ColorAlways))
	assert.False(t, l.colorsDisabled())
}

func TestColorsDisabledNever(t *testing.T) {
	l := New(NewOutput(io.Discard, ColorNever))
	assert.True(t, l.colorsDisabled())
}

func TestColorsDisabledAutoNonTTY(t *testing.T) {
	// io.Discard has no Fd() method, so it's treated as non-TTY -> colors disabled.
	l := New(NewOutput(io.Discard, ColorAuto))
	assert.True(t, l.colorsDisabled())
}

func TestColorsDisabledPackageLevel(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(NewOutput(io.Discard, ColorAlways))
	assert.False(t, ColorsDisabled())

	Default = New(NewOutput(io.Discard, ColorNever))
	assert.True(t, ColorsDisabled())
}
