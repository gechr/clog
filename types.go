package clog

import (
	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/fx"
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

// Quote controls how field values are quoted in log output.
type Quote = core.Quote

const (
	QuoteAuto   = core.QuoteAuto
	QuoteAlways = core.QuoteAlways
	QuoteNever  = core.QuoteNever
)

// QuotePair is an opening/closing delimiter pair used for smart quoting
// (see [Logger.SetSmartQuoteChars]). A zero Close means Close equals Open
// (symmetric quoting).
type QuotePair struct {
	Open  rune
	Close rune
}

// Wrap controls how log lines are wrapped when they exceed the terminal width.
type Wrap int

const (
	// WrapNone disables wrapping; lines are written as-is (default).
	WrapNone Wrap = iota
	// WrapHard breaks at the terminal width, even mid-word.
	WrapHard
	// WrapSoft breaks at word boundaries (spaces), falling back to
	// hard breaks only for words longer than the terminal width.
	WrapSoft
)

// Sort controls how fields are sorted in output.
type Sort = core.Sort

const (
	SortNone       = core.SortNone
	SortAscending  = core.SortAscending
	SortDescending = core.SortDescending
)

// FieldAlignment controls optional group-level field alignment behavior.
type FieldAlignment = fx.FieldAlignment

const (
	FieldAlignmentNone    = fx.FieldAlignmentNone
	FieldAlignmentMessage = fx.FieldAlignmentMessage
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
