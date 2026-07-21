package clog

import (
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPerFieldSettersAreIsolated(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetPercentPrecision(2)

	f := l.FieldFormats()
	assert.Equal(t, 2, f.PercentPrecision)
	// Everything else stays at the default snapshot.
	assert.Equal(t, time.Second, f.ElapsedMinimum)
	assert.Equal(t, ",", f.NumberGroupSeparator)
	assert.True(t, f.HyperlinkEnabled)
}

func TestSetHyperlinkFileFormatTargetsFileField(t *testing.T) {
	// Regression guard: the file setter must set the file field (and expand
	// the preset), leaving the line field untouched.
	l := NewWriter(io.Discard)
	l.SetHyperlinkFileFormat("vscode")

	f := l.FieldFormats()
	assert.Equal(t, "vscode://file{path}", f.HyperlinkFileFormat)
	assert.Empty(t, f.HyperlinkLineFormat, "line format must be untouched")
}

func TestSetHyperlinkLineFormatExpandsPreset(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetHyperlinkLineFormat("vscode")

	f := l.FieldFormats()
	assert.Equal(t, "vscode://file{path}:{line}", f.HyperlinkLineFormat)
	assert.Empty(t, f.HyperlinkFileFormat, "file format must be untouched")
}

func TestSetTimeScaleClearsFieldOverrides(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetDurationScale(TimeScale{{Round: time.Minute}})
	l.SetElapsedScale(TimeScale{})

	scale := TimeScale{{Round: 100 * time.Millisecond}}
	l.SetTimeScale(scale)

	f := l.FieldFormats()
	assert.Equal(t, scale, f.TimeScale)
	assert.Nil(t, f.DurationScale)
	assert.Nil(t, f.ElapsedScale)
}

func TestSetTimeGradientMaxSetsBoth(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetTimeGradientMax(30 * time.Second)

	f := l.FieldFormats()
	assert.Equal(t, 30*time.Second, f.DurationGradientMax)
	assert.Equal(t, 30*time.Second, f.ElapsedGradientMax)
}
