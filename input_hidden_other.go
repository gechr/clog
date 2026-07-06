//go:build !darwin && !linux

package clog

import (
	"context"
	"io"

	"golang.org/x/term"
)

// hiddenReadLine is the portable fallback: term.ReadPassword in a goroutine.
// Known limitation vs the unix implementation: the cancel-path restore can
// race the reader's own termios setup and leave echo disabled, and a
// partially typed secret is not flushed from the input queue.
func hiddenReadLine(ctx context.Context, fd int, w io.Writer) (string, error) {
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
