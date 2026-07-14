package clog

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestFieldFormatsPerLoggerIsolation is the acceptance test for scoping
// field formats to the Logger: two loggers in one process configured with
// different PercentPrecision must render differently.
func TestFieldFormatsPerLoggerIsolation(t *testing.T) {
	var bufA, bufB bytes.Buffer
	loggerA := New(TestOutput(&bufA))
	loggerB := New(TestOutput(&bufB))

	formatsA := DefaultFieldFormats()
	formatsA.PercentPrecision = 0
	loggerA.SetFieldFormats(formatsA)

	formatsB := DefaultFieldFormats()
	formatsB.PercentPrecision = 2
	loggerB.SetFieldFormats(formatsB)

	loggerA.Info().Percent("p", 0.7512).Msg("done")
	loggerB.Info().Percent("p", 0.7512).Msg("done")

	assert.Equal(t, "INF ℹ️ done p=75%\n", bufA.String())
	assert.Equal(t, "INF ℹ️ done p=75.12%\n", bufB.String())
}

func TestFieldFormatsGetterRoundTrip(t *testing.T) {
	logger := NewWriter(&bytes.Buffer{})

	f := DefaultFieldFormats()
	f.PercentMaximum = 100
	f.ElapsedMinimum = 0
	f.DurationGradientMax = 5 * time.Second
	logger.SetFieldFormats(f)

	got := logger.FieldFormats()
	assert.InDelta(t, 100.0, got.PercentMaximum, 0)
	assert.Equal(t, time.Duration(0), got.ElapsedMinimum)
	assert.Equal(t, 5*time.Second, got.DurationGradientMax)
}

func TestFieldFormatsDefaultWithoutSet(t *testing.T) {
	logger := NewWriter(&bytes.Buffer{})

	got := logger.FieldFormats()
	assert.Equal(t, DefaultFieldFormats(), got)
}

func TestDefaultFieldFormatsTimeScales(t *testing.T) {
	f := DefaultFieldFormats()

	assert.Zero(t, f.DurationMinimum)
	assert.Len(t, f.TimeScale, 3)
	assert.Nil(t, f.DurationScale)
	assert.NotNil(t, f.ElapsedScale)
	assert.Empty(t, f.ElapsedScale)
}

func TestFieldFormatsSubLoggerInheritsParent(t *testing.T) {
	var buf bytes.Buffer
	parent := New(TestOutput(&buf))

	f := DefaultFieldFormats()
	f.PercentPrecision = 1
	parent.SetFieldFormats(f)

	sub := parent.With().Str("component", "db").Logger()
	sub.Info().Percent("p", 0.5).Msg("done")

	assert.Equal(t, "INF ℹ️ done component=db p=50.0%\n", buf.String())
}

func TestFieldFormatsHyperlinkPresetExpansion(t *testing.T) {
	logger := New(NewOutput(&bytes.Buffer{}, ColorAlways))

	f := DefaultFieldFormats()
	f.HyperlinkLineFormat = "vscode"
	logger.SetFieldFormats(f)

	got := logger.FieldFormats()
	assert.Equal(t, "vscode://file{path}:{line}", got.HyperlinkLineFormat)

	link := logger.Output().PathLink("/tmp/main.go", 42, 0)
	assert.Equal(t, "\x1b]8;;vscode://file/tmp/main.go:42\x1b\\/tmp/main.go:42\x1b]8;;\x1b\\", link)
}

func TestFieldFormatsPercentMaximum(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	f := DefaultFieldFormats()
	f.PercentMaximum = 100
	logger.SetFieldFormats(f)

	logger.Info().Percent("p", 75).Msg("done")

	assert.Equal(t, "INF ℹ️ done p=75%\n", buf.String())
}
