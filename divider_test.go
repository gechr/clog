package clog

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDividerSend(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))

	l.Divider().Send()

	got := buf.String()
	assert.Equal(t, strings.Repeat("─", defaultDividerWidth)+"\n", got)
}

func TestDividerTitle(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))

	l.Divider().Msg("Build")

	got := buf.String()
	assert.Equal(
		t,
		"─── Build ──────────────────────────────────────────────────────────────────────\n",
		got,
	)
}

func TestDividerTitleAlignLeft(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))

	l.Divider().Align(AlignLeft).Msg("Test")

	got := strings.TrimRight(buf.String(), "\n")
	// Left-aligned: short leader on the left (dividerMinLeader = 3).
	leader := strings.Repeat("─", dividerMinLeader)
	assert.True(t, strings.HasPrefix(got, leader+" "),
		"expected left-aligned leader, got: %q", got)
}

func TestDividerTitleAlignRight(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))

	l.Divider().Align(AlignRight).Msg("Test")

	got := strings.TrimRight(buf.String(), "\n")
	// Right-aligned: short trailer on the right (dividerMinLeader = 3).
	trailer := " " + strings.Repeat("─", dividerMinLeader)
	assert.True(t, strings.HasSuffix(got, trailer),
		"expected right-aligned trailer, got: %q", got)
}

func TestDividerTitleAlignCenter(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))

	l.Divider().Align(AlignCenter).Msg("Mid")

	got := strings.TrimRight(buf.String(), "\n")
	// Centered: find the title position relative to total width.
	idx := strings.Index(got, " Mid ")
	assert.Greater(t, idx, dividerMinLeader,
		"expected centered title with substantial left leader, got: %q", got)
}

func TestDividerCustomChar(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))

	l.Divider().Char('═').Send()

	got := buf.String()
	assert.Equal(t, strings.Repeat("═", defaultDividerWidth)+"\n", got)
	assert.NotContains(t, got, "─")
}

func TestDividerCustomCharWithTitle(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))

	l.Divider().Char('═').Msg("Section")

	got := buf.String()
	assert.Equal(
		t,
		"═══ Section ════════════════════════════════════════════════════════════════════\n",
		got,
	)
}

func TestDividerWidthFallback(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))
	// TestOutput uses a bytes.Buffer which is non-TTY, Width() returns 0.

	l.Divider().Send()

	got := strings.TrimRight(buf.String(), "\n")
	// Count runes: should be defaultDividerWidth.
	assert.Len(t, []rune(got), defaultDividerWidth)
}

func TestDividerTitleLongerThanWidth(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))

	longTitle := strings.Repeat("X", defaultDividerWidth+10)
	l.Divider().Msg(longTitle)

	got := strings.TrimRight(buf.String(), "\n")
	// When the title is longer than width, just print the title.
	assert.Equal(t, longTitle, got)
}

func TestDividerTotalWidth(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))

	l.Divider().Msg("Hi")

	got := strings.TrimRight(buf.String(), "\n")
	// Total rune width should equal defaultDividerWidth.
	assert.Len(t, []rune(got), defaultDividerWidth)
}

func TestDividerCustomWidth(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))

	l.Divider().Width(40).Send()

	got := strings.TrimRight(buf.String(), "\n")
	assert.Len(t, []rune(got), 40)
	assert.Equal(t, strings.Repeat("─", 40), got)
}

func TestDividerCustomWidthWithTitle(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))

	l.Divider().Width(30).Msg("Hi")

	got := strings.TrimRight(buf.String(), "\n")
	assert.Len(t, []rune(got), 30)
	assert.Equal(t, "─── Hi ───────────────────────", got)
}

func TestDividerPackageLevel(t *testing.T) {
	var buf bytes.Buffer
	old := Default
	Default = New(TestOutput(&buf))

	defer func() { Default = old }()

	Divider().Send()

	got := buf.String()
	assert.Equal(t, strings.Repeat("─", defaultDividerWidth)+"\n", got)
}
