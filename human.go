package clog

// Byte formatting adapted from github.com/dustin/go-humanize (MIT).

import (
	"fmt"
	"math"
	"strings"
)

// SI byte size constants (base-1000).
const (
	kByte = 1000
	mByte = kByte * 1000
	gByte = mByte * 1000
	tByte = gByte * 1000
	pByte = tByte * 1000
	eByte = pByte * 1000
)

// IEC byte size constants (base-1024).
const (
	kiByte = 1024
	miByte = kiByte * 1024
	giByte = miByte * 1024
	tiByte = giByte * 1024
	piByte = tiByte * 1024
	eiByte = piByte * 1024
)

const (
	baseSI  = 1000 // SI base for byte formatting
	baseIEC = 1024 // IEC base for byte formatting

	humanMinBytes  = 10  // below this threshold, bytes are displayed as-is
	humanRoundHalf = 0.5 // half-up rounding offset
)

var (
	// siSizes is the ordered list of SI unit suffixes used by [humanBytes].
	siSizes = []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}

	// iecSizes is the ordered list of IEC unit suffixes used by [humanIBytes].
	iecSizes = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
)

// humanBytes formats a byte count as a human-readable SI string.
// digits controls significant digits; trailing zeros are always stripped.
// Below MB, decimals are suppressed.
func humanBytes(s uint64, digits int) string {
	return humanizeBytes(s, baseSI, digits, true, siSizes)
}

// humanBytesWidth returns the max formatted width (no zero stripping)
// for stable padding.
func humanBytesWidth(s uint64, digits int) int {
	return maxBytesWidth(s, baseSI, digits, siSizes)
}

// humanIBytes formats a byte count as a human-readable IEC string.
// digits controls significant digits; trailing zeros are always stripped.
// Below MiB, decimals are suppressed.
func humanIBytes(s uint64, digits int) string {
	return humanizeBytes(s, baseIEC, digits, true, iecSizes)
}

// humanIBytesWidth returns the max formatted width (no zero stripping)
// for stable padding.
func humanIBytesWidth(s uint64, digits int) int {
	return maxBytesWidth(s, baseIEC, digits, iecSizes)
}

// humanizeBytes is the shared implementation for [humanBytes] and [humanIBytes].
// Uses significant-digit rounding (half-up) adapted from go-humanize, with
// optional trailing-zero stripping controlled by strip.
func humanizeBytes(s uint64, base float64, digits int, strip bool, sizes []string) string {
	if s < humanMinBytes {
		return fmt.Sprintf("%d B", s)
	}

	e := math.Floor(math.Log(float64(s)) / math.Log(base))
	suffix := sizes[int(e)]
	rawVal := float64(s) / math.Pow(base, e)

	// Below MB/MiB (e < 2), decimal places are meaningless - show whole numbers.
	if e < 2 { //nolint:mnd // 0=B, 1=kB/KiB, 2=MB/MiB
		return fmt.Sprintf("%.0f %s", math.Round(rawVal), suffix)
	}

	// Compute decimal places for the target number of significant digits,
	// then round once with half-up to avoid double-rounding.
	decimals := max(digits-countDigits(int64(rawVal)), 0)
	rounding := math.Pow10(decimals)
	val := math.Floor(rawVal*rounding+humanRoundHalf) / rounding

	formatted := fmt.Sprintf("%.*f", decimals, val)
	if strip {
		formatted = trimDecimalZeros(formatted)
	}
	return formatted + " " + suffix
}

// maxBytesWidth returns the maximum formatted width for any value in the
// same unit range as s. With significant digits, a value with fewer integer
// digits may be wider (more decimal places), so we take the max of the
// actual total's width and the theoretical one-integer-digit width.
func maxBytesWidth(s uint64, base float64, digits int, sizes []string) int {
	totalWidth := len(humanizeBytes(s, base, digits, false, sizes))

	if s >= humanMinBytes {
		e := math.Floor(math.Log(float64(s)) / math.Log(base))
		if e >= 2 { //nolint:mnd // only MB/MiB+ have decimals
			suffix := sizes[int(e)]
			maxDecimals := max(digits-1, 0)
			// "X.XX… suffix" - 1 integer digit with maximum decimal places.
			oneDigitWidth := 1 + 1 + len(suffix) // "X" + " " + suffix
			if maxDecimals > 0 {
				oneDigitWidth += 1 + maxDecimals // "." + decimals
			}
			return max(totalWidth, oneDigitWidth)
		}
	}
	return totalWidth
}

// countDigits returns the number of decimal digits in n.
func countDigits(n int64) int {
	d := 0
	for n != 0 {
		n /= 10
		d++
	}
	return d
}

// trimDecimalZeros strips trailing zeros after a decimal point.
// "100.0" → "100", "82.90" → "82.9", "1.50" → "1.5", "42" → "42".
func trimDecimalZeros(s string) string {
	dot := strings.IndexByte(s, '.')
	if dot < 0 {
		return s
	}
	trimmed := strings.TrimRight(s[dot+1:], "0")
	if trimmed == "" {
		return s[:dot]
	}
	return s[:dot+1] + trimmed
}
