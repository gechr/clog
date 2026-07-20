package clog

import "github.com/gechr/clog/field/hyperlink"

// Hyperlink wraps text in an OSC 8 terminal hyperlink escape sequence,
// using the [Default] logger's output and hyperlink configuration.
// Returns plain text when colors or hyperlinks are disabled.
func Hyperlink(url, text string) string {
	return Default().Output().Hyperlink(url, text)
}

// PathLink creates a clickable terminal hyperlink for a file path, using
// the [Default] logger's output and hyperlink configuration.
// The line parameter is optional - pass 0 to omit line numbers.
func PathLink(path string, line int) string {
	return Default().Output().PathLink(path, line, 0)
}

// PathLinkText is like [PathLink] but renders text as the visible link label
// instead of deriving it from path. The hyperlink still targets path, so a
// caller can show an abbreviated or home-contracted path (e.g. ~/bin/foo)
// while linking to its full location. The line parameter is optional - pass 0
// to omit it from the resolved URL.
func PathLinkText(text, path string, line int) string {
	return Default().Output().PathLinkText(text, path, line, 0)
}

// Hyperlink wraps text in an OSC 8 terminal hyperlink, using the Output's
// color settings and hyperlink configuration.
// Satisfies [fx.Output].
func (o *Output) Hyperlink(url, text string) string {
	return o.hyperlink(url, text)
}

// PathLink creates a clickable terminal hyperlink for a file path, using the
// Output's color settings and hyperlink configuration.
// Satisfies [fx.Output].
func (o *Output) PathLink(path string, line, column int) string {
	return o.pathLink(path, line, column)
}

// PathLinkText is like [PathLink] but renders text as the visible link label
// instead of deriving it from path, while still targeting path.
func (o *Output) PathLinkText(text, path string, line, column int) string {
	return o.pathLinkText(text, path, line, column)
}

// setHyperlinks stores the hyperlink rendering configuration for this output.
func (o *Output) setHyperlinks(c hyperlink.Config) {
	o.hyperlinks.Store(&c)
}

// hyperlinkSettings returns the output's hyperlink configuration, falling
// back to the enabled default when none has been pushed.
func (o *Output) hyperlinkSettings() hyperlink.Config {
	if c := o.hyperlinks.Load(); c != nil {
		return *c
	}
	return hyperlink.DefaultConfig()
}

// hyperlink is like [Hyperlink] but uses the Output's color settings. An empty
// url has no link target, so the text is returned plain - callers can pass an
// optional url without guarding it.
func (o *Output) hyperlink(url, text string) string {
	if url == "" || !o.hyperlinkSettings().Enabled || o.ColorsDisabled() {
		return text
	}
	return hyperlink.OSC8(url, text)
}

// pathLink is like [PathLink] but uses the Output's color settings.
func (o *Output) pathLink(path string, line, column int) string {
	return o.pathLinkText(hyperlink.PathDisplayText(path, line, column), path, line, column)
}

// pathLinkText is like [pathLink] but uses text as the visible label rather
// than deriving it from path.
func (o *Output) pathLinkText(text, path string, line, column int) string {
	cfg := o.hyperlinkSettings()
	if !cfg.Enabled || o.ColorsDisabled() {
		return text
	}
	return hyperlink.OSC8(cfg.ResolvePathURL(path, line, column), text)
}
