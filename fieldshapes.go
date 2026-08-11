package clog

import "maps"

// FieldShape controls which tokens a field renders, not how they are styled.
// Shapes are structural: they apply with and without color, so piped output
// keeps the same tokens as a terminal. Prefix and Suffix additionally take the
// value's resolved style when styling is active, so the whole token reads as
// one unit (e.g. a green "[prod]" including its brackets).
type FieldShape struct {
	// OmitKey renders the value alone: no key and no separator.
	OmitKey bool
	// Prefix is written immediately before the (possibly quoted) value.
	Prefix string
	// Suffix is written immediately after the (possibly quoted) value.
	Suffix string
}

// FieldShapeMap maps field key names to their shapes.
type FieldShapeMap = map[string]FieldShape

// SetFieldShapes sets per-key token shaping for the built-in formatter.
// Like [Logger.SetParts], shapes replace any previously configured ones -
// pass nil to clear them. Keys absent from the map render normally.
//
// Shapes affect the built-in formatter (including animation task rows) only;
// a custom [Handler] receives the raw fields.
func (l *Logger) SetFieldShapes(shapes FieldShapeMap) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fieldShapes = maps.Clone(shapes)
}
