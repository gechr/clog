package clog

import "time"

// Handler processes log entries. Implement this interface to customise
// how log entries are formatted and output (e.g. JSON logging).
//
// When a Handler is set on a [Logger], the Logger handles level checking,
// field accumulation, timestamps, and mutex locking. The Handler only
// needs to format and write the entry.
type Handler interface {
	Log(Entry)
}

// HandlerFunc is an adapter to use ordinary functions as [Handler] values.
type HandlerFunc func(Entry)

// Log calls f(e).
func (f HandlerFunc) Log(e Entry) { f(e) }

// Field is a typed key-value pair attached to a log entry.
type Field struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// Entry represents a completed log entry passed to a [Handler].
type Entry struct {
	Time    time.Time `json:"time,omitzero"`
	Level   Level     `json:"level"`
	Prefix  string    `json:"prefix,omitempty"`
	Indent  int       `json:"indent,omitempty"`
	Message string    `json:"message"`
	Fields  []Field   `json:"fields,omitempty"`
}
