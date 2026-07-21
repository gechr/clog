package widget

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatETA(t *testing.T) {
	tests := []struct {
		name string
		dur  time.Duration
		want string
	}{
		{"zero", 0, "1s"},
		{"sub_second", 500 * time.Millisecond, "1s"},
		{"one_second", time.Second, "1s"},
		{"five_seconds", 5 * time.Second, "5s"},
		{"thirty_seconds", 30 * time.Second, "30s"},
		{"fifty_nine_seconds", 59 * time.Second, "59s"},
		{"one_minute", time.Minute, "1m"},
		{"one_minute_thirty", time.Minute + 30*time.Second, "1m30s"},
		{"two_minutes", 2 * time.Minute, "2m"},
		{"two_minutes_thirty", 2*time.Minute + 30*time.Second, "2m30s"},
		{"one_hour", time.Hour, "1h"},
		{"one_hour_two_min", time.Hour + 2*time.Minute, "1h2m"},
		{"two_hours_thirty_min", 2*time.Hour + 30*time.Minute, "2h30m"},
		{"negative", -5 * time.Second, "5s"},
		{"rounds_to_nearest_second", 4*time.Second + 600*time.Millisecond, "5s"},
		{"rounds_down", 4*time.Second + 400*time.Millisecond, "4s"},
		{"rounds_up_to_one", 200 * time.Millisecond, "1s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatETA(tt.dur))
		})
	}
}
