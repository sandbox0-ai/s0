package syncworker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sandbox0-ai/s0/internal/syncignore"
	"github.com/sandbox0-ai/s0/internal/syncstate"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestBuildUploadChangesRespectsGitignore(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("build/\n"), 0o600); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "build"), 0o755); err != nil {
		t.Fatalf("mkdir build: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "build", "artifact.txt"), []byte("ignored\n"), 0o600); err != nil {
		t.Fatalf("write ignored file: %v", err)
	}

	matcher, err := syncignore.Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	current, err := scanWorkspace(root)
	if err != nil {
		t.Fatalf("scanWorkspace() error = %v", err)
	}
	changes, err := buildUploadChanges(root, &syncstate.Manifest{Entries: map[string]syncstate.ManifestEntry{}}, current, matcher)
	if err != nil {
		t.Fatalf("buildUploadChanges() error = %v", err)
	}

	if len(changes) != 3 {
		t.Fatalf("len(changes) = %d, want 3", len(changes))
	}
	if got := changes[0].EventType; got != apispec.SyncEventTypeCreate {
		t.Fatalf("changes[0].EventType = %q, want create", got)
	}
	if got, _ := changes[0].Path.Get(); got != ".gitignore" {
		t.Fatalf("changes[0].Path = %q, want .gitignore", got)
	}
	if got, _ := changes[1].Path.Get(); got != "src" {
		t.Fatalf("changes[1].Path = %q, want src", got)
	}
	if got, _ := changes[2].Path.Get(); got != "src/main.go" {
		t.Fatalf("changes[2].Path = %q, want src/main.go", got)
	}
}

func TestBuildUploadChangesRespectsNestedGitignore(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "apps"), 0o755); err != nil {
		t.Fatalf("mkdir apps: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "apps", ".gitignore"), []byte("dist/\n"), 0o600); err != nil {
		t.Fatalf("write nested .gitignore: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "apps", "dist"), 0o755); err != nil {
		t.Fatalf("mkdir apps/dist: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "apps", "dist", "bundle.js"), []byte("ignored\n"), 0o600); err != nil {
		t.Fatalf("write ignored bundle: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "apps", "src"), 0o755); err != nil {
		t.Fatalf("mkdir apps/src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "apps", "src", "main.ts"), []byte("export {}\n"), 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	matcher, err := syncignore.Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	current, err := scanWorkspace(root)
	if err != nil {
		t.Fatalf("scanWorkspace() error = %v", err)
	}
	changes, err := buildUploadChanges(root, &syncstate.Manifest{Entries: map[string]syncstate.ManifestEntry{}}, current, matcher)
	if err != nil {
		t.Fatalf("buildUploadChanges() error = %v", err)
	}

	for _, change := range changes {
		if got, _ := change.Path.Get(); strings.HasPrefix(got, "apps/dist") {
			t.Fatalf("unexpected ignored path in upload changes: %q", got)
		}
	}
}

func TestApplyRemoteRename(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "old.txt"), []byte("hello\n"), 0o600); err != nil {
		t.Fatalf("write old file: %v", err)
	}

	changed, err := applyRemoteChange(root, apispec.SyncJournalEntry{
		EventType: apispec.NewOptSyncEventType(apispec.SyncEventTypeRename),
		Path:      apispec.NewOptString("docs/new.txt"),
		OldPath:   apispec.NewOptNilString("docs/old.txt"),
	})
	if err != nil {
		t.Fatalf("applyRemoteChange() error = %v", err)
	}
	if !changed {
		t.Fatalf("applyRemoteChange() changed = false, want true")
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "new.txt")); err != nil {
		t.Fatalf("new path missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("old path still exists or unexpected error: %v", err)
	}
}
