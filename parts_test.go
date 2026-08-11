package clog

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gechr/clog/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPart is a custom part value, outside the built-in range.
const testPart Part = 100

// registerTestPart registers a renderer for [testPart] and removes it when the
// test ends.
func registerTestPart(t *testing.T, render PartRenderer) {
	t.Helper()

	RegisterPart(testPart, render)
	t.Cleanup(func() { UnregisterPart(testPart) })
}

func TestRegisterPart(t *testing.T) {
	registerTestPart(t, func(Entry, *style.Config, bool) string {
		return "worker-3"
	})

	l, buf := newTestLogger()
	l.SetParts(PartLevel, testPart, PartMessage)

	l.Info().Msg("draining")

	assert.Equal(t, "INF worker-3 draining\n", buf.String())
}

func TestRegisterPartReceivesEntry(t *testing.T) {
	var got Entry
	var gotNoColor bool
	var gotStyles *style.Config

	registerTestPart(t, func(e Entry, styles *style.Config, noColor bool) string {
		got, gotStyles, gotNoColor = e, styles, noColor
		return ""
	})

	l, _ := newTestLogger()
	l.SetParts(PartLevel, testPart, PartMessage)

	l.Warn().Str("region", "emea").Msg("careful")

	assert.Equal(t, LevelWarn, got.Level)
	assert.Equal(t, "careful", got.Message)
	assert.Equal(t, []Field{{Key: "region", Value: "emea"}}, got.Fields)
	assert.True(t, got.Time.IsZero(), "timestamps are off by default")
	assert.True(t, gotNoColor, "TestOutput disables color")
	assert.NotNil(t, gotStyles)
}

func TestRegisterPartReceivesTimestamp(t *testing.T) {
	var got Entry

	registerTestPart(t, func(e Entry, _ *style.Config, _ bool) string {
		got = e
		return ""
	})

	stamp := time.Date(2026, time.August, 11, 9, 30, 0, 0, time.UTC)
	l, _ := newTestLogger()
	l.SetReportTimestamp(true)
	l.SetParts(PartLevel, testPart)

	l.LogFields(LevelInfo, stamp, "hello", nil)

	assert.Equal(t, stamp, got.Time.UTC())
}

func TestRegisterPartSharesLineTimestamp(t *testing.T) {
	// A renderer reading Entry.Time must agree with the rendered timestamp
	// part: both report one instant, not two calls to time.Now.
	registerTestPart(t, func(e Entry, _ *style.Config, _ bool) string {
		return e.Time.Format(time.RFC3339Nano)
	})

	l, buf := newTestLogger()
	l.SetReportTimestamp(true)
	l.SetTimeFormat(time.RFC3339Nano)
	l.SetParts(PartTimestamp, testPart)

	l.Info().Msg("")

	got := strings.Fields(strings.TrimSpace(buf.String()))
	require.Len(t, got, 2)
	assert.Equal(t, got[0], got[1], "built-in and custom parts must report the same instant")
}

func TestRegisterPartNoTimestampWhenUnreported(t *testing.T) {
	registerTestPart(t, func(e Entry, _ *style.Config, _ bool) string {
		if e.Time.IsZero() {
			return "no-time"
		}
		return "has-time"
	})

	l, buf := newTestLogger()
	l.SetParts(testPart, PartMessage)

	l.Info().Msg("hello")

	assert.Equal(t, "no-time hello\n", buf.String())
}

func TestRegisterPartAnimationRowVersusCompletion(t *testing.T) {
	// Two different renderers paint these lines, and only one of them can see
	// a custom part. The task row comes from fx.buildLine, which is handed
	// pre-rendered strings and drops parts it does not know; the completion
	// line from Msg goes through Logger.log and carries them. This asymmetry
	// is deliberate - pin it so a later fx change has to be a choice.
	registerTestPart(t, func(Entry, *style.Config, bool) string { return "worker-3" })

	var buf bytes.Buffer
	l := NewWriter(&buf) // &bytes.Buffer has no Fd(), so isTTY = false
	l.SetParts(testPart, PartMessage)

	result := l.Spinner("loading").
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(20 * time.Millisecond) // pass the delay gate
			return nil
		})
	require.NoError(t, result.Msg("done"))

	assert.Equal(t, "loading\nworker-3 done\n", buf.String())
}

func TestRegisterPartEmptyOutputIsOmitted(t *testing.T) {
	registerTestPart(t, func(Entry, *style.Config, bool) string { return "" })

	l, buf := newTestLogger()
	l.SetParts(PartLevel, testPart, PartMessage)

	l.Info().Msg("hello")

	assert.Equal(t, "INF hello\n", buf.String())
}

func TestRegisterPartUnregisteredIsSkipped(t *testing.T) {
	l, buf := newTestLogger()
	l.SetParts(PartLevel, testPart, PartMessage)

	l.Info().Msg("hello")

	assert.Equal(t, "INF hello\n", buf.String())
}

func TestRegisterPartPerEventOverride(t *testing.T) {
	registerTestPart(t, func(e Entry, _ *style.Config, _ bool) string {
		return "[" + e.Level.String() + "]"
	})

	l, buf := newTestLogger()

	l.Info().Parts(testPart, PartMessage).Msg("tagged")
	assert.Equal(t, "[INF] tagged\n", buf.String())

	// The logger's own order is unchanged.
	buf.Reset()
	l.Info().Msg("plain")
	assert.Equal(t, "INF ℹ️ plain\n", buf.String())
}

func TestRegisterPartOrdering(t *testing.T) {
	registerTestPart(t, func(Entry, *style.Config, bool) string { return "<>" })

	l, buf := newTestLogger()
	l.SetParts(PartMessage, PartFields, testPart)

	l.Info().Str("k", "v").Msg("hello")

	assert.Equal(t, "hello k=v <>\n", buf.String())
}

func TestRegisterPartStyles(t *testing.T) {
	registerTestPart(t, func(_ Entry, styles *style.Config, noColor bool) string {
		if noColor || styles.Timestamp == nil {
			return "plain"
		}
		return styles.Timestamp.Render("styled")
	})

	l, out := newTestLogger()
	l.SetParts(testPart, PartMessage)

	l.Info().Msg("hello")

	assert.Equal(t, "plain hello\n", out.String())
}

func TestRegisterPartReplacesRenderer(t *testing.T) {
	registerTestPart(t, func(Entry, *style.Config, bool) string { return "first" })
	RegisterPart(testPart, func(Entry, *style.Config, bool) string { return "second" })

	l, buf := newTestLogger()
	l.SetParts(testPart, PartMessage)

	l.Info().Msg("hello")

	assert.Equal(t, "second hello\n", buf.String())
}

func TestRegisterPartNilRendererUnregisters(t *testing.T) {
	registerTestPart(t, func(Entry, *style.Config, bool) string { return "shown" })
	RegisterPart(testPart, nil)

	require.Nil(t, lookupPartRenderer(testPart))

	l, buf := newTestLogger()
	l.SetParts(testPart, PartMessage)

	l.Info().Msg("hello")

	assert.Equal(t, "hello\n", buf.String())
}

func TestRegisterPartPanicsOnBuiltin(t *testing.T) {
	assert.PanicsWithValue(t, "clog: cannot override built-in part 0", func() {
		RegisterPart(PartTimestamp, func(Entry, *style.Config, bool) string { return "" })
	})

	assert.PanicsWithValue(t, "clog: cannot override built-in part 4", func() {
		RegisterPart(PartFields, func(Entry, *style.Config, bool) string { return "" })
	})
}

func TestUnregisterPartBuiltinIsNoop(t *testing.T) {
	UnregisterPart(PartMessage)

	l, buf := newTestLogger()
	l.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️ hello\n", buf.String())
}
