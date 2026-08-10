package kloghandler

import (
	"flag"
	"strconv"
	"strings"
	"time"

	"github.com/gechr/clog"
	"github.com/go-logr/logr"
	"k8s.io/klog/v2"
)

// SetKlog routes k8s.io/klog/v2 output through the given clog [clog.Logger].
// Once set, klog stops writing to its own files and streams, so there is no
// need to touch its flags. The one exception is klog.Fatal, which writes
// goroutine stacks straight to stderr before exiting whatever logger is
// installed.
//
//	kloghandler.SetKlog(clog.Default(), nil)
//	klog.V(2).InfoS("reconciled", "pod", name)
//
// The logger is also registered as klog's contextual logger, so
// [klog.Background] and [klog.FromContext] return it - the path
// client-go and controller-runtime use.
//
// klog only forwards a severity for its structured calls. Its unstructured
// ones (Info, Warning, Error, Fatal) are formatted first and arrive as plain
// Info, so klog.Warning is logged at [clog.LevelInfo]. Set
// [Options.KlogSeverity] to recover the real severity by reading klog's own
// formatted line instead. That carries two costs. Unstructured verbose calls
// such as klog.V(4).Info arrive as Info, because klog stamps them with an Info
// header. And klog hands the formatted line straight to the callback without
// consulting [Sink.Enabled], so [Options.Verbosity] and [Options.LevelFor] do
// not apply to it - klog's own -v gate and the logger's level still do.
//
// Records mapped to [clog.LevelFatal] do not call os.Exit from clog's side;
// klog.Fatal terminates the process itself, as it always has.
//
// klog applies its own -v flag before a verbose call ever reaches the sink, so
// a trace-level clog logger alone will not surface klog.V(2) entries. Setting
// [Options.Verbosity] raises klog's -v to match, wiring both ends of the
// bridge to the same ceiling. Leave it nil to keep whatever -v the program
// already parsed.
//
// Modifying klog's logger is not thread-safe: call this during program
// initialization, before other goroutines log.
func SetKlog(logger *clog.Logger, opts *Options) {
	s := newSink(logger, opts)

	loggerOpts := []klog.LoggerOption{klog.ContextualLogger(true)}
	if s.opts.KlogSeverity {
		loggerOpts = append(loggerOpts, klog.WriteKlogBuffer(s.writeKlogBuffer))
	}

	klog.SetLoggerWithOptions(logr.New(s), loggerOpts...)

	flags := map[string]string{}
	if s.opts.Verbosity != nil {
		flags["v"] = strconv.Itoa(*s.opts.Verbosity)
	}
	if s.opts.KlogSeverity {
		// The severity lives in the header, so -skip_headers would not merely
		// lose it: a message that happens to be header-shaped would then be
		// parsed as one, and forge its own severity.
		flags["skip_headers"] = "false"
	}

	setKlogFlags(flags)
}

// setKlogFlags sets klog's own flags. klog binds them to its globals, so
// registering them on a throwaway [flag.FlagSet] is enough to reach those
// without disturbing the program's real command line.
func setKlogFlags(flags map[string]string) {
	if len(flags) == 0 {
		return
	}

	fs := flag.NewFlagSet("kloghandler", flag.ContinueOnError)
	klog.InitFlags(fs)
	for name, value := range flags {
		_ = fs.Set(name, value)
	}
}

// ClearKlog restores klog's own output, undoing [SetKlog].
func ClearKlog() {
	klog.ClearLogger()
}

// writeKlogBuffer receives a fully formatted klog line - header included - for
// each unstructured klog call, and re-emits it as a clog record with the
// severity and source that the header carries.
func (s *Sink) writeKlogBuffer(data []byte) {
	level, msg, source := parseKlogLine(string(data))

	fields := make([]clog.Field, 0, len(s.fields)+derivedFields)
	fields = s.appendPreset(fields)
	if s.opts.AddSource && source != "" {
		fields = append(fields, clog.Field{Key: SourceKey, Value: source})
	}

	s.logger.LogFields(level, time.Now(), msg, fields)
}

const (
	// klogHeaderLen is the width of the fixed part of a klog header, up to and
	// including the space after the thread id:
	//
	//	Lmmdd hh:mm:ss.uuuuuu threadid file:line] msg
	klogHeaderLen = 30

	// klogHeaderEnd terminates the "file:line" part of the header.
	klogHeaderEnd = "] "
)

// klogSeverities maps klog's leading severity character to a clog level.
var klogSeverities = map[byte]clog.Level{
	'I': clog.LevelInfo,
	'W': clog.LevelWarn,
	'E': clog.LevelError,
	'F': clog.LevelFatal,
}

// parseKlogLine splits a formatted klog line into its level, message and
// source location. Lines without a recognisable header are returned whole at
// [clog.LevelInfo]; [SetKlog] forces headers on so that fallback stays a
// safeguard rather than a routine path.
func parseKlogLine(line string) (clog.Level, string, string) {
	// klog itself strips exactly one trailing newline before handing a record
	// to a logger, so a message ending in a blank line keeps it.
	line = strings.TrimSuffix(line, "\n")

	level, ok := klogHeaderLevel(line)
	if !ok {
		return clog.LevelInfo, line, ""
	}

	rest := line[klogHeaderLen:]
	end := headerEnd(rest)
	if end < 0 {
		return level, rest, ""
	}

	return level, rest[end+len(klogHeaderEnd):], rest[:end]
}

// headerEnd locates the terminator that closes the header's "file:line" part,
// or -1 if there is none. klog writes the line number immediately before it, so
// only a terminator preceded by a colon-delimited number is genuine - a path
// containing "] " would otherwise cut the header short.
func headerEnd(rest string) int {
	for offset := 0; offset < len(rest); {
		i := strings.Index(rest[offset:], klogHeaderEnd)
		if i < 0 {
			return -1
		}

		end := offset + i
		if endsWithLineNumber(rest[:end]) {
			return end
		}
		offset = end + len(klogHeaderEnd)
	}

	return -1
}

// endsWithLineNumber reports whether s ends in ":" followed by digits.
func endsWithLineNumber(s string) bool {
	i := strings.LastIndex(s, ":")
	if i < 0 || i == len(s)-1 {
		return false
	}

	for _, r := range s[i+1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// klogHeaderLevel reports the level of a klog header, and whether line starts
// with one at all. Every byte of the fixed part has a known shape, so checking
// the punctuation, the digit runs and the severity character together tells a
// header from a message that merely resembles one.
func klogHeaderLevel(line string) (clog.Level, bool) {
	if len(line) <= klogHeaderLen {
		return clog.LevelInfo, false
	}

	shaped := line[5] == ' ' && line[8] == ':' && line[11] == ':' &&
		line[14] == '.' && line[21] == ' ' && line[29] == ' ' &&
		allDigits(line[1:5]) && allDigits(line[6:8]) && allDigits(line[9:11]) &&
		allDigits(line[12:14]) && allDigits(line[15:21]) && isPaddedNumber(line[22:29])
	if !shaped {
		return clog.LevelInfo, false
	}

	level, ok := klogSeverities[line[0]]
	return level, ok
}

// allDigits reports whether every byte of s is an ASCII digit.
func allDigits(s string) bool {
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return len(s) > 0
}

// isPaddedNumber reports whether s is leading spaces followed by at least one
// digit, the shape klog gives the thread id field.
func isPaddedNumber(s string) bool {
	digits := strings.TrimLeft(s, " ")
	return allDigits(digits)
}
