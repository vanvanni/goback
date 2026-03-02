package logging

import (
	"io"
	"os"
	"testing"

	"github.com/rs/zerolog"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  zerolog.Level
	}{
		{input: "debug", want: zerolog.DebugLevel},
		{input: "DEBUG", want: zerolog.DebugLevel},
		{input: "warn", want: zerolog.WarnLevel},
		{input: "error", want: zerolog.ErrorLevel},
		{input: "fatal", want: zerolog.FatalLevel},
		{input: "unknown", want: zerolog.InfoLevel},
	}

	for _, tc := range tests {
		if got := parseLevel(tc.input); got != tc.want {
			t.Fatalf("parseLevel(%q): expected %v, got %v", tc.input, tc.want, got)
		}
	}
}

func TestInitSetsLevel(t *testing.T) {
	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = originalStdout
		_ = w.Close()
		_, _ = io.Copy(io.Discard, r)
		_ = r.Close()
	}()

	Init(zerolog.DebugLevel, "json")
	if got := Log.GetLevel(); got != zerolog.DebugLevel {
		t.Fatalf("expected level %v, got %v", zerolog.DebugLevel, got)
	}

	Init(zerolog.WarnLevel, "console")
	if got := Log.GetLevel(); got != zerolog.WarnLevel {
		t.Fatalf("expected level %v, got %v", zerolog.WarnLevel, got)
	}
}
