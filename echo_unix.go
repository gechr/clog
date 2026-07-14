//go:build darwin || linux

package clog

import (
	"github.com/gechr/clog/internal/core"
	"golang.org/x/sys/unix"
)

// echoController implements [core.EchoController] by toggling the ECHO local
// flag on a terminal fd's termios. Only ECHO is touched - canonical mode,
// ISIG, and every other flag are left alone - so line editing, Ctrl-C, and
// job control keep working while typed characters simply stop painting.
//
// Both methods are idempotent and are only ever called under the live
// region's lock, so no synchronization of its own is needed. If echo was
// already off when Suppress ran (e.g. another termios owner disabled it),
// Restore leaves it off rather than forcing it back on.
type echoController struct {
	fd         int
	suppressed bool
	echoWasOn  bool
}

// newEchoController returns an [core.EchoController] for the terminal fd.
func newEchoController(fd int) core.EchoController {
	return &echoController{fd: fd}
}

func (e *echoController) Suppress() {
	if e.suppressed {
		return
	}
	termios, err := unix.IoctlGetTermios(e.fd, ioctlReadTermios)
	if err != nil {
		return
	}
	e.echoWasOn = termios.Lflag&unix.ECHO != 0
	if e.echoWasOn {
		termios.Lflag &^= unix.ECHO
		if unix.IoctlSetTermios(e.fd, ioctlWriteTermios, termios) != nil {
			return
		}
	}
	e.suppressed = true
}

func (e *echoController) Restore() {
	if !e.suppressed {
		return
	}
	e.suppressed = false
	if !e.echoWasOn {
		return
	}
	// Set the single bit on the current state rather than restoring a saved
	// termios wholesale, so changes made by other owners in the meantime
	// (e.g. a hidden-input read) are not stomped.
	termios, err := unix.IoctlGetTermios(e.fd, ioctlReadTermios)
	if err != nil {
		return
	}
	termios.Lflag |= unix.ECHO
	_ = unix.IoctlSetTermios(e.fd, ioctlWriteTermios, termios)
}
