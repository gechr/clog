package clog

import "sync"

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

// Column adds a file path field with a line and column number as a clickable terminal hyperlink.
// Respects the logger's [ColorMode] setting.
func (c *Context) Column(key, path string, line, column int) *Context {
	if line < 1 {
		line = 1
	}

	if column < 1 {
		column = 1
	}

	c.Fields = append(
		c.Fields,
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
		c.Fields = append(c.Fields, Field{Key: key + "." + f.Key, Value: f.Value})
	}
	return c
}

// Line adds a file path field with a line number as a clickable terminal hyperlink.
// Respects the logger's [ColorMode] setting.
func (c *Context) Line(key, path string, line int) *Context {
	if line < 1 {
		line = 1
	}

	c.Fields = append(
		c.Fields,
		Field{Key: key, Value: c.logger.Output().pathLink(path, line, 0)},
	)
	return c
}

// Link adds a field as a clickable terminal hyperlink with custom URL and display text.
// Respects the logger's [ColorMode] setting.
func (c *Context) Link(key, url, text string) *Context {
	c.Fields = append(
		c.Fields,
		Field{Key: key, Value: c.logger.Output().hyperlink(url, text)},
	)
	return c
}

// Links adds a string slice field where each element is a hyperlink.
func (c *Context) Links(key string, links []Link) *Context {
	output := c.logger.Output()
	vals := make([]string, len(links))
	for i, l := range links {
		vals[i] = output.hyperlink(l.URL, l.Text)
	}
	c.Fields = append(c.Fields, Field{Key: key, Value: vals})
	return c
}

// Logger returns a new [Logger] with the accumulated fields and symbol.
// The returned Logger shares the parent's mutex to prevent interleaved output.
func (c *Context) Logger() *Logger {
	c.logger.mu.Lock()
	defer c.logger.mu.Unlock()
	l := c.logger.clone()
	l.mu = c.logger.mu  // share mutex
	l.fields = c.Fields // override with context Fields
	l.indent = c.indent // override with accumulated indent
	l.symbol = c.symbol // override with context symbol
	l.tree = c.tree     // override with accumulated tree
	l.atomicLevel.Store(
		int32(l.level), //nolint:gosec // Level values are small constants (-10 to 15)
	)
	return l
}

// Path adds a file path field as a clickable terminal hyperlink.
// Respects the logger's [ColorMode] setting.
func (c *Context) Path(key, path string) *Context {
	c.Fields = append(
		c.Fields,
		Field{Key: key, Value: c.logger.Output().pathLink(path, 0, 0)},
	)
	return c
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

// URL adds a field as a clickable terminal hyperlink where the URL is also the display text.
// Respects the logger's [ColorMode] setting.
func (c *Context) URL(key, url string) *Context {
	c.Fields = append(
		c.Fields,
		Field{Key: key, Value: c.logger.Output().hyperlink(url, url)},
	)
	return c
}

// URLs adds a string slice field where each element is a hyperlink
// with the URL as the display text.
func (c *Context) URLs(key string, urls []string) *Context {
	output := c.logger.Output()
	vals := make([]string, len(urls))
	for i, u := range urls {
		vals[i] = output.hyperlink(u, u)
	}
	c.Fields = append(c.Fields, Field{Key: key, Value: vals})
	return c
}

// clone returns a shallow copy of the Logger with all fields duplicated.
// The caller must hold l.mu. The returned Logger has its own mutex;
// callers that want to share the parent mutex should reassign l.mu after cloning.
func (l *Logger) clone() *Logger {
	return &Logger{
		mu: &sync.Mutex{}, // placeholder; callers typically override

		animationInterval:  l.animationInterval,
		exitFunc:           l.exitFunc,
		fields:             l.fields,
		fieldSort:          l.fieldSort,
		fieldStyleLevel:    l.fieldStyleLevel,
		fieldTimeFormat:    l.fieldTimeFormat,
		handler:            l.handler,
		hooks:              l.hooks,
		indent:             l.indent,
		indentPrefixes:     l.indentPrefixes,
		indentPrefixSep:    l.indentPrefixSep,
		indentWidth:        l.indentWidth,
		jsonIndent:         l.jsonIndent,
		jsonPrintMode:      l.jsonPrintMode,
		labels:             l.labels,
		labelsPadded:       l.labelsPadded,
		labelWidth:         l.labelWidth,
		level:              l.level,
		levelAlign:         l.levelAlign,
		nonTTYLevel:        l.nonTTYLevel,
		omitEmpty:          l.omitEmpty,
		omitZero:           l.omitZero,
		output:             l.output,
		parts:              l.parts,
		printIndent:        l.printIndent,
		quoteClose:         l.quoteClose,
		quoteMode:          l.quoteMode,
		quoteOpen:          l.quoteOpen,
		reportTimestamp:    l.reportTimestamp,
		separatorText:      l.separatorText,
		sliceClose:         l.sliceClose,
		sliceOpen:          l.sliceOpen,
		sliceSep:           l.sliceSep,
		styles:             l.styles,
		symbol:             l.symbol,
		symbols:            l.symbols,
		timeFormat:         l.timeFormat,
		timeLocation:       l.timeLocation,
		tree:               append([]TreePos{}, l.tree...),
		treeChars:          l.treeChars,
		yamlIndent:         l.yamlIndent,
		yamlIndentSequence: l.yamlIndentSequence,
	}
}
