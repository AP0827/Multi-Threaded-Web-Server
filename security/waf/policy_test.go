package waf

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPatternsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "policy.txt")
	content := "# comment\n\nunion select\n<script\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write policy: %v", err)
	}

	patterns, err := LoadPatternsFile(path)
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	if len(patterns) != 2 {
		t.Fatalf("expected 2 patterns, got %d", len(patterns))
	}
	if patterns[0] != "union select" || patterns[1] != "<script" {
		t.Fatalf("unexpected patterns: %#v", patterns)
	}
}
