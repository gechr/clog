package fx

import (
	"context"
	"strconv"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
)

func TestGroupFieldAlignmentOption(t *testing.T) {
	g := NewGroup(context.Background(), nil, WithFieldAlignment(FieldAlignmentMessage))

	assert.Equal(t, FieldAlignmentMessage, g.fieldAlignment)
}

func TestGroupParallelismOption(t *testing.T) {
	g := NewGroup(context.Background(), nil, WithParallelism(3))
	assert.Equal(t, 3, g.parallelism)

	g = NewGroup(context.Background(), nil, WithParallelism(0))
	assert.Zero(t, g.parallelism)

	g = NewGroup(context.Background(), nil, WithParallelism(-1))
	assert.Equal(t, -1, g.parallelism)
}

func TestGroupHeaderOption(t *testing.T) {
	cb := func(_, _ int, _ *Update) {}
	b := NewBuilder(BuilderConfig{Message: "header"})
	g := NewGroup(context.Background(), nil, WithHeader(b, cb))

	assert.NotNil(t, g.header)
	assert.Equal(t, "header", g.header.builder.message)
}

func TestGroupFooterOption(t *testing.T) {
	cb := func(_, _ int, _ *Update) {}
	b := NewBuilder(BuilderConfig{Message: "footer"})
	g := NewGroup(context.Background(), nil, WithFooter(b, cb))

	assert.NotNil(t, g.footer)
	assert.Equal(t, "footer", g.footer.builder.message)
}

func TestGroupTransientStatusOptions(t *testing.T) {
	g := NewGroup(context.Background(), nil, WithTransientHeader(), WithTransientFooter())

	assert.True(t, g.transientHeader)
	assert.True(t, g.transientFooter)
}

func TestGroupRenderDelayOption(t *testing.T) {
	g := NewGroup(context.Background(), nil, WithRenderDelay(250*time.Millisecond))

	assert.Equal(t, 250*time.Millisecond, g.renderDelay)
}

func TestGroupMaxHeightPercentOption(t *testing.T) {
	g := NewGroup(context.Background(), nil, WithMaxHeightPercent(0.5))
	assert.InDelta(t, 0.5, g.maxHeightPercent, 0.001)
}

func TestGroupMaxHeightPercentClamped(t *testing.T) {
	g := NewGroup(context.Background(), nil, WithMaxHeightPercent(1.5))
	assert.InDelta(t, 1.0, g.maxHeightPercent, 0.001)

	g = NewGroup(context.Background(), nil, WithMaxHeightPercent(-0.5))
	assert.InDelta(t, 0.0, g.maxHeightPercent, 0.001)
}

func TestGroupMaxLinesOption(t *testing.T) {
	g := NewGroup(context.Background(), nil, WithMaxLines(10))

	assert.Equal(t, 10, g.maxLines)
}

func TestGroupMonotonicOption(t *testing.T) {
	g := NewGroup(context.Background(), nil, WithMonotonic())

	assert.True(t, g.monotonic)
}

func TestGroupHideDoneOption(t *testing.T) {
	g := NewGroup(context.Background(), nil, WithHideDone())

	assert.True(t, g.hideDone)
}

func TestGroupClearOnCancelOption(t *testing.T) {
	g := NewGroup(context.Background(), nil, WithClearOnCancel())

	assert.True(t, g.clearOnCancel)
}

func TestGroupOverflowIndicatorOption(t *testing.T) {
	g := NewGroup(context.Background(), nil)
	assert.False(t, g.overflowIndicator, "indicator is disabled by default")

	g = NewGroup(context.Background(), nil, WithOverflowIndicator())
	assert.True(t, g.overflowIndicator)
	assert.Nil(t, g.overflowFunc)
	assert.Nil(t, g.overflowStyle)

	style := new(lipgloss.NewStyle().Faint(true))
	g = NewGroup(context.Background(), nil, WithOverflowIndicator(
		WithOverflowText(strconv.Itoa),
		WithOverflowStyle(style),
	))
	assert.True(t, g.overflowIndicator)
	assert.NotNil(t, g.overflowFunc)
	assert.Same(t, style, g.overflowStyle)
}

func TestGroupSyncAnimationsDefaultAndOptOut(t *testing.T) {
	g := NewGroup(context.Background(), nil)
	assert.True(t, g.syncAnimations)

	g = NewGroup(context.Background(), nil, WithoutSyncAnimations())
	assert.False(t, g.syncAnimations)
}
