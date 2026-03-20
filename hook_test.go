package clog

import (
	"bytes"
	"strings"
	"testing"
)

func TestAddHookBeforeWrite(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.SetLevel(LevelInfo)

	var called bool
	l.AddHook(HookBeforeWrite, func() { called = true })

	l.Info().Msg("hello")
	if !called {
		t.Fatal("HookBeforeWrite not called")
	}
	if !strings.Contains(buf.String(), "hello") {
		t.Fatalf("expected output, got %q", buf.String())
	}
}

func TestAddHookAfterWrite(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.SetLevel(LevelInfo)

	var called bool
	l.AddHook(HookAfterWrite, func() { called = true })

	l.Info().Msg("hello")
	if !called {
		t.Fatal("HookAfterWrite not called")
	}
}

func TestMultipleHooksRunInOrder(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.SetLevel(LevelInfo)

	var order []int
	l.AddHook(HookBeforeWrite, func() { order = append(order, 1) })
	l.AddHook(HookBeforeWrite, func() { order = append(order, 2) })
	l.AddHook(HookBeforeWrite, func() { order = append(order, 3) })

	l.Info().Msg("test")
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Fatalf("hooks ran in wrong order: %v", order)
	}
}

func TestClearHooks(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.SetLevel(LevelInfo)

	var called bool
	l.AddHook(HookBeforeWrite, func() { called = true })
	l.ClearHooks(HookBeforeWrite)

	l.Info().Msg("test")
	if called {
		t.Fatal("hook should not have been called after ClearHooks")
	}
}

func TestClearAllHooks(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.SetLevel(LevelInfo)

	var beforeCalled, afterCalled bool
	l.AddHook(HookBeforeWrite, func() { beforeCalled = true })
	l.AddHook(HookAfterWrite, func() { afterCalled = true })
	l.ClearAllHooks()

	l.Info().Msg("test")
	if beforeCalled || afterCalled {
		t.Fatal("hooks should not have been called after ClearAllHooks")
	}
}

func TestClearHooksOnlyAffectsPoint(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.SetLevel(LevelInfo)

	var beforeCalled, afterCalled bool
	l.AddHook(HookBeforeWrite, func() { beforeCalled = true })
	l.AddHook(HookAfterWrite, func() { afterCalled = true })
	l.ClearHooks(HookBeforeWrite)

	l.Info().Msg("test")
	if beforeCalled {
		t.Fatal("before hook should not have been called")
	}
	if !afterCalled {
		t.Fatal("after hook should still have been called")
	}
}

func TestHookWithHandler(t *testing.T) {
	var beforeCalled, afterCalled bool
	var handlerCalled bool

	l := New(nil)
	l.SetLevel(LevelInfo)
	l.SetHandler(HandlerFunc(func(Entry) { handlerCalled = true }))
	l.AddHook(HookBeforeWrite, func() { beforeCalled = true })
	l.AddHook(HookAfterWrite, func() { afterCalled = true })

	l.Info().Msg("test")
	if !handlerCalled {
		t.Fatal("handler not called")
	}
	if !beforeCalled {
		t.Fatal("before hook not called with custom handler")
	}
	if !afterCalled {
		t.Fatal("after hook not called with custom handler")
	}
}

func TestHooksInheritedBySubLogger(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.SetLevel(LevelInfo)

	var called bool
	l.AddHook(HookBeforeWrite, func() { called = true })

	sub := l.With().Str("component", "sub").Logger()
	sub.Info().Msg("from sub")

	if !called {
		t.Fatal("hook should be inherited by sub-logger")
	}
}

func TestDefaultLoggerAddHook(t *testing.T) {
	old := Default
	defer func() { Default = old }()

	var buf bytes.Buffer
	Default = NewWriter(&buf)
	Default.SetLevel(LevelInfo)

	var called bool
	AddHook(HookBeforeWrite, func() { called = true })
	Info().Msg("test")

	if !called {
		t.Fatal("default AddHook not called")
	}
}

func TestDefaultLoggerClearHooks(t *testing.T) {
	old := Default
	defer func() { Default = old }()

	var buf bytes.Buffer
	Default = NewWriter(&buf)
	Default.SetLevel(LevelInfo)

	var called bool
	AddHook(HookBeforeWrite, func() { called = true })
	ClearHooks(HookBeforeWrite)
	Info().Msg("test")

	if called {
		t.Fatal("hook should not be called after ClearHooks")
	}
}
