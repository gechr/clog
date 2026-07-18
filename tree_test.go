package clog

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Basic single-level tree positions
// ---------------------------------------------------------------------------

func TestTreeMiddle(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Tree(TreeMiddle).Logger()
	sub.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️ ├── hello\n", buf.String())
}

func TestTreeFirst(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Tree(TreeFirst).Logger()
	sub.Info().Msg("hello")

	// TreeFirst renders the same as TreeMiddle by default.
	assert.Equal(t, "INF ℹ️ ├── hello\n", buf.String())
}

func TestTreeLast(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Tree(TreeLast).Logger()
	sub.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️ └── hello\n", buf.String())
}

// ---------------------------------------------------------------------------
// Multi-level nesting with correct continuation lines
// ---------------------------------------------------------------------------

func TestTreeTwoLevelsMiddleMiddle(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().
		Tree(TreeMiddle).
		Tree(TreeMiddle).
		Logger()
	sub.Info().Msg("hello")

	// Parent is middle → "│   ", child is middle → "├── ".
	assert.Equal(t, "INF ℹ️ │   ├── hello\n", buf.String())
}

func TestTreeTwoLevelsMiddleLast(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().
		Tree(TreeMiddle).
		Tree(TreeLast).
		Logger()
	sub.Info().Msg("hello")

	// Parent is middle → "│   ", child is last → "└── ".
	assert.Equal(t, "INF ℹ️ │   └── hello\n", buf.String())
}

func TestTreeTwoLevelsLastLast(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().
		Tree(TreeLast).
		Tree(TreeLast).
		Logger()
	sub.Info().Msg("hello")

	// Parent is last → "    ", child is last → "└── ".
	assert.Equal(t, "INF ℹ️     └── hello\n", buf.String())
}

func TestTreeTwoLevelsLastMiddle(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().
		Tree(TreeLast).
		Tree(TreeMiddle).
		Logger()
	sub.Info().Msg("hello")

	// Parent is last → "    ", child is middle → "├── ".
	assert.Equal(t, "INF ℹ️     ├── hello\n", buf.String())
}

func TestTreeThreeLevels(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().
		Tree(TreeMiddle).
		Tree(TreeLast).
		Tree(TreeMiddle).
		Logger()
	sub.Info().Msg("hello")

	// Ancestor 0 (middle) → "│   ", ancestor 1 (last) → "    ", leaf (middle) → "├── ".
	assert.Equal(t, "INF ℹ️ │       ├── hello\n", buf.String())
}

// ---------------------------------------------------------------------------
// Realistic tree output (multiple lines)
// ---------------------------------------------------------------------------

func TestTreeRealisticOutput(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))

	l.Info().Msg("Project")
	buf.Reset()

	mid := l.With().Tree(TreeMiddle).Logger()
	mid.Info().Msg("src/")
	line1 := buf.String()
	buf.Reset()

	nested1 := mid.With().Tree(TreeMiddle).Logger()
	nested1.Info().Msg("main.go")
	line2 := buf.String()
	buf.Reset()

	nested2 := mid.With().Tree(TreeLast).Logger()
	nested2.Info().Msg("util.go")
	line3 := buf.String()
	buf.Reset()

	last := l.With().Tree(TreeLast).Logger()
	last.Info().Msg("go.mod")
	line4 := buf.String()

	assert.Equal(t, "INF ℹ️ ├── src/\n", line1)
	assert.Equal(t, "INF ℹ️ │   ├── main.go\n", line2)
	assert.Equal(t, "INF ℹ️ │   └── util.go\n", line3)
	assert.Equal(t, "INF ℹ️ └── go.mod\n", line4)
}

// ---------------------------------------------------------------------------
// Additive via sub-loggers
// ---------------------------------------------------------------------------

func TestTreeAdditive(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub1 := l.With().Tree(TreeMiddle).Logger()
	sub2 := sub1.With().Tree(TreeLast).Logger()
	sub2.Info().Msg("nested")

	assert.Equal(t, "INF ℹ️ │   └── nested\n", buf.String())
}

// ---------------------------------------------------------------------------
// Composition with indent
// ---------------------------------------------------------------------------

func TestTreeWithIndent(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().
		Indent().
		Tree(TreeMiddle).
		Logger()
	sub.Info().Msg("hello")

	// 2 spaces (indent) + "├── " (tree) + message.
	assert.Equal(t, "INF ℹ️   ├── hello\n", buf.String())
}

func TestTreeWithIndentTwoLevels(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().
		Indent().
		Tree(TreeMiddle).
		Tree(TreeLast).
		Logger()
	sub.Info().Msg("hello")

	// 2 spaces (indent) + "│   " (tree ancestor) + "└── " (tree leaf).
	assert.Equal(t, "INF ℹ️   │   └── hello\n", buf.String())
}

// ---------------------------------------------------------------------------
// Custom TreeChars
// ---------------------------------------------------------------------------

func TestTreeCustomChars(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetTreeChars(TreeChars{
		First:    "┌─ ",
		Middle:   "├─ ",
		Last:     "└─ ",
		Continue: "│  ",
		Blank:    "   ",
	})

	sub := l.With().Tree(TreeFirst).Logger()
	sub.Info().Msg("first")
	line1 := buf.String()
	buf.Reset()

	sub = l.With().Tree(TreeMiddle).Logger()
	sub.Info().Msg("middle")
	line2 := buf.String()
	buf.Reset()

	sub = l.With().Tree(TreeLast).Logger()
	sub.Info().Msg("last")
	line3 := buf.String()

	assert.Equal(t, "INF ℹ️ ┌─ first\n", line1)
	assert.Equal(t, "INF ℹ️ ├─ middle\n", line2)
	assert.Equal(t, "INF ℹ️ └─ last\n", line3)
}

func TestTreeCustomCharsContinuation(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetTreeChars(TreeChars{
		First:    "┌─ ",
		Middle:   "├─ ",
		Last:     "└─ ",
		Continue: "│  ",
		Blank:    "   ",
	})

	sub := l.With().
		Tree(TreeFirst).
		Tree(TreeLast).
		Logger()
	sub.Info().Msg("nested")

	// First parent → Continue "│  ", last leaf → "└─ ".
	assert.Equal(t, "INF ℹ️ │  └─ nested\n", buf.String())
}

// ---------------------------------------------------------------------------
// Clone preservation - parent mutation after clone must not affect child
// ---------------------------------------------------------------------------

func TestTreeClonePreservesTree(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	child := l.With().Tree(TreeMiddle).Logger()

	// Logging on child after creation should still have the tree.
	child.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️ ├── hello\n", buf.String())
}

func TestTreeCloneIndependentTree(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	child := l.With().Tree(TreeMiddle).Logger()

	// Creating a sub-logger from child should not affect child.
	_ = child.With().Tree(TreeLast).Logger()

	child.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️ ├── hello\n", buf.String())
}

func TestTreeClonePreservesChars(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetTreeChars(TreeChars{
		First:    "F ",
		Middle:   "M ",
		Last:     "L ",
		Continue: "C ",
		Blank:    "B ",
	})

	child := l.With().Tree(TreeMiddle).Logger()

	// Mutate parent chars after cloning.
	l.SetTreeChars(TreeChars{
		First:    "X ",
		Middle:   "X ",
		Last:     "X ",
		Continue: "X ",
		Blank:    "X ",
	})

	child.Info().Msg("hello")

	// Child should use original "M " chars.
	assert.Equal(t, "INF ℹ️ M hello\n", buf.String())
}

// ---------------------------------------------------------------------------
// Handler integration (Entry.Tree)
// ---------------------------------------------------------------------------

func TestTreeHandler(t *testing.T) {
	l := NewWriter(nil)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	sub := l.With().
		Tree(TreeMiddle).
		Tree(TreeLast).
		Logger()
	sub.Info().Msg("test")

	assert.Equal(t, []TreePos{TreeMiddle, TreeLast}, got.Tree)
	assert.Equal(t, "test", got.Message)
}

func TestTreeHandlerNil(t *testing.T) {
	l := NewWriter(nil)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	l.Info().Msg("test")

	assert.Nil(t, got.Tree)
}

// ---------------------------------------------------------------------------
// Tree across log levels
// ---------------------------------------------------------------------------

func TestTreeAllLevels(t *testing.T) {
	tests := []struct {
		name    string
		method  func(*Logger) *Event
		wantLvl string
		symbol  string
	}{
		{"trace", (*Logger).Trace, "TRC", "🔍"},
		{"debug", (*Logger).Debug, "DBG", "🐞"},
		{"info", (*Logger).Info, "INF", "ℹ️"},
		{"dry", (*Logger).Dry, "DRY", "🚧"},
		{"warn", (*Logger).Warn, "WRN", "⚠️"},
		{"error", (*Logger).Error, "ERR", "❌"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			l := New(TestOutput(&buf))
			l.SetLevel(LevelTrace)

			sub := l.With().Tree(TreeMiddle).Logger()
			tt.method(sub).Msg("test")

			assert.Equal(t, tt.wantLvl+" "+tt.symbol+" ├── test\n", buf.String())
		})
	}
}

// ---------------------------------------------------------------------------
// Tree combined with fields
// ---------------------------------------------------------------------------

func TestTreeWithFields(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Tree(TreeMiddle).Logger()
	sub.Info().Str("key", "val").Msg("hello")

	assert.Equal(t, "INF ℹ️ ├── hello key=val\n", buf.String())
}

func TestTreeWithContextFields(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().
		Tree(TreeLast).
		Str("ctx", "field").
		Logger()
	sub.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️ └── hello ctx=field\n", buf.String())
}

// ---------------------------------------------------------------------------
// Tree combined with custom symbol
// ---------------------------------------------------------------------------

func TestTreeWithCustomSymbol(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().
		Tree(TreeMiddle).
		Symbol(">>>").
		Logger()
	sub.Info().Msg("hello")

	assert.Equal(t, "INF >>> ├── hello\n", buf.String())
}

// ---------------------------------------------------------------------------
// Tree empty message
// ---------------------------------------------------------------------------

func TestTreeEmptyMessage(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Tree(TreeMiddle).Logger()
	sub.Info().Msg("")

	// With tree set, the tree connectors still appear as the message part.
	assert.Equal(t, "INF ℹ️ ├── \n", buf.String())
}

// ---------------------------------------------------------------------------
// computeTreeIndent edge cases
// ---------------------------------------------------------------------------

func TestComputeTreeIndentEmpty(t *testing.T) {
	assert.Empty(t, computeTreeIndent(nil, DefaultTreeChars()))
}

func TestComputeTreeIndentSingleMiddle(t *testing.T) {
	assert.Equal(t, "├── ", computeTreeIndent([]TreePos{TreeMiddle}, DefaultTreeChars()))
}

func TestComputeTreeIndentSingleLast(t *testing.T) {
	assert.Equal(t, "└── ", computeTreeIndent([]TreePos{TreeLast}, DefaultTreeChars()))
}

func TestComputeTreeIndentSingleFirst(t *testing.T) {
	assert.Equal(t, "├── ", computeTreeIndent([]TreePos{TreeFirst}, DefaultTreeChars()))
}

func TestComputeTreeIndentDeep(t *testing.T) {
	tree := []TreePos{TreeMiddle, TreeMiddle, TreeMiddle, TreeLast}

	// Three ancestors (all middle → "│   ") + leaf (last → "└── ").
	assert.Equal(t, "│   │   │   └── ", computeTreeIndent(tree, DefaultTreeChars()))
}

func TestComputeTreeIndentMixedAncestors(t *testing.T) {
	tree := []TreePos{TreeMiddle, TreeLast, TreeFirst, TreeMiddle}

	// Ancestor 0 (middle) → "│   "
	// Ancestor 1 (last)   → "    "
	// Ancestor 2 (first)  → "│   "
	// Leaf (middle)       → "├── "
	assert.Equal(t, "│       │   ├── ", computeTreeIndent(tree, DefaultTreeChars()))
}

// ---------------------------------------------------------------------------
// Animation tree (non-TTY path - TestOutput is non-TTY)
// ---------------------------------------------------------------------------

func TestSpinnerTree(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))

	require.NoError(t, l.Spinner("loading").Tree(TreeMiddle).
		Wait(context.Background(), func(_ context.Context) error {
			return nil
		}).Msg("done"))

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, nl), nl)
	require.Len(t, lines, 2)

	assert.Equal(t, "INF ⏳ ├── loading\n", lines[0]+nl)
	assert.Equal(t, "INF ℹ️ ├── done\n", lines[1]+nl)
}

func TestSpinnerTreeOnTreeLogger(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Tree(TreeMiddle).Logger()

	require.NoError(t, sub.Spinner("loading").Tree(TreeLast).
		Wait(context.Background(), func(_ context.Context) error {
			return nil
		}).Msg("done"))

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, nl), nl)
	require.Len(t, lines, 2)

	// Logger tree [middle] + builder tree [last] → "│   └── ".
	assert.Equal(t, "INF ⏳ │   └── loading\n", lines[0]+nl)
	assert.Equal(t, "INF ℹ️ │   └── done\n", lines[1]+nl)
}

// ---------------------------------------------------------------------------
// Group task tree (non-TTY)
// ---------------------------------------------------------------------------

func TestGroupTaskTree(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))

	g := l.Group(context.Background())
	r := g.Add(l.Spinner("task").Tree(TreeLast)).
		Run(func(_ context.Context) error {
			return nil
		})
	g.Wait()

	require.NoError(t, r.Msg("done"))

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, nl), nl)
	require.Len(t, lines, 2)

	assert.Equal(t, "INF ⏳ └── task\n", lines[0]+nl)
	assert.Equal(t, "INF ℹ️ └── done\n", lines[1]+nl)
}

// ---------------------------------------------------------------------------
// Package-level SetTreeChars
// ---------------------------------------------------------------------------

func TestPackageLevelSetTreeChars(t *testing.T) {
	orig := Default()
	defer func() { SetDefault(orig) }()

	var buf bytes.Buffer

	SetDefault(New(TestOutput(&buf)))
	SetTreeChars(TreeChars{
		First:    "F ",
		Middle:   "M ",
		Last:     "L ",
		Continue: "C ",
		Blank:    "B ",
	})

	sub := With().Tree(TreeLast).Logger()
	sub.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️ L hello\n", buf.String())
}
