package clog

import "golang.org/x/sys/unix"

// Terminal ioctls for reading/writing termios state on linux.
const (
	ioctlReadTermios  = unix.TCGETS
	ioctlWriteTermios = unix.TCSETS
)

// flushTerminalInput discards typed-but-unread bytes from the terminal input
// queue.
func flushTerminalInput(fd int) error {
	return unix.IoctlSetInt(fd, unix.TCFLSH, unix.TCIFLUSH)
}
