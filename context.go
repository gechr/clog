package clog

import "sync"

// Context builds a sub-logger with preset fields.
// Created by [Logger.With]. Finalise with [Context.Logger].
type Context struct {
	fieldBuilder[Context]

	logger *Logger
	prefix *string // nil = inherit from parent logger
}

// clone returns a shallow copy of the Logger with all fields duplicated.
// The caller must hold l.mu. The returned Logger has its own mutex;
// callers that want to share the parent mutex should reassign l.mu after cloning.
func (l *Logger) clone() *Logger {
	return &Logger{
		mu: &sync.Mutex{}, // placeholder; callers typically override

		elapsedFormatFunc:       l.elapsedFormatFunc,
		elapsedMinimum:          l.elapsedMinimum,
		elapsedPrecision:        l.elapsedPrecision,
		elapsedRound:            l.elapsedRound,
		exitFunc:                l.exitFunc,
		fieldSort:               l.fieldSort,
		fieldStyleLevel:         l.fieldStyleLevel,
		fieldTimeFormat:         l.fieldTimeFormat,
		fields:                  l.fields,
		handler:                 l.handler,
		labelWidth:              l.labelWidth,
		labels:                  l.labels,
		level:                   l.level,
		levelAlign:              l.levelAlign,
		omitEmpty:               l.omitEmpty,
		omitZero:                l.omitZero,
		output:                  l.output,
		labelsPadded:            l.labelsPadded,
		parts:                   l.parts,
		percentFormatFunc:       l.percentFormatFunc,
		percentPrecision:        l.percentPrecision,
		prefix:                  l.prefix,
		prefixes:                l.prefixes,
		quantityUnitsIgnoreCase: l.quantityUnitsIgnoreCase,
		quoteClose:              l.quoteClose,
		quoteMode:               l.quoteMode,
		quoteOpen:               l.quoteOpen,
		reportTimestamp:         l.reportTimestamp,
		separatorText:           l.separatorText,
		styles:                  l.styles,
		timeFormat:              l.timeFormat,
		timeLocation:            l.timeLocation,
	}
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
		Field{Key: key, Value: c.logger.Output().pathLink(path, line, column)},
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
		Field{Key: key, Value: c.logger.Output().pathLink(path, line, 0)},
	)
	return c
}

// Link adds a field as a clickable terminal hyperlink with custom URL and display text.
// Respects the logger's [ColorMode] setting.
func (c *Context) Link(key, url, text string) *Context {
	c.fields = append(
		c.fields,
		Field{Key: key, Value: c.logger.Output().hyperlink(url, text)},
	)
	return c
}

// Logger returns a new [Logger] with the accumulated fields and prefix.
// The returned Logger shares the parent's mutex to prevent interleaved output.
func (c *Context) Logger() *Logger {
	c.logger.mu.Lock()
	defer c.logger.mu.Unlock()
	l := c.logger.clone()
	l.mu = c.logger.mu                  // share mutex
	l.fields = c.fields                 // override with context fields
	l.prefix = c.prefix                 // override with context prefix
	l.atomicLevel.Store(int32(l.level)) //nolint:gosec // Level values are small constants (0-6)
	return l
}

// Path adds a file path field as a clickable terminal hyperlink.
// Respects the logger's [ColorMode] setting.
func (c *Context) Path(key, path string) *Context {
	c.fields = append(
		c.fields,
		Field{Key: key, Value: c.logger.Output().pathLink(path, 0, 0)},
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
		Field{Key: key, Value: c.logger.Output().hyperlink(url, url)},
	)
	return c
}
