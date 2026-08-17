package hyperlink_test

import (
	"testing"

	"github.com/gechr/clog/field/hyperlink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFallbackRender(t *testing.T) {
	tests := []struct {
		name     string
		fallback hyperlink.Fallback
		want     string
	}{
		{name: "url", fallback: hyperlink.FallbackURL, want: "https://example.com"},
		{
			name:     "expanded",
			fallback: hyperlink.FallbackExpanded,
			want:     "docs (https://example.com)",
		},
		{
			name:     "markdown",
			fallback: hyperlink.FallbackMarkdown,
			want:     "[docs](https://example.com)",
		},
		{name: "text", fallback: hyperlink.FallbackText, want: "docs"},
		{name: "zero_value_is_url", want: "https://example.com"},
		{
			name:     "unknown_defaults_to_url",
			fallback: hyperlink.Fallback(99),
			want:     "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.fallback.Render("https://example.com", "docs"))
		})
	}
}

func TestParseFallback(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  hyperlink.Fallback
	}{
		{name: "url", value: "url", want: hyperlink.FallbackURL},
		{name: "expanded", value: "expanded", want: hyperlink.FallbackExpanded},
		{name: "markdown", value: "markdown", want: hyperlink.FallbackMarkdown},
		{name: "text", value: "text", want: hyperlink.FallbackText},
		{
			name:  "case_and_space_insensitive",
			value: "  Markdown ",
			want:  hyperlink.FallbackMarkdown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hyperlink.ParseFallback(tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseFallbackUnknown(t *testing.T) {
	got, err := hyperlink.ParseFallback("plain")

	assert.Equal(t, `clog: unknown hyperlink fallback "plain"`, err.Error())
	assert.Equal(t, hyperlink.FallbackURL, got)
}

func TestFallbackString(t *testing.T) {
	assert.Equal(t, "url", hyperlink.FallbackURL.String())
	assert.Equal(t, "expanded", hyperlink.FallbackExpanded.String())
	assert.Equal(t, "markdown", hyperlink.FallbackMarkdown.String())
	assert.Equal(t, "text", hyperlink.FallbackText.String())
}
