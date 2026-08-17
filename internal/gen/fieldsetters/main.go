// Command fieldsetters generates the per-knob [FieldFormats] convenience
// setters on Logger (fieldformats_setters.go) and their package-level
// Default-logger mirrors (fieldformats_setters_default.go) from the single
// spec table below. Adding a knob is one table entry; the two files stay in
// sync by construction.
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"log"
	"os"
	"strings"
)

// setter describes one per-knob convenience setter. The generated Logger
// method is named "Set"+name and mutates a copy of the current formats
// snapshot via mutateFieldFormats (or mutateHyperlinks when hyperlink is
// set, which also pushes the hyperlink subset to the output).
type setter struct {
	name      string // method name without the "Set" prefix
	params    string // parameter list of the setter
	args      string // argument list forwarded by the Default mirror
	doc       string // Logger method doc, starting after "Set<name> "
	mirrorDoc string // Default mirror doc, starting after "Set<name> "
	body      string // statements applied to the *FieldFormats copy
	hyperlink bool   // mutate via mutateHyperlinks instead of mutateFieldFormats
}

var setters = []setter{
	{
		name:   "DurationFormat",
		params: "format func(time.Duration) string",
		args:   "format",
		doc: `sets a custom formatter for [time.Duration] fields (also
used for elapsed fields when no elapsed formatter is set). nil restores the
built-in format.`,
		mirrorDoc: "sets a custom duration formatter on the [Default] logger.",
		body:      "f.DurationFormat = format",
	},
	{
		name:   "DurationGradientMax",
		params: "maximum time.Duration",
		args:   "maximum",
		doc: `sets the duration mapped to the end of the duration
gradient. 0 disables the gradient.`,
		mirrorDoc: "sets the duration gradient maximum on the [Default] logger.",
		body:      "f.DurationGradientMax = maximum",
	},
	{
		name:   "DurationMinimum",
		params: "minimum time.Duration",
		args:   "minimum",
		doc: `hides duration fields below this duration. 0 shows all
values and is the default.`,
		mirrorDoc: "sets the minimum duration shown on the [Default] logger.",
		body:      "f.DurationMinimum = minimum",
	},
	{
		name:   "DurationScale",
		params: "scale TimeScale",
		args:   "scale",
		doc: `sets the magnitude-keyed rounding and precision scale for
duration fields. nil inherits [FieldFormats.TimeScale]; a non-nil empty
scale disables rounding and decimal display. See [TimeScale].`,
		mirrorDoc: "sets the duration rounding/precision scale on the [Default] logger.",
		body:      "f.DurationScale = scale",
	},
	{
		name:   "ElapsedFormat",
		params: "format func(time.Duration) string",
		args:   "format",
		doc: `sets a custom formatter for elapsed-time fields (takes
priority over the duration formatter). nil restores the built-in format.`,
		mirrorDoc: "sets a custom elapsed formatter on the [Default] logger.",
		body:      "f.ElapsedFormat = format",
	},
	{
		name:   "ElapsedGradientMax",
		params: "maximum time.Duration",
		args:   "maximum",
		doc: `sets the duration mapped to the end of the elapsed
gradient. 0 disables the gradient.`,
		mirrorDoc: "sets the elapsed gradient maximum on the [Default] logger.",
		body:      "f.ElapsedGradientMax = maximum",
	},
	{
		name:   "ElapsedMinimum",
		params: "minimum time.Duration",
		args:   "minimum",
		doc: `hides elapsed fields below this duration. 0 shows all
values. Defaults to [time.Second].`,
		mirrorDoc: "sets the minimum elapsed duration shown on the [Default] logger.",
		body:      "f.ElapsedMinimum = minimum",
	},
	{
		name:   "ElapsedScale",
		params: "scale TimeScale",
		args:   "scale",
		doc: `sets the magnitude-keyed rounding and precision scale for
elapsed (and deadline) fields. nil inherits [FieldFormats.TimeScale]; a
non-nil empty scale disables rounding and decimal display. See [TimeScale].`,
		mirrorDoc: "sets the elapsed and deadline rounding/precision scale on the [Default] logger.",
		body:      "f.ElapsedScale = scale",
	},
	{
		name:   "TimeScale",
		params: "scale TimeScale",
		args:   "scale",
		doc: `sets the shared time scale and clears the duration and elapsed
overrides so all time fields inherit it. nil disables scale-based rounding
and decimal display for all time fields.`,
		mirrorDoc: "sets the shared rounding/precision scale for all time fields on the [Default] logger.",
		body: `f.TimeScale = scale
f.DurationScale = nil
f.ElapsedScale = nil`,
	},
	{
		name:   "TimeGradientMax",
		params: "maximum time.Duration",
		args:   "maximum",
		doc: `sets both [FieldFormats.DurationGradientMax] and
[FieldFormats.ElapsedGradientMax] to max in one call, since duration and
elapsed fields are usually given the same gradient ceiling. 0 disables both
gradients.`,
		mirrorDoc: "sets both the duration and elapsed gradient maxima on the [Default] logger.",
		body: `f.DurationGradientMax = maximum
f.ElapsedGradientMax = maximum`,
	},
	{
		name:      "HyperlinkEnabled",
		params:    "enabled bool",
		args:      "enabled",
		doc:       `enables or disables all hyperlink rendering.`,
		mirrorDoc: "enables or disables hyperlink rendering on the [Default] logger.",
		body:      "f.HyperlinkEnabled = enabled",
		hyperlink: true,
	},
	{
		name:   "HyperlinkFallback",
		params: "fallback hyperlink.Fallback",
		args:   "fallback",
		doc: `sets how links render where OSC 8 sequences cannot be
emitted - piped output, NO_COLOR, or [ColorNever]. Path fields are unaffected.`,
		mirrorDoc: "sets the hyperlink fallback mode on the [Default] logger.",
		body:      "f.HyperlinkFallback = fallback",
		hyperlink: true,
	},
	{
		name:   "HyperlinkColumnFormat",
		params: "format string",
		args:   "format",
		doc: `sets the URL format for file+line+column links.
Accepts a full format string or a preset name (e.g. "vscode").`,
		mirrorDoc: "sets the file+line+column hyperlink format on the [Default] logger.",
		body:      `f.HyperlinkColumnFormat = hyperlink.Expand(format, "column")`,
		hyperlink: true,
	},
	{
		name:   "HyperlinkDirFormat",
		params: "format string",
		args:   "format",
		doc: `sets the URL format for directory links.
Accepts a full format string or a preset name (e.g. "vscode").`,
		mirrorDoc: "sets the directory hyperlink format on the [Default] logger.",
		body:      `f.HyperlinkDirFormat = hyperlink.Expand(format, "path")`,
		hyperlink: true,
	},
	{
		name:   "HyperlinkFileFormat",
		params: "format string",
		args:   "format",
		doc: `sets the URL format for file-only links.
Accepts a full format string or a preset name (e.g. "vscode").`,
		mirrorDoc: "sets the file-only hyperlink format on the [Default] logger.",
		body:      `f.HyperlinkFileFormat = hyperlink.Expand(format, "path")`,
		hyperlink: true,
	},
	{
		name:   "HyperlinkLineFormat",
		params: "format string",
		args:   "format",
		doc: `sets the URL format for file+line links.
Accepts a full format string or a preset name (e.g. "vscode").`,
		mirrorDoc: "sets the file+line hyperlink format on the [Default] logger.",
		body:      `f.HyperlinkLineFormat = hyperlink.Expand(format, "line")`,
		hyperlink: true,
	},
	{
		name:   "HyperlinkPathFormat",
		params: "format string",
		args:   "format",
		doc: `sets the generic fallback URL format for any path.
Accepts a full format string or a preset name (e.g. "vscode").`,
		mirrorDoc: "sets the generic path hyperlink format on the [Default] logger.",
		body:      `f.HyperlinkPathFormat = hyperlink.Expand(format, "path")`,
		hyperlink: true,
	},
	{
		name:   "PercentFormat",
		params: "format func(float64) string",
		args:   "format",
		doc: `sets a custom formatter for percent fields; it receives the
display value already scaled to 0-100. nil restores the built-in format.`,
		mirrorDoc: "sets a custom percent formatter on the [Default] logger.",
		body:      "f.PercentFormat = format",
	},
	{
		name:   "PercentMaximum",
		params: "maximum float64",
		args:   "maximum",
		doc: `sets the input range maximum for percent values. 0 means
the default of 1.0 (fractions); set 100 for 0-100 input.`,
		mirrorDoc: "sets the percent input maximum on the [Default] logger.",
		body:      "f.PercentMaximum = maximum",
	},
	{
		name:   "PercentPrecision",
		params: "precision int",
		args:   "precision",
		doc: `sets the decimal precision for percent display
(0 = "75%", 1 = "75.0%").`,
		mirrorDoc: "sets the percent display precision on the [Default] logger.",
		body:      "f.PercentPrecision = precision",
	},
	{
		name:   "PercentReverseGradient",
		params: "reverse bool",
		args:   "reverse",
		doc: `flips the percent gradient direction (green at 0%,
red at 100%) for usage-style metrics.`,
		mirrorDoc: "flips the percent gradient direction on the [Default] logger.",
		body:      "f.PercentReverseGradient = reverse",
	},
	{
		name:   "QuantityUnitsIgnoreCase",
		params: "ignore bool",
		args:   "ignore",
		doc: `enables or disables case-insensitive quantity
unit matching.`,
		mirrorDoc: "toggles case-insensitive quantity units on the [Default] logger.",
		body:      "f.QuantityUnitsIgnoreCase = ignore",
	},
	{
		name:   "NumberFormat",
		params: "format NumberFormat",
		args:   "format",
		doc: `sets how integer fields and both halves of fraction fields
are rendered. It applies to fractions too unless [Logger.SetFractionFormat]
overrides them. Defaults to [NumberPlain].`,
		mirrorDoc: "sets the numeric format for integer and fraction fields on the [Default] logger.",
		body:      "f.NumberFormat = format",
	},
	{
		name:   "FractionFormat",
		params: "format NumberFormat",
		args:   "format",
		doc: `overrides the numeric format for fraction fields only.
When unset, fractions fall back to [Logger.SetNumberFormat].`,
		mirrorDoc: "overrides the numeric format for fraction fields on the [Default] logger.",
		body:      "f.FractionFormat = &format",
	},
	{
		name:   "NumberGroupSeparator",
		params: "sep string",
		args:   "sep",
		doc: `sets the separator inserted between digit groups for
[NumberGrouped] (e.g. "," for "1,234,567"). Defaults to ",".`,
		mirrorDoc: "sets the digit-group separator on the [Default] logger.",
		body:      "f.NumberGroupSeparator = sep",
	},
	{
		name:   "NumberCompactMinimum",
		params: "minimum int64",
		args:   "minimum",
		doc: `sets the smallest magnitude that [NumberCompact]
abbreviates; values below it render using the compact fallback (see
[Logger.SetNumberCompactFallback]). Defaults to 1000.`,
		mirrorDoc: "sets the minimum magnitude for compact abbreviation on the [Default] logger.",
		body:      "f.NumberCompactMinimum = minimum",
	},
	{
		name:   "NumberCompactFallback",
		params: "format NumberFormat",
		args:   "format",
		doc: `sets how [NumberCompact] renders values below the
compact minimum: [NumberGrouped] (the default, e.g. "9,999") or
[NumberPlain] (e.g. "9999").`,
		mirrorDoc: "sets how compact mode renders sub-minimum values on the [Default] logger.",
		body:      "f.NumberCompactFallback = format",
	},
}

const header = "// Code generated by internal/gen/fieldsetters; DO NOT EDIT.\n\npackage clog\n\n"

// docComment renders text as a doc comment for func Set<name>, prefixing the
// first line with the function name.
func docComment(name, text string) string {
	lines := strings.Split(text, "\n")
	var b strings.Builder
	fmt.Fprintf(&b, "// Set%s %s\n", name, lines[0])
	for _, line := range lines[1:] {
		fmt.Fprintf(&b, "// %s\n", line)
	}
	return b.String()
}

func genSetters() []byte {
	var b bytes.Buffer
	b.WriteString(header)
	b.WriteString("import (\n\t\"time\"\n\n\t\"github.com/gechr/clog/field/hyperlink\"\n)\n")

	for _, s := range setters {
		mutator := "mutateFieldFormats"
		if s.hyperlink {
			mutator = "mutateHyperlinks"
		}

		b.WriteString("\n")
		b.WriteString(docComment(s.name, s.doc))
		fmt.Fprintf(&b, "func (l *Logger) Set%s(%s) {\n", s.name, s.params)
		if lines := strings.Split(
			s.body,
			"\n",
		); len(lines) > 1 ||
			strings.Contains(s.body, "hyperlink.Expand") {
			fmt.Fprintf(&b, "\tl.%s(func(f *FieldFormats) {\n", mutator)
			for _, line := range lines {
				fmt.Fprintf(&b, "\t\t%s\n", line)
			}
			b.WriteString("\t})\n")
		} else {
			fmt.Fprintf(&b, "\tl.%s(func(f *FieldFormats) { %s })\n", mutator, s.body)
		}
		b.WriteString("}\n")
	}
	return b.Bytes()
}

func genMirrors() []byte {
	var b bytes.Buffer
	b.WriteString(header)
	b.WriteString("import (\n\t\"time\"\n\n\t\"github.com/gechr/clog/field/hyperlink\"\n)\n")

	for _, s := range setters {
		b.WriteString("\n")
		b.WriteString(docComment(s.name, s.mirrorDoc))
		fmt.Fprintf(
			&b,
			"func Set%s(%s) { Default().Set%s(%s) }\n",
			s.name,
			s.params,
			s.name,
			s.args,
		)
	}
	return b.Bytes()
}

func writeFile(name string, src []byte) {
	formatted, err := format.Source(src)
	if err != nil {
		log.Fatalf("format %s: %v", name, err)
	}
	if err := os.WriteFile(name, formatted, 0o600); err != nil {
		log.Fatalf("write %s: %v", name, err)
	}
}

func main() {
	writeFile("fieldformats_setters.go", genSetters())
	writeFile("fieldformats_setters_default.go", genMirrors())
}
