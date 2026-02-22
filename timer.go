package clog

import "time"

// Elapsed adds an elapsed-time field at the current position in the field
// list. The duration is measured from the first Elapsed call on this event
// until the event is finalised with [Event.Send], [Event.Msg], or
// [Event.Msgf].
//
// The key parameter is the field name (e.g. "elapsed"). The field uses the
// same formatting and styling as [AnimationBuilder.Elapsed].
//
//	e := clog.Info().Str("step", "migrate").Elapsed("elapsed")
//	runMigrations()
//	e.Msg("done")
//	// Output: INF ℹ️ done step=migrate elapsed=2s
func (e *Event) Elapsed(key string) *Event {
	if e == nil {
		return e
	}
	if e.elapsedStart.IsZero() {
		e.elapsedStart = time.Now()
	}
	e.fields = append(e.fields, Field{Key: key, Value: elapsed(0)})
	return e
}

// resolveElapsed replaces any elapsed(0) placeholder values in the event's
// fields with the actual elapsed duration since the first [Event.Elapsed] call.
func (e *Event) resolveElapsed() {
	if e.elapsedStart.IsZero() {
		return
	}
	dur := elapsed(time.Since(e.elapsedStart))
	for i := range e.fields {
		if v, ok := e.fields[i].Value.(elapsed); ok && v == 0 {
			e.fields[i].Value = dur
		}
	}
}
