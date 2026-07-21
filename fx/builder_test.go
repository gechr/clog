package fx

import (
	"slices"
	"testing"
	"time"

	"github.com/gechr/clog/field/deadline"
	"github.com/gechr/clog/field/duration"
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
			Key: "timeout",
			Value: core.DeadlineField{
				Remaining:  12 * time.Second,
				From:       15 * time.Second,
				OmitOnDone: true,
			},
		},
		{Key: "url", Value: "https://example.com"},
	}, out)
}

func TestResolveDynamicFieldsDeadlineClampsAtZero(t *testing.T) {
	b := NewBuilder(BuilderConfig{})
	b.Deadline("timeout", 15*time.Second)

	out := b.ResolveDynamicFields(b.Fields, 20*time.Second)

	assert.Equal(t, []core.Field{
		{
			Key: "timeout",
			Value: core.DeadlineField{
				Remaining:  0,
				From:       15 * time.Second,
				OmitOnDone: true,
			},
		},
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
				Remaining:  12 * time.Second,
				From:       15 * time.Second,
				OmitOnDone: true,
				Trailing:   true,
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

func TestResolveDoneFieldsOmitsDeadlineByDefault(t *testing.T) {
	b := NewBuilder(BuilderConfig{})
	b.Deadline("timeout", 15*time.Second)

	fields := append(slices.Clone(b.Fields),
		core.Field{Key: "visible", Value: "yes"},
	)

	done := b.ResolveDoneFields(fields, 3*time.Second)
	assert.Equal(t, []core.Field{
		{Key: "visible", Value: "yes"},
	}, done)
}

func TestResolveDoneFieldsKeepsDeadlineWhenRequested(t *testing.T) {
	b := NewBuilder(BuilderConfig{})
	b.Deadline("timeout", 15*time.Second, deadline.WithOmitOnDone(false))

	done := b.ResolveDoneFields(b.Fields, 3*time.Second)
	assert.Equal(t, []core.Field{
		{
			Key: "timeout",
			Value: core.DeadlineField{
				Remaining: 12 * time.Second,
				From:      15 * time.Second,
			},
		},
	}, done)
}

func TestResolveDoneFieldsOmitsDefaultDeadlineAndOptInTimers(t *testing.T) {
	b := NewBuilder(BuilderConfig{})
	b.Elapsed("elapsed", elapsed.WithOmitOnDone(true))
	b.Deadline("timeout", 15*time.Second)
	b.Duration("latency", 3*time.Second, duration.WithOmitOnDone(true))

	fields := append(slices.Clone(b.Fields),
		core.Field{Key: "visible", Value: "yes"},
	)

	live := b.ResolveDynamicFields(fields, 3*time.Second)
	assert.Equal(t, []core.Field{
		{
			Key: "elapsed",
			Value: core.ElapsedField{
				Value:      3 * time.Second,
				OmitOnDone: true,
			},
		},
		{
			Key: "timeout",
			Value: core.DeadlineField{
				Remaining:  12 * time.Second,
				From:       15 * time.Second,
				OmitOnDone: true,
			},
		},
		{
			Key: "latency",
			Value: core.DurationField{
				Value:      3 * time.Second,
				OmitOnDone: true,
			},
		},
		{Key: "visible", Value: "yes"},
	}, live)

	done := b.ResolveDoneFields(fields, 3*time.Second)
	assert.Equal(t, []core.Field{
		{Key: "visible", Value: "yes"},
	}, done)
}

func TestResolveDoneFieldsDurationOption(t *testing.T) {
	b := NewBuilder(BuilderConfig{})
	b.Duration("latency", 3*time.Second, duration.WithOmitOnDone(true))

	done := b.ResolveDoneFields(b.Fields, 3*time.Second)
	assert.Empty(t, done)
}

func TestGroupRenderResolveDynamicFieldsDeadline(t *testing.T) {
	b := NewBuilder(BuilderConfig{})
	b.Deadline("timeout", 15*time.Second)

	out := resolveDynamicFields(b.Fields, b, 6*time.Second, 0, 0)

	assert.Equal(t, []core.Field{
		{
			Key: "timeout",
			Value: core.DeadlineField{
				Remaining:  9 * time.Second,
				From:       15 * time.Second,
				OmitOnDone: true,
			},
		},
	}, out)

	// Past the deadline the remaining time clamps at zero.
	out = resolveDynamicFields(b.Fields, b, time.Minute, 0, 0)

	assert.Equal(t, []core.Field{
		{
			Key: "timeout",
			Value: core.DeadlineField{
				Remaining:  0,
				From:       15 * time.Second,
				OmitOnDone: true,
			},
		},
	}, out)
}

func TestResolveDynamicFieldsUpdateDeadline(t *testing.T) {
	// No builder-level deadline key: the update-scoped countdown must still
	// resolve, keyed by nothing but its value type.
	b := NewBuilder(BuilderConfig{})

	fields := []core.Field{
		{Key: "url", Value: "https://example.com"},
		{
			Key: "wait",
			Value: core.DeadlineField{
				Remaining:  10 * time.Second,
				From:       15 * time.Second, // attached 5s into the task
				OmitOnDone: true,
			},
		},
	}

	out := b.ResolveDynamicFields(fields, 8*time.Second)

	assert.Equal(t, []core.Field{
		{Key: "url", Value: "https://example.com"},
		{
			Key: "wait",
			Value: core.DeadlineField{
				Remaining:  7 * time.Second,
				From:       15 * time.Second,
				OmitOnDone: true,
			},
		},
	}, out)
}

func TestResolveDynamicFieldsUpdateDeadlineClampsAtZero(t *testing.T) {
	b := NewBuilder(BuilderConfig{})

	fields := []core.Field{
		{
			Key: "wait",
			Value: core.DeadlineField{
				Remaining:  10 * time.Second,
				From:       15 * time.Second,
				OmitOnDone: true,
			},
		},
	}

	out := b.ResolveDynamicFields(fields, 20*time.Second)

	assert.Equal(t, core.DeadlineField{
		Remaining:  0,
		From:       15 * time.Second,
		OmitOnDone: true,
	}, out[0].Value)
}

func TestResolveDynamicFieldsUpdateElapsed(t *testing.T) {
	// No builder-level elapsed key: the update-scoped stopwatch must still
	// resolve, keyed by nothing but its value type.
	b := NewBuilder(BuilderConfig{})

	fields := []core.Field{
		{Key: "url", Value: "https://example.com"},
		{
			Key: "waited",
			Value: core.ElapsedField{
				Value:      2 * time.Second,
				Start:      3 * time.Second, // attached 5s in, backdated 2s
				OmitOnDone: true,
			},
		},
	}

	out := b.ResolveDynamicFields(fields, 8*time.Second)

	assert.Equal(t, []core.Field{
		{Key: "url", Value: "https://example.com"},
		{
			Key: "waited",
			Value: core.ElapsedField{
				Value:      5 * time.Second,
				Start:      3 * time.Second,
				OmitOnDone: true,
			},
		},
	}, out)
}

func TestResolveDynamicFieldsUpdateElapsedClampsAtZero(t *testing.T) {
	b := NewBuilder(BuilderConfig{})

	fields := []core.Field{
		{
			Key: "waited",
			Value: core.ElapsedField{
				Start:      3 * time.Second,
				OmitOnDone: true,
			},
		},
	}

	out := b.ResolveDynamicFields(fields, 1*time.Second)

	assert.Equal(t, core.ElapsedField{
		Value:      0,
		Start:      3 * time.Second,
		OmitOnDone: true,
	}, out[0].Value)
}
