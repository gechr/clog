package style_test

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/style"
	"github.com/stretchr/testify/require"
)

func TestRenderBackticks(t *testing.T) {
	base := new(lipgloss.NewStyle().Foreground(lipgloss.Color("4")))   // blue
	code := new(lipgloss.NewStyle().Foreground(lipgloss.Color("183"))) // lavender

	t.Run("styles code spans and strips delimiters", func(t *testing.T) {
		got := style.RenderBackticks("see `inline` now", base, code)
		want := base.Render("see ") + code.Render("inline") + base.Render(" now")
		require.Equal(t, want, got)
	})

	t.Run("nil code leaves backticks intact, styled by base", func(t *testing.T) {
		got := style.RenderBackticks("a `b` c", base, nil)
		require.Equal(t, base.Render("a `b` c"), got)
	})

	t.Run("no backticks renders whole with base", func(t *testing.T) {
		got := style.RenderBackticks("plain text", base, code)
		require.Equal(t, base.Render("plain text"), got)
	})

	t.Run("nil base renders ordinary text plain", func(t *testing.T) {
		got := style.RenderBackticks("plain `code`", nil, code)
		require.Equal(t, "plain "+code.Render("code"), got)
	})

	t.Run("unmatched trailing backtick is not a delimiter", func(t *testing.T) {
		got := style.RenderBackticks("a `b", base, code)
		require.Equal(t, base.Render("a `b"), got)
	})
}
