package clog

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
	l := &Logger{
		mu: c.logger.mu,

		exitFunc:        c.logger.exitFunc,
		fieldStyleLevel: c.logger.fieldStyleLevel,
		fieldTimeFormat: c.logger.fieldTimeFormat,
		fields:          c.fields,
		handler:         c.logger.handler,
		labelWidth:      c.logger.labelWidth,
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
	l.atomicLevel.Store(
		int32(c.logger.level), //nolint:gosec // Level values are small constants (0-6)
	)
	return l
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

// URL adds a field as a clickable terminal hyperlink where the URL is also the display text.
// Respects the logger's [ColorMode] setting.
func (c *Context) URL(key, url string) *Context {
	c.fields = append(
		c.fields,
		Field{Key: key, Value: c.logger.output.hyperlink(url, url)},
	)
	return c
}
