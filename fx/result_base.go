package fx

import (
	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/internal/core"
)

// resultBase holds common fields and fluent-config methods shared by
// WaitResult, TaskResult, and GroupResult.
type resultBase[T any] struct {
	core.FieldBuilder[T]

	ErrorMsg     *string
	LevelError   core.Level
	Log          Logger
	MsgStyle     *lipgloss.Style
	PartOverride *[]core.Part
	SuccessLevel core.Level
	SuccessMsg   string
	SymbolStr    *string
}

// send logs a success or error event, drawing the level, style, and part
// configuration from the base. fields, successMsg, and err vary per caller.
func (b *resultBase[T]) send(fields []core.Field, successMsg string, err error) {
	var lvl core.Level
	var msg string
	var errField error

	switch {
	case err == nil:
		lvl = b.SuccessLevel
		msg = successMsg
	case b.ErrorMsg != nil:
		lvl = b.LevelError
		msg = *b.ErrorMsg
		errField = err
	default:
		lvl = b.LevelError
		msg = err.Error()
	}

	b.Log.Done(DoneEvent{
		Level:    lvl,
		Fields:   fields,
		MsgStyle: b.MsgStyle,
		Parts:    b.PartOverride,
		Symbol:   b.SymbolStr,
		Msg:      msg,
		Err:      errField,
	})
}

// OnErrorLevel sets the log level for the error case.
func (b *resultBase[T]) OnErrorLevel(lvl core.Level) *T {
	b.LevelError = lvl
	return b.Self
}

// OnErrorMessage sets a custom message for the error case.
func (b *resultBase[T]) OnErrorMessage(msg string) *T {
	b.ErrorMsg = &msg
	return b.Self
}

// OnSuccessLevel sets the log level for the success case.
func (b *resultBase[T]) OnSuccessLevel(lvl core.Level) *T {
	b.SuccessLevel = lvl
	return b.Self
}

// OnSuccessMessage sets the message for the success case.
func (b *resultBase[T]) OnSuccessMessage(msg string) *T {
	b.SuccessMsg = msg
	return b.Self
}

// MessageStyle overrides the message text style for the completion log line,
// taking precedence over the global and per-level styles without mutating them.
// An empty [lipgloss.NewStyle] renders the message plain.
func (b *resultBase[T]) MessageStyle(s *lipgloss.Style) *T {
	b.MsgStyle = s
	return b.Self
}

// Parts overrides the log-line part order for the completion message.
func (b *resultBase[T]) Parts(parts ...core.Part) *T {
	b.PartOverride = new(parts)
	return b.Self
}

// Symbol sets a custom emoji symbol for the completion log message.
func (b *resultBase[T]) Symbol(symbol string) *T {
	b.SymbolStr = &symbol
	return b.Self
}
