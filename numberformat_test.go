package clog

import (
	"io"
	"testing"

	"github.com/gechr/clog/field/fraction"
	"github.com/gechr/clog/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fmtVal(v any, f *FieldFormats) (string, valueKind) {
	return formatValue(
		v,
		sliceFormat{open: "[", close: "]", sep: ", "},
		QuoteAuto,
		0,
		0,
		nil,
		"",
		f,
	)
}

func TestNumberFormatString(t *testing.T) {
	assert.Equal(t, "plain", NumberPlain.String())
	assert.Equal(t, "grouped", NumberGrouped.String())
	assert.Equal(t, "compact", NumberCompact.String())
}

func TestNumberFormatMarshalRoundTrip(t *testing.T) {
	for _, mode := range []NumberFormat{NumberPlain, NumberGrouped, NumberCompact} {
		text, err := mode.MarshalText()
		require.NoError(t, err)

		var got NumberFormat
		require.NoError(t, got.UnmarshalText(text))
		assert.Equal(t, mode, got)
	}
}

func TestNumberFormatUnmarshalInvalid(t *testing.T) {
	var mode NumberFormat
	require.Error(t, mode.UnmarshalText([]byte("bogus")))
}

func TestFormatValueNumberModes(t *testing.T) {
	plain := DefaultFieldFormats()
	grouped := DefaultFieldFormats()
	grouped.NumberFormat = NumberGrouped
	compact := DefaultFieldFormats()
	compact.NumberFormat = NumberCompact

	cases := []struct {
		name  string
		value any
		fmts  *FieldFormats
		want  string
	}{
		{"int_plain", 1234567, &plain, "1234567"},
		{"int_grouped", 1234567, &grouped, "1,234,567"},
		{"int_compact", 1234567, &compact, "1.2M"},
		{"int64_grouped", int64(1234567), &grouped, "1,234,567"},
		{"uint_grouped", uint(1234567), &grouped, "1,234,567"},
		{"uint64_grouped", uint64(1234567), &grouped, "1,234,567"},
		{"compact_below_minimum", 999, &compact, "999"},
		{"compact_at_minimum", 1000, &compact, "1K"},
		{"negative_grouped", -1234567, &grouped, "-1,234,567"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, kind := fmtVal(tc.value, tc.fmts)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, kindNumber, kind)
		})
	}
}

func TestFormatValueUint64Overflow(t *testing.T) {
	// Values above math.MaxInt64 cannot be grouped safely, so they render
	// as a plain decimal string rather than producing a bogus sign flip.
	grouped := DefaultFieldFormats()
	grouped.NumberFormat = NumberGrouped
	got, kind := fmtVal(uint64(18446744073709551615), &grouped)
	assert.Equal(t, "18446744073709551615", got)
	assert.Equal(t, kindNumber, kind)
}

func TestFormatValueFractionModes(t *testing.T) {
	grouped := DefaultFieldFormats()
	grouped.NumberFormat = NumberGrouped
	compact := DefaultFieldFormats()
	compact.NumberFormat = NumberCompact

	got, _ := fmtVal(core.Fraction{Current: 1234567, Total: 9999999}, &grouped)
	assert.Equal(t, "1,234,567/9,999,999", got)

	got, _ = fmtVal(core.Fraction{Current: 1234567, Total: 9999999}, &compact)
	assert.Equal(t, "1.2M/10M", got)
}

func TestFractionFormatOverridesNumberFormat(t *testing.T) {
	// NumberFormat is grouped, but FractionFormat forces compact for fractions.
	f := DefaultFieldFormats()
	f.NumberFormat = NumberGrouped
	fracMode := NumberCompact
	f.FractionFormat = &fracMode

	gotInt, _ := fmtVal(1234567, &f)
	assert.Equal(t, "1,234,567", gotInt, "plain int still uses NumberFormat")

	gotFrac, _ := fmtVal(core.Fraction{Current: 1234567, Total: 9999999}, &f)
	assert.Equal(t, "1.2M/10M", gotFrac, "fraction uses FractionFormat override")
}

func TestFractionFormatFallsBackToNumberFormat(t *testing.T) {
	// No FractionFormat set: fractions inherit NumberFormat.
	f := DefaultFieldFormats()
	f.NumberFormat = NumberGrouped

	got, _ := fmtVal(core.Fraction{Current: 1234567, Total: 9999999}, &f)
	assert.Equal(t, "1,234,567/9,999,999", got)
}

func TestFractionPerFieldFormatWins(t *testing.T) {
	// Per-field WithFormat beats both FractionFormat and NumberFormat.
	f := DefaultFieldFormats()
	f.NumberFormat = NumberGrouped
	fracMode := NumberGrouped
	f.FractionFormat = &fracMode

	frac := core.Fraction{Current: 1234567, Total: 9999999}
	fraction.Apply(&frac, fraction.WithFormat(NumberCompact))

	got, _ := fmtVal(frac, &f)
	assert.Equal(t, "1.2M/10M", got)
}

func TestCustomGroupSeparator(t *testing.T) {
	f := DefaultFieldFormats()
	f.NumberFormat = NumberGrouped
	f.NumberGroupSeparator = " "

	got, _ := fmtVal(1234567, &f)
	assert.Equal(t, "1 234 567", got)
}

func TestCompactMinimumThreshold(t *testing.T) {
	f := DefaultFieldFormats()
	f.NumberFormat = NumberCompact
	f.NumberCompactMinimum = 1_000_000

	below, _ := fmtVal(999999, &f)
	assert.Equal(t, "999,999", below, "below minimum falls back to grouped")

	atOrAbove, _ := fmtVal(1500000, &f)
	assert.Equal(t, "1.5M", atOrAbove)
}

func TestCompactGroupsBelowThreshold(t *testing.T) {
	// With a 10k minimum, values below it are grouped and values at or above
	// are abbreviated, so the series reads 9,999 -> 10K -> 11K -> 12K.
	f := DefaultFieldFormats()
	f.NumberFormat = NumberCompact
	f.NumberCompactMinimum = 10_000

	cases := map[int]string{
		9999:  "9,999",
		10000: "10K",
		11000: "11K",
		12000: "12K",
	}
	for in, want := range cases {
		got, _ := fmtVal(in, &f)
		assert.Equal(t, want, got, "value %d", in)
	}
}

func TestCompactFallbackPlain(t *testing.T) {
	// NumberCompactFallback = NumberPlain keeps sub-threshold values verbatim.
	f := DefaultFieldFormats()
	f.NumberFormat = NumberCompact
	f.NumberCompactMinimum = 10_000
	f.NumberCompactFallback = NumberPlain

	below, _ := fmtVal(9999, &f)
	assert.Equal(t, "9999", below, "plain fallback below threshold")

	atOrAbove, _ := fmtVal(10000, &f)
	assert.Equal(t, "10K", atOrAbove)
}

func TestSetNumberFormatSetters(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetNumberFormat(NumberGrouped)
	l.SetNumberGroupSeparator("_")
	l.SetFractionFormat(NumberCompact)
	l.SetNumberCompactMinimum(500)
	l.SetNumberCompactFallback(NumberPlain)

	f := l.FieldFormats()
	assert.Equal(t, NumberGrouped, f.NumberFormat)
	assert.Equal(t, "_", f.NumberGroupSeparator)
	require.NotNil(t, f.FractionFormat)
	assert.Equal(t, NumberCompact, *f.FractionFormat)
	assert.Equal(t, int64(500), f.NumberCompactMinimum)
	assert.Equal(t, NumberPlain, f.NumberCompactFallback)
}
