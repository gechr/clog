package core

import (
	"fmt"
	"strings"
)

// NumberFormat controls how numeric field values (plain integers and both
// halves of a [Fraction]) are rendered.
//
// NumberFormat implements [encoding.TextMarshaler] and
// [encoding.TextUnmarshaler], so it works directly with [flag.TextVar] and
// most flag libraries.
//
//go:generate go tool golang.org/x/tools/cmd/stringer -type=NumberFormat -linecomment
type NumberFormat int

const (
	// NumberPlain renders numbers verbatim with no grouping (the default),
	// e.g. "1234567".
	NumberPlain NumberFormat = iota // plain
	// NumberGrouped inserts a separator every three digits, e.g.
	// "1,234,567". The separator is configurable.
	NumberGrouped // grouped
	// NumberCompact renders an abbreviated form using K/M/B/T suffixes,
	// e.g. "1.2M". Values below the configured minimum render with digit
	// grouping instead (e.g. "9,999" before "10K").
	NumberCompact // compact
)

// MarshalText implements [encoding.TextMarshaler].
func (f NumberFormat) MarshalText() ([]byte, error) {
	return []byte(f.String()), nil
}

// UnmarshalText implements [encoding.TextUnmarshaler].
func (f *NumberFormat) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case NumberPlain.String():
		*f = NumberPlain
	case NumberGrouped.String():
		*f = NumberGrouped
	case NumberCompact.String():
		*f = NumberCompact
	default:
		return fmt.Errorf("unknown number format: %q (valid: %q, %q, %q)",
			text, NumberPlain, NumberGrouped, NumberCompact)
	}
	return nil
}
