package helper

import (
	"regexp"
	"strings"
	"time"
)

func SafeFilename(name string) string {
	safe := strings.ReplaceAll(name, " ", "_")
	reg := regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	safe = reg.ReplaceAllString(safe, "")
	safe = strings.Trim(safe, ".-")
	if safe == "" {
		safe = time.Now().Format("20060102-150405")
	}

	return safe
}
