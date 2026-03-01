package clog

import "github.com/gechr/clog/field/hyperlink"

// Hyperlink wraps text in an OSC 8 terminal hyperlink escape sequence.
// Returns plain text when colours or hyperlinks are disabled globally.
func Hyperlink(url, text string) string {
	if !hyperlink.Enabled() || ColorsDisabled() {
		return text
	}
	return hyperlink.OSC8(url, text)
}

// PathLink creates a clickable terminal hyperlink for a file path.
// The line parameter is optional - pass 0 to omit line numbers.
func PathLink(path string, line int) string {
	display := hyperlink.PathDisplayText(path, line, 0)

	if !hyperlink.Enabled() || ColorsDisabled() {
		return display
	}
	return Hyperlink(hyperlink.ResolvePathURL(path, line, 0), display)
}

// Hyperlink wraps text in an OSC 8 terminal hyperlink, using the Output's colour settings.
// Satisfies [fx.Output].
func (o *Output) Hyperlink(url, text string) string {
	return o.hyperlink(url, text)
}

// PathLink creates a clickable terminal hyperlink for a file path, using the Output's colour settings.
// Satisfies [fx.Output].
func (o *Output) PathLink(path string, line, column int) string {
	return o.pathLink(path, line, column)
}

// hyperlink is like [Hyperlink] but uses the Output's colour settings.
func (o *Output) hyperlink(url, text string) string {
	if !hyperlink.Enabled() || o.ColorsDisabled() {
		return text
	}
	return hyperlink.OSC8(url, text)
}

// pathLink is like [PathLink] but uses the Output's colour settings.
func (o *Output) pathLink(path string, line, column int) string {
	display := hyperlink.PathDisplayText(path, line, column)

	if !hyperlink.Enabled() || o.ColorsDisabled() {
		return display
	}
	return hyperlink.OSC8(hyperlink.ResolvePathURL(path, line, column), display)
}
