package tomledit

import "testing"

func TestReplaceKeyValue(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		key      string
		newValue string
		want     string
		wantOK   bool
	}{
		{"basic", `version = "1.0.0"`, "version", "1.2.3", `version = "1.2.3"`, true},
		{"indented", "    version = \"1.0.0\"", "version", "1.2.3", "    version = \"1.2.3\"", true},
		{"preserves inline comment", `version = "1.0.0" # bump me`, "version", "2.0.0", `version = "2.0.0" # bump me`, true},
		{"preserves tab indent", "\tname = \"vcl-core\"", "name", "x", "\tname = \"x\"", true},
		{"wrong key", `version = "1.0.0"`, "name", "x", `version = "1.0.0"`, false},
		{"extra spaces not matched", `version  =  "1.0.0"`, "version", "2.0.0", `version  =  "1.0.0"`, false},
		{"single quotes not matched", `version = '1.0.0'`, "version", "2.0.0", `version = '1.0.0'`, false},
		{"no spaces not matched", `version="1.0.0"`, "version", "2.0.0", `version="1.0.0"`, false},
		{"unterminated quote", `version = "1.0.0`, "version", "2.0.0", `version = "1.0.0`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ReplaceKeyValue(tt.line, tt.key, tt.newValue)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadKeyValue(t *testing.T) {
	tests := []struct {
		name   string
		line   string
		key    string
		want   string
		wantOK bool
	}{
		{"basic", `name = "vcl-core"`, "name", "vcl-core", true},
		{"indented", "  version = \"0.1.0\"", "version", "0.1.0", true},
		{"with comment", `name = "vcl-cli" # the cli`, "name", "vcl-cli", true},
		{"wrong key", `name = "vcl-core"`, "version", "", false},
		{"single quotes not matched", `name = 'vcl-core'`, "name", "", false},
		{"unterminated", `name = "vcl-core`, "name", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ReadKeyValue(tt.line, tt.key)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
