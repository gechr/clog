package fx

import (
	"slices"
	"testing"
	"time"

	"github.com/gechr/clog/field/deadline"
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

func TestResolveDynamicFieldsDeadline(t *testing.T) {
	b := NewBuilder(BuilderConfig{})
	b.Deadline("timeout", 15*time.Second)

	fields := append(slices.Clone(b.Fields),
		core.Field{Key: "url", Value: "https://example.com"},
	)

	out := b.ResolveDynamicFields(fields, 3*time.Second)

	assert.Equal(t, []core.Field{
		{
			Key:   "timeout",
			Value: core.DeadlineField{Remaining: 12 * time.Second, From: 15 * time.Second},
		},
		{Key: "url", Value: "https://example.com"},
	}, out)
}

func TestResolveDynamicFieldsDeadlineClampsAtZero(t *testing.T) {
	b := NewBuilder(BuilderConfig{})
	b.Deadline("timeout", 15*time.Second)

	out := b.ResolveDynamicFields(b.Fields, 20*time.Second)

	assert.Equal(t, []core.Field{
		{Key: "timeout", Value: core.DeadlineField{Remaining: 0, From: 15 * time.Second}},
	}, out)
}

func TestResolveDynamicFieldsDeadlineTrailing(t *testing.T) {
	b := NewBuilder(BuilderConfig{})
	b.Deadline("timeout", 15*time.Second, deadline.WithTrailing())

	// Runtime-added fields land after the builder's base fields.
	fields := append(slices.Clone(b.Fields),
		core.Field{Key: "url", Value: "https://example.com"},
		core.Field{Key: "region", Value: "us-east-1"},
	)

	out := b.ResolveDynamicFields(fields, 3*time.Second)

	assert.Equal(t, []core.Field{
		{Key: "url", Value: "https://example.com"},
		{Key: "region", Value: "us-east-1"},
		{
			Key: "timeout",
			Value: core.DeadlineField{
				Remaining: 12 * time.Second,
				From:      15 * time.Second,
				Trailing:  true,
			},
		},
	}, out)
}

func TestStripDynamicFieldsDeadline(t *testing.T) {
	b := NewBuilder(BuilderConfig{})
	b.Deadline("timeout", 15*time.Second)

	fields := append(slices.Clone(b.Fields),
		core.Field{Key: "url", Value: "https://example.com"},
	)

	out := b.StripDynamicFields(fields)

	assert.Equal(t, []core.Field{
		{Key: "url", Value: "https://example.com"},
	}, out)
}

func TestGroupRenderResolveDynamicFieldsDeadline(t *testing.T) {
	b := NewBuilder(BuilderConfig{})
	b.Deadline("timeout", 15*time.Second)

	out := resolveDynamicFields(b.Fields, b, 6*time.Second, 0, 0)

	assert.Equal(t, []core.Field{
		{
			Key:   "timeout",
			Value: core.DeadlineField{Remaining: 9 * time.Second, From: 15 * time.Second},
		},
	}, out)

	// Past the deadline the remaining time clamps at zero.
	out = resolveDynamicFields(b.Fields, b, time.Minute, 0, 0)

	assert.Equal(t, []core.Field{
		{Key: "timeout", Value: core.DeadlineField{Remaining: 0, From: 15 * time.Second}},
	}, out)
}
