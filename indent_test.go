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
// Basic indentation via With().Indent() / With().Depth()
// ---------------------------------------------------------------------------

func TestIndentOneLevel(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Indent().Logger()
	sub.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️   hello\n", buf.String())
}

func TestIndentTwoLevelsChained(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Indent().Indent().Logger()
	sub.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️     hello\n", buf.String())
}

func TestDepth(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Depth(3).Logger()
	sub.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️       hello\n", buf.String())
}

func TestIndentAdditive(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub1 := l.With().Indent().Logger()
	sub2 := sub1.With().Indent().Logger()
	sub2.Info().Msg("nested")

	assert.Equal(t, "INF ℹ️     nested\n", buf.String())
}

func TestIndentLevelColumnFixed(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))

	l.Info().Msg("top")
	top := buf.String()
	buf.Reset()

	sub := l.With().Indent().Logger()
	sub.Info().Msg("child")
	child := buf.String()

	assert.Equal(t, "INF ℹ️ top\n", top)
	assert.Equal(t, "INF ℹ️   child\n", child)
}

func TestZeroIndentNoChange(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️ hello\n", buf.String())
}

func TestIndentEmptyMessage(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Indent().Logger()
	sub.Info().Msg("")

	// With indent > 0, the indent spaces still appear as the message part.
	assert.Equal(t, "INF ℹ️   \n", buf.String())
}

// ---------------------------------------------------------------------------
// SetIndent (direct) — bypasses With().Indent() context chain
// ---------------------------------------------------------------------------

func TestSetIndentDirect(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetIndent(2)
	l.Info().Msg("hello")

	// 2 levels × 2 spaces/level = 4 spaces before message.
	assert.Equal(t, "INF ℹ️     hello\n", buf.String())
}

func TestSetIndentOverridesContext(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Indent().Logger() // depth 1
	sub.SetIndent(3)                  // override to depth 3
	sub.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️       hello\n", buf.String())
}

func TestSetIndentZeroClears(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetIndent(5)
	l.SetIndent(0) // clear
	l.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️ hello\n", buf.String())
}

// ---------------------------------------------------------------------------
// Custom indent width
// ---------------------------------------------------------------------------

func TestIndentCustomWidth(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetIndentWidth(4)

	sub := l.With().Indent().Logger()
	sub.Info().Msg("hello")

	// 1 level × 4 spaces = 4 spaces.
	assert.Equal(t, "INF ℹ️     hello\n", buf.String())
}

func TestIndentWidthOne(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetIndentWidth(1)

	sub := l.With().Depth(3).Logger()
	sub.Info().Msg("hello")

	// 3 levels × 1 space = 3 spaces.
	assert.Equal(t, "INF ℹ️    hello\n", buf.String())
}

// ---------------------------------------------------------------------------
// Indent prefixes
// ---------------------------------------------------------------------------

func TestIndentPrefixes(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetIndentPrefixes([]string{"|"})

	sub := l.With().Indent().Logger()
	sub.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️   | hello\n", buf.String())
}

func TestIndentPrefixesDeep(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetIndentPrefixes([]string{">"})

	sub := l.With().Depth(2).Logger()
	sub.Info().Msg("hello")

	// 4 spaces (2 levels * 2 width) + "> " prefix.
	assert.Equal(t, "INF ℹ️     > hello\n", buf.String())
}

func TestIndentPrefixesCycle(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetIndentPrefixes([]string{"A", "B"})

	d1 := l.With().Indent().Logger()
	d1.Info().Msg("one")
	line1 := buf.String()
	buf.Reset()

	d2 := l.With().Depth(2).Logger()
	d2.Info().Msg("two")
	line2 := buf.String()
	buf.Reset()

	d3 := l.With().Depth(3).Logger()
	d3.Info().Msg("three")
	line3 := buf.String()

	// depth 1 → symbols[0]="A ", depth 2 → symbols[1]="B ", depth 3 → symbols[0]="A " (wraps)
	assert.Equal(t, "INF ℹ️   A one\n", line1)
	assert.Equal(t, "INF ℹ️     B two\n", line2)
	assert.Equal(t, "INF ℹ️       A three\n", line3)
}

func TestIndentPrefixSeparator(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetIndentPrefixes([]string{"|"})
	l.SetIndentPrefixSeparator("--")

	sub := l.With().Indent().Logger()
	sub.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️   |--hello\n", buf.String())
}

func TestIndentPrefixSeparatorEmpty(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetIndentPrefixes([]string{"|"})
	l.SetIndentPrefixSeparator("")

	sub := l.With().Indent().Logger()
	sub.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️   |hello\n", buf.String())
}

func TestIndentPrefixesNil(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetIndentPrefixes([]string{"|"})
	l.SetIndentPrefixes(nil) // clear prefixes

	sub := l.With().Indent().Logger()
	sub.Info().Msg("hello")

	// nil clears prefixes, falls back to spaces only.
	assert.Equal(t, "INF ℹ️   hello\n", buf.String())
}

// ---------------------------------------------------------------------------
// Handler integration (Entry.Indent)
// ---------------------------------------------------------------------------

func TestIndentHandler(t *testing.T) {
	l := NewWriter(nil)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	sub := l.With().Depth(3).Logger()
	sub.Info().Msg("test")

	assert.Equal(t, 3, got.Indent)
	assert.Equal(t, "test", got.Message)
}

func TestIndentHandlerZero(t *testing.T) {
	l := NewWriter(nil)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	l.Info().Msg("test")

	assert.Equal(t, 0, got.Indent)
}

func TestIndentHandlerViaSetIndent(t *testing.T) {
	l := NewWriter(nil)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	l.SetIndent(5)
	l.Info().Msg("test")

	assert.Equal(t, 5, got.Indent)
	assert.Equal(t, "test", got.Message)
}

// ---------------------------------------------------------------------------
// Indent across log levels
// ---------------------------------------------------------------------------

func TestIndentAllLevels(t *testing.T) {
	tests := []struct {
		name    string
		method  func(*Logger) *Event
		wantLvl string
		prefix  string
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
			l.SetLevel(TraceLevel)

			sub := l.With().Indent().Logger()
			tt.method(sub).Msg("test")

			assert.Equal(t, tt.wantLvl+" "+tt.prefix+"   test\n", buf.String())
		})
	}
}

// ---------------------------------------------------------------------------
// Indent combined with fields
// ---------------------------------------------------------------------------

func TestIndentWithFields(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Indent().Logger()
	sub.Info().Str("key", "val").Msg("hello")

	// Indent appears before message, fields appear after.
	assert.Equal(t, "INF ℹ️   hello key=val\n", buf.String())
}

func TestIndentWithContextFields(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Indent().Str("ctx", "field").Logger()
	sub.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️   hello ctx=field\n", buf.String())
}

func TestIndentWithBothFieldTypes(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Indent().Str("ctx", "one").Logger()
	sub.Info().Str("event", "two").Msg("hello")

	assert.Equal(t, "INF ℹ️   hello ctx=one event=two\n", buf.String())
}

// ---------------------------------------------------------------------------
// Indent combined with custom prefix
// ---------------------------------------------------------------------------

func TestIndentWithCustomPrefix(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Indent().Prefix(">>>").Logger()
	sub.Info().Msg("hello")

	assert.Equal(t, "INF >>>   hello\n", buf.String())
}

func TestIndentWithEventPrefix(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Indent().Logger()
	sub.Info().Prefix(">>>").Msg("hello")

	assert.Equal(t, "INF >>>   hello\n", buf.String())
}

// ---------------------------------------------------------------------------
// Clone preservation — child must not be affected by parent mutations
// ---------------------------------------------------------------------------

func TestIndentClonePreservesSettings(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetIndentWidth(4)
	l.SetIndentPrefixes([]string{"|"})
	l.SetIndentPrefixSeparator("->")

	// Clone via With().Logger().
	child := l.With().Indent().Logger()

	// Mutate parent after cloning.
	l.SetIndentWidth(8)
	l.SetIndentPrefixes([]string{"X"})
	l.SetIndentPrefixSeparator("~~")

	child.Info().Msg("hello")

	// Child should use the original settings: width=4, prefix="|", sep="->".
	// 1 level × 4 spaces = "    " + "|" + "->" = "    |->"
	assert.Equal(t, "INF ℹ️     |->hello\n", buf.String())
}

func TestIndentCloneIndependentDepth(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Indent().Logger() // depth 1

	// Mutating parent's indent doesn't affect the child.
	l.SetIndent(10)

	sub.Info().Msg("hello")

	// Child stays at depth 1.
	assert.Equal(t, "INF ℹ️   hello\n", buf.String())
}

// ---------------------------------------------------------------------------
// computeIndent edge cases
// ---------------------------------------------------------------------------

func TestComputeIndentZeroDepth(t *testing.T) {
	assert.Empty(t, computeIndent(0, 2, nil, nil))
}

func TestComputeIndentNegativeDepth(t *testing.T) {
	assert.Empty(t, computeIndent(-1, 2, nil, nil))
}

func TestComputeIndentNoPrefixes(t *testing.T) {
	// 3 levels × 2 width = 6 spaces.
	assert.Equal(t, "      ", computeIndent(3, 2, nil, nil))
}

func TestComputeIndentWithPrefix(t *testing.T) {
	sep := " "
	// 2 levels × 2 width = 4 spaces, then prefix ">" + " ".
	assert.Equal(t, "    > ", computeIndent(2, 2, []string{">"}, &sep))
}

func TestComputeIndentPrefixCycling(t *testing.T) {
	prefixes := []string{"A", "B", "C"}
	// depth 1 → A, depth 2 → B, depth 3 → C, depth 4 → A (wraps).
	assert.Equal(t, "  A ", computeIndent(1, 2, prefixes, nil))
	assert.Equal(t, "    B ", computeIndent(2, 2, prefixes, nil))
	assert.Equal(t, "      C ", computeIndent(3, 2, prefixes, nil))
	assert.Equal(t, "        A ", computeIndent(4, 2, prefixes, nil))
}

func TestComputeIndentCustomSep(t *testing.T) {
	sep := "--"
	assert.Equal(t, "  |--", computeIndent(1, 2, []string{"|"}, &sep))
}

func TestComputeIndentEmptySep(t *testing.T) {
	sep := ""
	assert.Equal(t, "  |", computeIndent(1, 2, []string{"|"}, &sep))
}

func TestComputeIndentZeroWidth(t *testing.T) {
	// Zero-width indent: no leading spaces, just the prefix.
	assert.Equal(t, "> ", computeIndent(3, 0, []string{">"}, nil))
}

func TestComputeIndentLargeDepth(t *testing.T) {
	s := computeIndent(100, 2, nil, nil)
	assert.Len(t, s, 200)
	assert.Equal(t, strings.Repeat(" ", 200), s)
}

// ---------------------------------------------------------------------------
// Depth(0) should be a no-op
// ---------------------------------------------------------------------------

func TestDepthZeroNoOp(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Depth(0).Logger()
	sub.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️ hello\n", buf.String())
}

// ---------------------------------------------------------------------------
// Indent + Depth mixed on Context
// ---------------------------------------------------------------------------

func TestIndentAndDepthMixed(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Indent().Depth(2).Logger() // 1 + 2 = 3
	sub.Info().Msg("hello")

	// 3 levels × 2 spaces = 6 spaces.
	assert.Equal(t, "INF ℹ️       hello\n", buf.String())
}

// ---------------------------------------------------------------------------
// Animation indent (non-TTY path — TestOutput is non-TTY)
// ---------------------------------------------------------------------------

func TestSpinnerIndent(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))

	require.NoError(t, l.Spinner("loading").Indent().
		Wait(context.Background(), func(_ context.Context) error {
			return nil
		}).Msg("done"))

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	require.Len(t, lines, 2)

	// Non-TTY initial line: indent should be present in message.
	assert.Equal(t, "INF ⏳   loading\n", lines[0]+"\n")
	// Completion line: indentedLogger applies indent to the result.
	assert.Equal(t, "INF ℹ️   done\n", lines[1]+"\n")
}

func TestSpinnerDepth(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))

	require.NoError(t, l.Spinner("loading").Depth(2).
		Wait(context.Background(), func(_ context.Context) error {
			return nil
		}).Msg("done"))

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	require.Len(t, lines, 2)

	// 2 levels × 2 width = 4 spaces.
	assert.Equal(t, "INF ⏳     loading\n", lines[0]+"\n")
	assert.Equal(t, "INF ℹ️     done\n", lines[1]+"\n")
}

func TestSpinnerIndentOnIndentedLogger(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Indent().Logger() // logger at depth 1

	require.NoError(t, sub.Spinner("loading").Indent(). // animation adds 1 more
								Wait(context.Background(), func(_ context.Context) error {
			return nil
		}).Msg("done"))

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	require.Len(t, lines, 2)

	// Logger depth 1 + builder depth 1 = 2 total → 4 spaces.
	assert.Equal(t, "INF ⏳     loading\n", lines[0]+"\n")
	assert.Equal(t, "INF ℹ️     done\n", lines[1]+"\n")
}

func TestSpinnerIndentWithError(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))

	err := l.Spinner("loading").Indent().
		Wait(context.Background(), func(_ context.Context) error {
			return assert.AnError
		}).Err()

	require.ErrorIs(t, err, assert.AnError)

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	require.Len(t, lines, 2)

	// Error completion line should also be indented.
	assert.Equal(t, "INF ⏳   loading\n", lines[0]+"\n")
	assert.Equal(t, "ERR ❌   "+assert.AnError.Error()+"\n", lines[1]+"\n")
}

// ---------------------------------------------------------------------------
// Group task indent (non-TTY)
// ---------------------------------------------------------------------------

func TestGroupTaskIndent(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))

	g := l.Group(context.Background())
	r := g.Add(l.Spinner("task").Indent()).
		Run(func(_ context.Context) error {
			return nil
		})
	g.Wait()

	require.NoError(t, r.Msg("done"))

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	require.Len(t, lines, 2)

	// Non-TTY group initial line is indented.
	assert.Equal(t, "INF ⏳   task\n", lines[0]+"\n")
	// Completion message is indented via indentedLogger.
	assert.Equal(t, "INF ℹ️   done\n", lines[1]+"\n")
}

func TestGroupTaskDepth(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))

	g := l.Group(context.Background())
	r := g.Add(l.Spinner("task").Depth(2)).
		Run(func(_ context.Context) error {
			return nil
		})
	g.Wait()

	require.NoError(t, r.Msg("done"))

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	require.Len(t, lines, 2)

	assert.Equal(t, "INF ⏳     task\n", lines[0]+"\n")
	assert.Equal(t, "INF ℹ️     done\n", lines[1]+"\n")
}

func TestGroupTaskIndentOnIndentedLogger(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Indent().Logger()

	g := sub.Group(context.Background())
	r := g.Add(sub.Spinner("task").Indent()).
		Run(func(_ context.Context) error {
			return nil
		})
	g.Wait()

	require.NoError(t, r.Msg("done"))

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	require.Len(t, lines, 2)

	// Logger depth 1 + builder depth 1 = 2 → 4 spaces.
	assert.Equal(t, "INF ⏳     task\n", lines[0]+"\n")
	assert.Equal(t, "INF ℹ️     done\n", lines[1]+"\n")
}

func TestGroupTaskIndentWithError(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))

	g := l.Group(context.Background())
	r := g.Add(l.Spinner("task").Indent()).
		Run(func(_ context.Context) error {
			return assert.AnError
		})
	g.Wait()

	err := r.Err()
	require.ErrorIs(t, err, assert.AnError)

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	require.Len(t, lines, 2)

	assert.Equal(t, "INF ⏳   task\n", lines[0]+"\n")
	assert.Equal(t, "ERR ❌   "+assert.AnError.Error()+"\n", lines[1]+"\n")
}

// ---------------------------------------------------------------------------
// Animation indent with fields
// ---------------------------------------------------------------------------

func TestSpinnerIndentWithFields(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))

	require.NoError(t, l.Spinner("loading").Indent().Str("file", "main.go").
		Wait(context.Background(), func(_ context.Context) error {
			return nil
		}).Msg("done"))

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	require.Len(t, lines, 2)

	// Non-TTY initial line should have indent + message + field.
	assert.Equal(t, "INF ⏳   loading file=main.go\n", lines[0]+"\n")
	assert.Equal(t, "INF ℹ️   done file=main.go\n", lines[1]+"\n")
}

// ---------------------------------------------------------------------------
// Animation indent with prefixes
// ---------------------------------------------------------------------------

func TestSpinnerIndentWithPrefixes(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetIndentPrefixes([]string{"|"})

	require.NoError(t, l.Spinner("loading").Indent().
		Wait(context.Background(), func(_ context.Context) error {
			return nil
		}).Msg("done"))

	out := buf.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	require.Len(t, lines, 2)

	// indent (2 spaces) + prefix "|" + sep " " = "  | "
	assert.Equal(t, "INF ⏳   | loading\n", lines[0]+"\n")
	assert.Equal(t, "INF ℹ️   | done\n", lines[1]+"\n")
}

// ---------------------------------------------------------------------------
// Package-level indent functions (operate on Default logger)
// ---------------------------------------------------------------------------

func TestPackageLevelSetIndent(t *testing.T) {
	orig := Default
	defer func() { Default = orig }()

	var buf bytes.Buffer

	Default = New(TestOutput(&buf))
	SetIndent(1)
	Info().Msg("hello")

	assert.Equal(t, "INF ℹ️   hello\n", buf.String())
}

func TestPackageLevelSetIndentWidth(t *testing.T) {
	orig := Default
	defer func() { Default = orig }()

	var buf bytes.Buffer

	Default = New(TestOutput(&buf))
	SetIndentWidth(4)
	SetIndent(1)
	Info().Msg("hello")

	assert.Equal(t, "INF ℹ️     hello\n", buf.String())
}

func TestPackageLevelSetIndentPrefixes(t *testing.T) {
	orig := Default
	defer func() { Default = orig }()

	var buf bytes.Buffer

	Default = New(TestOutput(&buf))
	SetIndentPrefixes([]string{"|"})
	SetIndent(1)
	Info().Msg("hello")

	assert.Equal(t, "INF ℹ️   | hello\n", buf.String())
}

func TestPackageLevelSetIndentPrefixSeparator(t *testing.T) {
	orig := Default
	defer func() { Default = orig }()

	var buf bytes.Buffer

	Default = New(TestOutput(&buf))
	SetIndentPrefixes([]string{"|"})
	SetIndentPrefixSeparator("->")
	SetIndent(1)
	Info().Msg("hello")

	assert.Equal(t, "INF ℹ️   |->hello\n", buf.String())
}
