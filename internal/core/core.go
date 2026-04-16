// Package core provides shared value types used by both the root clog
// package and subpackages like fx/. It lives under internal/ to prevent
// external consumers from depending on it directly.
package core

import "github.com/gechr/clog/level"

// ErrorKey is the default field key used by Err methods.
const ErrorKey = "error"

// Level represents a log level.
type Level = level.Level

// Field is a typed key-value pair attached to a log entry.
type Field struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// Part identifies a component of a formatted log line.
type Part int

const (
	// PartTimestamp is the timestamp component.
	PartTimestamp Part = iota
	// PartLevel is the level label component.
	PartLevel
	// PartSymbol is the emoji symbol component.
	PartSymbol
	// PartMessage is the log message component.
	PartMessage
	// PartFields is the structured fields component.
	PartFields
)

// Sort controls how fields are sorted in output.
type Sort int

const (
	// SortNone preserves the insertion order of fields (default).
	SortNone Sort = iota
	// SortAscending sorts fields by key A→Z.
	SortAscending
	// SortDescending sorts fields by key Z→A.
	SortDescending
)

// Quote controls how field values are quoted in log output.
type Quote int

const (
	// QuoteAuto quotes values only when they contain spaces, unprintable
	// characters, or embedded quotes. This is the default.
	QuoteAuto Quote = iota
	// QuoteAlways quotes all string, error, and default-kind values.
	QuoteAlways
	// QuoteNever disables quoting entirely.
	QuoteNever
)

// TreePos identifies a node's position among its siblings in a tree.
type TreePos int

const (
	// TreeFirst marks the first sibling.
	TreeFirst TreePos = iota
	// TreeMiddle marks a middle sibling (more siblings follow).
	TreeMiddle
	// TreeLast marks the last sibling (no more siblings follow).
	TreeLast
)

// TreeChars defines the box-drawing characters used by tree indentation.
type TreeChars struct {
	First    string // connector for TreeFirst  (default "├── ")
	Middle   string // connector for TreeMiddle (default "├── ")
	Last     string // connector for TreeLast   (default "└── ")
	Continue string // ancestor line when parent is First/Middle (default "│   ")
	Blank    string // ancestor line when parent is Last         (default "    ")
}
