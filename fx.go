package clog

import (
	"github.com/gechr/clog/fx"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/style"
)

// Convenience aliases so callers can reference common fx types without
// importing subpackages directly.
type (
	Update     = fx.Update
	TaskResult = fx.TaskResult
)

// fxLogger adapts *Logger to the fx.Logger interface. This allows the fx
// package types (Builder, WaitResult, Group, etc.) to call back into root
// clog's logging infrastructure without importing root clog.
type fxLogger struct{ l *Logger }

// Ensure fxLogger satisfies fx.Logger at compile time.
var _ fx.Logger = fxLogger{}

// Ensure *Output satisfies fx.RenderOutput (and fx.Output) at compile time.
var _ fx.RenderOutput = (*Output)(nil)

func (f fxLogger) Done(evt fx.DoneEvent) {
	e := f.l.newEvent(evt.Level)
	if e == nil {
		return
	}
	e = e.withFields(evt.Fields)
	if evt.Parts != nil {
		e = e.withParts(evt.Parts)
	}
	if evt.Symbol != nil {
		e = e.withSymbol(*evt.Symbol)
	}
	if evt.MsgStyle != nil {
		e = e.MessageStyle(evt.MsgStyle)
	}
	if evt.Err != nil {
		e.Err(evt.Err).Msg(evt.Msg)
	} else {
		e.Msg(evt.Msg)
	}
}

func (f fxLogger) WithIndent(depth int, tree []core.TreePos) fx.Logger {
	ctx := f.l.With()
	if depth > 0 {
		ctx = ctx.Depth(depth)
	}
	for _, pos := range tree {
		ctx = ctx.Tree(pos)
	}
	return fxLogger{ctx.Logger()}
}

func (f fxLogger) Output() fx.Output {
	return f.l.Output()
}

// TaskConfig locks the logger, snapshots every setting per-tick rendering
// needs, and wraps the root-only formatting and styling logic in closures so
// the fx render loops never touch the logger again.
func (f fxLogger) TaskConfig(b *fx.Builder) fx.TaskConfig {
	l := f.l
	l.mu.Lock()
	l.resolvePrintThemeLocked()
	level := b.LogLevel()
	order := l.parts
	if partOrder, ok := b.PartOrder(); ok {
		order = partOrder
	}
	combinedTree := l.tree
	if treePos := b.TreePositions(); len(treePos) > 0 {
		combinedTree = append(append([]TreePos{}, l.tree...), treePos...)
	}
	noColor := l.output.ColorsDisabled()
	styles := l.styles
	label := l.formatLabel(level)
	labels := l.allPaddedLabels()
	cfg := fx.TaskConfig{
		AnimationInterval: l.animationInterval,
		Indentation: computeIndent(
			l.indent+b.IndentLevel(),
			l.indentWidth,
			l.indentPrefixes,
			l.indentPrefixSep,
		) + computeTreeIndent(combinedTree, l.treeChars),
		IsTTY: l.output.IsTTY(),
		Label: label,
		NonTTYSilent: b.SuppressesNonTTY() ||
			(l.nonTTYLevel != UnsetLevel && level < l.nonTTYLevel),
		Order:           order,
		Out:             l.output.Writer(),
		Output:          l.output,
		ReportTimestamp: l.reportTimestamp,
		TimeFormat:      l.timeFormat,
		TimeLocation:    l.timeLocation,
	}
	fieldOpts := formatFieldsOpts{
		fieldSort:       l.fieldSort,
		fieldStyleLevel: l.fieldStyleLevel,
		formats:         l.loadFieldFormats(),
		level:           level,
		noColor:         noColor,
		quoteOpen:       l.quoteOpen,
		quoteClose:      l.quoteClose,
		quoteMode:       l.quoteMode,
		quoteSmart:      l.smartQuotePairs(),
		separatorText:   l.separatorText,
		sliceClose:      l.sliceClose,
		sliceOpen:       l.sliceOpen,
		sliceSep:        l.sliceSep,
		styles:          l.styles,
		timeFormat:      l.fieldTimeFormat,
	}
	l.mu.Unlock()

	// Styled level symbol for the builder's own level.
	if s := styles.Levels[level]; s != nil && !noColor {
		cfg.LevelSymbol = s.Render(label)
	} else {
		cfg.LevelSymbol = label
	}

	cfg.FormatFields = func(fields []core.Field) string {
		return formatFields(fields, fieldOpts)
	}
	cfg.StyleLevel = func(lvl core.Level) string {
		lab := labels[lvl]
		if s := styles.Levels[lvl]; s != nil && !noColor {
			return s.Render(lab)
		}
		return lab
	}
	msgStyleOverride := b.MessageStyleOverride()
	cfg.StyleMessage = func(msg string, lvl core.Level) string {
		if msgStyleOverride != nil {
			if noColor {
				return msg
			}
			return msgStyleOverride.Render(msg)
		}
		return styledMsg(msg, lvl, styles, noColor)
	}
	cfg.StyleSymbol = func(symbol string, lvl core.Level) string {
		return styledSymbol(symbol, lvl, styles, noColor)
	}
	cfg.StyleTimestamp = func(ts string) string {
		if styles.Timestamp != nil && !noColor {
			return styles.Timestamp.Render(ts)
		}
		return ts
	}
	return cfg
}

// styledMsg applies the message style for the given level, if any.
func styledMsg(msg string, level Level, styles *style.Config, noColor bool) string {
	if noColor {
		return msg
	}
	if s := styles.Messages[level]; s != nil {
		return s.Render(msg)
	}
	if styles.Message != nil {
		return styles.Message.Render(msg)
	}
	return msg
}

func styledSymbol(symbol string, level Level, styles *style.Config, noColor bool) string {
	if !noColor {
		if s := styles.Symbols[level]; s != nil {
			return s.Render(symbol)
		}
	}
	return symbol
}
