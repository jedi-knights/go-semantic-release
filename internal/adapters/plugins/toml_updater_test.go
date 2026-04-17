package plugins

import (
	"strings"
	"testing"
)

func TestUpdateTOMLKey_BasicSection(t *testing.T) {
	t.Parallel()
	input := `[tool.poetry]
name = "myproject"
version = "1.0.0"
description = "A package"
`
	got, err := updateTOMLKey([]byte(input), "tool.poetry.version", "2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(got), `version = "2.0.0"`) {
		t.Errorf("expected version = \"2.0.0\" in output, got:\n%s", got)
	}
	if strings.Contains(string(got), `version = "1.0.0"`) {
		t.Error("old version should be replaced")
	}
}

func TestUpdateTOMLKey_PreservesOtherContent(t *testing.T) {
	t.Parallel()
	input := `[tool.poetry]
name = "myproject"
version = "1.0.0"
description = "A package"
`
	got, err := updateTOMLKey([]byte(input), "tool.poetry.version", "2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gotStr := string(got)
	if !strings.Contains(gotStr, `name = "myproject"`) {
		t.Error("other keys should be preserved")
	}
	if !strings.Contains(gotStr, `description = "A package"`) {
		t.Error("other keys should be preserved")
	}
}

func TestUpdateTOMLKey_PreservesInlineComment(t *testing.T) {
	t.Parallel()
	input := `[tool.poetry]
version = "1.0.0"  # the version
`
	got, err := updateTOMLKey([]byte(input), "tool.poetry.version", "2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(got), `version = "2.0.0"  # the version`) {
		t.Errorf("inline comment should be preserved, got:\n%s", got)
	}
}

func TestUpdateTOMLKey_SectionNotFound(t *testing.T) {
	t.Parallel()
	input := `[other.section]
version = "1.0.0"
`
	_, err := updateTOMLKey([]byte(input), "tool.poetry.version", "2.0.0")
	if err == nil {
		t.Fatal("expected error when section not found")
	}
}

func TestUpdateTOMLKey_KeyNotInSection(t *testing.T) {
	t.Parallel()
	input := `[tool.poetry]
name = "myproject"
`
	_, err := updateTOMLKey([]byte(input), "tool.poetry.version", "2.0.0")
	if err == nil {
		t.Fatal("expected error when key not found in section")
	}
}

func TestUpdateTOMLKey_TopLevelKey(t *testing.T) {
	t.Parallel()
	input := `version = "1.0.0"
name = "myproject"
`
	got, err := updateTOMLKey([]byte(input), "version", "2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(got), `version = "2.0.0"`) {
		t.Errorf("expected version = \"2.0.0\", got:\n%s", got)
	}
}

func TestUpdateTOMLKey_OnlyUpdatesTargetSection(t *testing.T) {
	t.Parallel()
	input := "[tool.ruff]\nversion = \"1.0.0\"\n\n[tool.poetry]\nversion = \"1.0.0\"\n"
	got, err := updateTOMLKey([]byte(input), "tool.poetry.version", "2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gotStr := string(got)
	if !strings.Contains(gotStr, "[tool.ruff]\nversion = \"1.0.0\"") {
		t.Errorf("[tool.ruff] version should be unchanged, got:\n%s", gotStr)
	}
	if !strings.Contains(gotStr, "[tool.poetry]\nversion = \"2.0.0\"") {
		t.Errorf("expected version = \"2.0.0\" under [tool.poetry], got:\n%s", gotStr)
	}
}

func TestUpdateTOMLKey_IndentedValue(t *testing.T) {
	t.Parallel()
	input := `[tool.poetry]
    version = "1.0.0"
`
	got, err := updateTOMLKey([]byte(input), "tool.poetry.version", "2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(got), `version = "2.0.0"`) {
		t.Errorf("expected indented version to be updated, got:\n%s", got)
	}
}

func TestReplaceKeyValue_SimpleCase(t *testing.T) {
	t.Parallel()
	line := `version = "1.0.0"`
	got, ok := replaceKeyValue(line, "version", "2.0.0")
	if !ok {
		t.Fatal("expected match")
	}
	if got != `version = "2.0.0"` {
		t.Errorf("got %q, want %q", got, `version = "2.0.0"`)
	}
}

func TestReplaceKeyValue_WithTrailingComment(t *testing.T) {
	t.Parallel()
	line := `version = "1.0.0"  # pinned`
	got, ok := replaceKeyValue(line, "version", "2.0.0")
	if !ok {
		t.Fatal("expected match")
	}
	if got != `version = "2.0.0"  # pinned` {
		t.Errorf("got %q, want %q", got, `version = "2.0.0"  # pinned`)
	}
}

func TestReplaceKeyValue_NoMatch(t *testing.T) {
	t.Parallel()
	line := `name = "myproject"`
	_, ok := replaceKeyValue(line, "version", "2.0.0")
	if ok {
		t.Error("should not match a different key")
	}
}
