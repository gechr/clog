package fx

import (
	"context"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gechr/clog/internal/core"
	xansi "github.com/gechr/x/ansi"
)

const nl = "\n"

// writeString writes s to w, discarding the return values.
func writeString(w io.Writer, s string) {
	_, _ = io.WriteString(w, s)
}

// liveRegionProvider is the optional capability probed for on the task's
// [RenderOutput]. The root clog Output implements it; external Output
// implementations (and test stubs) that don't are served by the direct
// in-place render loop below, so the interface contracts stay unchanged.
type liveRegionProvider interface {
	LiveRegion() *core.LiveRegion
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

	// Live-region path: when the output exposes a shared LiveRegion, register
	// this animation as one slot of the region's stacked block. The region
	// owns the render cadence and cursor visibility, coordinates concurrent
	// standalone animations, and lets log lines displace the block instead of
	// being overpainted by the next frame.
	if p, ok := gt.cfg.Output.(liveRegionProvider); ok {
		if region := p.LiveRegion(); region != nil {
			return runAnimationRegion(ctx, b, gt, done, region)
		}
	}

	// Direct path: this animation owns the terminal and repaints in place.
	// Hide cursor during animation.
	writeString(gt.cfg.Out, xansi.HideCursor)
	defer writeString(gt.cfg.Out, xansi.ShowCursor)

	ticker := time.NewTicker(gt.tickRate)
	defer ticker.Stop()

	var frameBuf strings.Builder
	rendered := false
	prevLineCount := 1
	lastLine := ""
	lastWidth := 0

	// prevRows returns the physical row count of the block currently on
	// screen: zero before the first frame is rendered.
	prevRows := func() int {
		if rendered {
			return prevLineCount
		}
		return 0
	}

	for {
		select {
		case err := <-done:
			// For bar animations, render one final frame so 100% is visible
			// before the line is cleared and replaced with the completion message.
			if b.mode == AnimationBar && err == nil {
				resetBarWidgetState(gt)
				line := renderTaskLine(gt, false, time.Now(), nil)
				frameBuf.Reset()
				prevLineCount = appendRepaint(
					&frameBuf, []string{line}, prevRows(), gt.cfg.Output.Width(),
				)
				writeString(gt.cfg.Out, frameBuf.String())
				rendered = true
			}
			if rendered {
				writeString(gt.cfg.Out, syncFrame(
					xansi.CursorUp(prevLineCount)+xansi.EraseScreenBelow,
				))
			} else {
				writeString(gt.cfg.Out, xansi.ClearLine)
			}
			return err
		case now := <-ticker.C:
			line := renderTaskLine(gt, false, now, nil)
			width := gt.cfg.Output.Width()
			// Skip identical frames: nothing on screen would change, so a
			// write would be wasted bandwidth.
			if rendered && line == lastLine && width == lastWidth {
				continue
			}
			frameBuf.Reset()
			prevLineCount = appendRepaint(&frameBuf, []string{line}, prevRows(), width)
			writeString(gt.cfg.Out, frameBuf.String())
			rendered = true
			lastLine = line
			lastWidth = width
		case <-ctx.Done():
			// Preserve the in-progress line as scrollback.
			writeString(gt.cfg.Out, nl)
			return ctx.Err()
		}
	}
}

// runAnimationRegion runs a single animation as one slot of the output's
// shared [core.LiveRegion], blocking until the task completes or the context
// is cancelled. The region's render loop calls the slot's render closure
// under the region lock, so this goroutine must not call renderTaskLine while
// the slot is registered (the closure mutates per-tick caches on gt).
// Done and cancel semantics mirror the direct render loop: a successful bar shows a
// final 100% frame before the line is erased, and the completion message is
// printed by the caller through the logger, landing cleanly via displacement.
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
		if b.mode == AnimationBar && err == nil {
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
