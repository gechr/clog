package clog

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/gechr/clog/fx"
	"github.com/gechr/clog/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAfterSetsDelay(t *testing.T) {
	b := Spinner("test").After(500 * time.Millisecond)
	assert.Equal(t, 500*time.Millisecond, b.Delay())
}

func TestAfterTaskFinishesBeforeDelay(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	start := time.Now()
	result := Spinner("loading").
		After(1*time.Second).
		Wait(context.Background(), func(_ context.Context) error {
			// Task completes immediately, well before the 1s delay.
			return nil
		})

	require.NoError(t, result.TaskErr)
	// Should return almost immediately since the task finishes before the delay.
	assert.Less(t, time.Since(start), 500*time.Millisecond)
}

func TestAfterTaskFinishesAfterDelay(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	result := Spinner("loading").
		After(10*time.Millisecond).
		Wait(context.Background(), func(_ context.Context) error {
			// Task takes longer than the delay, so animation would appear.
			time.Sleep(50 * time.Millisecond)
			return nil
		})

	require.NoError(t, result.TaskErr)
}

func TestAfterTaskErrorBeforeDelay(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	testErr := assert.AnError
	result := Spinner("loading").
		After(1*time.Second).
		Wait(context.Background(), func(_ context.Context) error {
			return testErr
		})

	require.ErrorIs(t, result.TaskErr, testErr)
}

func TestAfterContextCancelledDuringDelay(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	result := Spinner("loading").
		After(1*time.Second).
		Wait(ctx, func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})

	require.ErrorIs(t, result.TaskErr, context.Canceled)
}

func TestNonTTYSilentSetsField(t *testing.T) {
	b := Spinner("test").NonTTYSilent(true)
	assert.True(t, b.SuppressesNonTTY())

	b2 := Spinner("test").NonTTYSilent(false)
	assert.False(t, b2.SuppressesNonTTY())
}

func TestNonTTYSilentSuppressesOutput(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf) // &bytes.Buffer has no Fd(), so isTTY = false

	result := l.Spinner("loading").
		NonTTYSilent(true).
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(20 * time.Millisecond) // long enough to pass the delay gate
			return nil
		})

	require.NoError(t, result.Silent())
	assert.Empty(t, buf.String(), "NonTTYSilent(true) should produce no output on a non-TTY writer")
}

func TestNonTTYSilentFalseStillPrints(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)

	result := l.Spinner("loading").
		NonTTYSilent(false).
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(20 * time.Millisecond)
			return nil
		})

	require.NoError(t, result.Silent())
	assert.NotEmpty(
		t,
		buf.String(),
		"NonTTYSilent(false) should still print the static line on a non-TTY writer",
	)
}

func TestNonTTYLevelViaLogger(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.SetNonTTYLevel(LevelError)

	result := l.Spinner("loading").
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(20 * time.Millisecond)
			return nil
		})

	require.NoError(t, result.Silent())
	assert.Empty(
		t,
		buf.String(),
		"SetNonTTYLevel(LevelError) should suppress info-level animation output",
	)
}

func TestNonTTYLevelSuppressesLogEvents(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.SetNonTTYLevel(LevelError)

	l.Info().Msg("should be suppressed")
	l.Warn().Msg("should be suppressed")
	assert.Empty(t, buf.String(), "Info and Warn should be suppressed below LevelError on non-TTY")

	l.Error().Msg("should appear")
	assert.NotEmpty(t, buf.String(), "Error should pass through at LevelError threshold")
}

func TestNonTTYLevelResetWithUnsetLevel(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.SetNonTTYLevel(LevelError)
	l.SetNonTTYLevel(UnsetLevel)

	l.Info().Msg("should appear after reset")
	assert.NotEmpty(t, buf.String(), "UnsetLevel restores normal non-TTY output")
}

func TestElapsedFieldOrdering(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *fx.Builder
		wantKeys []string
	}{
		{
			name: "elapsed between other fields",
			build: func() *fx.Builder {
				return Spinner("test").
					Str("a", "1").
					Elapsed("elapsed").
					Int("b", 2)
			},
			wantKeys: []string{"a", "elapsed", "b"},
		},
		{
			name: "elapsed first",
			build: func() *fx.Builder {
				return Spinner("test").
					Elapsed("timer").
					Str("x", "y")
			},
			wantKeys: []string{"timer", "x"},
		},
		{
			name: "elapsed last",
			build: func() *fx.Builder {
				return Spinner("test").
					Str("x", "y").
					Int("n", 1).
					Elapsed("dur")
			},
			wantKeys: []string{"x", "n", "dur"},
		},
		{
			name: "elapsed only",
			build: func() *fx.Builder {
				return Spinner("test").
					Elapsed("t")
			},
			wantKeys: []string{"t"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.build()
			require.Len(t, b.Fields, len(tt.wantKeys))
			for i, key := range tt.wantKeys {
				assert.Equal(t, key, b.Fields[i].Key)
			}
			// The elapsed placeholder must have the elapsed type.
			for _, f := range b.Fields {
				if f.Key == b.ElapsedFieldKey() {
					_, ok := f.Value.(core.ElapsedField)
					assert.True(t, ok, "elapsed field should have elapsed type, got %T", f.Value)
				}
			}
		})
	}
}
