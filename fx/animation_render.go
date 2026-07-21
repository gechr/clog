package fx

import (
	"context"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gechr/clog/internal/core"
)

const nl = "\n"

// writeString writes s to w, discarding the return values.
func writeString(w io.Writer, s string) {
	_, _ = io.WriteString(w, s)
}

// liveRegionProvider is the optional capability probed for on the task's
// [RenderOutput]. The root clog Output implements it; external Output
// implementations (and test stubs) that don't get a private region wrapping
// their writer, so the interface contracts stay unchanged.
type liveRegionProvider interface {
	LiveRegion() *core.LiveRegion
}

// liveRegionFor returns the output's shared [core.LiveRegion] when it
// provides one, or a private region wrapping the output's writer otherwise.
// A private region coordinates the animations of a single run; only the
// shared region additionally displaces log lines written on the same output.
func liveRegionFor(output RenderOutput) *core.LiveRegion {
	if p, ok := output.(liveRegionProvider); ok {
		if region := p.LiveRegion(); region != nil {
			return region
		}
	}
	return core.NewLiveRegion(output.Writer(), output.Width)
}

// runAnimation runs the render loop for a single animation, blocking until
// the task completes or the context is cancelled.
func runAnimation(
	ctx context.Context,
	b *Builder,
	task TaskFunc,
	msgPtr *atomic.Pointer[string],
	fields *atomic.Pointer[[]core.Field],
	levelPtr *atomic.Int64,
	symbolPtr *atomic.Pointer[string],
	startTime time.Time,
) error {
	// Run the task in a goroutine.
	done := make(chan error, 1)
	go func() {
		done <- task(ctx)
	}()

	// If a delay is configured, wait for it to elapse before showing
	// any animation. If the task completes first, return immediately.
	if b.delayDur > 0 {
		timer := time.NewTimer(b.delayDur)
		select {
		case err := <-done:
			timer.Stop()
			return err
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	// Build the gt and snapshot the logger's settings.
	gt := &renderTask{
		groupTask: &groupTask{
			builder:   b,
			fieldsPtr: fields,
			levelPtr:  levelPtr,
			msgPtr:    msgPtr,
			start:     startTime,
			symbolPtr: symbolPtr,
		},
	}
	captureTaskConfig(gt)

	// A level-disabled task still runs, but renders nothing on any writer.
	if gt.cfg.Silent {
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Don't animate if not a TTY (CI, piped output, etc.).
	// Print the initial message so the user knows something is in progress,
	// unless NonTTYSilent() was set, in which case suppress all output.
	// Dynamic fields (elapsed, bar percent) are stripped because their
	// initial zero values are meaningless without live updates.
	if !gt.cfg.IsTTY {
		if !gt.cfg.NonTTYSilent {
			fieldsStr := strings.TrimLeft(
				gt.cfg.FormatFields(b.StripDynamicFields(*fields.Load())),
				" ",
			)
			line := buildLine(
				gt.cfg.Order,
				gt.cfg.ReportTimestamp,
				time.Now().In(gt.cfg.TimeLocation).Format(gt.cfg.TimeFormat),
				gt.cfg.Label,
				*gt.symbolPtr.Load(),
				gt.cfg.Indentation+*msgPtr.Load(),
				fieldsStr,
			)
			writeString(gt.cfg.Out, line+nl)
		}

		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Register this animation as one slot of the region's stacked block. The
	// region owns the render cadence and cursor visibility, coordinates
	// concurrent standalone animations, and (on a shared region) lets log
	// lines displace the block instead of being overpainted by the next frame.
	return runAnimationRegion(ctx, b, gt, done, liveRegionFor(gt.cfg.Output))
}

// runAnimationRegion runs a single animation as one slot of the output's
// [core.LiveRegion], blocking until the task completes or the context is
// cancelled. The region's render loop calls the slot's render closure
// under the region lock, so this goroutine must not call renderTaskLine while
// the slot is registered (the closure mutates per-tick caches on gt).
// On success a bar shows a final 100% frame before the line is erased, and
// the completion message is printed by the caller through the logger,
// landing cleanly via displacement.
func runAnimationRegion(
	ctx context.Context,
	b *Builder,
	gt *renderTask,
	done <-chan error,
	region *core.LiveRegion,
) error {
	id := region.Register(func(now time.Time) string {
		return renderTaskLine(gt, false, now, nil)
	}, gt.tickRate)

	select {
	case err := <-done:
		// For bar animations, render one final frame so 100% is visible
		// before the line is cleared and replaced with the completion message.
		if b.cfg.Mode == AnimationBar && err == nil {
			resetBarWidgetState(gt)
			region.RenderFrame(time.Now())
		}
		region.Unregister(id)
		return err
	case <-ctx.Done():
		// Unregister first so the region stops calling the render closure;
		// after that this goroutine owns gt again.
		region.Unregister(id)
		// Preserve the in-progress line as scrollback - the last frame stays
		// on screen. The frozen frame is written as a regular displaced line
		// so it lands above any animations still live in the region.
		region.WriteLines(renderTaskLine(gt, false, time.Now(), nil) + nl)
		return ctx.Err()
	}
}
