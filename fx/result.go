package fx

import (
	"context"
	"errors"
	"fmt"

	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/level"
)

// WaitResult holds the result of a [Builder.Wait] or [Builder.Progress]
// operation and allows chaining additional fields before finalising the log output.
type WaitResult struct {
	resultBase[WaitResult]

	TaskErr error
}

// NewWaitResult creates a WaitResult with default settings.
func NewWaitResult(
	err error,
	logger Logger,
	parts *[]core.Part,
	lvl core.Level,
	msg string,
) *WaitResult {
	w := &WaitResult{
		resultBase: resultBase[WaitResult]{
			Log:          logger,
			PartOverride: parts,
			SuccessLevel: lvl,
			SuccessMsg:   msg,
			LevelError:   level.Error,
		},
		TaskErr: err,
	}
	w.InitSelf(w)
	return w
}

// Err returns the error, logging success at info level or failure at error
// level using the original animation message.
func (w *WaitResult) Err() error {
	return w.Send()
}

// Msg logs at info level with the given message on success, or at error
// level with the error string on failure. Returns the error.
func (w *WaitResult) Msg(msg string) error {
	w.SuccessMsg = msg
	return w.Send()
}

// Msgf is like [WaitResult.Msg] but accepts a format string.
func (w *WaitResult) Msgf(format string, a ...any) error {
	return w.Msg(fmt.Sprintf(format, a...))
}

// Send finalises the result, logging at the configured success or error
// level. Context cancellation errors are silenced since they indicate
// an interrupt, not a task failure. Returns the error from the task.
func (w *WaitResult) Send() error {
	if w.TaskErr != nil && errors.Is(w.TaskErr, context.Canceled) {
		return w.TaskErr
	}

	var lvl core.Level
	var msg string
	var errField error

	switch {
	case w.TaskErr == nil:
		lvl = w.SuccessLevel
		msg = w.SuccessMsg
	case w.ErrorMsg != nil:
		lvl = w.LevelError
		msg = *w.ErrorMsg
		errField = w.TaskErr
	default:
		lvl = w.LevelError
		msg = w.TaskErr.Error()
	}

	evt := DoneEvent{
		Level:    lvl,
		Fields:   w.Fields,
		MsgStyle: w.MsgStyle,
		Parts:    w.PartOverride,
		Symbol:   w.SymbolStr,
		Msg:      msg,
	}
	if errField != nil {
		evt.Err = errField
	}
	w.Log.Done(evt)
	return w.TaskErr
}

// Silent returns just the error without logging anything.
func (w *WaitResult) Silent() error {
	return w.TaskErr
}
