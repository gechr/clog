package clog

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
)

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

	// Don't animate if colours are disabled (CI, piped output, etc.).
	// Print the initial title so the user knows something is in progress.
	if noColor {
		parts := make([]string, 0, len(order))
		for _, p := range order {
			var part string
			switch p {
			case PartTimestamp:
				if !reportTS {
					continue
				}
				part = time.Now().In(timeLoc).Format(timeFmt)
			case PartLevel:
				part = label
			case PartPrefix:
				part = "⏳"
			case PartMessage:
				part = *title.Load()
			case PartFields:
				part = strings.TrimLeft(
					formatFields(*fields.Load(), formatFieldsOpts{
						noColor:    true,
						quoteClose: quoteClose,
						quoteMode:  quoteMode,
						quoteOpen:  quoteOpen,
						timeFormat: fieldTimeFormat,
					}), " ",
				)
			}
			if part != "" {
				parts = append(parts, part)
			}
		}

		_, _ = fmt.Fprintf(out, "%s\n", strings.Join(parts, " "))
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

	startTime := time.Now()
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			termOut.ClearLine()
			_, _ = fmt.Fprint(out, "\r")
			return err
		case now := <-ticker.C:
			elapsed := now.Sub(startTime)
			frame := int(elapsed / s.FPS)
			char := s.Frames[frame%len(s.Frames)]

			parts := make([]string, 0, len(order))

			for _, p := range order {
				var part string

				switch p {
				case PartTimestamp:
					if !reportTS {
						continue
					}

					ts := now.In(timeLoc).Format(timeFmt)
					if styles.Timestamp != nil {
						part = styles.Timestamp.Render(ts)
					} else {
						part = ts
					}
				case PartLevel:
					part = levelPrefix
				case PartPrefix:
					part = char
				case PartMessage:
					msg := *title.Load()
					if hasPulse {
						half := 0.5
						t := half * (1.0 + math.Sin(2*math.Pi*elapsed.Seconds()*pulseSpeed-math.Pi/2))
						msg = pulseText(msg, t, pulseStops)
					} else if hasShimmer {
						phase := math.Mod(elapsed.Seconds()*shimmerSpeed, 1.0)
						msg = shimmerText(msg, phase, shimmerStops, shimmerDir)
					}
					part = msg
				case PartFields:
					part = strings.TrimLeft(formatFields(*fields.Load(), formatFieldsOpts{
						fieldStyleLevel: fieldStyleLevel,
						styles:          styles,
						level:           InfoLevel,
						quoteMode:       quoteMode,
						quoteOpen:       quoteOpen,
						quoteClose:      quoteClose,
						timeFormat:      fieldTimeFormat,
					}), " ")
				}

				if part != "" {
					parts = append(parts, part)
				}
			}

			termOut.ClearLine()
			_, _ = fmt.Fprintf(out, "\r%s", strings.Join(parts, " "))
		case <-ctx.Done():
			termOut.ClearLine()
			_, _ = fmt.Fprint(out, "\r")
			return ctx.Err()
		}
	}
}
