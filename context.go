package clog

import "sync/atomic"

// Context builds a sub-logger with preset fields.
// Created by [Logger.With]. Finalise with [Context.Logger].
type Context struct {
	fieldBuilder[Context]

	indent int // accumulated indent level
	logger *Logger
	symbol *string   // nil = inherit from parent logger
	tree   []TreePos // accumulated tree positions
}

// Dedent removes one indent level from the sub-logger, down to a minimum of
// zero. Useful for pulling a child logger back toward the root when a parent
// logger is already indented.
func (c *Context) Dedent() *Context {
	if c.indent > 0 {
		c.indent--
	}
	return c
}

// Depth adds multiple indent levels at once. Equivalent to calling
// [Context.Indent] n times.
func (c *Context) Depth(n int) *Context {
	c.indent += n
	return c
}

// Indent adds one indent level to the sub-logger. Chainable:
// With().Indent().Indent().Logger() produces two levels of indentation.
func (c *Context) Indent() *Context {
	c.indent++
	return c
}

// Logger returns a new [Logger] with the accumulated fields and symbol.
// The returned Logger shares the parent's mutex to prevent interleaved output.
func (c *Context) Logger() *Logger {
	c.logger.mu.Lock()
	defer c.logger.mu.Unlock()
	l := c.logger.clone()
	l.fields = c.Fields // override with context Fields
	l.indent = c.indent // override with accumulated indent
	l.symbol = c.symbol // override with context symbol
	l.tree = c.tree     // override with accumulated tree
	return l
}

// Symbol sets a custom symbol for the sub-logger.
func (c *Context) Symbol(symbol string) *Context {
	c.symbol = new(symbol)
	return c
}

// Tree adds one tree-nesting level with the given position. Each call
// deepens the tree by one level, drawing box-drawing connectors (├──, └──, │)
// automatically. Combine with [Context.Indent] to add space-based indent
// before the tree connectors.
func (c *Context) Tree(pos TreePos) *Context {
	c.tree = append(c.tree, pos)
	return c
}

// clone returns a copy of the Logger. The caller must hold l.mu. Every field
// is copied wholesale - new Logger fields are inherited by sub-loggers
// automatically - so only shared mutable state needs explicit handling below.
// The returned Logger shares the parent's mutex (its only caller,
// [Context.Logger], relies on this to prevent interleaved output).
func (l *Logger) clone() *Logger {
	c := *l

	// Fresh atomics: the sub-logger's level and field-format snapshot must be
	// settable independently of the parent.
	c.atomicLevel = &atomic.Int32{}
	c.atomicLevel.Store(l.atomicLevel.Load())
	c.fieldFormats = &atomic.Pointer[FieldFormats]{}
	c.fieldFormats.Store(l.fieldFormats.Load())

	// Deep-copy the slice the sub-logger appends to.
	c.tree = append([]TreePos{}, l.tree...)
	return &c
}
