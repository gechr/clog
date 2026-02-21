package clog

import (
	"io"
	"testing"
	"time"
)

func BenchmarkLogDisabled(b *testing.B) {
	b.ReportAllocs()

	l := New(NewOutput(io.Discard, ColorNever))
	l.SetLevel(ErrorLevel)

	for b.Loop() {
		l.Info().Str("k", "v").Msg("hello")
	}
}

func BenchmarkLogSimple(b *testing.B) {
	b.ReportAllocs()

	l := New(NewOutput(io.Discard, ColorNever))

	for b.Loop() {
		l.Info().Str("key", "value").Msg("hello")
	}
}

func BenchmarkLogMultiField(b *testing.B) {
	b.ReportAllocs()

	l := New(NewOutput(io.Discard, ColorNever))

	for b.Loop() {
		l.Info().
			Str("name", "test").
			Int("count", 42).
			Bool("ok", true).
			Duration("elapsed", time.Second).
			Float64("rate", 3.14).
			Msg("multi")
	}
}

func BenchmarkLogContext(b *testing.B) {
	b.ReportAllocs()

	l := New(NewOutput(io.Discard, ColorNever))
	sub := l.With().Str("app", "bench").Int("pid", 1234).Bool("debug", false).Logger()

	for b.Loop() {
		sub.Info().Str("action", "test").Msg("context log")
	}
}

func BenchmarkFormatFields(b *testing.B) {
	b.ReportAllocs()

	l := New(NewOutput(io.Discard, ColorNever))

	for b.Loop() {
		l.Info().
			Str("name", "test").
			Int("count", 42).
			Bool("ok", true).
			Duration("elapsed", time.Second).
			Float64("rate", 3.14).
			Str("extra", "value").
			Msg("many fields")
	}
}

func BenchmarkHighlightJSON(b *testing.B) {
	b.ReportAllocs()

	l := New(NewOutput(io.Discard, ColorNever))
	jsonStr := []byte(`{"name":"test","count":42,"active":true}`)

	for b.Loop() {
		l.Info().RawJSON("data", jsonStr).Msg("json log")
	}
}

func BenchmarkPulseText(b *testing.B) {
	b.ReportAllocs()

	stops := DefaultPulseGradient()
	text := "pulse benchmark text"

	for b.Loop() {
		pulseText(text, 0.3, stops)
	}
}
