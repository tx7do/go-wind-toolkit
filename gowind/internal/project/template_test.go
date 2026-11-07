package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsLocalTemplateDir(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"existing absolute dir", dir, true},
		{"existing relative dir", ".", true}, // CWD 总是目录
		{"dot prefix nonexistent", "./no-such-dir", false},
		{"separator but missing", filepath.Join(dir, "missing"), false},
		{"bare name treated as repo", "go-wind-admin-template", false},
		{"https url treated as repo", "https://github.com/x/y.git", false},
		{"empty", "", false},
		{"file not dir", filepath.Join(dir, "file.txt"), false},
	}
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLocalTemplateDir(tt.input); got != tt.want {
				t.Fatalf("isLocalTemplateDir(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
