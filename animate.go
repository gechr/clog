package clog

import (
	"context"
	"io"
	"math"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/charmbracelet/bubbles/spinner"
)

// clearLine is the ANSI escape to erase the entire current line (EL2),
// followed by a carriage return to reset the cursor to column 0.
const clearLine = "\x1b[2K\r"

//nolint:cyclop // animation loop has inherent complexity
func runAnimation(
	ctx context.Context,
	title *atomic.Pointer[string],
	fields *atomic.Pointer[[]Field],
	s spinner.Spinner,
	shimmerStops []ColorStop,
	shimmerDir Direction,
	pulseStops []ColorStop,
	task Task,
) error {
	// Run the task in a goroutine.
	done := make(chan error, 1)
	go func() {
		done <- task(ctx)
	}()

	// Snapshot Default's settings under the mutex to avoid data races.
	Default.mu.Lock()
	fieldStyleLevel := Default.fieldStyleLevel
	fieldTimeFormat := Default.fieldTimeFormat
	label := Default.formatLabel(InfoLevel)
	noColor := Default.output.ColorsDisabled()
	order := Default.parts
	out := Default.output.Writer()
	termOut := Default.output.Renderer().Output()
	quoteClose := Default.quoteClose
	quoteMode := Default.quoteMode
	quoteOpen := Default.quoteOpen
	reportTS := Default.reportTimestamp
	styles := Default.styles
	timeFmt := Default.timeFormat
	timeLoc := Default.timeLocation
	Default.mu.Unlock()

	// buildParts assembles the display parts slice from the configured order.
	// It takes dynamic values (timestamp, prefix, message, fieldsStr) that
	// change per frame, while static values (label/levelPrefix) are captured
	// in the closure.
	buildParts := func(order []Part, reportTS bool, tsStr, levelStr, prefix, msg, fieldsStr string) string {
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
			case PartPrefix:
				part = prefix
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

	// Don't animate if colours are disabled (CI, piped output, etc.).
	// Print the initial title so the user knows something is in progress.
	if noColor {
		fieldsStr := strings.TrimLeft(
			formatFields(*fields.Load(), formatFieldsOpts{
				noColor:    true,
				quoteClose: quoteClose,
				quoteMode:  quoteMode,
				quoteOpen:  quoteOpen,
				timeFormat: fieldTimeFormat,
			}), " ",
		)
		line := buildParts(order, reportTS,
			time.Now().In(timeLoc).Format(timeFmt),
			label, "⏳", *title.Load(), fieldsStr)
		_, _ = io.WriteString(out, line+"\n")
		return <-done
	}

	// Hide cursor during animation.
	termOut.HideCursor()
	defer termOut.ShowCursor()

	var levelPrefix string
	if style := styles.Levels[InfoLevel]; style != nil {
		levelPrefix = style.Render(label)
	} else {
		levelPrefix = label
	}

	// Use a faster tick rate when shimmer or pulse is active for smooth animation.
	// The spinner frame advances at its own FPS regardless of the tick rate.
	tickRate := s.FPS
	hasShimmer := len(shimmerStops) > 0
	hasPulse := len(pulseStops) > 0
	if hasShimmer && shimmerTickRate < tickRate {
		tickRate = shimmerTickRate
	}
	if hasPulse && pulseTickRate < tickRate {
		tickRate = pulseTickRate
	}

	// Pre-compute the shimmer LUT once — it's phase-independent.
	var sLUT *shimmerLUT
	if hasShimmer {
		sLUT = buildShimmerLUT(shimmerStops)
	}

	// Cache formatted fields — only re-format when the atomic pointer changes.
	var cachedFieldsPtr unsafe.Pointer
	var cachedFieldsStr string
	fieldOpts := formatFieldsOpts{
		fieldStyleLevel: fieldStyleLevel,
		styles:          styles,
		level:           InfoLevel,
		quoteMode:       quoteMode,
		quoteOpen:       quoteOpen,
		quoteClose:      quoteClose,
		timeFormat:      fieldTimeFormat,
	}

	startTime := time.Now()
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	// Compose each frame into a single buffer to reduce syscalls.
	var frameBuf strings.Builder

	for {
		select {
		case err := <-done:
			_, _ = io.WriteString(out, clearLine)
			return err
		case now := <-ticker.C:
			elapsed := now.Sub(startTime)
			frame := int(elapsed / s.FPS)
			char := s.Frames[frame%len(s.Frames)]

			msg := *title.Load()
			if hasPulse {
				half := 0.5
				t := half * (1.0 + math.Sin(2*math.Pi*elapsed.Seconds()*pulseSpeed-math.Pi/2))
				msg = pulseText(msg, t, pulseStops)
			} else if hasShimmer {
				phase := math.Mod(elapsed.Seconds()*shimmerSpeed, 1.0)
				msg = shimmerText(msg, phase, shimmerDir, sLUT)
			}

			// Only re-format fields when the pointer changes.
			fp := fields.Load()
			fpRaw := unsafe.Pointer(fp)
			if fpRaw != cachedFieldsPtr {
				cachedFieldsPtr = fpRaw
				cachedFieldsStr = strings.TrimLeft(formatFields(*fp, fieldOpts), " ")
			}

			var tsStr string
			if reportTS {
				ts := now.In(timeLoc).Format(timeFmt)
				if styles.Timestamp != nil {
					tsStr = styles.Timestamp.Render(ts)
				} else {
					tsStr = ts
				}
			}

			line := buildParts(order, reportTS, tsStr, levelPrefix, char, msg, cachedFieldsStr)
			frameBuf.Reset()
			frameBuf.WriteString(clearLine)
			frameBuf.WriteString(line)
			_, _ = io.WriteString(out, frameBuf.String())
		case <-ctx.Done():
			_, _ = io.WriteString(out, clearLine)
			return ctx.Err()
		}
	}
}
