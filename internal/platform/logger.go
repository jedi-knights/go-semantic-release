package platform

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance check.
var _ ports.Logger = (*ConsoleLogger)(nil)

// ConsoleLogger implements ports.Logger with formatted console output.
type ConsoleLogger struct {
	out   io.Writer
	level LogLevel
}

// LogLevel controls minimum log level.
type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarn
	LogError
)

// NewConsoleLogger creates a logger that writes to the given writer.
func NewConsoleLogger(out io.Writer, level LogLevel) *ConsoleLogger {
	return &ConsoleLogger{out: out, level: level}
}

// DefaultLogger creates a logger writing to stderr at info level.
func DefaultLogger() *ConsoleLogger {
	return NewConsoleLogger(os.Stderr, LogInfo)
}

func (l *ConsoleLogger) Debug(msg string, keysAndValues ...any) {
	if l.level <= LogDebug {
		l.log("DEBUG", msg, keysAndValues...)
	}
}

func (l *ConsoleLogger) Info(msg string, keysAndValues ...any) {
	if l.level <= LogInfo {
		l.log("INFO", msg, keysAndValues...)
	}
}

func (l *ConsoleLogger) Warn(msg string, keysAndValues ...any) {
	if l.level <= LogWarn {
		l.log("WARN", msg, keysAndValues...)
	}
}

func (l *ConsoleLogger) Error(msg string, keysAndValues ...any) {
	if l.level <= LogError {
		l.log("ERROR", msg, keysAndValues...)
	}
}

func (l *ConsoleLogger) log(level, msg string, keysAndValues ...any) {
	var sb strings.Builder
	_, _ = fmt.Fprintf(&sb, "[%s] %s", level, msg)

	for i := 0; i+1 < len(keysAndValues); i += 2 {
		_, _ = fmt.Fprintf(&sb, " %v=%v", keysAndValues[i], keysAndValues[i+1])
	}
	sb.WriteString("\n")

	_, _ = fmt.Fprint(l.out, sb.String())
}
