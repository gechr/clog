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
	l.SetElapsedMinimum(0)
	l.SetElapsedFormatFunc(func(time.Duration) string { return "1s" })
	return l
}

func TestTimedSend(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Timed("database migration").Send()

	assert.Equal(t, "INF ℹ️ database migration elapsed=1s\n", buf.String())
}

func TestTimedMsg(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Timed("upload").Msg("file uploaded")

	assert.Equal(t, "INF ℹ️ file uploaded elapsed=1s\n", buf.String())
}

func TestTimedErrNil(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Timed("compile").Err(nil)

	assert.Equal(t, "INF ℹ️ compile elapsed=1s\n", buf.String())
}

func TestTimedErrNonNil(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Timed("compile").Err(errors.New("syntax error"))

	assert.Equal(t, "ERR ❌ compile error=\"syntax error\" elapsed=1s\n", buf.String())
}

func TestTimedFields(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Timed("batch").
		Str("name", "imports").
		Int("items", 100).
		Send()

	assert.Equal(t, "INF ℹ️ batch name=imports items=100 elapsed=1s\n", buf.String())
}

func TestTimedElapsedKey(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Timed("compile").ElapsedKey("duration").Send()

	assert.Equal(t, "INF ℹ️ compile duration=1s\n", buf.String())
}

func TestTimedElapsedAppearsAfterFields(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Timed("process").Str("a", "1").Int("b", 2).Send()

	assert.Equal(t, "INF ℹ️ process a=1 b=2 elapsed=1s\n", buf.String())
}

func TestTimedErrFieldOrdering(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Timed("upload").Str("file", "a.txt").Err(errors.New("timeout"))

	assert.Equal(t, "ERR ❌ upload file=a.txt error=timeout elapsed=1s\n", buf.String())
}

func TestTimedLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)
	l.SetLevel(WarnLevel)

	l.Timed("ignored").Send()

	assert.Empty(t, buf.String())
}

func TestTimedErrLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)
	l.SetLevel(FatalLevel)

	l.Timed("ignored").Err(errors.New("boom"))

	assert.Empty(t, buf.String())
}

func TestTimedPackageLevel(t *testing.T) {
	var buf bytes.Buffer
	orig := Default
	defer func() { Default = orig }()
	Default = newTimerTestLogger(&buf)

	Timed("package level").Send()

	assert.Equal(t, "INF ℹ️ package level elapsed=1s\n", buf.String())
}

func TestTimedElapsedFormatFunc(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))
	l.SetElapsedMinimum(0)
	l.SetElapsedFormatFunc(func(time.Duration) string { return "custom" })

	l.Timed("test").Send()

	assert.Equal(t, "INF ℹ️ test elapsed=custom\n", buf.String())
}

func TestTimedMsgWithFields(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)

	l.Timed("original").
		Str("key", "val").
		Msg("replacement")

	assert.Equal(t, "INF ℹ️ replacement key=val elapsed=1s\n", buf.String())
}

func TestTimedSubLogger(t *testing.T) {
	var buf bytes.Buffer
	l := newTimerTestLogger(&buf)
	sub := l.With().Str("component", "db").Logger()

	sub.Timed("query").Send()

	assert.Equal(t, "INF ℹ️ query component=db elapsed=1s\n", buf.String())
}
