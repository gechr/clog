package clog

import (
	"bytes"
	"testing"
	"time"

	"github.com/gechr/clog/internal/core"
	"github.com/stretchr/testify/assert"
)

// newDeadlineTestLogger creates a logger with a fixed elapsed format so tests
// can assert exact output. The deadline field always renders as "15s".
func newDeadlineTestLogger(buf *bytes.Buffer) *Logger {
	l := New(TestOutput(buf))
	f := DefaultFieldFormats()
	f.ElapsedFormat = func(time.Duration) string { return "15s" }
	l.SetFieldFormats(f)
	return l
}

func TestDeadlineMsg(t *testing.T) {
	var buf bytes.Buffer
	l := newDeadlineTestLogger(&buf)

	l.Info().Deadline("timeout", 15*time.Second).Msg("done")

	assert.Equal(t, "INF ℹ️ done timeout=15s\n", buf.String())
}

func TestDeadlineFields(t *testing.T) {
	var buf bytes.Buffer
	l := newDeadlineTestLogger(&buf)

	l.Info().
		Str("job", "upload").
		Deadline("timeout", 15*time.Second).
		Msg("waiting")

	assert.Equal(t, "INF ℹ️ waiting job=upload timeout=15s\n", buf.String())
}

func TestDeadlineResolvesRemaining(t *testing.T) {
	l, buf := newTestLogger()

	// The event is finalised immediately, so with a one-hour deadline the
	// remaining time rounds back up to the full hour.
	l.Info().Deadline("timeout", time.Hour).Msg("test")

	assert.Equal(t, "INF ℹ️ test timeout=1h\n", buf.String())
}

func TestDeadlineExpiredShowsZero(t *testing.T) {
	l, buf := newTestLogger()

	// A zero-duration deadline is already expired; unlike elapsed fields it
	// is not hidden by ElapsedMinimum - the countdown shows "0s".
	l.Info().Deadline("timeout", 0).Msg("test")

	assert.Equal(t, "INF ℹ️ test timeout=0s\n", buf.String())
}

func TestDeadlineNoCallNoResolve(t *testing.T) {
	var buf bytes.Buffer
	l := newDeadlineTestLogger(&buf)

	// No Deadline call - should not add any deadline field.
	l.Info().Str("a", "1").Msg("plain")

	assert.Equal(t, "INF ℹ️ plain a=1\n", buf.String())
}

func TestFieldFormatsDeadlineRoundCeils(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	f := DefaultFieldFormats()
	f.ElapsedScale = TimeScale{{Round: time.Second}}
	l.SetFieldFormats(f)

	// 14.2s remaining rounds UP to 15s - a running countdown never shows a
	// step early (or "0s" while time remains).
	l.mu.Lock()
	l.fields = []Field{{
		Key:   "timeout",
		Value: core.DeadlineField{Remaining: 14200 * time.Millisecond, From: 15 * time.Second},
	}}
	l.mu.Unlock()

	l.Info().Msg("test")

	assert.Equal(t, "INF ℹ️ test timeout=15s\n", buf.String())
}

func TestFieldFormatsDeadlineIgnoresMinimum(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	f := DefaultFieldFormats()
	f.ElapsedMinimum = 2 * time.Second
	f.ElapsedScale = TimeScale{}
	l.SetFieldFormats(f)

	// A deadline below the elapsed minimum stays visible - hiding a countdown
	// as it nears expiry would defeat its purpose.
	l.mu.Lock()
	l.fields = []Field{{
		Key:   "timeout",
		Value: core.DeadlineField{Remaining: time.Second, From: 15 * time.Second},
	}}
	l.mu.Unlock()

	l.Info().Msg("test")

	assert.Equal(t, "INF ℹ️ test timeout=1s\n", buf.String())
}

func TestDeadlineEventFormatFunc(t *testing.T) {
	l, buf := newTestLogger()
	f := DefaultFieldFormats()
	f.ElapsedFormat = func(time.Duration) string { return "custom" }
	l.SetFieldFormats(f)

	l.Info().Deadline("timeout", 15*time.Second).Msg("test")

	assert.Equal(t, "INF ℹ️ test timeout=custom\n", buf.String())
}

func TestDeadlineNilEvent(t *testing.T) {
	var e *Event

	assert.Nil(t, e.Deadline("timeout", 15*time.Second))
}
