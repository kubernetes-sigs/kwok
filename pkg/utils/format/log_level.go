package format

import (
	"strconv"

	"golang.org/x/exp/slog"
)

// AbsString returns the absolute value of x.
func AbsStringifyLevel(level int) string {
	if level < 0 {
		level = -level
	}
	return strconv.Itoa(level)
}

func StringifyLevel(l int) string {
	level := slog.Level(l)
	switch {
	case level < slog.InfoLevel:
		return "debug"
	case level < slog.WarnLevel:
		return "info"
	case level < slog.ErrorLevel:
		return "warn"
	default:
		return "error"
	}
}
