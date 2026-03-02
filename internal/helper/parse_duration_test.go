package helper

import (
	"strings"
	"testing"
	"time"
)

func TestParseIntervalValid(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Duration
	}{
		{name: "trimmed duration", input: " 90s ", want: 90 * time.Second},
		{name: "compound duration", input: "1m30s", want: 90 * time.Second},
		{name: "sub-second rounds down to second precision", input: "1500ms", want: 1 * time.Second},
	}

	for _, tc := range tests {
		got, err := ParseInterval(tc.input)
		if err != nil {
			t.Fatalf("%s: expected nil error, got %v", tc.name, err)
		}
		if got != tc.want {
			t.Fatalf("%s: expected %v, got %v", tc.name, tc.want, got)
		}
	}
}

func TestParseIntervalErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{name: "empty", input: " ", wantErr: "interval is empty"},
		{name: "invalid format", input: "abc", wantErr: "invalid interval"},
		{name: "zero", input: "0s", wantErr: "interval must be > 0"},
		{name: "negative", input: "-1s", wantErr: "interval must be > 0"},
		{name: "below one second", input: "500ms", wantErr: "interval must be at least 1s"},
	}

	for _, tc := range tests {
		_, err := ParseInterval(tc.input)
		if err == nil {
			t.Fatalf("%s: expected error", tc.name)
		}
		if !strings.Contains(err.Error(), tc.wantErr) {
			t.Fatalf("%s: expected error containing %q, got %v", tc.name, tc.wantErr, err)
		}
	}
}
