package clog

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// newTimerTestLogger creates a logger with a fixed elapsed format so tests
// can assert exact output. The elapsed field always renders as "1s".
func newTimerTestLogger(buf *bytes.Buffer) *Logger {
	l := New(TestOutput(buf))
	f := DefaultFieldFormats()
	f.ElapsedMinimum = 0
	f.ElapsedFormat = func(time.Duration) string { return "1s" }
	l.SetFieldFormats(f)
	return l
}

func TestElapsedSend(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Info().Elapsed("elapsed").Send()

	assert.Equal(t, "INF ℹ️ elapsed=1s\n", buf.String())
}

func TestElapsedMsg(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Info().Elapsed("elapsed").Msg("file uploaded")

	assert.Equal(t, "INF ℹ️ file uploaded elapsed=1s\n", buf.String())
}

func TestElapsedMsgf(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Info().Elapsed("elapsed").Msgf("uploaded %d files", 3)

	assert.Equal(t, "INF ℹ️ uploaded 3 files elapsed=1s\n", buf.String())
}

func TestElapsedErrField(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Error().Elapsed("elapsed").Err(errors.New("syntax error")).Msg("compile")

	assert.Equal(t, "ERR ❌ compile elapsed=1s error=\"syntax error\"\n", buf.String())
}

func TestElapsedFields(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Info().
		Str("name", "imports").
		Int("items", 100).
		Elapsed("elapsed").
		Msg("batch")

	assert.Equal(t, "INF ℹ️ batch name=imports items=100 elapsed=1s\n", buf.String())
}

func TestElapsedCustomKey(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Info().Elapsed("duration").Msg("compile")

	assert.Equal(t, "INF ℹ️ compile duration=1s\n", buf.String())
}

func TestElapsedPositionFirst(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Info().Elapsed("elapsed").Str("a", "1").Int("b", 2).Msg("process")

	assert.Equal(t, "INF ℹ️ process elapsed=1s a=1 b=2\n", buf.String())
}

func TestElapsedPositionMiddle(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Info().Str("a", "1").Elapsed("elapsed").Int("b", 2).Msg("process")

	assert.Equal(t, "INF ℹ️ process a=1 elapsed=1s b=2\n", buf.String())
}

func TestElapsedPositionLast(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Info().Str("a", "1").Int("b", 2).Elapsed("elapsed").Msg("process")

	assert.Equal(t, "INF ℹ️ process a=1 b=2 elapsed=1s\n", buf.String())
}

func TestElapsedCalledTwice(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Info().Elapsed("elapsed").Str("a", "1").Elapsed("elapsed").Str("b", "2").Msg("double")

	assert.Equal(t, "INF ℹ️ double elapsed=1s a=1 elapsed=1s b=2\n", buf.String())
}

func TestElapsedLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)
	l.SetLevel(LevelWarn)

	l.Info().Elapsed("elapsed").Msg("ignored")

	assert.Empty(t, buf.String())
}

func TestElapsedEventFormatFunc(t *testing.T) {
	l, buf := newTestLogger()
	f := DefaultFieldFormats()
	f.ElapsedMinimum = 0
	f.ElapsedFormat = func(time.Duration) string { return "custom" }
	l.SetFieldFormats(f)

	l.Info().Elapsed("elapsed").Msg("test")

	assert.Equal(t, "INF ℹ️ test elapsed=custom\n", buf.String())
}

func TestElapsedWithSubLogger(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)
	sub := l.With().Str("component", "db").Logger()

	sub.Info().Elapsed("elapsed").Msg("query")

	assert.Equal(t, "INF ℹ️ query component=db elapsed=1s\n", buf.String())
}

func TestElapsedNoCallNoResolve(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	// No Elapsed call - should not add any elapsed field.
	l.Info().Str("a", "1").Msg("plain")

	assert.Equal(t, "INF ℹ️ plain a=1\n", buf.String())
}
