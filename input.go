package clog

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// inputConfig holds resolved input configuration.
type inputConfig struct {
	// sensitive suppresses echo of typed characters, for password-style input.
	sensitive bool
}

// InputOption configures how [Logger.Input] reads and displays input.
type InputOption func(*inputConfig)

// WithSensitive returns an [InputOption] that suppresses echo of typed
// characters, masking password-style input. [Logger.Password] applies this
// automatically.
func WithSensitive(v bool) InputOption {
	return func(c *inputConfig) {
		c.sensitive = v
	}
}

// inputSource pairs a reader with a [bufio.Reader] wrapping it, cached on the
// [Logger] and reused across [Logger.Input]/[Logger.Password] calls. Without
// this, a fresh bufio.Reader per call would read ahead into the underlying
// reader and silently discard whatever it buffered past the first line the
// moment the function returns - harmless against a live terminal (canonical
// mode only ever has one line ready at a time) but silently eats every
// prompt after the first against a pipe or file, where many lines are
// available in a single read. raw is kept alongside buf purely so
// [Logger.Password] can still type-assert it to *os.File for TTY/fd checks;
// buf never exposes the underlying reader.
type inputSource struct {
	raw io.Reader
	buf *bufio.Reader
}

func newInputSource(r io.Reader) *inputSource {
	return &inputSource{raw: r, buf: bufio.NewReader(r)}
}

// Input prompts on the logger's output and reads a line from the logger's
// input (see [Logger.SetInput]), defaulting to [os.Stdin]. Ctrl-C terminates
// the process as usual (the terminal's normal SIGINT handling is left
// untouched), so a prompt can never hang waiting for input that will never
// come.
//
// Input is not safe for concurrent use: simultaneous calls share the
// logger's buffered input reader, and log lines emitted from other
// goroutines while a prompt is pending may interleave with it. Prompt from
// one goroutine at a time.
//
//	name, err := clog.Input("Name: ")
func (l *Logger) Input(prompt string, opts ...InputOption) (string, error) {
	var cfg inputConfig
	for _, o := range opts {
		o(&cfg)
	}

	l.mu.Lock()
	w := l.output.Writer()
	if l.input == nil {
		l.input = newInputSource(os.Stdin)
	}
	src := l.input
	l.mu.Unlock()

	return readInput(src, w, prompt, cfg.sensitive)
}

// Password is [Logger.Input] with [WithSensitive] applied: on a terminal,
// typed characters are not echoed, via [term.ReadPassword]. Like
// [Logger.Input], it is not safe for concurrent use.
//
//	pass, err := clog.Password("Password: ")
func (l *Logger) Password(prompt string, opts ...InputOption) (string, error) {
	return l.Input(prompt, append(opts, WithSensitive(true))...)
}

// Input prompts and reads a line using the [Default] logger.
func Input(prompt string, opts ...InputOption) (string, error) {
	return Default.Input(prompt, opts...)
}

// Password prompts and reads a line without echo using the [Default] logger.
func Password(prompt string, opts ...InputOption) (string, error) {
	return Default.Password(prompt, opts...)
}

// readInput writes prompt to w, then reads a single line from src. Sensitive
// reads on a terminal defer entirely to [term.ReadPassword] rather than a
// bespoke raw-mode reader: it is the battle-tested primitive for this exact
// job, and - critically - it disables only local echo, leaving ICANON and
// ISIG (Ctrl-C, Ctrl-Z, ...) working exactly as the terminal normally would.
// A hand-rolled raw-mode loop would need to give up ISIG to read byte-by-byte,
// which trades a well-understood default for a self-implemented one. Every
// other case (plain prompts, and any non-TTY reader such as a pipe or a test
// double) uses a plain newline-delimited read: canonical mode already gives
// free line editing (backspace, Ctrl-U, Ctrl-W) at the kernel level, so there
// is nothing left for clog to reimplement.
func readInput(src *inputSource, w io.Writer, prompt string, sensitive bool) (string, error) {
	writeString(w, prompt)

	if sensitive {
		if f, ok := src.raw.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
			b, err := term.ReadPassword(int(f.Fd()))
			writeString(w, nl)
			if err != nil {
				return "", err
			}
			return string(b), nil
		}
	}

	return readLineCooked(src.buf)
}

// readLineCooked reads a single newline-delimited line from r.
func readLineCooked(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	line = strings.TrimRight(line, "\r\n")
	if err != nil {
		if errors.Is(err, io.EOF) && line != "" {
			return line, nil
		}
		return "", err
	}
	return line, nil
}
