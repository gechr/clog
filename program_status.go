package clog

import "github.com/charmbracelet/x/ansi"

// ProgramStatus describes a terminal program status report (OSC 7501).
// Every report replaces the addressed record, so repeat App and Title on
// each update. Progress is indeterminate unless HasProgress is true.
type ProgramStatus = ansi.ProgramStatus

// ProgramState describes a program's current activity.
type ProgramState = ansi.ProgramState

// ProgramStatusKind describes why a blocked program needs user input.
type ProgramStatusKind = ansi.ProgramStatusKind

const (
	ProgramStateIdle    = ansi.ProgramStateIdle
	ProgramStateWorking = ansi.ProgramStateWorking
	ProgramStateDone    = ansi.ProgramStateDone
	ProgramStateBlocked = ansi.ProgramStateBlocked
	ProgramStateError   = ansi.ProgramStateError
	ProgramStateClear   = ansi.ProgramStateClear

	ProgramStatusKindPermission = ansi.ProgramStatusKindPermission
	ProgramStatusKindQuestion   = ansi.ProgramStatusKindQuestion
	ProgramStatusKindAuth       = ansi.ProgramStatusKindAuth
)

// SetProgramStatus reports status to the logger's terminal independently of
// log levels and handlers. It emits nothing for a nil logger or a non-TTY
// output, even with ColorAlways. ColorNever does not disable status reports.
// Reports are serialized with live animations without adding a visible line.
//
// Callers own the lifecycle: report done or error on completion, idle on
// cancellation, and blocked only when user action is needed. Reports are not
// inferred from log messages or animation completion. Unsupported terminals
// ignore the sequence, so no feature query or input reader is required.
func (l *Logger) SetProgramStatus(status ProgramStatus) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output.SetProgramStatus(status)
}

// SetProgramStatus writes a sanitized OSC 7501 report to a terminal output.
// Invalid states or IDs are ignored. The Charm ANSI encoder removes control
// characters and limits text lengths. Never include sensitive data in reports.
func (o *Output) SetProgramStatus(status ProgramStatus) {
	if !o.isTTY {
		return
	}
	if sequence := ansi.SetProgramStatus(status); sequence != "" {
		o.LiveRegion().WriteControl(sequence)
	}
}
