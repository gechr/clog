//go:build darwin || linux

package clog

import (
	"context"
	"errors"
	"io"
	"strings"

	"golang.org/x/sys/unix"
)

// hiddenReadLine reads one line from the terminal fd with echo disabled,
// honouring ctx. Unlike term.ReadPassword-in-a-goroutine, the CALLER is the
// only termios writer: echo is disabled before the reader goroutine starts
// and restored exactly once by this function, so a cancellation can never
// race the reader's own termios changes and leave the terminal echo-off.
// On cancellation the terminal input queue is flushed first, so a partially
// typed secret is not delivered to the shell after the prompt is gone.
//
// On cancellation the reader goroutine remains blocked on the fd until the
// process exits or input arrives; a post-cancel line is zeroed and dropped,
// never retained.
func hiddenReadLine(ctx context.Context, fd int, w io.Writer) (string, error) {
	old, err := unix.IoctlGetTermios(fd, ioctlReadTermios)
	if err != nil {
		return "", err
	}
	// Mirror term.ReadPassword's state: no echo, but keep canonical mode
	// (kernel line editing) and ISIG (Ctrl-C stays a signal).
	hidden := *old
	hidden.Lflag &^= unix.ECHO
	hidden.Lflag |= unix.ICANON | unix.ISIG
	hidden.Iflag |= unix.ICRNL
	if err := unix.IoctlSetTermios(fd, ioctlWriteTermios, &hidden); err != nil {
		return "", err
	}
	restore := func() error {
		return unix.IoctlSetTermios(fd, ioctlWriteTermios, old)
	}

	type result struct {
		line []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, rerr := readLineRaw(fd)
		if ctx.Err() != nil {
			// The prompt was already abandoned: zero the secret rather than
			// parking it in a channel nobody will read.
			clear(line)
			return
		}
		ch <- result{line: line, err: rerr}
	}()

	select {
	case <-ctx.Done():
		flushErr := flushTerminalInput(fd)
		restoreErr := restore()
		writeString(w, nl)
		// Join keeps errors.Is(err, ctx.Err()) working while still surfacing
		// a failed flush/restore instead of swallowing it.
		return "", errors.Join(ctx.Err(), flushErr, restoreErr)
	case r := <-ch:
		restoreErr := restore()
		writeString(w, nl)
		if r.err != nil {
			return "", r.err
		}
		if restoreErr != nil {
			return "", restoreErr
		}
		return strings.TrimRight(string(r.line), "\r\n"), nil
	}
}

// readLineRaw performs blocking reads on the fd until a newline arrives (in
// canonical mode the kernel delivers at most one line per read) or the
// stream ends. It never touches termios.
func readLineRaw(fd int) ([]byte, error) {
	// One canonical-mode line per read; generous for any PIN/passphrase.
	const readChunk = 256
	var line []byte
	buf := make([]byte, readChunk)
	for {
		n, err := unix.Read(fd, buf)
		if n > 0 {
			line = append(line, buf[:n]...)
			if line[len(line)-1] == '\n' {
				return line, nil
			}
		}
		switch {
		case errors.Is(err, unix.EINTR):
			continue
		case err != nil:
			return line, err
		case n == 0:
			// EOF: mirror term.ReadPassword, which returns what was read.
			return line, nil
		}
	}
}
