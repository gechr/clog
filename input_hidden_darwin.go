package clog

import "golang.org/x/sys/unix"

// Terminal ioctls for reading/writing termios state on darwin (and the BSDs
// they descend from).
const (
	ioctlReadTermios  = unix.TIOCGETA
	ioctlWriteTermios = unix.TIOCSETA
)

// fread selects the read (input) queue for TIOCFLUSH; from <sys/file.h>
// (x/sys/unix does not export it).
const fread = 0x1

// flushTerminalInput discards typed-but-unread bytes from the terminal input
// queue.
func flushTerminalInput(fd int) error {
	return unix.IoctlSetPointerInt(fd, unix.TIOCFLUSH, fread)
}
