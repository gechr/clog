package clog

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gechr/clog/fx"
	"github.com/gechr/clog/internal/core"
	xansi "github.com/gechr/x/ansi"
)

// Convenience aliases so callers can reference common fx types without
// importing subpackages directly.
type (
	Update     = fx.Update
	TaskResult = fx.TaskResult
)

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

// fxLogger adapts *Logger to the fx.Logger interface. This allows the fx
// package types (Builder, WaitResult, Group, etc.) to call back into root
// clog's logging and animation infrastructure without importing root clog.
type fxLogger struct{ l *Logger }

// Ensure fxLogger satisfies fx.Logger at compile time.
var _ fx.Logger = fxLogger{}

// Ensure *Output satisfies fx.Output at compile time.
var _ fx.Output = (*Output)(nil)

func (f fxLogger) Done(evt fx.DoneEvent) {
	e := f.l.newEvent(evt.Level)
	if e == nil {
		return
	}
	e = e.withFields(evt.Fields)
	if evt.Parts != nil {
		e = e.withParts(evt.Parts)
	}
	if evt.Symbol != nil {
		e = e.withSymbol(*evt.Symbol)
	}
	if evt.Err != nil {
		e.Err(evt.Err).Msg(evt.Msg)
	} else {
		e.Msg(evt.Msg)
	}
}

func (f fxLogger) WithIndent(depth int, tree []core.TreePos) fx.Logger {
	ctx := f.l.With()
	if depth > 0 {
		ctx = ctx.Depth(depth)
	}
	for _, pos := range tree {
		ctx = ctx.Tree(pos)
	}
	return fxLogger{ctx.Logger()}
}

func (f fxLogger) Output() fx.Output {
	return f.l.Output()
}

func (f fxLogger) RunAnimation(ctx context.Context, cfg fx.AnimationConfig) error {
	return runAnimation(
		ctx,
		cfg.Builder,
		cfg.Task,
		cfg.MsgPtr,
		cfg.FieldsPtr,
		cfg.LevelPtr,
		cfg.SymbolPtr,
		cfg.StartTime,
	)
}

func (f fxLogger) RunGroup(ctx context.Context, g *fx.Group) error {
	return runGroupLoop(ctx, g)
}

func runAnimation(
	ctx context.Context,
	b *fx.Builder,
	task fx.TaskFunc,
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
	if b.DelayDur > 0 {
		timer := time.NewTimer(b.DelayDur)
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
	gt := &groupTask{
		GroupTask: &fx.GroupTask{
			Builder:   b,
			FieldsPtr: fields,
			LevelPtr:  levelPtr,
			MsgPtr:    msgPtr,
			StartTime: startTime,
			SymbolPtr: symbolPtr,
		},
	}
	captureTaskConfig(gt)

	// Don't animate if not a TTY (CI, piped output, etc.).
	// Print the initial message so the user knows something is in progress,
	// unless NonTTYSilent() was set, in which case suppress all output.
	// Dynamic fields (elapsed, bar percent) are stripped because their
	// initial zero values are meaningless without live updates.
	if !gt.cfg.isTTY {
		if !gt.cfg.nonTTYSilent {
			fieldsStr := strings.TrimLeft(
				formatFields(b.StripDynamicFields(*fields.Load()), gt.fieldOpts),
				" ",
			)
			line := buildLine(
				gt.cfg.order,
				gt.cfg.reportTS,
				time.Now().In(gt.cfg.timeLoc).Format(gt.cfg.timeFmt),
				gt.cfg.label,
				*gt.SymbolPtr.Load(),
				gt.cfg.indentation+*msgPtr.Load(),
				fieldsStr,
			)
			writeString(gt.cfg.out, line+nl)
		}

		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Hide cursor during animation.
	writeString(gt.cfg.out, xansi.HideCursor)
	defer writeString(gt.cfg.out, xansi.ShowCursor)

	ticker := time.NewTicker(gt.tickRate)
	defer ticker.Stop()

	var frameBuf strings.Builder
	rendered := false
	prevLineCount := 1

	for {
		select {
		case err := <-done:
			// For bar animations, render one final frame so 100% is visible
			// before the line is cleared and replaced with the completion message.
			if b.Mode == fx.AnimationBar && err == nil {
				resetBarWidgetState(gt)
				line := renderTaskLine(gt, false, time.Now(), nil)
				frameBuf.Reset()
				if rendered {
					frameBuf.WriteString(xansi.CursorUp(prevLineCount))
					frameBuf.WriteString(xansi.EraseScreenBelow)
				} else {
					frameBuf.WriteString(xansi.ClearLine)
				}
				frameBuf.WriteString(line)
				frameBuf.WriteString(nl)
				writeString(gt.cfg.out, frameBuf.String())
				rendered = true
				prevLineCount = frameRows(line, gt.cfg.output.Width())
			}
			if rendered {
				writeString(gt.cfg.out, xansi.CursorUp(prevLineCount)+xansi.EraseScreenBelow)
			} else {
				writeString(gt.cfg.out, xansi.ClearLine)
			}
			return err
		case now := <-ticker.C:
			line := renderTaskLine(gt, false, now, nil)
			frameBuf.Reset()
			if rendered {
				frameBuf.WriteString(xansi.CursorUp(prevLineCount))
				frameBuf.WriteString(xansi.EraseScreenBelow)
			} else {
				frameBuf.WriteString(xansi.ClearLine)
			}
			frameBuf.WriteString(line)
			frameBuf.WriteString(nl)
			writeString(gt.cfg.out, frameBuf.String())
			rendered = true
			prevLineCount = frameRows(line, gt.cfg.output.Width())
		case <-ctx.Done():
			switch {
			case b.ClearOnCancel && rendered:
				writeString(gt.cfg.out, xansi.CursorUp(prevLineCount)+xansi.EraseScreenBelow)
			case !b.ClearOnCancel:
				writeString(gt.cfg.out, nl)
			default:
				writeString(gt.cfg.out, xansi.ClearLine)
			}
			return ctx.Err()
		}
	}
}
