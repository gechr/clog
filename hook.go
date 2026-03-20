package clog

// HookPoint identifies when a hook fires during the log write lifecycle.
type HookPoint int

const (
	// HookBeforeWrite fires just before each log line is written to the output.
	HookBeforeWrite HookPoint = iota
	// HookAfterWrite fires just after each log line is written to the output.
	HookAfterWrite
)
