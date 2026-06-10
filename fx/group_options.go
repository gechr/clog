package fx

import "time"

// GroupOption configures a group before it starts rendering.
type GroupOption func(*Group)

// WithFieldAlignment sets the group-level field alignment mode.
func WithFieldAlignment(alignment FieldAlignment) GroupOption {
	return func(g *Group) {
		g.FieldAlignment = alignment
	}
}

// WithParallelism limits how many group tasks may run at once.
// Values less than or equal to zero disable the limit.
func WithParallelism(parallelism int) GroupOption {
	return func(g *Group) {
		g.Parallelism = parallelism
	}
}

// WithMonotonic prevents grouped bar fills and their associated percentage
// text from rendering lower than the highest progress fraction previously
// shown for each task.
func WithMonotonic() GroupOption {
	return func(g *Group) {
		g.Monotonic = true
	}
}

// WithFooter adds a status line below the task block, updated each tick.
// The builder provides initial config (level, symbol, message, fields).
// The callback updates the message and fields each tick based on progress.
func WithFooter(b *Builder, fn GroupStatusFunc) GroupOption {
	return func(g *Group) {
		g.Footer = &GroupStatus{Builder: b, Callback: fn}
	}
}

// WithTransientFooter hides the footer when no task rows are visible.
func WithTransientFooter() GroupOption {
	return func(g *Group) {
		g.TransientFooter = true
	}
}

// WithHeader adds a status line above the task block, updated each tick.
// The builder provides initial config (level, symbol, message, fields).
// The callback updates the message and fields each tick based on progress.
func WithHeader(b *Builder, fn GroupStatusFunc) GroupOption {
	return func(g *Group) {
		g.Header = &GroupStatus{Builder: b, Callback: fn}
	}
}

// WithTransientHeader hides the header when no task rows are visible.
func WithTransientHeader() GroupOption {
	return func(g *Group) {
		g.TransientHeader = true
	}
}

// WithRenderDelay suppresses group rendering until d has elapsed. If all
// tasks finish before the delay, the group produces no live-render output.
func WithRenderDelay(d time.Duration) GroupOption {
	return func(g *Group) {
		g.RenderDelay = d
	}
}

// WithMaxLines caps the number of visible lines in the group render block.
// When set, this takes precedence over the automatic terminal height cap.
// Values less than or equal to zero are ignored.
func WithMaxLines(n int) GroupOption {
	return func(g *Group) {
		g.MaxLines = n
	}
}

// WithMaxHeightPercent caps the group render block to a percentage of the
// terminal height (e.g. 0.5 for half). Clamped to (0, 1].
// When both WithMaxLines and WithMaxHeightPercent are set, the smaller wins.
func WithMaxHeightPercent(percent float64) GroupOption {
	return func(g *Group) {
		g.MaxHeightPercent = max(0, min(percent, 1))
	}
}

// WithHideDone removes completed tasks from the rendered block so that
// only active and pending tasks remain visible. Completed tasks reappear
// in the caller's own logging (e.g. via [WaitResult.Msg]).
func WithHideDone() GroupOption {
	return func(g *Group) {
		g.HideDone = true
	}
}

// WithClearOnCancel controls whether the rendered block is erased when
// the context is cancelled. By default the last frame is preserved so the
// user can see what was on screen when the interrupt arrived.
func WithClearOnCancel() GroupOption {
	return func(g *Group) {
		g.ClearOnCancel = true
	}
}

// WithoutSyncAnimations stops animations in the group from sharing a
// common epoch, so spinners, pulses, and shimmers no longer stay in
// lockstep. Sync is enabled by default.
func WithoutSyncAnimations() GroupOption {
	return func(g *Group) {
		g.SyncAnimations = false
	}
}
