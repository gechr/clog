package clog

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
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
	hyperlinksEnabled.Store(true)
	loadAllFromEnv()
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

func loadAllFromEnv() {
	loadNoColorFromEnv()
	loadLogLevelFromEnv()
	loadHyperlinkFormatsFromEnv()
}

func loadLogLevelFromEnv() {
	level := strings.TrimSpace(getEnv(envLogLevel))
	if level == "" {
		return
	}

	switch strings.ToLower(level) {
	case LevelTrace:
		Default.SetLevel(TraceLevel)
		Default.SetReportTimestamp(true)
	case LevelDebug:
		Default.SetLevel(DebugLevel)
		Default.SetReportTimestamp(true)
	case LevelInfo:
		Default.SetLevel(InfoLevel)
	case LevelDry:
		Default.SetLevel(DryLevel)
	case LevelWarn, "warning":
		Default.SetLevel(WarnLevel)
	case LevelError:
		Default.SetLevel(ErrorLevel)
	case LevelFatal, "critical":
		Default.SetLevel(FatalLevel)
	default:
		// Build the env var name for the error message.
		envVar := DefaultEnvPrefix + "_" + envLogLevel
		if p, ok := envPrefix.Load().(string); ok && p != "" {
			envVar = p + "_" + envLogLevel
		}
		fmt.Fprintf(os.Stderr, "clog: unrecognised log level %q in %s\n", level, envVar)
	}
}

func loadHyperlinkFormatsFromEnv() {
	// HYPERLINK_FORMAT (preset) is applied first; individual format vars override it.
	if v := getEnv(envHyperlinkFormat); v != "" {
		if err := SetHyperlinkPreset(v); err != nil {
			envVar := DefaultEnvPrefix + "_" + envHyperlinkFormat
			if p, ok := envPrefix.Load().(string); ok && p != "" {
				envVar = p + "_" + envHyperlinkFormat
			}
			fmt.Fprintf(os.Stderr, "clog: unrecognised hyperlink preset %q in %s\n", v, envVar)
		}
	}

	if v := getEnv(envHyperlinkPathFormat); v != "" {
		SetHyperlinkPathFormat(v)
	}

	if v := getEnv(envHyperlinkFileFormat); v != "" {
		SetHyperlinkFileFormat(v)
	}

	if v := getEnv(envHyperlinkDirFormat); v != "" {
		SetHyperlinkDirFormat(v)
	}

	if v := getEnv(envHyperlinkLineFormat); v != "" {
		SetHyperlinkLineFormat(v)
	}

	if v := getEnv(envHyperlinkColumnFormat); v != "" {
		SetHyperlinkColumnFormat(v)
	}
}

func loadNoColorFromEnv() {
	// Check NO_COLOR per https://no-color.org/ -> presence of the variable
	// (regardless of value, including empty) disables colours.
	_, set := os.LookupEnv("NO_COLOR")
	noColorEnvSet.Store(set)
}
