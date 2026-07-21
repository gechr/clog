package fx

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gechr/clog/field/deadline"
	"github.com/gechr/clog/field/elapsed"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/level"
)

// TaskFunc is a function executed by [Builder.Wait].
type TaskFunc func(context.Context) error

// UpdateFunc is a function executed by [Builder.Progress].
// The [Update] allows updating the animation's message and fields.
type UpdateFunc func(context.Context, *Update) error

// Update is a fluent builder for updating an animation's message and fields
// during an [UpdateFunc]. Call [Update.Msg] and field methods to
// build the update, then [Update.Send] to apply it atomically.
type Update struct {
	core.FieldBuilder[Update]

	base              []core.Field
	elapsed           func() time.Duration // task elapsed time, for anchoring update-scoped deadlines; nil-safe
	fieldsPtr         *atomic.Pointer[[]core.Field]
	levelPtr          *atomic.Int64 // overridden level; nil when not updatable
	msgPtr            *atomic.Pointer[string]
	msgText           string
	progressPtr       *atomic.Int64           // bar mode: current progress value; nil for non-bar modes
	symbolOverridePtr *atomic.Bool            // when set to true, disables animated spinner in favour of static symbol
	symbolPtr         *atomic.Pointer[string] // symbol icon; nil when not updatable
	totalPtr          *atomic.Int64           // bar mode: total progress value; nil for non-bar modes
}

// Progress returns the current progress value for a bar animation,
// or 0 if this is not a bar animation.
func (p *Update) Progress() int {
	if p.progressPtr == nil {
		return 0
	}
	return int(p.progressPtr.Load())
}

// Message returns the animation's currently displayed message (the last
// value applied with [Update.Send], or the initial message).
func (p *Update) Message() string {
	if m := p.msgPtr.Load(); m != nil {
		return *m
	}
	return ""
}

// SetProgress sets the current progress value for a bar animation.
// Values are clamped to [0, total]. No-op if this is not a bar animation.
func (p *Update) SetProgress(current int) *Update {
	if p.progressPtr != nil {
		if current < 0 {
			current = 0
		}
		if p.totalPtr != nil {
			if total := int(p.totalPtr.Load()); current > total {
				current = total
			}
		}
		p.progressPtr.Store(int64(current))
	}
	return p
}

// SetTotal updates the total progress value for a bar animation.
// No-op if this is not a bar animation.
func (p *Update) SetTotal(total int) *Update {
	if p.totalPtr != nil {
		if total <= 0 {
			total = 1
		}
		p.totalPtr.Store(int64(total))
	}
	return p
}

// AddTotal atomically adds delta to the total progress value for a bar animation.
// The result is clamped to a minimum of 1.
// No-op if this is not a bar animation.
func (p *Update) AddTotal(delta int) *Update {
	if p.totalPtr != nil {
		newVal := p.totalPtr.Add(int64(delta))
		if newVal < 1 {
			p.totalPtr.CompareAndSwap(newVal, 1)
		}
	}
	return p
}

// SetLevel overrides the log level used when rendering the final done line.
// No-op if the animation does not support level updates.
func (p *Update) SetLevel(level core.Level) *Update {
	if p.levelPtr != nil {
		p.levelPtr.Store(int64(level))
	}
	return p
}

// SetSymbol changes the icon displayed beside the message during animation.
// The change takes effect on the next render tick.
// No-op if the animation does not support symbol updates.
func (p *Update) SetSymbol(symbol string) *Update {
	if p.symbolPtr != nil {
		p.symbolPtr.Store(&symbol)
	}
	if p.symbolOverridePtr != nil {
		p.symbolOverridePtr.Store(true)
	}
	return p
}

// Msg sets the animation's displayed message.
func (p *Update) Msg(msg string) *Update {
	p.msgText = msg
	return p
}

// Msgf sets the animation's displayed message with formatting.
func (p *Update) Msgf(format string, args ...any) *Update {
	return p.Msg(fmt.Sprintf(format, args...))
}

// Deadline adds an auto-updating countdown field that displays the time
// remaining until from has elapsed, clamped at 0. Unlike [Builder.Deadline],
// whose countdown is anchored to the animation's start, the countdown is
// anchored to the moment this method is called - so a deadline can be scoped
// to one phase of a task and attached only when that phase begins. Like any
// other update field, it lasts until a later [Update.Send] omits it.
// Coloring runs against the logger's deadline gradient by consumed time; use
// options from the [deadline] package to override rendering for this field.
// The field is omitted from the done row by default
// (deadline.WithOmitOnDone). On a non-animated (non-TTY) line the field
// renders statically as the full window.
func (p *Update) Deadline(
	key string,
	from time.Duration,
	opts ...deadline.Option,
) *Update {
	f := core.DeadlineField{From: from, Remaining: from, OmitOnDone: true}
	deadline.Apply(&f, opts...)
	if p.elapsed != nil {
		// The renderer computes Remaining = From - taskElapsed, so folding
		// the elapsed-so-far into From anchors the countdown at now.
		f.From += p.elapsed()
	}
	p.Fields = append(p.Fields, core.Field{Key: key, Value: f})
	return p
}

// Elapsed adds an auto-updating stopwatch field that displays the time since
// it started counting. Unlike [Builder.Elapsed], whose timer is anchored to
// the animation's start, the stopwatch is anchored to the moment this method
// is called, backdated by since - so a timer can be scoped to one phase of a
// task (since 0), or continue a count that began before the field was
// attached. Like any other update field, it lasts until a later [Update.Send]
// omits it - and keeps counting between Sends, so a stalled loop still renders
// live progress. Coloring runs against the logger's elapsed gradient; use
// options from the [elapsed] package to override rendering for this field.
// The field is omitted from the done row by default
// (elapsed.WithOmitOnDone). On a non-animated (non-TTY) line the field
// renders statically as since.
func (p *Update) Elapsed(
	key string,
	since time.Duration,
	opts ...elapsed.Option,
) *Update {
	f := core.ElapsedField{Value: since, OmitOnDone: true}
	elapsed.Apply(&f, opts...)
	// The renderer computes Value = taskElapsed - Start, so subtracting the
	// already-elapsed count anchors the stopwatch since ago.
	f.Start = -since
	if p.elapsed != nil {
		f.Start += p.elapsed()
	}
	p.Fields = append(p.Fields, core.Field{Key: key, Value: f})
	return p
}

// Clear empties the row - the message, every field (including the builder's
// base fields), and any level override - and applies immediately, without a
// separate [Update.Send]. A task that finalizes cleared logs no done line:
// the explicit "nothing durable to say", for a loop that would otherwise
// freeze a stale mid-state snapshot into the terminal history when it
// returns on a teardown.
func (p *Update) Clear() {
	empty := ""
	p.msgPtr.Store(&empty)
	merged := []core.Field{}
	p.fieldsPtr.Store(&merged)
	p.Fields = nil
	p.msgText = ""
	if p.levelPtr != nil {
		p.levelPtr.Store(int64(level.Unset))
	}
}

// Send applies the accumulated message and field changes to the animation atomically.
func (p *Update) Send() {
	msg := p.msgText
	p.msgPtr.Store(&msg)
	merged := core.MergeFields(p.base, p.Fields)
	p.fieldsPtr.Store(&merged)
	p.Fields = nil // reset for reuse
}
