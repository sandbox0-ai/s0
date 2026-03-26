package syncworker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListWatchDirectoriesSkipsGitInternals(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git", "objects"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.git/objects) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "docs", "guides"), 0o755); err != nil {
		t.Fatalf("MkdirAll(docs/guides) error = %v", err)
	}

	directories, err := listWatchDirectories(root)
	if err != nil {
		t.Fatalf("listWatchDirectories() error = %v", err)
	}

	watched := make(map[string]struct{}, len(directories))
	for _, directory := range directories {
		watched[directory] = struct{}{}
	}

	assertWatchedDirectory(t, watched, root, true)
	assertWatchedDirectory(t, watched, filepath.Join(root, "docs"), true)
	assertWatchedDirectory(t, watched, filepath.Join(root, "docs", "guides"), true)
	assertWatchedDirectory(t, watched, filepath.Join(root, ".git"), false)
	assertWatchedDirectory(t, watched, filepath.Join(root, ".git", "objects"), false)
}

func TestWorkspaceWatcherSyncTracksDirectoryLifecycle(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll(docs) error = %v", err)
	}

	watcher, err := newWorkspaceWatcher(root)
	if err != nil {
		t.Fatalf("newWorkspaceWatcher() error = %v", err)
	}
	defer watcher.Close()

	assertWatchedDirectory(t, watcher.watched, root, true)
	assertWatchedDirectory(t, watcher.watched, filepath.Join(root, "docs"), true)

	if err := os.MkdirAll(filepath.Join(root, "notes", "daily"), 0o755); err != nil {
		t.Fatalf("MkdirAll(notes/daily) error = %v", err)
	}
	if err := watcher.Sync(); err != nil {
		t.Fatalf("watcher.Sync() after create error = %v", err)
	}
	assertWatchedDirectory(t, watcher.watched, filepath.Join(root, "notes"), true)
	assertWatchedDirectory(t, watcher.watched, filepath.Join(root, "notes", "daily"), true)

	if err := os.RemoveAll(filepath.Join(root, "docs")); err != nil {
		t.Fatalf("RemoveAll(docs) error = %v", err)
	}
	if err := watcher.Sync(); err != nil {
		t.Fatalf("watcher.Sync() after remove error = %v", err)
	}
	assertWatchedDirectory(t, watcher.watched, filepath.Join(root, "docs"), false)
}

func assertWatchedDirectory(t *testing.T, watched map[string]struct{}, path string, want bool) {
	t.Helper()
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("Abs(%s) error = %v", path, err)
	}
	_, got := watched[absolutePath]
	if got != want {
		t.Fatalf("watched[%s] = %v, want %v", absolutePath, got, want)
	}
}
