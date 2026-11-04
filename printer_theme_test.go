package clog

import (
	"bytes"
	"testing"

	"github.com/gechr/clog/style"
	"github.com/gechr/clog/theme"
	"github.com/stretchr/testify/require"
)

func TestRawPrintersResolveThemeBeforeFirstWrite(t *testing.T) {
	printers := map[string]func(*Printer){
		"json": func(p *Printer) { p.RawJSON([]byte(`{"message":"hello"}`)) },
		"yaml": func(p *Printer) { p.RawYAML([]byte("message: hello")) },
		"toml": func(p *Printer) { p.RawTOML([]byte(`message = "hello"`)) },
		"hcl":  func(p *Printer) { p.RawHCL([]byte(`message = "hello"`)) },
	}
	for name, render := range printers {
		t.Run(name, func(t *testing.T) {
			for _, palette := range []*theme.Theme{theme.Light(), theme.Dark(), theme.Monokai()} {
				t.Run(palette.Name(), func(t *testing.T) {
					var got, want bytes.Buffer
					logger := New(NewOutput(&got, ColorAlways))
					logger.SetTheme(theme.Single(palette))
					render(logger.Print())

					// Explicit styles provide the expected rendering independently
					// of the lazy theme-resolution path being tested.
					expected := New(NewOutput(&want, ColorAlways))
					expected.SetStyles(&style.Config{
						JSON: style.NewJSON(palette), YAML: style.NewYAML(palette),
						TOML: style.NewTOML(palette), HCL: style.NewHCL(palette),
					})
					render(expected.Print())
					require.Equal(t, want.String(), got.String())
					require.False(t, logger.printThemeDirty)
				})
			}
		})
	}
}

func TestRawJSONDetectsLightBackground(t *testing.T) {
	var got, want bytes.Buffer
	output := NewOutput(&got, ColorAlways)
	output.bgDone, output.bgOK, output.bg = true, true, theme.BackgroundLight
	logger := New(output)
	logger.Print().RawJSON([]byte(`{"message":"hello","ok":true,"count":42}`))

	expected := New(NewOutput(&want, ColorAlways))
	expected.SetStyles(&style.Config{JSON: style.NewJSON(theme.Light())})
	expected.Print().RawJSON([]byte(`{"message":"hello","ok":true,"count":42}`))
	require.Equal(t, want.String(), got.String())
}

func TestRawJSONKeepsCustomStylesAndSkipsColorlessDetection(t *testing.T) {
	var got bytes.Buffer
	logger := New(NewOutput(&got, ColorAlways))
	logger.SetTheme(theme.Single(theme.Light()))
	custom := style.NewJSON(theme.Monokai())
	logger.SetStyles(&style.Config{JSON: custom})
	logger.Print().RawJSON([]byte(`{"message":"hello"}`))
	require.Equal(t, custom, logger.styles.JSON)

	got.Reset()
	logger = New(NewOutput(&got, ColorNever))
	logger.Print().Mode(JSONFlat).RawJSON([]byte(`{"message":"hello"}`))
	require.JSONEq(t, `{"message":"hello"}`, got.String())
	require.True(t, logger.printThemeDirty)
	require.False(t, logger.output.bgDone)
}
