package style_test

import (
	"testing"

	"github.com/gechr/clog/style"
	"github.com/gechr/clog/theme"
	"github.com/stretchr/testify/require"
)

func TestBacktickForBackground(t *testing.T) {
	require.Equal(t,
		"\x1b[38;2;240;171;252mcode\x1b[m",
		style.BacktickFor(theme.BackgroundDark).Render("code"),
	)
	require.Equal(t,
		"\x1b[38;2;162;28;175mcode\x1b[m",
		style.BacktickFor(theme.BackgroundLight).Render("code"),
	)
}
