package logging

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

var Log zerolog.Logger

func Init(level zerolog.Level, format string) {
	zerolog.TimeFieldFormat = time.RFC3339

	var out zerolog.Logger
	switch strings.ToLower(format) {
	case "json":
		out = zerolog.New(os.Stdout)
	default:
		out = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		})
	}

	out = out.Level(level)
	Log = out.With().Timestamp().Logger()
}

func parseLevel(l string) zerolog.Level {
	switch strings.ToLower(l) {
	case "debug":
		return zerolog.DebugLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}
