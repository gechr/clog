package fx

import (
	"slices"
	"testing"
	"time"

	"github.com/gechr/clog/field/elapsed"
	"github.com/gechr/clog/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestResolveDynamicFieldsTrailing(t *testing.T) {
	b := NewBuilder(BuilderConfig{})
	b.Elapsed("elapsed", elapsed.Trailing())

	// Runtime-added fields land after the builder's base fields.
	fields := append(slices.Clone(b.Fields),
		core.Field{Key: "url", Value: "https://example.com"},
		core.Field{Key: "region", Value: "us-east-1"},
	)

	out := b.ResolveDynamicFields(fields, 3*time.Second)

	assert.Equal(t, []core.Field{
		{Key: "url", Value: "https://example.com"},
		{Key: "region", Value: "us-east-1"},
		{Key: "elapsed", Value: core.ElapsedField{Value: 3 * time.Second, Trailing: true}},
	}, out)
}

func TestResolveDynamicFieldsNoTrailing(t *testing.T) {
	b := NewBuilder(BuilderConfig{})
	b.Elapsed("elapsed")

	fields := append(slices.Clone(b.Fields),
		core.Field{Key: "url", Value: "https://example.com"},
	)

	out := b.ResolveDynamicFields(fields, 3*time.Second)

	assert.Equal(t, []core.Field{
		{Key: "elapsed", Value: core.ElapsedField{Value: 3 * time.Second}},
		{Key: "url", Value: "https://example.com"},
	}, out)
}
