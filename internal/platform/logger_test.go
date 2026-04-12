package platform_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/platform"
)

func TestConsoleLogger_Info_WritesMessage(t *testing.T) {
	var buf bytes.Buffer
	l := platform.NewConsoleLogger(&buf, platform.LogInfo)
	l.Info("hello world")

	out := buf.String()
	if !strings.Contains(out, "[INFO]") {
		t.Errorf("output %q should contain [INFO]", out)
	}
	if !strings.Contains(out, "hello world") {
		t.Errorf("output %q should contain message", out)
	}
}

func TestConsoleLogger_Debug_SuppressedAtInfoLevel(t *testing.T) {
	var buf bytes.Buffer
	l := platform.NewConsoleLogger(&buf, platform.LogInfo)
	l.Debug("should not appear")

	if buf.Len() != 0 {
		t.Errorf("Debug at info level should produce no output, got %q", buf.String())
	}
}

func TestConsoleLogger_Debug_AppearsAtDebugLevel(t *testing.T) {
	var buf bytes.Buffer
	l := platform.NewConsoleLogger(&buf, platform.LogDebug)
	l.Debug("debug message")

	out := buf.String()
	if !strings.Contains(out, "[DEBUG]") {
		t.Errorf("output %q should contain [DEBUG]", out)
	}
	if !strings.Contains(out, "debug message") {
		t.Errorf("output %q should contain message", out)
	}
}

func TestConsoleLogger_Warn_WritesMessage(t *testing.T) {
	var buf bytes.Buffer
	l := platform.NewConsoleLogger(&buf, platform.LogWarn)
	l.Warn("something is off")

	out := buf.String()
	if !strings.Contains(out, "[WARN]") {
		t.Errorf("output %q should contain [WARN]", out)
	}
}

func TestConsoleLogger_Error_WritesMessage(t *testing.T) {
	var buf bytes.Buffer
	l := platform.NewConsoleLogger(&buf, platform.LogError)
	l.Error("critical failure")

	out := buf.String()
	if !strings.Contains(out, "[ERROR]") {
		t.Errorf("output %q should contain [ERROR]", out)
	}
}

func TestConsoleLogger_KeyValuePairs_AppendedToOutput(t *testing.T) {
	var buf bytes.Buffer
	l := platform.NewConsoleLogger(&buf, platform.LogInfo)
	l.Info("event occurred", "key", "value", "count", 42)

	out := buf.String()
	if !strings.Contains(out, "key=value") {
		t.Errorf("output %q should contain key=value pair", out)
	}
	if !strings.Contains(out, "count=42") {
		t.Errorf("output %q should contain count=42 pair", out)
	}
}

func TestConsoleLogger_Warn_SuppressedAboveLevel(t *testing.T) {
	var buf bytes.Buffer
	l := platform.NewConsoleLogger(&buf, platform.LogError)
	l.Warn("suppressed warning")

	if buf.Len() != 0 {
		t.Errorf("Warn at error level should produce no output, got %q", buf.String())
	}
}
