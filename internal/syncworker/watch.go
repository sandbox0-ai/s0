package syncworker

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

type workspaceWatcher struct {
	root    string
	watcher *fsnotify.Watcher
	watched map[string]struct{}
}

func newWorkspaceWatcher(root string) (*workspaceWatcher, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		_ = watcher.Close()
		return nil, err
	}

	workspaceWatcher := &workspaceWatcher{
		root:    absoluteRoot,
		watcher: watcher,
		watched: map[string]struct{}{},
	}
	if err := workspaceWatcher.Sync(); err != nil {
		_ = watcher.Close()
		return nil, err
	}
	return workspaceWatcher, nil
}

func (w *workspaceWatcher) Close() error {
	if w == nil || w.watcher == nil {
		return nil
	}
	return w.watcher.Close()
}

func (w *workspaceWatcher) Events() <-chan fsnotify.Event {
	if w == nil || w.watcher == nil {
		return nil
	}
	return w.watcher.Events
}

func (w *workspaceWatcher) Errors() <-chan error {
	if w == nil || w.watcher == nil {
		return nil
	}
	return w.watcher.Errors
}

func (w *workspaceWatcher) HandleEvent(event fsnotify.Event) (bool, error) {
	if w == nil {
		return false, nil
	}
	if !w.shouldTrackPath(event.Name) {
		return false, nil
	}
	if event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
		if err := w.Sync(); err != nil {
			return false, err
		}
	}
	return shouldTriggerReconcile(event), nil
}

func (w *workspaceWatcher) Sync() error {
	if w == nil {
		return nil
	}
	directories, err := listWatchDirectories(w.root)
	if err != nil {
		return err
	}

	next := make(map[string]struct{}, len(directories))
	for _, directory := range directories {
		next[directory] = struct{}{}
		if _, ok := w.watched[directory]; ok {
			continue
		}
		if err := w.watcher.Add(directory); err != nil {
			return err
		}
	}

	for directory := range w.watched {
		if _, ok := next[directory]; ok {
			continue
		}
		if err := w.watcher.Remove(directory); err != nil && !errorsIsWatchMissing(err) {
			return err
		}
	}

	w.watched = next
	return nil
}

func listWatchDirectories(root string) ([]string, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absoluteRoot, 0o755); err != nil {
		return nil, err
	}

	directories := []string{absoluteRoot}
	err = filepath.WalkDir(absoluteRoot, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == absoluteRoot {
			return nil
		}
		if isGitInternalPath(absoluteRoot, current) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return filepath.SkipDir
		}
		directories = append(directories, current)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return directories, nil
}

func (w *workspaceWatcher) shouldTrackPath(path string) bool {
	if w == nil {
		return false
	}
	if strings.TrimSpace(path) == "" {
		return true
	}
	return !isGitInternalPath(w.root, path)
}

func isGitInternalPath(root, current string) bool {
	relative, err := filepath.Rel(root, current)
	if err != nil {
		return false
	}
	relative = filepath.ToSlash(relative)
	return relative == ".git" || strings.HasPrefix(relative, ".git/")
}

func shouldTriggerReconcile(event fsnotify.Event) bool {
	return event.Has(fsnotify.Create) ||
		event.Has(fsnotify.Write) ||
		event.Has(fsnotify.Remove) ||
		event.Has(fsnotify.Rename) ||
		event.Has(fsnotify.Chmod)
}

func errorsIsWatchMissing(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "can't remove non-existent watch") ||
		strings.Contains(message, "can't remove non-existent kqueue watch") ||
		strings.Contains(message, "no such file or directory")
}
