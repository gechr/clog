package clog

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	"github.com/gechr/clog/field/hyperlink"
	"github.com/gechr/clog/level"
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

var envPrefix atomic.Value // stores string; "" means no custom prefix

func init() {
	hyperlink.SetEnabled(true)
	loadAllFromEnv()
}

func loadAllFromEnv() {
	loadNoColorFromEnv()
	loadLogLevelFromEnv()
	loadHyperlinkFormatsFromEnv()
}

// SetEnvPrefix sets a custom environment variable prefix. Env vars are
// checked with the custom prefix first, then "CLOG" as fallback.
//
//	clog.SetEnvPrefix("MYAPP")
//	// Now checks MYAPP_LOG_LEVEL, then CLOG_LOG_LEVEL
//	// Now checks MYAPP_HYPERLINK_PATH_FORMAT, then CLOG_HYPERLINK_PATH_FORMAT
//	// etc.
func SetEnvPrefix(prefix string) {
	envPrefix.Store(strings.TrimRight(prefix, "_"))
	loadAllFromEnv()
}

// getEnv reads an env var by suffix, checking custom prefix first, then CLOG.
func getEnv(suffix string) string {
	if p, ok := envPrefix.Load().(string); ok && p != "" {
		if v := os.Getenv(p + "_" + suffix); v != "" {
			return v
		}
	}
	return os.Getenv(DefaultEnvPrefix + "_" + suffix)
}

func loadLogLevelFromEnv() {
	lvl := strings.TrimSpace(getEnv(envLogLevel))
	if lvl == "" {
		return
	}

	parsed, err := level.Parse(lvl)
	if err != nil {
		envVar := DefaultEnvPrefix + "_" + envLogLevel
		if p, ok := envPrefix.Load().(string); ok && p != "" {
			envVar = p + "_" + envLogLevel
		}
		fmt.Fprintf(os.Stderr, "clog: unrecognised log level %q in %s\n", lvl, envVar)
		return
	}

	Default.SetLevel(parsed)
	if parsed <= LevelDebug {
		Default.SetReportTimestamp(true)
	}
}

func loadHyperlinkFormatsFromEnv() {
	// HYPERLINK_FORMAT (preset) is applied first; individual format vars override it.
	if v := getEnv(envHyperlinkFormat); v != "" {
		if err := hyperlink.SetPreset(v); err != nil {
			envVar := DefaultEnvPrefix + "_" + envHyperlinkFormat
			if p, ok := envPrefix.Load().(string); ok && p != "" {
				envVar = p + "_" + envHyperlinkFormat
			}
			fmt.Fprintf(os.Stderr, "clog: unrecognised hyperlink preset %q in %s\n", v, envVar)
		}
	}

	if v := getEnv(envHyperlinkPathFormat); v != "" {
		hyperlink.SetPathFormat(v)
	}

	if v := getEnv(envHyperlinkFileFormat); v != "" {
		hyperlink.SetFileFormat(v)
	}

	if v := getEnv(envHyperlinkDirFormat); v != "" {
		hyperlink.SetDirFormat(v)
	}

	if v := getEnv(envHyperlinkLineFormat); v != "" {
		hyperlink.SetLineFormat(v)
	}

	if v := getEnv(envHyperlinkColumnFormat); v != "" {
		hyperlink.SetColumnFormat(v)
	}
}

func loadNoColorFromEnv() {
	// Check NO_COLOR per https://no-color.org/ -> presence of the variable
	// (regardless of value, including empty) disables colours.
	_, set := os.LookupEnv("NO_COLOR")
	noColorEnvSet.Store(set)
}
