package fx

import (
	"time"

	"charm.land/lipgloss/v2"
	xmath "github.com/gechr/x/math"
)

// GroupOption configures a group before it starts rendering.
type GroupOption func(*Group)

type suspendOptions struct {
	showCursor bool
}

// SuspendOption configures [Group.Suspend].
type SuspendOption func(*suspendOptions)

// WithShowCursor controls whether [Group.Suspend] shows the cursor while the
// terminal is released. By default Suspend preserves the current cursor
// visibility; final cleanup still restores the cursor.
func WithShowCursor(show bool) SuspendOption {
	return func(o *suspendOptions) {
		o.showCursor = show
	}
}

// WithFieldAlignment sets the group-level field alignment mode.
func WithFieldAlignment(alignment FieldAlignment) GroupOption {
	return func(g *Group) {
		g.fieldAlignment = alignment
	}
}

// WithParallelism limits how many group tasks may run at once.
// Values less than or equal to zero disable the limit.
func WithParallelism(parallelism int) GroupOption {
	return func(g *Group) {
		g.parallelism = parallelism
	}
}

// WithMonotonic prevents grouped bar fills and their associated percentage
// text from rendering lower than the highest progress fraction previously
// shown for each task.
func WithMonotonic() GroupOption {
	return func(g *Group) {
		g.monotonic = true
	}
}

// WithFooter adds a status line below the task block, updated each tick.
// The builder provides initial config (level, symbol, message, fields).
// The callback updates the message and fields each tick based on progress.
func WithFooter(b *Builder, fn GroupStatusFunc) GroupOption {
	return func(g *Group) {
		g.footer = &groupStatus{builder: b, callback: fn}
	}
}

// WithTransientFooter hides the footer when no task rows are visible.
func WithTransientFooter() GroupOption {
	return func(g *Group) {
		g.transientFooter = true
	}
}

// WithHeader adds a status line above the task block, updated each tick.
// The builder provides initial config (level, symbol, message, fields).
// The callback updates the message and fields each tick based on progress.
func WithHeader(b *Builder, fn GroupStatusFunc) GroupOption {
	return func(g *Group) {
		g.header = &groupStatus{builder: b, callback: fn}
	}
}

// WithTransientHeader hides the header when no task rows are visible.
func WithTransientHeader() GroupOption {
	return func(g *Group) {
		g.transientHeader = true
	}
}

// WithRenderDelay suppresses group rendering until d has elapsed. If all
// tasks finish before the delay, the group produces no live-render output.
func WithRenderDelay(d time.Duration) GroupOption {
	return func(g *Group) {
		g.renderDelay = d
	}
}

// WithMaxLines caps the number of visible lines in the group render block.
// When set, this takes precedence over the automatic terminal height cap.
// Values less than or equal to zero are ignored.
func WithMaxLines(n int) GroupOption {
	return func(g *Group) {
		g.maxLines = n
	}
}

// WithMaxHeightPercent caps the group render block to a percentage of the
// terminal height (e.g. 0.5 for half). Clamped to (0, 1].
// When both WithMaxLines and WithMaxHeightPercent are set, the smaller wins.
func WithMaxHeightPercent(percent float64) GroupOption {
	return func(g *Group) {
		g.maxHeightPercent = xmath.Clamp01(percent)
	}
}

// WithHideDone removes completed tasks from the rendered block so that
// only active and pending tasks remain visible. Completed tasks reappear
// in the caller's own logging (e.g. via [WaitResult.Msg]).
func WithHideDone() GroupOption {
	return func(g *Group) {
		g.hideDone = true
	}
}

// WithClearOnCancel controls whether the rendered block is erased when
// the context is cancelled. By default the last frame is preserved so the
// user can see what was on screen when the interrupt arrived.
func WithClearOnCancel() GroupOption {
	return func(g *Group) {
		g.clearOnCancel = true
	}
}

// OverflowIndicatorFunc formats the overflow indicator's message for the
// number of tasks hidden by the row budget.
type OverflowIndicatorFunc func(hidden int) string

// OverflowOption configures the overflow indicator line enabled by
// [WithOverflowIndicator].
type OverflowOption func(*overflowSettings)

type overflowSettings struct {
	format OverflowIndicatorFunc
	style  *lipgloss.Style
}

// WithOverflowText sets the formatter for the indicator's message text.
// The default renders "N more".
func WithOverflowText(fn OverflowIndicatorFunc) OverflowOption {
	return func(o *overflowSettings) {
		o.format = fn
	}
}

// WithOverflowStyle sets the style applied to the indicator's message text
// (e.g. dim via Faint), taking precedence over the level message style.
func WithOverflowStyle(s *lipgloss.Style) OverflowOption {
	return func(o *overflowSettings) {
		o.style = s
	}
}

// WithOverflowIndicator appends an "… N more" line to the task block when the
// terminal height (or [WithMaxLines] / [WithMaxHeightPercent]) leaves some
// tasks unrendered. Without this option hidden tasks are dropped silently.
// The message text and style are configurable via [WithOverflowText] and
// [WithOverflowStyle].
func WithOverflowIndicator(opts ...OverflowOption) GroupOption {
	return func(g *Group) {
		var o overflowSettings
		for _, opt := range opts {
			opt(&o)
		}
		g.overflowIndicator = true
		g.overflowFunc = o.format
		g.overflowStyle = o.style
	}
}

// WithoutSyncAnimations stops animations in the group from sharing a
// common epoch, so spinners, pulses, and shimmers no longer stay in
// lockstep. Sync is enabled by default.
func WithoutSyncAnimations() GroupOption {
	return func(g *Group) {
		g.syncAnimations = false
	}
}
