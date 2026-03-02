package helper

import (
	"regexp"
	"testing"
)

func TestSafeFilenameSanitizes(t *testing.T) {
	got := SafeFilename("a b/c?.txt")
	if got != "a_bc.txt" {
		t.Fatalf("expected sanitized file name %q, got %q", "a_bc.txt", got)
	}
}

func TestSafeFilenameTrimsDotsAndDashes(t *testing.T) {
	got := SafeFilename("..-backup-..")
	if got != "backup" {
		t.Fatalf("expected trimmed file name %q, got %q", "backup", got)
	}
}

func TestSafeFilenameFallbackTimestamp(t *testing.T) {
	got := SafeFilename("!!!")
	matched, err := regexp.MatchString(`^\d{8}-\d{6}$`, got)
	if err != nil {
		t.Fatalf("unexpected regex error: %v", err)
	}
	if !matched {
		t.Fatalf("expected timestamp-like fallback, got %q", got)
	}
}
