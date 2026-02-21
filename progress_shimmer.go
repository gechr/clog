package clog

import (
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

// Direction controls which way an animation travels.
type Direction int

const (
	// DirectionRight moves the shimmer wave from left to right (default).
	DirectionRight Direction = iota
	// DirectionLeft moves the shimmer wave from right to left.
	DirectionLeft
	// DirectionMiddleIn sends the shimmer wave inward from both edges.
	DirectionMiddleIn
	// DirectionMiddleOut sends the shimmer wave outward from the center.
	DirectionMiddleOut
	// DirectionBounceIn sends the shimmer wave inward from both edges, then
	// bounces it back outward, creating a ping-pong effect.
	DirectionBounceIn
	// DirectionBounceOut sends the shimmer wave outward from the center, then
	// bounces it back inward, creating a ping-pong effect.
	DirectionBounceOut
)

const (
	// shimmerSpeed is the number of full gradient cycles per second.
	shimmerSpeed = 0.5

	// shimmerTickRate is the repaint interval when shimmer is active (~30fps).
	shimmerTickRate = 33 * time.Millisecond

	shimmerLUTSize = 64
)

// shimmerLUT is a pre-computed gradient lookup table of hex color strings.
type shimmerLUT [shimmerLUTSize]string

// shimmerStyleLUT is a pre-computed lookup table of lipgloss styles, built
// from a [shimmerLUT]. Reusing styles across frames eliminates per-frame
// style allocations entirely.
type shimmerStyleLUT [shimmerLUTSize]lipgloss.Style

// DefaultShimmerGradient returns a wave-shaped gradient for shimmer effects:
// a subtle red-to-green-to-blue cycle that wraps seamlessly.
func DefaultShimmerGradient() []ColorStop {
	rR, rG, rB := 1.0, 0.7, 0.7
	gR, gG, gB := 0.7, 1.0, 0.75
	bR, bG, bB := 0.7, 0.8, 1.0
	third, twoThirds := 0.33, 0.67
	red := colorful.Color{R: rR, G: rG, B: rB}
	green := colorful.Color{R: gR, G: gG, B: gB}
	blue := colorful.Color{R: bR, G: bG, B: bB}
	return []ColorStop{
		{Position: 0, Color: red},
		{Position: third, Color: green},
		{Position: twoThirds, Color: blue},
		{Position: 1.0, Color: red},
	}
}

// Shimmer creates a new [AnimationBuilder] using the [Default] logger with an
// animated gradient shimmer on the message text.
// Each character is coloured independently based on its position in the wave.
// With no arguments, the default shimmer gradient is used. Custom gradient
// stops can be passed to override the default.
func Shimmer(msg string, stops ...ColorStop) *AnimationBuilder {
	return Default.Shimmer(msg, stops...)
}

// Shimmer creates a new [AnimationBuilder] with an animated gradient shimmer on the message text.
// Each character is coloured independently based on its position in the wave.
// With no arguments, the default shimmer gradient is used. Custom gradient
// stops can be passed to override the default.
func (l *Logger) Shimmer(msg string, stops ...ColorStop) *AnimationBuilder {
	if len(stops) == 0 {
		stops = DefaultShimmerGradient()
	}
	b := &AnimationBuilder{
		level:        InfoLevel,
		logger:       l,
		mode:         animationShimmer,
		msg:          msg,
		shimmerStops: stops,
		speed:        shimmerSpeed,
		spinner:      DefaultSpinnerStyle(),
	}
	b.initSelf(b)
	return b
}

// ShimmerDirection sets the direction the shimmer wave travels.
// Defaults to [DirectionRight]. Use [DirectionLeft] to reverse
// or [DirectionMiddleIn] for a wave entering from both edges.
// Only meaningful when the builder was created with [Shimmer].
func (b *AnimationBuilder) ShimmerDirection(d Direction) *AnimationBuilder {
	b.shimmerDir = d
	return b
}

// Speed sets the number of full animation cycles per second.
// For [Shimmer] this controls how fast the gradient wave sweeps across the text
// (default 0.5). For [Pulse] this controls the oscillation rate (default 0.5).
// Higher values produce faster animation. Values <= 0 are treated as the default.
func (b *AnimationBuilder) Speed(speed Speed) *AnimationBuilder {
	if speed <= 0 {
		switch b.mode { //nolint:exhaustive // only pulse and shimmer have configurable speed
		case animationPulse:
			speed = pulseSpeed
		default:
			speed = shimmerSpeed
		}
	}
	b.speed = speed
	return b
}

// buildShimmerLUT pre-computes a gradient lookup table of hex color strings
// from the given color stops. The LUT is phase-independent and can be reused
// across frames.
func buildShimmerLUT(stops []ColorStop) *shimmerLUT {
	var lut shimmerLUT
	for i := range lut {
		t := float64(i) / float64(shimmerLUTSize-1)
		//nolint:gosec // i is bounded by range lut
		lut[i] = interpolateGradient(t, stops).Clamped().Hex()
	}
	return &lut
}

// buildShimmerStyleLUT pre-computes a lipgloss style for every entry in the
// hex LUT. Call once after [buildShimmerLUT] and pass the result to
// [shimmerText] to avoid style allocations in the render loop.
func buildShimmerStyleLUT(lut *shimmerLUT) *shimmerStyleLUT {
	var s shimmerStyleLUT
	for i, hex := range lut {
		//nolint:gosec // i is bounded by range lut
		s[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
	}
	return &s
}

// shimmerText renders each character of text with a gradient-interpolated
// foreground color, creating an animated shimmer when called with advancing
// phase values. Spaces are passed through unstyled. The caller must supply a
// pre-built LUT from [buildShimmerLUT]. Passing a pre-built [shimmerStyleLUT]
// from [buildShimmerStyleLUT] eliminates per-frame style allocations; pass
// nil to create styles on the fly.
func shimmerText(
	text string,
	phase float64,
	dir Direction,
	lut *shimmerLUT,
	styleLUT *shimmerStyleLUT,
) string {
	n := utf8.RuneCountInString(text)
	if n == 0 {
		return text
	}

	// Map each character position to a LUT index, then batch adjacent
	// characters that share the same index into a single style.Render call.
	// This reduces style creations from ~N/frame to ~5-10/frame (or zero
	// when a pre-built shimmerStyleLUT is supplied).
	var buf strings.Builder
	var (
		runByteStart int
		runIdx       int
		runIsSpace   bool
		charPos      int
	)

	flushRun := func(byteEnd int) {
		run := text[runByteStart:byteEnd]
		if runIsSpace {
			buf.WriteString(run)
		} else {
			var style lipgloss.Style
			if styleLUT != nil {
				style = styleLUT[runIdx]
			} else {
				style = lipgloss.NewStyle().Foreground(lipgloss.Color(lut[runIdx]))
			}
			buf.WriteString(style.Render(run))
		}
	}

	for byteIdx, r := range text {
		curIsSpace := unicode.IsSpace(r)
		var curIdx int
		if !curIsSpace {
			curIdx = shimmerCharIdx(charPos, n, phase, dir)
		}

		if charPos == 0 {
			runIsSpace = curIsSpace
			runIdx = curIdx
		} else if curIsSpace != runIsSpace || (!curIsSpace && curIdx != runIdx) {
			// Start a new run when transitioning between space/non-space or
			// when the quantized LUT index changes.
			flushRun(byteIdx)
			runByteStart = byteIdx
			runIsSpace = curIsSpace
			runIdx = curIdx
		}
		charPos++
	}
	// Flush final run.
	flushRun(len(text))

	return buf.String()
}

// shimmerCharIdx returns the LUT index for character at position i of n,
// given the animation phase and direction.
func shimmerCharIdx(i, n int, phase float64, dir Direction) int {
	pos := float64(i) / float64(n)

	var t float64
	switch dir {
	case DirectionLeft:
		t = math.Mod(pos+phase, 1.0)
	case DirectionMiddleIn:
		fold := math.Abs(2*pos - 1.0)
		t = math.Mod(fold+phase, 1.0)
	case DirectionMiddleOut:
		fold := 1.0 - math.Abs(2*pos-1.0)
		t = math.Mod(fold+phase, 1.0)
	case DirectionRight:
		t = math.Mod(pos-phase+1.0, 1.0)
	case DirectionBounceIn:
		//nolint:mnd // triangle wave amplitude for ping-pong phase
		bounce := 0.75 * (1.0 - math.Abs(2*phase-1.0))
		fold := math.Abs(2*pos - 1.0)
		t = math.Mod(fold+bounce, 1.0)
	case DirectionBounceOut:
		//nolint:mnd // triangle wave amplitude for ping-pong phase
		bounce := 0.75 * (1.0 - math.Abs(2*phase-1.0))
		fold := 1.0 - math.Abs(2*pos-1.0)
		t = math.Mod(fold+bounce, 1.0)
	}

	idx := int(t * float64(shimmerLUTSize-1))
	if idx >= shimmerLUTSize {
		idx = shimmerLUTSize - 1
	}
	if idx < 0 {
		idx = 0
	}
	return idx
}
