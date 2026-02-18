package clog

// IsTerminal returns true if the [Default] logger's output is connected to a terminal.
func IsTerminal() bool {
	Default.mu.Lock()
	defer Default.mu.Unlock()
	return Default.output.IsTTY()
}
