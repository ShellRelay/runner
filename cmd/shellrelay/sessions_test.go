package main

import (
	"testing"
	"time"
)

func TestFmtDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{name: "zero", d: 0, want: "0s"},
		{name: "one second", d: 1 * time.Second, want: "1s"},
		{name: "499ms rounds down", d: 499 * time.Millisecond, want: "0s"},
		{name: "500ms rounds up", d: 500 * time.Millisecond, want: "1s"},
		{name: "1500ms rounds up", d: 1500 * time.Millisecond, want: "2s"},
		{name: "59 seconds", d: 59 * time.Second, want: "59s"},
		{name: "one minute", d: 60 * time.Second, want: "1m00s"},
		{name: "one minute one second", d: 61 * time.Second, want: "1m01s"},
		{name: "five minutes thirty", d: 5*time.Minute + 30*time.Second, want: "5m30s"},
		{name: "one hour", d: time.Hour, want: "1h00m00s"},
		{name: "one hour one min one sec", d: time.Hour + time.Minute + time.Second, want: "1h01m01s"},
		{name: "two hours thirty min", d: 2*time.Hour + 30*time.Minute, want: "2h30m00s"},
		{name: "large duration", d: 25*time.Hour + 59*time.Minute + 59*time.Second, want: "25h59m59s"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fmtDuration(tc.d)
			if got != tc.want {
				t.Errorf("fmtDuration(%v) = %q, want %q", tc.d, got, tc.want)
			}
		})
	}
}

func TestFmtSize(t *testing.T) {
	tests := []struct {
		name string
		b    int64
		want string
	}{
		{name: "zero bytes", b: 0, want: "0B"},
		{name: "one byte", b: 1, want: "1B"},
		{name: "999 bytes", b: 999, want: "999B"},
		{name: "1023 bytes", b: 1023, want: "1023B"},
		{name: "exactly 1KB", b: 1024, want: "1.0KB"},
		{name: "1.5KB", b: 1536, want: "1.5KB"},
		{name: "10KB", b: 10 * 1024, want: "10.0KB"},
		{name: "1023KB", b: 1023 * 1024, want: "1023.0KB"},
		{name: "exactly 1MB", b: 1 << 20, want: "1.0MB"},
		{name: "1.5MB", b: 3 * (1 << 19), want: "1.5MB"},
		{name: "100MB", b: 100 * (1 << 20), want: "100.0MB"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := fmtSize(tc.b)
			if got != tc.want {
				t.Errorf("fmtSize(%d) = %q, want %q", tc.b, got, tc.want)
			}
		})
	}
}
