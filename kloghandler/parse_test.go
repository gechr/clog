package kloghandler

import (
	"testing"

	"github.com/gechr/clog"
	"github.com/stretchr/testify/assert"
)

func TestParseKlogLine(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantLevel  clog.Level
		wantMsg    string
		wantSource string
	}{
		{
			name:       "info",
			line:       "I0115 10:30:00.123456    1234 server.go:42] listening\n",
			wantLevel:  clog.LevelInfo,
			wantMsg:    "listening",
			wantSource: "server.go:42",
		},
		{
			name:       "warning",
			line:       "W0115 10:30:00.123456    1234 client.go:7] retrying\n",
			wantLevel:  clog.LevelWarn,
			wantMsg:    "retrying",
			wantSource: "client.go:7",
		},
		{
			name:       "error",
			line:       "E0115 10:30:00.123456    1234 client.go:9] request failed\n",
			wantLevel:  clog.LevelError,
			wantMsg:    "request failed",
			wantSource: "client.go:9",
		},
		{
			name:       "fatal",
			line:       "F0115 10:30:00.123456    1234 main.go:1] giving up\n",
			wantLevel:  clog.LevelFatal,
			wantMsg:    "giving up",
			wantSource: "main.go:1",
		},
		{
			name:       "message containing the header terminator",
			line:       "I0115 10:30:00.123456    1234 server.go:42] got [1] result\n",
			wantLevel:  clog.LevelInfo,
			wantMsg:    "got [1] result",
			wantSource: "server.go:42",
		},
		{
			name:       "path containing the header terminator",
			line:       "I0115 10:30:00.123456    1234 odd] dir/server.go:42] listening\n",
			wantLevel:  clog.LevelInfo,
			wantMsg:    "listening",
			wantSource: "odd] dir/server.go:42",
		},
		{
			name:       "trailing blank line is part of the message",
			line:       "I0115 10:30:00.123456    1234 server.go:42] listening\n\n",
			wantLevel:  clog.LevelInfo,
			wantMsg:    "listening\n",
			wantSource: "server.go:42",
		},
		{
			name:      "no header, as under -skip_headers",
			line:      "listening\n",
			wantLevel: clog.LevelInfo,
			wantMsg:   "listening",
		},
		{
			name:      "header-shaped message with non-digit fields",
			line:      "I0x15 10:30:00.12ab56    1234 server.go:42] listening",
			wantLevel: clog.LevelInfo,
			wantMsg:   "I0x15 10:30:00.12ab56    1234 server.go:42] listening",
		},
		{
			name:      "long line that is not a header",
			line:      "this message is longer than a klog header but has no header\n",
			wantLevel: clog.LevelInfo,
			wantMsg:   "this message is longer than a klog header but has no header",
		},
		{
			name:      "unknown severity character",
			line:      "X0115 10:30:00.123456    1234 server.go:42] listening\n",
			wantLevel: clog.LevelInfo,
			wantMsg:   "X0115 10:30:00.123456    1234 server.go:42] listening",
		},
		{
			name:      "header without a terminator",
			line:      "I0115 10:30:00.123456    1234 truncated\n",
			wantLevel: clog.LevelInfo,
			wantMsg:   "truncated",
		},
		{
			name:      "empty",
			line:      "",
			wantLevel: clog.LevelInfo,
			wantMsg:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, msg, source := parseKlogLine(tt.line)

			assert.Equal(t, tt.wantLevel, level)
			assert.Equal(t, tt.wantMsg, msg)
			assert.Equal(t, tt.wantSource, source)
		})
	}
}

func TestParseKlogLinePaddedThreadID(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantLevel clog.Level
		wantMsg   string
	}{
		{
			name:      "unpadded wide thread id",
			line:      "W0115 10:30:00.123456 1234567 client.go:7] retrying",
			wantLevel: clog.LevelWarn,
			wantMsg:   "retrying",
		},
		{
			name:      "non-numeric thread id is not a header",
			line:      "W0115 10:30:00.123456    abcd client.go:7] retrying",
			wantLevel: clog.LevelInfo,
			wantMsg:   "W0115 10:30:00.123456    abcd client.go:7] retrying",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, msg, _ := parseKlogLine(tt.line)

			assert.Equal(t, tt.wantLevel, level)
			assert.Equal(t, tt.wantMsg, msg)
		})
	}
}
