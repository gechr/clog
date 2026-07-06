package clog

import (
	"bufio"
	"context"
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
	// fields, when non-nil, collects clog-styled key=value fields that are
	// rendered into the prompt after the message, followed by ": ".
	fields func(*Event)
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

// WithFields returns an [InputOption] that renders clog-styled key=value
// fields into the prompt. The prompt string becomes the message part and is
// followed by the fields and a ": " suffix, styled exactly like a log line's
// message and fields:
//
//	pass, err := clog.Password("Enter passphrase", clog.WithFields(func(e *clog.Event) {
//		e.Str("user", "alice").Int("attempts", 3)
//	}))
//	// Enter passphrase user=alice attempts=3: _
func WithFields(fn func(*Event)) InputOption {
	return func(c *inputConfig) {
		c.fields = fn
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
// come. When the process intercepts SIGINT instead of dying from it (e.g.
// [os/signal.NotifyContext]), use [Logger.InputContext] so the cancellation
// can abort the prompt.
//
// Input is not safe for concurrent use: simultaneous calls share the
// logger's buffered input reader, and log lines emitted from other
// goroutines while a prompt is pending may interleave with it. Prompt from
// one goroutine at a time.
//
//	name, err := clog.Input("Name: ")
func (l *Logger) Input(prompt string, opts ...InputOption) (string, error) {
	return l.InputContext(context.Background(), prompt, opts...)
}

// InputContext is [Logger.Input] with cancellation: when ctx is cancelled
// while the read is blocked, the terminal state is restored (a sensitive
// read disables echo), the prompt line is terminated, and ctx's error is
// returned. This matters for processes that intercept SIGINT (e.g.
// [os/signal.NotifyContext]): the terminal's Ctrl-C then cancels a context
// instead of killing the process, and a blocked read observes neither.
//
// On cancellation the reader goroutine remains blocked on the input until
// the process exits or the input produces data; callers must treat an
// aborted prompt as terminal for that input. In particular, do not close an
// *os.File input after an aborted prompt: for non-pollable files such as
// /dev/tty, Close blocks until the pending Read returns (golang/go#26593).
func (l *Logger) InputContext(
	ctx context.Context,
	prompt string,
	opts ...InputOption,
) (string, error) {
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

	if cfg.fields != nil {
		ev := l.Dict()
		cfg.fields(ev)
		prompt = l.renderPrompt(prompt, ev.fields)
	}

	return readInput(ctx, src, w, prompt, cfg.sensitive)
}

// Password is [Logger.Input] with [WithSensitive] applied: on a terminal,
// typed characters are not echoed, via [term.ReadPassword]. Like
// [Logger.Input], it is not safe for concurrent use.
//
//	pass, err := clog.Password("Password: ")
func (l *Logger) Password(prompt string, opts ...InputOption) (string, error) {
	return l.PasswordContext(context.Background(), prompt, opts...)
}

// PasswordContext is [Logger.Password] with cancellation; see
// [Logger.InputContext] for the semantics.
func (l *Logger) PasswordContext(
	ctx context.Context,
	prompt string,
	opts ...InputOption,
) (string, error) {
	return l.InputContext(ctx, prompt, append(opts, WithSensitive(true))...)
}

// Input prompts and reads a line using the [Default] logger.
func Input(prompt string, opts ...InputOption) (string, error) {
	return Default.Input(prompt, opts...)
}

// InputContext prompts and reads a line with cancellation using the
// [Default] logger.
func InputContext(ctx context.Context, prompt string, opts ...InputOption) (string, error) {
	return Default.InputContext(ctx, prompt, opts...)
}

// Password prompts and reads a line without echo using the [Default] logger.
func Password(prompt string, opts ...InputOption) (string, error) {
	return Default.Password(prompt, opts...)
}

// PasswordContext prompts and reads a line without echo, with cancellation,
// using the [Default] logger.
func PasswordContext(ctx context.Context, prompt string, opts ...InputOption) (string, error) {
	return Default.PasswordContext(ctx, prompt, opts...)
}

// renderPrompt renders a prompt line from a message and fields using the
// logger's styling - the same primitives as the log formatter's message and
// fields parts (backtick-aware message style, key=value field styling) -
// followed by a ": " suffix.
func (l *Logger) renderPrompt(msg string, fields []Field) string {
	l.mu.Lock()
	defer l.mu.Unlock()

	noColor := l.colorsDisabled()
	l.resolvePrintThemeLocked()

	var b strings.Builder
	if noColor || l.styles.Message == nil {
		b.WriteString(msg)
	} else {
		b.WriteString(l.styles.BacktickMode.Render(msg, l.styles.Message, l.styles.Backtick))
	}
	rendered := strings.TrimLeft(formatFields(fields, formatFieldsOpts{
		fieldSort:       l.fieldSort,
		fieldStyleLevel: l.fieldStyleLevel,
		formats:         l.loadFieldFormats(),
		level:           LevelInfo,
		noColor:         noColor,
		quoteOpen:       l.quoteOpen,
		quoteClose:      l.quoteClose,
		quoteMode:       l.quoteMode,
		quoteSmart:      l.smartQuotePairs(),
		separatorText:   l.separatorText,
		sliceClose:      l.sliceClose,
		sliceOpen:       l.sliceOpen,
		sliceSep:        l.sliceSep,
		styles:          l.styles,
		timeFormat:      l.fieldTimeFormat,
	}), " ")
	if rendered != "" {
		b.WriteString(" ")
		b.WriteString(rendered)
	}
	b.WriteString(": ")
	return b.String()
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
//
// A cancellable ctx runs the blocking read in a goroutine so ctx can abort
// the prompt; see [Logger.InputContext]. With context.Background() (the
// plain Input/Password path) the read stays on the calling goroutine.
func readInput(
	ctx context.Context,
	src *inputSource,
	w io.Writer,
	prompt string,
	sensitive bool,
) (string, error) {
	writeString(w, prompt)

	if sensitive {
		if f, ok := src.raw.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
			return readPassword(ctx, int(f.Fd()), w)
		}
	}

	if ctx.Done() == nil {
		return readLineCooked(src.buf)
	}
	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := readLineCooked(src.buf)
		ch <- result{line: line, err: err}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case r := <-ch:
		return r.line, r.err
	}
}

// readPassword reads without echo from the terminal fd, honouring ctx: on
// cancellation the pre-read terminal state is restored (ReadPassword only
// restores it when its read returns, which an aborted prompt never waits
// for) and the prompt line is terminated.
func readPassword(ctx context.Context, fd int, w io.Writer) (string, error) {
	if ctx.Done() == nil {
		b, err := term.ReadPassword(fd)
		writeString(w, nl)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	state, err := term.GetState(fd)
	if err != nil {
		return "", err
	}
	type result struct {
		b   []byte
		err error
	}
	ch := make(chan result, 1)
	go func() {
		b, rerr := term.ReadPassword(fd)
		ch <- result{b: b, err: rerr}
	}()
	select {
	case <-ctx.Done():
		restoreErr := term.Restore(fd, state)
		_ = restoreErr
		writeString(w, nl)
		return "", ctx.Err()
	case r := <-ch:
		writeString(w, nl)
		if r.err != nil {
			return "", r.err
		}
		return string(r.b), nil
	}
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
