package clog

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProgramStatusTerminalOnly(t *testing.T) {
	for _, mode := range []ColorMode{ColorAuto, ColorAlways, ColorNever} {
		var buf bytes.Buffer
		out := NewOutput(&buf, mode)
		logger := New(out)
		logger.SetProgramStatus(ProgramStatus{State: ProgramStateWorking, App: "example"})
		assert.Empty(t, buf.String())
		assert.Nil(t, out.region.Load(), "redirected output must not create a live region")
	}
}

func TestProgramStatusBypassesFormatting(t *testing.T) {
	var buf bytes.Buffer
	out := fakeTTYOutput(t, &buf) // ColorNever still permits status reports.
	logger := New(out)
	logger.SetLevel(LevelFatal)
	logger.SetHandler(HandlerFunc(func(Entry) { t.Error("status reached log handler") }))
	logger.AddHook(HookBeforeWrite, func() { t.Error("status reached log hook") })
	logger.SetProgramStatus(ProgramStatus{
		State:       ProgramStateWorking,
		App:         "example",
		Message:     "Processing",
		Progress:    25,
		HasProgress: true,
	})
	assert.Equal(
		t,
		"\x1b]7501;state=working:progress=25:app=example:msg=UHJvY2Vzc2luZw==\a",
		buf.String(),
	)

	buf.Reset()
	logger.SetProgramStatus(ProgramStatus{State: ProgramStateWorking, Progress: 25})
	assert.Equal(
		t,
		"\x1b]7501;state=working\a",
		buf.String(),
		"unset progress must remain indeterminate",
	)
	buf.Reset()
	logger.SetProgramStatus(ProgramStatus{State: "invalid"})
	logger.SetProgramStatus(ProgramStatus{State: ProgramStateWorking, ID: "bad:id"})
	assert.Empty(t, buf.String())
}

func TestProgramStatusDefaultAndNilLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := New(fakeTTYOutput(t, &buf))
	previous := Default()
	SetDefault(logger)
	t.Cleanup(func() { SetDefault(previous) })
	SetProgramStatus(ProgramStatus{State: ProgramStateDone, App: "example"})
	assert.Equal(t, "\x1b]7501;state=done:app=example\a", buf.String())

	var absent *Logger
	require.NotPanics(t, func() {
		absent.SetProgramStatus(ProgramStatus{State: ProgramStateError})
	})
}

func TestProgramStatusDuringAnimation(t *testing.T) {
	var buf bytes.Buffer
	out := fakeTTYOutput(t, &buf)
	region := out.LiveRegion()
	id := region.Register(func(time.Time) string { return "Processing" }, time.Hour)
	t.Cleanup(func() { region.Unregister(id) })
	buf.Reset()
	out.SetProgramStatus(ProgramStatus{State: ProgramStateBlocked, Kind: ProgramStatusKindAuth})
	assert.Equal(t, "\x1b]7501;state=blocked:kind=auth\a", buf.String())
	buf.Reset()
	region.RenderFrame(time.Now())
	assert.Empty(t, buf.String(), "status must not invalidate the painted frame")
	region.Suspend(false)
	buf.Reset()
	out.SetProgramStatus(ProgramStatus{State: ProgramStateIdle})
	assert.Equal(t, "\x1b]7501;state=idle\a", buf.String())
}

func TestProgramStatusConcurrentOutput(t *testing.T) {
	var buf bytes.Buffer
	out := fakeTTYOutput(t, &buf)
	region := out.LiveRegion()
	id := region.Register(func(time.Time) string { return "Processing" }, time.Hour)
	var workers sync.WaitGroup
	for range 20 {
		workers.Go(func() {
			out.SetProgramStatus(ProgramStatus{State: ProgramStateWorking})
			out.WriteLine("log line\n")
			region.RenderFrame(time.Now())
		})
	}
	workers.Wait()
	region.Unregister(id)
	assert.Equal(t, 20, strings.Count(buf.String(), "\x1b]7501;state=working\a"))
	assert.Equal(t, 20, strings.Count(buf.String(), "log line\n"))
}
