package clog

import (
	"fmt"
	"reflect"
)

// Context builds a sub-logger with preset fields.
// Created by [Logger.With]. Finalise with [Context.Logger].
type Context struct {
	fieldBuilder[Context]

	logger *Logger
	prefix *string // nil = inherit from parent logger
}

// Column adds a file path field with a line and column number as a clickable terminal hyperlink.
// Respects the logger's [ColorMode] setting.
func (c *Context) Column(key, path string, line, column int) *Context {
	if line < 1 {
		line = 1
	}

	if column < 1 {
		column = 1
	}

	c.fields = append(
		c.fields,
		Field{Key: key, Value: c.logger.output.pathLink(path, line, column)},
	)
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
	if dict == nil {
		return c
	}

	for _, f := range dict.fields {
		c.fields = append(c.fields, Field{Key: key + "." + f.Key, Value: f.Value})
	}
	return c
}

// Err adds an error field with key "error" to the context. No-op if err is nil.
func (c *Context) Err(err error) *Context {
	if err == nil {
		return c
	}

	c.fields = append(c.fields, Field{Key: ErrorKey, Value: err})
	return c
}

// Line adds a file path field with a line number as a clickable terminal hyperlink.
// Respects the logger's [ColorMode] setting.
func (c *Context) Line(key, path string, line int) *Context {
	if line < 1 {
		line = 1
	}

	c.fields = append(
		c.fields,
		Field{Key: key, Value: c.logger.output.pathLink(path, line, 0)},
	)
	return c
}

// Link adds a field as a clickable terminal hyperlink with custom URL and display text.
// Respects the logger's [ColorMode] setting.
func (c *Context) Link(key, url, text string) *Context {
	c.fields = append(
		c.fields,
		Field{Key: key, Value: c.logger.output.hyperlink(url, text)},
	)
	return c
}

// Logger returns a new [Logger] with the accumulated fields and prefix.
// The returned Logger shares the parent's mutex to prevent interleaved output.
func (c *Context) Logger() *Logger {
	c.logger.mu.Lock()
	defer c.logger.mu.Unlock()
	return &Logger{
		mu: c.logger.mu,

		exitFunc:        c.logger.exitFunc,
		fieldStyleLevel: c.logger.fieldStyleLevel,
		fieldTimeFormat: c.logger.fieldTimeFormat,
		fields:          c.fields,
		handler:         c.logger.handler,
		labels:          c.logger.labels,
		level:           c.logger.level,
		levelAlign:      c.logger.levelAlign,
		omitEmpty:       c.logger.omitEmpty,
		omitZero:        c.logger.omitZero,
		output:          c.logger.output,
		parts:           c.logger.parts,
		prefix:          c.prefix,
		prefixes:        c.logger.prefixes,
		quoteClose:      c.logger.quoteClose,
		quoteMode:       c.logger.quoteMode,
		quoteOpen:       c.logger.quoteOpen,
		reportTimestamp: c.logger.reportTimestamp,
		styles:          c.logger.styles,
		timeFormat:      c.logger.timeFormat,
		timeLocation:    c.logger.timeLocation,
	}
}

// Path adds a file path field as a clickable terminal hyperlink.
// Respects the logger's [ColorMode] setting.
func (c *Context) Path(key, path string) *Context {
	c.fields = append(
		c.fields,
		Field{Key: key, Value: c.logger.output.pathLink(path, 0, 0)},
	)
	return c
}

// Prefix sets a custom prefix for the sub-logger.
func (c *Context) Prefix(prefix string) *Context {
	c.prefix = new(prefix)
	return c
}

// Stringer adds a field by calling the value's String method. No-op if val is nil.
func (c *Context) Stringer(key string, val fmt.Stringer) *Context {
	if val == nil {
		return c
	}

	// Detect typed nils (e.g. (*bytes.Buffer)(nil) passed as fmt.Stringer).
	rv := reflect.ValueOf(val)
	if rv.Kind() == reflect.Pointer && rv.IsNil() {
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
			strs[i] = Nil
		} else {
			strs[i] = v.String()
		}
	}

	c.fields = append(c.fields, Field{Key: key, Value: strs})
	return c
}

// URL adds a field as a clickable terminal hyperlink where the URL is also the display text.
// Respects the logger's [ColorMode] setting.
func (c *Context) URL(key, url string) *Context {
	c.fields = append(
		c.fields,
		Field{Key: key, Value: c.logger.output.hyperlink(url, url)},
	)
	return c
}
