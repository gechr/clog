package clog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
