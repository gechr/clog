package clog

import "fmt"

// Context builds a sub-logger with preset fields.
// Created by [Logger.With]. Finalise with [Context.Logger].
type Context struct {
	fieldBuilder[Context]

	logger *Logger
	prefix *string // nil = inherit from parent logger
}

// Logger returns a new [Logger] with the accumulated fields and prefix.
// The returned Logger shares the parent's mutex to prevent interleaved output.
func (c *Context) Logger() *Logger {
	c.logger.mu.Lock()
	defer c.logger.mu.Unlock()

	return &Logger{
		mu: c.logger.mu,

		exitFunc:        c.logger.exitFunc,
		fields:          c.fields,
		handler:         c.logger.handler,
		labels:          c.logger.labels,
		level:           c.logger.level,
		levelAlign:      c.logger.levelAlign,
		omitEmpty:       c.logger.omitEmpty,
		omitZero:        c.logger.omitZero,
		quoteMode:       c.logger.quoteMode,
		out:             c.logger.out,
		quoteOpen:       c.logger.quoteOpen,
		quoteClose:      c.logger.quoteClose,
		prefix:          c.prefix,
		prefixes:        c.logger.prefixes,
		reportTimestamp: c.logger.reportTimestamp,
		styles:          c.logger.styles,
		timeFormat:      c.logger.timeFormat,
		timeLocation:    c.logger.timeLocation,
		colorMode:       c.logger.colorMode,
		parts:           c.logger.parts,
	}
}

// Err adds an error field with key "error" to the context. No-op if err is nil.
func (c *Context) Err(err error) *Context {
	if err == nil {
		return c
	}

	c.fields = append(c.fields, Field{Key: ErrorKey, Value: err})
	return c
}

// Dict adds a group of fields under a key prefix using dot notation.
// Build the nested fields using [Dict] to create a field-only Event:
//
//	logger := clog.With().Dict("db", clog.Dict().
//	    Str("host", "localhost").
//	    Int("port", 5432),
//	).Logger()
func (c *Context) Dict(key string, dict *Event) *Context {
	for _, f := range dict.fields {
		c.fields = append(c.fields, Field{Key: key + "." + f.Key, Value: f.Value})
	}

	return c
}

// Path adds a file path field as a clickable terminal hyperlink.
// Respects the logger's [ColorMode] setting.
func (c *Context) Path(key, path string) *Context {
	c.fields = append(
		c.fields,
		Field{Key: key, Value: pathLinkWithMode(path, 0, 0, c.logger.colorMode)},
	)
	return c
}

// Line adds a file path field with a line number as a clickable terminal hyperlink.
// Respects the logger's [ColorMode] setting.
func (c *Context) Line(key, path string, lineNumber int) *Context {
	if lineNumber < 1 {
		lineNumber = 1
	}

	c.fields = append(
		c.fields,
		Field{Key: key, Value: pathLinkWithMode(path, lineNumber, 0, c.logger.colorMode)},
	)
	return c
}

// Column adds a file path field with a line and column number as a clickable terminal hyperlink.
// Respects the logger's [ColorMode] setting.
func (c *Context) Column(key, path string, lineNumber, column int) *Context {
	if lineNumber < 1 {
		lineNumber = 1
	}

	if column < 1 {
		column = 1
	}

	c.fields = append(
		c.fields,
		Field{Key: key, Value: pathLinkWithMode(path, lineNumber, column, c.logger.colorMode)},
	)
	return c
}

// Link adds a field as a clickable terminal hyperlink with custom URL and display text.
// Respects the logger's [ColorMode] setting.
func (c *Context) Link(key, url, text string) *Context {
	c.fields = append(
		c.fields,
		Field{Key: key, Value: hyperlinkWithMode(url, text, c.logger.colorMode)},
	)
	return c
}

// Stringer adds a field by calling the value's String method. No-op if val is nil.
func (c *Context) Stringer(key string, val fmt.Stringer) *Context {
	if val == nil {
		return c
	}

	c.fields = append(c.fields, Field{Key: key, Value: val.String()})
	return c
}

// Stringers adds a field with a slice of [fmt.Stringer] values.
func (c *Context) Stringers(key string, vals []fmt.Stringer) *Context {
	strs := make([]string, len(vals))
	for i, v := range vals {
		if v == nil {
			strs[i] = nilStr
		} else {
			strs[i] = v.String()
		}
	}

	c.fields = append(c.fields, Field{Key: key, Value: strs})
	return c
}

// Prefix sets a custom prefix for the sub-logger.
func (c *Context) Prefix(prefix string) *Context {
	c.prefix = new(prefix)
	return c
}
