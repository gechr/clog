package clog

import (
	"fmt"
	"os"
	"strings"

	"github.com/gechr/clog/field/hyperlink"
	"github.com/gechr/clog/internal/env"
	"github.com/gechr/clog/level"
	"github.com/gechr/clog/theme"
)

// DefaultEnvPrefix is the default environment variable prefix.
const DefaultEnvPrefix = "CLOG"

// Env var suffixes (appended to prefix + "_").
const (
	envLogLevel              = "LOG_LEVEL"
	envHyperlinkFormat       = "HYPERLINK_FORMAT"
	envHyperlinkPathFormat   = "HYPERLINK_PATH_FORMAT"
	envHyperlinkFileFormat   = "HYPERLINK_FILE_FORMAT"
	envHyperlinkDirFormat    = "HYPERLINK_DIR_FORMAT"
	envHyperlinkLineFormat   = "HYPERLINK_LINE_FORMAT"
	envHyperlinkColumnFormat = "HYPERLINK_COLUMN_FORMAT"
)

func init() {
	loadAllFromEnv()
}

func loadAllFromEnv() {
	loadNoColorFromEnv()
	loadLogLevelFromEnv()
	loadHyperlinkFormatsFromEnv()
	loadThemeFromEnv()
}

// SetEnvPrefix sets a custom environment variable prefix. Env vars are
// checked with the custom prefix first, then "CLOG" as fallback.
//
//	clog.SetEnvPrefix("MYAPP")
//	// Now checks MYAPP_LOG_LEVEL, then CLOG_LOG_LEVEL
//	// Now checks MYAPP_HYPERLINK_PATH_FORMAT, then CLOG_HYPERLINK_PATH_FORMAT
//	// etc.
func SetEnvPrefix(prefix string) {
	env.SetPrefix(prefix) // shared with the theme package
	loadAllFromEnv()
}

// getEnv reads an env var by suffix, checking custom prefix first, then CLOG.
func getEnv(suffix string) string {
	v, _ := env.Lookup(suffix)
	return v
}

func loadLogLevelFromEnv() {
	raw, envVar := env.Lookup(envLogLevel)
	lvl := strings.TrimSpace(raw)
	if lvl == "" {
		return
	}

	parsed, err := level.Parse(lvl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "clog: unrecognised log level %q in %s\n", lvl, envVar)
		return
	}

	Default().SetLevel(parsed)
	if parsed <= LevelDebug {
		Default().SetReportTimestamp(true)
	}
}

func loadHyperlinkFormatsFromEnv() {
	f := Default().FieldFormats()
	changed := false

	// HYPERLINK_FORMAT (preset) is applied first; individual format vars override it.
	if v, envVar := env.Lookup(envHyperlinkFormat); v != "" {
		if cfg, err := hyperlink.Preset(v); err != nil {
			fmt.Fprintf(os.Stderr, "clog: unrecognised hyperlink preset %q in %s\n", v, envVar)
		} else {
			f.HyperlinkPathFormat = cfg.PathFormat
			f.HyperlinkFileFormat = cfg.FileFormat
			f.HyperlinkDirFormat = cfg.DirFormat
			f.HyperlinkLineFormat = cfg.LineFormat
			f.HyperlinkColumnFormat = cfg.ColumnFormat
			changed = true
		}
	}

	for _, h := range []struct {
		suffix string
		dst    *string
	}{
		{envHyperlinkPathFormat, &f.HyperlinkPathFormat},
		{envHyperlinkFileFormat, &f.HyperlinkFileFormat},
		{envHyperlinkDirFormat, &f.HyperlinkDirFormat},
		{envHyperlinkLineFormat, &f.HyperlinkLineFormat},
		{envHyperlinkColumnFormat, &f.HyperlinkColumnFormat},
	} {
		if v := getEnv(h.suffix); v != "" {
			*h.dst = v
			changed = true
		}
	}

	if changed {
		Default().SetFieldFormats(f)
	}
}

// loadThemeFromEnv applies printer-theme configuration to the [Default] logger.
//
//	<PREFIX>_THEME              an explicit theme (e.g. "monokai") applied
//	                            on both backgrounds via [theme.Single].
//	<PREFIX>_THEME_LIGHT/_DARK  a light/dark pair selected by the terminal
//	                            background on first write.
//
// When no theme variables are set the existing theme is left untouched.
func loadThemeFromEnv() {
	pair, err := theme.FromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "clog: %v\n", err)
		return
	}
	if pair != nil {
		Default().SetTheme(pair)
	}
}

func loadNoColorFromEnv() {
	// Check NO_COLOR per https://no-color.org/ -> presence of the variable
	// (regardless of value, including empty) disables colors.
	_, set := os.LookupEnv("NO_COLOR")
	noColorEnvSet.Store(set)
}
