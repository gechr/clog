package fx

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/gechr/clog/internal/core"
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
	fieldsPtr         *atomic.Pointer[[]core.Field]
	msgPtr            *atomic.Pointer[string]
	msgText           string
	progressPtr       *atomic.Int64           // bar mode: current progress value; nil for non-bar modes
	levelPtr          *atomic.Int64           // overridden level; nil when not updatable
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

// Send applies the accumulated message and field changes to the animation atomically.
func (p *Update) Send() {
	msg := p.msgText
	p.msgPtr.Store(&msg)
	merged := core.MergeFields(p.base, p.Fields)
	p.fieldsPtr.Store(&merged)
	p.Fields = nil // reset for reuse
}
