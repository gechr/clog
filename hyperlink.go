package clog

import "github.com/gechr/clog/field/hyperlink"

// Hyperlink wraps text in an OSC 8 terminal hyperlink escape sequence,
// using the [Default] logger's output and hyperlink configuration.
// Returns plain text when colors or hyperlinks are disabled.
func Hyperlink(url, text string) string {
	return Default.Output().Hyperlink(url, text)
}

// PathLink creates a clickable terminal hyperlink for a file path, using
// the [Default] logger's output and hyperlink configuration.
// The line parameter is optional - pass 0 to omit line numbers.
func PathLink(path string, line int) string {
	return Default.Output().PathLink(path, line, 0)
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

// hyperlink is like [Hyperlink] but uses the Output's color settings.
func (o *Output) hyperlink(url, text string) string {
	if !o.hyperlinkSettings().Enabled || o.ColorsDisabled() {
		return text
	}
	return hyperlink.OSC8(url, text)
}

// pathLink is like [PathLink] but uses the Output's color settings.
func (o *Output) pathLink(path string, line, column int) string {
	display := hyperlink.PathDisplayText(path, line, column)

	cfg := o.hyperlinkSettings()
	if !cfg.Enabled || o.ColorsDisabled() {
		return display
	}
	return hyperlink.OSC8(cfg.ResolvePathURL(path, line, column), display)
}
