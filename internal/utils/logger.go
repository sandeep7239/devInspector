package utils

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type Level int

const (
	Info Level = iota
	Warn
	Error
)

type Logger struct {
	out   io.Writer
	level Level
}

func NewLogger(level string) *Logger {
	return &Logger{out: os.Stderr, level: parseLevel(level)}
}

func (l *Logger) Info(format string, args ...any)  { l.log(Info, "INFO", format, args...) }
func (l *Logger) Warn(format string, args ...any)  { l.log(Warn, "WARN", format, args...) }
func (l *Logger) Error(format string, args ...any) { l.log(Error, "ERROR", format, args...) }

func (l *Logger) log(level Level, label, format string, args ...any) {
	if level < l.level {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.out, "%s [%s] %s\n", time.Now().Format(time.RFC3339), label, msg)
}

func parseLevel(level string) Level {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "WARN":
		return Warn
	case "ERROR":
		return Error
	default:
		return Info
	}
}
