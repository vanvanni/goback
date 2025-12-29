package helper

import (
	"fmt"
	"strings"
	"time"
)

func ParseDuration(duration string) (time.Duration, error) {
	s := strings.TrimSpace(duration)
	if s == "" {
		return 0, fmt.Errorf("duration is empty")
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", s, err)
	}

	if d <= 0 {
		return 0, fmt.Errorf("duration must be > 0, got %v", d)
	}

	secs := time.Duration(d.Seconds()) * time.Second
	if secs == 0 {
		return 0, fmt.Errorf("duration must be at least 1s, got %v", d)
	}

	return secs, nil
}
