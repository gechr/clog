package clog

import "time"

// Timer measures the duration of an operation and logs it on completion.
// Created by [Logger.Timed] or the package-level [Timed] function.
//
// Use Timer when you want elapsed-time logging without animations. The
// elapsed field uses the same formatting and styling as
// [AnimationBuilder.Elapsed].
//
//	t := clog.Timed("database migration")
//	runMigrations()
//	t.Send()
//	// Output: INF ℹ️ database migration elapsed=2s
type Timer struct {
	fieldBuilder[Timer]

	logger     *Logger
	message    string
	start      time.Time
	elapsedKey string
}

// Timed creates a new [Timer] that records the current time. The returned
// Timer logs the elapsed duration when finalised with [Timer.Send],
// [Timer.Err], or [Timer.Msg].
func (l *Logger) Timed(msg string) *Timer {
	t := &Timer{
		logger:     l,
		message:    msg,
		start:      time.Now(),
		elapsedKey: "elapsed",
	}
	t.initSelf(t)
	return t
}

// ElapsedKey sets the field key for the elapsed duration. Defaults to "elapsed".
func (t *Timer) ElapsedKey(key string) *Timer {
	t.elapsedKey = key
	return t
}

// Send finalises the timer, logging at [InfoLevel] with the original message.
func (t *Timer) Send() {
	t.log(InfoLevel, t.message, nil)
}

// Err finalises the timer. If err is nil, logs at [InfoLevel] with the
// original message. If err is non-nil, logs at [ErrorLevel] with the
// original message and an error field.
func (t *Timer) Err(err error) {
	if err != nil {
		t.log(ErrorLevel, t.message, err)
		return
	}
	t.log(InfoLevel, t.message, nil)
}

// Msg finalises the timer, logging at [InfoLevel] with a custom message.
func (t *Timer) Msg(msg string) {
	t.log(InfoLevel, msg, nil)
}

// log creates an event at the given level and writes it with the accumulated
// fields plus the elapsed duration.
func (t *Timer) log(level Level, msg string, err error) {
	e := t.logger.newEvent(level)
	if e == nil {
		return
	}
	e.fields = append(e.fields, t.fields...)
	if err != nil {
		e.fields = append(e.fields, Field{Key: ErrorKey, Value: err})
	}
	e.fields = append(e.fields, Field{Key: t.elapsedKey, Value: elapsed(time.Since(t.start))})
	e.Msg(msg)
}

// Timed creates a new [Timer] on the [Default] logger.
func Timed(msg string) *Timer { return Default.Timed(msg) }
