//go:build !darwin && !linux

package clog

import "github.com/gechr/clog/internal/core"

// newEchoController returns nil on platforms without termios support: the
// live region treats a nil controller as "echo suppression unavailable".
func newEchoController(int) core.EchoController {
	return nil
}
