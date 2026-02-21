package clog

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAfterSetsDelay(t *testing.T) {
	b := Spinner("test").After(500 * time.Millisecond)
	assert.Equal(t, 500*time.Millisecond, b.delay)
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

	require.NoError(t, result.err)
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

	require.NoError(t, result.err)
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

	require.ErrorIs(t, result.err, testErr)
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

	require.ErrorIs(t, result.err, context.Canceled)
}

func TestElapsedFieldOrdering(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *AnimationBuilder
		wantKeys []string
	}{
		{
			name: "elapsed between other fields",
			build: func() *AnimationBuilder {
				return Spinner("test").
					Str("a", "1").
					Elapsed("elapsed").
					Int("b", 2)
			},
			wantKeys: []string{"a", "elapsed", "b"},
		},
		{
			name: "elapsed first",
			build: func() *AnimationBuilder {
				return Spinner("test").
					Elapsed("timer").
					Str("x", "y")
			},
			wantKeys: []string{"timer", "x"},
		},
		{
			name: "elapsed last",
			build: func() *AnimationBuilder {
				return Spinner("test").
					Str("x", "y").
					Int("n", 1).
					Elapsed("dur")
			},
			wantKeys: []string{"x", "n", "dur"},
		},
		{
			name: "elapsed only",
			build: func() *AnimationBuilder {
				return Spinner("test").
					Elapsed("t")
			},
			wantKeys: []string{"t"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.build()
			require.Len(t, b.fields, len(tt.wantKeys))
			for i, key := range tt.wantKeys {
				assert.Equal(t, key, b.fields[i].Key)
			}
			// The elapsed placeholder must have the elapsed type.
			for _, f := range b.fields {
				if f.Key == b.elapsedKey {
					_, ok := f.Value.(elapsed)
					assert.True(t, ok, "elapsed field should have elapsed type, got %T", f.Value)
				}
			}
		})
	}
}
