package clog

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/gechr/clog/internal/core"
)

// Style is a pointer type (*lipgloss.Style). A nil Style means "no style"
// and is safe to pass wherever a Style is accepted.
type Style = *lipgloss.Style

// fieldBuilder is the package-local alias for [core.FieldBuilder].
// All fluent builders (Context, fx.Builder, WaitResult, etc.)
// embed this type.
type fieldBuilder[T any] = core.FieldBuilder[T]

// LabelMap maps levels to strings (used for labels, symbols, etc.).
type LabelMap map[Level]string

// Align controls how text is aligned within a fixed-width column.
type Align int

const (
	// AlignNone disables alignment padding.
	AlignNone Align = iota
	// AlignLeft left-aligns text (pads with trailing spaces).
	AlignLeft
	// AlignRight right-aligns text (pads with leading spaces).
	AlignRight
	// AlignCenter center-aligns text (pads with leading and trailing spaces).
	AlignCenter
)

// QuoteMode controls how field values are quoted in log output.
type QuoteMode = core.QuoteMode

const (
	QuoteAuto   = core.QuoteAuto
	QuoteAlways = core.QuoteAlways
	QuoteNever  = core.QuoteNever
)

// Sort controls how fields are sorted in output.
type Sort = core.Sort

const (
	SortNone       = core.SortNone
	SortAscending  = core.SortAscending
	SortDescending = core.SortDescending
)

// Part identifies a component of a formatted log line.
type Part = core.Part

const (
	PartTimestamp = core.PartTimestamp
	PartLevel     = core.PartLevel
	PartSymbol    = core.PartSymbol
	PartMessage   = core.PartMessage
	PartFields    = core.PartFields
)

// TreePos identifies a node's position among its siblings in a tree.
type TreePos = core.TreePos

const (
	TreeFirst  = core.TreeFirst
	TreeMiddle = core.TreeMiddle
	TreeLast   = core.TreeLast
)

// TreeChars defines the box-drawing characters used by tree indentation.
// Override with [Logger.SetTreeChars].
type TreeChars = core.TreeChars
