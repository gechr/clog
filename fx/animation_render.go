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

// frameRows returns the number of physical terminal rows the rendered line
// occupies once wrapping at termWidth is accounted for. ANSI escape codes
// are stripped before measuring. Falls back to 1 when the width is unknown
// (e.g. Output.Width() == 0).
func frameRows(line string, termWidth int) int {
	if termWidth <= 0 {
		return 1
	}
	w := xansi.StringWidth(line)
	if w <= 0 {
		return 1
	}
	return (w + termWidth - 1) / termWidth
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
			switch {
			case b.clearOnCancel && rendered:
				writeString(gt.cfg.Out, syncFrame(
					xansi.CursorUp(prevLineCount)+xansi.EraseScreenBelow,
				))
			case !b.clearOnCancel:
				writeString(gt.cfg.Out, nl)
			default:
				writeString(gt.cfg.Out, xansi.ClearLine)
			}
			return ctx.Err()
		}
	}
}
