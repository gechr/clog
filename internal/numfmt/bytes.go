// Package numfmt provides human-readable formatting for byte counts.
//
// Adapted from github.com/dustin/go-humanize (MIT).
package numfmt

import (
	"fmt"
	"math"
	"strings"
)

const (
	baseSI  = 1000 // SI base for byte formatting
	baseIEC = 1024 // IEC base for byte formatting

	minBytes  = 10  // below this threshold, bytes are displayed as-is
	roundHalf = 0.5 // half-up rounding offset
)

var (
	siSizes  = []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}
	iecSizes = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
)

// Bytes formats a byte count as a human-readable SI string (base-1000).
// digits controls significant digits; trailing zeros are always stripped.
// Below MB, decimals are suppressed.
func Bytes(s uint64, digits int) string {
	return format(s, baseSI, digits, true, siSizes)
}

// BytesWidth returns the max formatted width (no zero stripping)
// for stable padding.
func BytesWidth(s uint64, digits int) int {
	return maxWidth(s, baseSI, digits, siSizes)
}

// IBytes formats a byte count as a human-readable IEC string (base-1024).
// digits controls significant digits; trailing zeros are always stripped.
// Below MiB, decimals are suppressed.
func IBytes(s uint64, digits int) string {
	return format(s, baseIEC, digits, true, iecSizes)
}

// IBytesWidth returns the max formatted width (no zero stripping)
// for stable padding.
func IBytesWidth(s uint64, digits int) int {
	return maxWidth(s, baseIEC, digits, iecSizes)
}

// format is the shared implementation for [Bytes] and [IBytes].
func format(s uint64, base float64, digits int, strip bool, sizes []string) string {
	if s < minBytes {
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
	decimals := max(digits-CountDigits(int64(rawVal)), 0)
	rounding := math.Pow10(decimals)
	val := math.Floor(rawVal*rounding+roundHalf) / rounding

	formatted := fmt.Sprintf("%.*f", decimals, val)
	if strip {
		formatted = TrimDecimalZeros(formatted)
	}
	return formatted + " " + suffix
}

// maxWidth returns the maximum formatted width for any value in the
// same unit range as s.
func maxWidth(s uint64, base float64, digits int, sizes []string) int {
	totalWidth := len(format(s, base, digits, false, sizes))

	if s >= minBytes {
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

// CountDigits returns the number of decimal digits in n.
func CountDigits(n int64) int {
	if n == 0 {
		return 1
	}
	d := 0
	for n != 0 {
		n /= 10
		d++
	}
	return d
}

// TrimDecimalZeros strips trailing zeros after a decimal point.
// "100.0" → "100", "82.90" → "82.9", "1.50" → "1.5", "42" → "42".
func TrimDecimalZeros(s string) string {
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
