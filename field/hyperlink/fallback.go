package hyperlink

import (
	"fmt"
	"strings"
)

//go:generate go tool golang.org/x/tools/cmd/stringer -type=Fallback -linecomment

// Fallback selects how a hyperlink renders where OSC 8 sequences cannot be
// emitted - piped output, `NO_COLOR`, or [github.com/gechr/clog.ColorNever].
// It applies to links whose target is independent of their label ([OSC8] via
// a logger's Hyperlink, Link, Links, URL and URLs fields). Path fields are
// exempt: their label already carries the path, so they render it alone.
type Fallback int

const (
	// FallbackURL renders the URL alone, dropping the label. It is the
	// default because a label usually abbreviates its URL, so the URL is what
	// a piped log line needs to stay actionable.
	FallbackURL Fallback = iota // url
	// FallbackExpanded renders `label (url)`.
	FallbackExpanded // expanded
	// FallbackMarkdown renders `[label](url)`.
	FallbackMarkdown // markdown
	// FallbackText renders the label alone, dropping the URL.
	FallbackText // text
)

// ParseFallback resolves a fallback mode by name: "url", "expanded",
// "markdown" or "text". Matching ignores case and surrounding space.
func ParseFallback(name string) (Fallback, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "url":
		return FallbackURL, nil
	case "expanded":
		return FallbackExpanded, nil
	case "markdown":
		return FallbackMarkdown, nil
	case "text":
		return FallbackText, nil
	default:
		return FallbackURL, fmt.Errorf("clog: unknown hyperlink fallback %q", name)
	}
}

// Render renders url and text as plain text in this mode. An unknown mode
// renders as [FallbackURL].
func (f Fallback) Render(url, text string) string {
	switch f {
	case FallbackExpanded:
		return text + " (" + url + ")"
	case FallbackMarkdown:
		return "[" + text + "](" + url + ")"
	case FallbackText:
		return text
	case FallbackURL:
		return url
	default:
		return url
	}
}
