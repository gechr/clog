package fx

import (
	"context"
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

	Base        []core.Field
	FieldsPtr   *atomic.Pointer[[]core.Field]
	MsgPtr      *atomic.Pointer[string]
	MsgText     string
	ProgressPtr *atomic.Int64           // bar mode: current progress value; nil for non-bar modes
	LevelPtr    *atomic.Int64           // overridden level; nil when not updatable
	SymbolPtr   *atomic.Pointer[string] // symbol icon; nil when not updatable
	TotalPtr    *atomic.Int64           // bar mode: total progress value; nil for non-bar modes
}

// SetProgress sets the current progress value for a bar animation.
// Values are clamped to [0, total]. No-op if this is not a bar animation.
func (p *Update) SetProgress(current int) *Update {
	if p.ProgressPtr != nil {
		if current < 0 {
			current = 0
		}
		if p.TotalPtr != nil {
			if total := int(p.TotalPtr.Load()); current > total {
				current = total
			}
		}
		p.ProgressPtr.Store(int64(current))
	}
	return p
}

// SetTotal updates the total progress value for a bar animation.
// No-op if this is not a bar animation.
func (p *Update) SetTotal(total int) *Update {
	if p.TotalPtr != nil {
		if total <= 0 {
			total = 1
		}
		p.TotalPtr.Store(int64(total))
	}
	return p
}

// AddTotal atomically adds delta to the total progress value for a bar animation.
// The result is clamped to a minimum of 1.
// No-op if this is not a bar animation.
func (p *Update) AddTotal(delta int) *Update {
	if p.TotalPtr != nil {
		newVal := p.TotalPtr.Add(int64(delta))
		if newVal < 1 {
			p.TotalPtr.CompareAndSwap(newVal, 1)
		}
	}
	return p
}

// SetLevel overrides the log level used when rendering the final done line.
// No-op if the animation does not support level updates.
func (p *Update) SetLevel(level core.Level) *Update {
	if p.LevelPtr != nil {
		p.LevelPtr.Store(int64(level))
	}
	return p
}

// SetSymbol changes the icon displayed beside the message during animation.
// The change takes effect on the next render tick.
// No-op if the animation does not support symbol updates.
func (p *Update) SetSymbol(symbol string) *Update {
	if p.SymbolPtr != nil {
		p.SymbolPtr.Store(&symbol)
	}
	return p
}

// Msg sets the animation's displayed message.
func (p *Update) Msg(msg string) *Update {
	p.MsgText = msg
	return p
}

// Send applies the accumulated message and field changes to the animation atomically.
func (p *Update) Send() {
	msg := p.MsgText
	p.MsgPtr.Store(&msg)
	merged := core.MergeFields(p.Base, p.Fields)
	p.FieldsPtr.Store(&merged)
	p.Fields = nil // reset for reuse
}
