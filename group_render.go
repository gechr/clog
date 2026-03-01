package clog

import (
	"context"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gechr/clog/fx"
	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/fx/pulse"
	"github.com/gechr/clog/fx/shimmer"
	"github.com/gechr/clog/fx/spinner"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/style"
	"github.com/muesli/termenv"
)

// taskConfig is an immutable snapshot of logger settings captured under the
// logger's mutex. It stores exactly the fields needed for per-tick rendering
// so the animation loop never touches the logger after the initial capture.
type taskConfig struct {
	indentation  string    // pre-computed indent string before message
	isTTY        bool      // output.IsTTY()
	label        string    // pre-computed padded label
	levelSymbol  string    // styled label (via styles.Levels[level])
	noColor      bool      // output.ColorsDisabled()
	nonTTYSilent bool      // builder.SuppressNonTTY || level < logger.nonTTYLevel
	order        []Part    // l.parts
	out          io.Writer // output.Writer()
	output       *Output   // for Width() in bar mode
	reportTS     bool
	styles       *style.Config
	termOut      *termenv.Output // output.Renderer().Output()
	timeFmt      string
	timeLoc      *time.Location
}

// groupTask holds per-animation mutable state for both the single-animation
// (runAnimation) and multi-animation (Group) paths. It embeds *fx.GroupTask
// for shared state and adds rendering-specific fields.
type groupTask struct {
	*fx.GroupTask

	cfg      taskConfig
	symbol   string // resolved icon (builder.Symbol or "⏳")
	tickRate time.Duration

	// per-tick mutable state
	cachedFieldsPtr *[]core.Field    // dedup: last-formatted fields pointer
	cachedFieldsStr string           // dedup: last-formatted fields string
	fieldOpts       formatFieldsOpts // pre-built from taskConfig
	hexLUT          *shimmer.LUT     // shimmer only, immutable after init
	pCache          pulse.Cache
	styleLUT        *shimmer.StyleLUT // shimmer only, immutable after init

	// gradient cache (bar mode with ProgressGradient only)
	gradientProgress float64
	gradientStyle    lipgloss.Style
	gradientValid    bool
}

// captureTaskConfig locks the builder's logger, snapshots all fields into
// s.cfg, and pre-computes s.tickRate, s.symbol, s.fieldOpts, s.cfg.levelSymbol,
// and shimmer LUTs.
func captureTaskConfig(gt *groupTask) {
	b := gt.Builder
	l := b.Log.(fxLogger).l //nolint:errcheck,forcetypeassert // fxLogger is the only Logger impl
	l.mu.Lock()
	animInterval := l.animationInterval
	order := l.parts
	if b.PartOverrides != nil {
		order = *b.PartOverrides
	}
	combinedTree := l.tree
	if len(b.TreePos) > 0 {
		combinedTree = append(append([]TreePos{}, l.tree...), b.TreePos...)
	}
	gt.cfg = taskConfig{
		indentation: computeIndent(
			l.indent+b.IndentDepth,
			l.indentWidth,
			l.indentPrefixes,
			l.indentPrefixSep,
		) + computeTreeIndent(combinedTree, l.treeChars),
		isTTY:        l.output.IsTTY(),
		label:        l.formatLabel(b.Level),
		noColor:      l.output.ColorsDisabled(),
		nonTTYSilent: b.SuppressNonTTY || (l.nonTTYLevel != UnsetLevel && b.Level < l.nonTTYLevel),
		order:        order,
		out:          l.output.Writer(),
		output:       l.output,
		reportTS:     l.reportTimestamp,
		styles:       l.styles,
		termOut:      l.output.Renderer().Output(),
		timeFmt:      l.timeFormat,
		timeLoc:      l.timeLocation,
	}
	gt.fieldOpts = formatFieldsOpts{
		fieldSort:       l.fieldSort,
		fieldStyleLevel: l.fieldStyleLevel,
		level:           b.Level,
		noColor:         l.output.ColorsDisabled(),
		quoteOpen:       l.quoteOpen,
		quoteClose:      l.quoteClose,
		quoteMode:       l.quoteMode,
		separatorText:   l.separatorText,
		styles:          l.styles,
		timeFormat:      l.fieldTimeFormat,
	}
	l.mu.Unlock()

	// Styled level symbol.
	if style := gt.cfg.styles.Levels[b.Level]; style != nil && !gt.cfg.noColor {
		gt.cfg.levelSymbol = style.Render(gt.cfg.label)
	} else {
		gt.cfg.levelSymbol = gt.cfg.label
	}

	// Resolve the symbol icon.
	gt.symbol = b.SymbolIcon
	if gt.symbol == "" {
		gt.symbol = "⏳"
	}

	// Determine tick rate and pre-compute mode-specific resources.
	switch b.Mode {
	case fx.AnimationSpinner:
		gt.tickRate = b.SpinnerStyle.Interval
	case fx.AnimationPulse:
		gt.tickRate = pulse.TickRate
	case fx.AnimationShimmer:
		gt.tickRate = shimmer.TickRate
		gt.hexLUT = shimmer.BuildLUT(b.ShimmerStops)
		gt.styleLUT = shimmer.BuildStyleLUT(gt.hexLUT, gt.cfg.output.Renderer())
	case fx.AnimationBar:
		gt.tickRate = bar.TickRate
	}

	// Guard against invalid spinner.Style values.
	if b.Mode == fx.AnimationSpinner && len(b.SpinnerStyle.Frames) == 0 {
		b.SpinnerStyle.Frames = spinner.DefaultStyle().Frames
	}
	if b.Mode == fx.AnimationSpinner && b.SpinnerStyle.Boomerang {
		b.SpinnerStyle.Frames = spinner.BoomerangFrames(b.SpinnerStyle.Frames)
	}
	if gt.tickRate <= 0 {
		gt.tickRate = spinner.DefaultStyle().Interval
	}
	if animInterval > 0 && gt.tickRate < animInterval {
		gt.tickRate = animInterval
	}
}

// buildLine assembles a log line from the configured parts order.
func buildLine(order []Part, reportTS bool, tsStr, levelStr, symbol, msg, fieldsStr string) string {
	parts := make([]string, 0, len(order))
	for _, p := range order {
		var part string
		switch p {
		case PartTimestamp:
			if !reportTS {
				continue
			}
			part = tsStr
		case PartLevel:
			part = levelStr
		case PartSymbol:
			part = symbol
		case PartMessage:
			part = msg
		case PartFields:
			part = fieldsStr
		}
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, " ")
}

// styledMsg applies the message style for the given level, if any.
func styledMsg(msg string, level Level, styles *style.Config, noColor bool) string {
	if s := styles.Messages[level]; s != nil && !noColor {
		return s.Render(msg)
	}
	return msg
}

// renderTaskFields formats the fields for a task, caching the result when
// the atomic pointer has not changed.
func renderTaskFields(gt *groupTask, dur time.Duration) string {
	b := gt.Builder
	fp := gt.FieldsPtr.Load()
	if b.ElapsedKey != "" || b.BarPercentKey != "" {
		resolved := b.ResolveDynamicFields(*fp, dur)
		gt.cachedFieldsStr = strings.TrimLeft(formatFields(resolved, gt.fieldOpts), " ")
	} else if fp != gt.cachedFieldsPtr {
		gt.cachedFieldsStr = strings.TrimLeft(formatFields(*fp, gt.fieldOpts), " ")
	}
	gt.cachedFieldsPtr = fp
	return gt.cachedFieldsStr
}

// renderTaskTimestamp returns the styled timestamp string for a task.
func renderTaskTimestamp(gt *groupTask) string {
	if !gt.cfg.reportTS {
		return ""
	}
	ts := time.Now().In(gt.cfg.timeLoc).Format(gt.cfg.timeFmt)
	if gt.cfg.styles.Timestamp != nil && !gt.cfg.noColor {
		return gt.cfg.styles.Timestamp.Render(ts)
	}
	return ts
}

// renderTaskLine renders a single animation frame line for a task.
// For done tasks, it renders the frozen final state with the level's default symbol.
// For active tasks, it renders the current animation frame.
// It does not perform any I/O.
func renderTaskLine(gt *groupTask, isDone bool, now time.Time) string {
	b := gt.Builder
	dur := now.Sub(gt.StartTime)
	fieldsStr := renderTaskFields(gt, dur)
	tsStr := renderTaskTimestamp(gt)

	if isDone {
		// Show the frozen final line with the level's default symbol.
		msg := gt.cfg.indentation + styledMsg(
			*gt.MsgPtr.Load(),
			b.Level,
			gt.cfg.styles,
			gt.cfg.noColor,
		)
		levelSymbol := gt.cfg.levelSymbol
		// Use a checkmark or the builder symbol for completed items.
		doneSymbol := gt.symbol
		return buildLine(
			gt.cfg.order,
			gt.cfg.reportTS,
			tsStr,
			levelSymbol,
			doneSymbol,
			msg,
			fieldsStr,
		)
	}

	// Bar mode has its own rendering path.
	if b.Mode == fx.AnimationBar {
		return renderTaskBarLine(gt, fieldsStr, tsStr, now)
	}

	msg := *gt.MsgPtr.Load()
	var char string

	switch b.Mode { //nolint:exhaustive // animationBar handled above
	case fx.AnimationSpinner:
		n := len(b.SpinnerStyle.Frames)
		i := int(dur/b.SpinnerStyle.Interval) % n
		if b.SpinnerStyle.Reverse {
			i = n - 1 - i
		}
		char = b.SpinnerStyle.Frames[i]
		msg = styledMsg(msg, b.Level, gt.cfg.styles, gt.cfg.noColor)
	case fx.AnimationPulse:
		char = gt.symbol
		t := (1.0 + math.Sin(2*math.Pi*dur.Seconds()*b.Speed-math.Pi/2)) / 2 //nolint:mnd // half-wave normalisation
		msg = pulse.TextCached(msg, t, b.PulseStops, &gt.pCache, gt.cfg.output.Renderer())
	case fx.AnimationShimmer:
		char = gt.symbol
		phase := math.Mod(dur.Seconds()*b.Speed, 1.0)
		msg = shimmer.Text(msg, phase, b.ShimmerDir, gt.hexLUT, gt.styleLUT)
	}

	return buildLine(
		gt.cfg.order,
		gt.cfg.reportTS,
		tsStr,
		gt.cfg.levelSymbol,
		char,
		gt.cfg.indentation+msg,
		fieldsStr,
	)
}

// renderTaskBarLine renders a bar-animation frame for a task. Factored out to
// keep renderTaskLine focused.
func renderTaskBarLine(gt *groupTask, fieldsStr, tsStr string, now time.Time) string {
	b := gt.Builder
	msg := gt.cfg.indentation + styledMsg(*gt.MsgPtr.Load(), b.Level, gt.cfg.styles, gt.cfg.noColor)

	current := int(b.BarProgressPtr.Load())
	total := int(b.BarTotalPtr.Load())

	// Cache the gradient style to avoid lipgloss.NewStyle() per frame.
	barStyle := b.BarStyle
	if len(barStyle.ProgressGradient) > 0 {
		progress := float64(current) / float64(max(total, 1))
		if !gt.gradientValid || gt.gradientProgress != progress {
			c := style.InterpolateGradient(progress, barStyle.ProgressGradient)
			gt.gradientStyle = gt.cfg.output.Renderer().
				NewStyle().
				Foreground(lipgloss.Color(c.Clamped().Hex()))
			gt.gradientProgress = progress
			gt.gradientValid = true
		}
		barStyle.StyleFill = &gt.gradientStyle
		barStyle.ProgressGradient = nil // prevent renderBar from recomputing
	}
	barStr := bar.Render(current, total, barStyle, gt.cfg.output.Width())
	sep := b.BarStyle.Separator
	if sep == "" {
		sep = " "
	}

	elapsed := now.Sub(gt.StartTime)
	var rate float64
	if secs := elapsed.Seconds(); secs > 0 && current > 0 {
		rate = float64(current) / secs
	}
	state := bar.State{Current: current, Total: total, Elapsed: elapsed, Rate: rate}

	var leftText, rightText string
	if b.BarStyle.WidgetLeft != nil {
		leftText = b.BarStyle.WidgetLeft(state)
	}
	if b.BarStyle.WidgetRight != nil && b.BarPercentKey == "" {
		rightText = b.BarStyle.WidgetRight(state)
	} else if b.BarStyle.WidgetLeft == nil && b.BarStyle.WidgetRight == nil && b.BarPercentKey == "" {
		// Default: padded percent on the right when no widgets are configured
		// and no BarPercent field is set.
		rightText = bar.FormatPercent(current, total, 0, true)
	}

	barFull := barStr
	if leftText != "" {
		barFull = leftText + sep + barFull
	}
	if rightText != "" {
		barFull = barFull + sep + rightText
	}

	// writeFrame equivalent: build the complete line string.
	if b.BarStyle.Placement == bar.PlaceInline {
		parts := buildLine(
			gt.cfg.order,
			gt.cfg.reportTS,
			tsStr,
			gt.cfg.levelSymbol,
			gt.symbol,
			msg+sep+barFull,
			fieldsStr,
		)
		return parts
	}
	parts := buildLine(
		gt.cfg.order,
		gt.cfg.reportTS,
		tsStr,
		gt.cfg.levelSymbol,
		gt.symbol,
		msg,
		fieldsStr,
	)
	return bar.FormatLine(parts, barFull, sep, b.BarStyle.Placement, gt.cfg.output.Width())
}

// runGroupLoop runs the group render loop, blocking until all tasks complete
// or the context is cancelled. Called by fxLogger.RunGroup.
func runGroupLoop(ctx context.Context, g *fx.Group) error {
	g.Mu.Lock()
	fxTasks := g.Tasks
	g.Mu.Unlock()

	if len(fxTasks) == 0 {
		return nil
	}

	// Wrap each fx.GroupTask with rendering state.
	gts := make([]*groupTask, len(fxTasks))
	for i, ft := range fxTasks {
		gt := &groupTask{GroupTask: ft}
		captureTaskConfig(gt)
		gts[i] = gt
	}

	// Non-TTY: print each task's initial line, then block on all results.
	// Dynamic fields (elapsed, bar percent) are stripped because their
	// initial zero values are meaningless without live updates.
	if !gts[0].cfg.isTTY {
		for _, gt := range gts {
			b := gt.Builder
			fieldsStr := strings.TrimLeft(
				formatFields(b.StripDynamicFields(*gt.FieldsPtr.Load()), gt.fieldOpts), " ",
			)
			line := buildLine(gt.cfg.order, gt.cfg.reportTS,
				time.Now().In(gt.cfg.timeLoc).Format(gt.cfg.timeFmt),
				gt.cfg.label, gt.symbol, gt.cfg.indentation+*gt.MsgPtr.Load(), fieldsStr)
			writeString(gt.cfg.out, line+"\n")
		}
		for _, ft := range fxTasks {
			select {
			case ft.Err = <-ft.DoneErr:
			case <-ctx.Done():
				for _, ft2 := range fxTasks {
					if ft2.Err == nil {
						select {
						case ft2.Err = <-ft2.DoneErr:
						default:
							ft2.Err = ctx.Err()
						}
					}
				}
				return ctx.Err()
			}
		}
		return nil
	}

	// Tick rate = fastest task's rate.
	tickRate := gts[0].tickRate
	for _, gt := range gts[1:] {
		tickRate = min(tickRate, gt.tickRate)
	}

	termOut := gts[0].cfg.termOut
	termOut.HideCursor()
	defer termOut.ShowCursor()

	out := gts[0].cfg.out
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	numLines := 0
	done := make([]bool, len(gts))
	remaining := len(gts)
	var frameBuf strings.Builder

	for remaining > 0 {
		select {
		case <-ctx.Done():
			clearBlock(out, numLines)
			for i, ft := range fxTasks {
				if !done[i] {
					ft.Err = ctx.Err()
				}
			}
			return ctx.Err()
		case <-ticker.C:
			now := time.Now()
			// Drain completed tasks.
			for i, ft := range fxTasks {
				if done[i] {
					continue
				}
				select {
				case err := <-ft.DoneErr:
					ft.Err = err
					done[i] = true
					remaining--
				default:
				}
			}
			// Batch all writes into a single string.
			frameBuf.Reset()
			if numLines > 1 {
				fmt.Fprintf(&frameBuf, cursorUpFmt, numLines-1)
			}
			for i, gt := range gts {
				line := renderTaskLine(gt, done[i], now)
				if i < len(gts)-1 {
					fmt.Fprintf(&frameBuf, "%s%s\n", clearLine, line)
				} else {
					fmt.Fprintf(&frameBuf, "%s%s", clearLine, line)
				}
			}
			writeString(out, frameBuf.String())
			numLines = len(gts)
			// If all done, break out after one final render.
			if remaining == 0 {
				break
			}
		}
	}

	clearBlock(out, numLines)
	return nil
}

// clearBlock erases n lines starting from the current cursor line and
// repositions the cursor back to the first cleared line. The cursor is
// expected to be on the last line of the block (no trailing newline).
func clearBlock(out io.Writer, n int) {
	if n == 0 {
		return
	}
	var buf strings.Builder
	if n > 1 {
		fmt.Fprintf(&buf, cursorUpFmt, n-1)
	}
	for range n {
		buf.WriteString(clearLine + "\n")
	}
	fmt.Fprintf(&buf, cursorUpFmt, n)
	writeString(out, buf.String())
}
