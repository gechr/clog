package printer_test

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/printer"
	"github.com/stretchr/testify/assert"
)

func TestEmitStyled(t *testing.T) {
	t.Run("nil_style", func(t *testing.T) {
		var buf strings.Builder
		printer.EmitStyled(&buf, "text", nil)
		assert.Equal(t, "text", buf.String())
	})

	t.Run("with_style", func(t *testing.T) {
		st := lipgloss.NewStyle().Bold(true)
		var buf strings.Builder
		printer.EmitStyled(&buf, "text", &st)
		assert.Equal(t, st.Render("text"), buf.String())
	})

	t.Run("preserves_surrounding_whitespace", func(t *testing.T) {
		st := lipgloss.NewStyle().Bold(true)
		var buf strings.Builder
		printer.EmitStyled(&buf, "  hello\n", &st)
		assert.Equal(t, "  "+st.Render("hello")+"\n", buf.String())
	})

	t.Run("whitespace_only", func(t *testing.T) {
		st := lipgloss.NewStyle().Bold(true)
		var buf strings.Builder
		printer.EmitStyled(&buf, "  \n", &st)
		assert.Equal(t, "  \n", buf.String())
	})
}
