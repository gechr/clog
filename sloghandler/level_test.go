package sloghandler

import (
	"log/slog"
	"testing"

	"github.com/gechr/clog"
	"github.com/stretchr/testify/assert"
)

func TestLevelMapping(t *testing.T) {
	tests := []struct {
		slogLevel slog.Level
		clogLevel clog.Level
	}{
		{slog.Level(-8), clog.LevelTrace}, // below debug
		{slog.LevelDebug - 1, clog.LevelTrace},
		{slog.LevelDebug, clog.LevelDebug},
		{slog.LevelDebug + 1, clog.LevelDebug},
		{slog.LevelInfo - 1, clog.LevelDebug},
		{slog.LevelInfo, clog.LevelInfo},
		{slog.LevelInfo + 1, clog.LevelInfo},
		{slog.LevelWarn - 1, clog.LevelInfo},
		{slog.LevelWarn, clog.LevelWarn},
		{slog.LevelWarn + 1, clog.LevelWarn},
		{slog.LevelError - 1, clog.LevelWarn},
		{slog.LevelError, clog.LevelError},
		{slog.LevelError + 1, clog.LevelFatal},
		{slog.Level(12), clog.LevelFatal}, // well above error
	}

	for _, tt := range tests {
		got := slogLevelToClog(tt.slogLevel)
		assert.Equal(t, tt.clogLevel, got, "slog.Level(%d)", tt.slogLevel)
	}
}
