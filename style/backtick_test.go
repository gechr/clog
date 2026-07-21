package style_test

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/style"
	"github.com/stretchr/testify/require"
)

func TestBacktickStripRender(t *testing.T) {
	base := new(lipgloss.NewStyle().Foreground(lipgloss.Color("4")))   // blue
	code := new(lipgloss.NewStyle().Foreground(lipgloss.Color("183"))) // lavender

	t.Run("styles code spans and strips delimiters", func(t *testing.T) {
		got := style.BacktickStrip.Render("see `inline` now", base, code)
		want := base.Render("see ") + code.Render("inline") + base.Render(" now")
		require.Equal(t, want, got)
	})

	t.Run("nil code leaves backticks intact, styled by base", func(t *testing.T) {
		got := style.BacktickStrip.Render("a `b` c", base, nil)
		require.Equal(t, base.Render("a `b` c"), got)
	})

	t.Run("no backticks renders whole with base", func(t *testing.T) {
		got := style.BacktickStrip.Render("plain text", base, code)
		require.Equal(t, base.Render("plain text"), got)
	})

	t.Run("nil base renders ordinary text plain", func(t *testing.T) {
		got := style.BacktickStrip.Render("plain `code`", nil, code)
		require.Equal(t, "plain "+code.Render("code"), got)
	})

	t.Run("unmatched trailing backtick is not a delimiter", func(t *testing.T) {
		got := style.BacktickStrip.Render("a `b", base, code)
		require.Equal(t, base.Render("a `b"), got)
	})
}

func TestBacktickModeRender(t *testing.T) {
	base := new(lipgloss.NewStyle().Foreground(lipgloss.Color("4")))   // blue
	code := new(lipgloss.NewStyle().Foreground(lipgloss.Color("183"))) // lavender

	t.Run("keep preserves visible width of pre-aligned content", func(t *testing.T) {
		msg := "open   `probe`   check" // grid-padded columns
		got := style.BacktickKeep.Render(msg, base, code)
		require.Equal(t, base.Render("open   ")+code.Render("`probe`")+base.Render("   check"), got)
		require.Equal(t, lipgloss.Width(msg), lipgloss.Width(got))
	})

	t.Run("strip shrinks each span by two columns", func(t *testing.T) {
		msg := "open   `probe`   check"
		got := style.BacktickStrip.Render(msg, base, code)
		require.Equal(t, lipgloss.Width(msg)-2, lipgloss.Width(got))
	})

	t.Run("unset renders like strip", func(t *testing.T) {
		msg := "see `inline` now"
		require.Equal(t,
			style.BacktickStrip.Render(msg, base, code),
			style.BacktickUnset.Render(msg, base, code),
		)
	})

	t.Run("lone unpaired backtick is untouched in both modes", func(t *testing.T) {
		msg := "a `b"
		require.Equal(t, base.Render(msg), style.BacktickStrip.Render(msg, base, code))
		require.Equal(t, base.Render(msg), style.BacktickKeep.Render(msg, base, code))
	})
}

func TestConfigMergeSegmentStyle(t *testing.T) {
	num := new(lipgloss.NewStyle().Bold(true))
	unit := new(lipgloss.NewStyle().Faint(true))
	c := &style.Config{FieldDuration: style.SegmentStyle{Number: num, Unit: unit}}

	// A partially-set override replaces only its own segment.
	override := new(lipgloss.NewStyle().Italic(true))
	c.Merge(&style.Config{FieldDuration: style.SegmentStyle{Number: override}})
	require.Equal(t, style.SegmentStyle{Number: override, Unit: unit}, c.FieldDuration)
}

func TestConfigMergeBacktickMode(t *testing.T) {
	c := &style.Config{BacktickMode: style.BacktickKeep}

	// An unset mode in the merged config keeps the current one.
	c.Merge(&style.Config{})
	require.Equal(t, style.BacktickKeep, c.BacktickMode)

	// An explicit strip overrides an inherited keep.
	c.Merge(&style.Config{BacktickMode: style.BacktickStrip})
	require.Equal(t, style.BacktickStrip, c.BacktickMode)
}
