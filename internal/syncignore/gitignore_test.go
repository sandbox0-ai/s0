package syncignore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatcher(t *testing.T) {
	root := t.TempDir()
	content := "" +
		"# comment\n" +
		"*.log\n" +
		"build/\n" +
		"/root-only.txt\n" +
		"!keep.log\n" +
		"nested/cache/\n"
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(content), 0o600); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	matcher, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	tests := []struct {
		path    string
		isDir   bool
		ignored bool
	}{
		{path: "debug.log", ignored: true},
		{path: "keep.log", ignored: false},
		{path: "build", isDir: true, ignored: true},
		{path: "build/output.txt", ignored: true},
		{path: "root-only.txt", ignored: true},
		{path: "nested/root-only.txt", ignored: false},
		{path: "nested/cache", isDir: true, ignored: true},
		{path: "nested/cache/data.bin", ignored: true},
		{path: "src/main.go", ignored: false},
	}

	for _, tt := range tests {
		if got := matcher.Match(tt.path, tt.isDir); got != tt.ignored {
			t.Fatalf("Match(%q, %t) = %t, want %t", tt.path, tt.isDir, got, tt.ignored)
		}
	}
}

func TestMatcherLoadsNestedGitignoreAndInfoExclude(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "apps", "web"), 0o755); err != nil {
		t.Fatalf("mkdir nested app: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".git", "info"), 0o755); err != nil {
		t.Fatalf("mkdir .git/info: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "apps", ".gitignore"), []byte("dist/\n!dist/keep.txt\n"), 0o600); err != nil {
		t.Fatalf("write nested .gitignore: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "info", "exclude"), []byte("tmp/\n"), 0o600); err != nil {
		t.Fatalf("write info exclude: %v", err)
	}

	matcher, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	tests := []struct {
		path    string
		isDir   bool
		ignored bool
	}{
		{path: "apps/dist", isDir: true, ignored: true},
		{path: "apps/dist/app.js", ignored: true},
		{path: "apps/dist/keep.txt", ignored: false},
		{path: "dist/app.js", ignored: false},
		{path: "tmp", isDir: true, ignored: true},
		{path: "tmp/cache.bin", ignored: true},
	}

	for _, tt := range tests {
		if got := matcher.Match(tt.path, tt.isDir); got != tt.ignored {
			t.Fatalf("Match(%q, %t) = %t, want %t", tt.path, tt.isDir, got, tt.ignored)
		}
	}
}
