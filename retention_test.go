package main

import (
	"testing"
	"time"
)

func TestParseRetention(t *testing.T) {
	tests := []struct {
		in         string
		want       time.Duration
		wantErrStr string
	}{
		{"0s", 0, ""},
		{"1h", time.Hour, ""},
		{"1h30m", 90 * time.Minute, ""},
		{"7d", 7 * 24 * time.Hour, ""},
		{"2d", 48 * time.Hour, ""},
		{"1.5d", 36 * time.Hour, ""},
		{"d", 0, `invalid duration "d"`},
		{"banana", 0, `invalid duration "banana"`},
		{"2x", 0, `invalid duration "2x"`},
	}
	for _, tt := range tests {
		got, err := ParseRetention(tt.in)
		if tt.wantErrStr != "" {
			if err == nil || err.Error() != tt.wantErrStr {
				t.Errorf("ParseRetention(%q) error = %v, want %q", tt.in, err, tt.wantErrStr)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseRetention(%q) unexpected error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseRetention(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
