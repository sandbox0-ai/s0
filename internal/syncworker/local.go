package syncworker

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sandbox0-ai/s0/internal/syncignore"
	"github.com/sandbox0-ai/s0/internal/syncstate"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func scanWorkspace(root string) (*syncstate.Manifest, error) {
	manifest := &syncstate.Manifest{Entries: map[string]syncstate.ManifestEntry{}}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}

	err := filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if current == root {
			return nil
		}

		relative, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative == ".git" {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(relative, ".git/") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return unsupportedWorkspaceEntryError(relative, info.Mode())
		}

		item := syncstate.ManifestEntry{
			Path: relative,
			Mode: uint32(info.Mode().Perm()),
		}
		switch {
		case entry.IsDir():
			item.Kind = "directory"
		case info.Mode().IsRegular():
			sum, size, err := hashFile(current)
			if err != nil {
				return err
			}
			item.Kind = "file"
			item.Size = size
			item.SHA256 = sum
		default:
			return unsupportedWorkspaceEntryError(relative, info.Mode())
		}
		manifest.Entries[item.Path] = item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func buildUploadChanges(root string, previous, current *syncstate.Manifest, matcher *syncignore.Matcher) ([]apispec.ChangeRequest, error) {
	if previous == nil {
		previous = &syncstate.Manifest{Entries: map[string]syncstate.ManifestEntry{}}
	}
	if current == nil {
		current = &syncstate.Manifest{Entries: map[string]syncstate.ManifestEntry{}}
	}
	if previous.Entries == nil {
		previous.Entries = map[string]syncstate.ManifestEntry{}
	}
	if current.Entries == nil {
		current.Entries = map[string]syncstate.ManifestEntry{}
	}

	creates := make([]apispec.ChangeRequest, 0)
	writes := make([]apispec.ChangeRequest, 0)
	removes := make([]apispec.ChangeRequest, 0)

	allPaths := make([]string, 0, len(previous.Entries)+len(current.Entries))
	seen := map[string]struct{}{}
	for relative := range previous.Entries {
		allPaths = append(allPaths, relative)
		seen[relative] = struct{}{}
	}
	for relative := range current.Entries {
		if _, ok := seen[relative]; ok {
			continue
		}
		allPaths = append(allPaths, relative)
	}
	sort.Strings(allPaths)

	for _, relative := range allPaths {
		prev, hadPrev := previous.Entries[relative]
		curr, hasCurr := current.Entries[relative]
		isDir := hasCurr && curr.Kind == "directory"
		if !hasCurr {
			isDir = hadPrev && prev.Kind == "directory"
		}
		if matcher != nil && matcher.Match(relative, isDir) {
			continue
		}

		switch {
		case !hadPrev && hasCurr:
			change, err := createChange(root, curr)
			if err != nil {
				return nil, err
			}
			creates = append(creates, change)
		case hadPrev && !hasCurr:
			removes = append(removes, apispec.ChangeRequest{
				EventType: apispec.SyncEventTypeRemove,
				Path:      apispec.NewOptString(relative),
			})
		case hadPrev && hasCurr:
			if prev.Kind != curr.Kind {
				removes = append(removes, apispec.ChangeRequest{
					EventType: apispec.SyncEventTypeRemove,
					Path:      apispec.NewOptString(relative),
				})
				change, err := createChange(root, curr)
				if err != nil {
					return nil, err
				}
				creates = append(creates, change)
				continue
			}
			if curr.Kind == "file" && (prev.SHA256 != curr.SHA256 || prev.Size != curr.Size) {
				change, err := writeChange(root, curr)
				if err != nil {
					return nil, err
				}
				writes = append(writes, change)
			}
		}
	}

	sort.SliceStable(creates, func(i, j int) bool {
		left, _ := creates[i].Path.Get()
		right, _ := creates[j].Path.Get()
		return pathDepth(left) < pathDepth(right)
	})
	sort.SliceStable(removes, func(i, j int) bool {
		left, _ := removes[i].Path.Get()
		right, _ := removes[j].Path.Get()
		return pathDepth(left) > pathDepth(right)
	})

	changes := make([]apispec.ChangeRequest, 0, len(creates)+len(writes)+len(removes))
	changes = append(changes, creates...)
	changes = append(changes, writes...)
	changes = append(changes, removes...)
	return changes, nil
}

func createChange(root string, entry syncstate.ManifestEntry) (apispec.ChangeRequest, error) {
	change := apispec.ChangeRequest{
		EventType: apispec.SyncEventTypeCreate,
		Path:      apispec.NewOptString(entry.Path),
		Mode:      apispec.NewOptNilInt64(int64(entry.Mode)),
	}
	switch entry.Kind {
	case "directory":
		change.EntryKind = apispec.NewOptChangeRequestEntryKind(apispec.ChangeRequestEntryKindDirectory)
		return change, nil
	case "file":
		change.EntryKind = apispec.NewOptChangeRequestEntryKind(apispec.ChangeRequestEntryKindFile)
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(entry.Path)))
		if err != nil {
			return apispec.ChangeRequest{}, err
		}
		change.ContentBase64 = apispec.NewOptNilString(base64.StdEncoding.EncodeToString(content))
		change.ContentSHA256 = apispec.NewOptString(entry.SHA256)
		change.SizeBytes = apispec.NewOptInt64(entry.Size)
		return change, nil
	default:
		return apispec.ChangeRequest{}, &UnsupportedWorkspaceEntryError{
			Path:      entry.Path,
			EntryType: strings.TrimSpace(entry.Kind),
		}
	}
}

func writeChange(root string, entry syncstate.ManifestEntry) (apispec.ChangeRequest, error) {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(entry.Path)))
	if err != nil {
		return apispec.ChangeRequest{}, err
	}
	return apispec.ChangeRequest{
		EventType:     apispec.SyncEventTypeWrite,
		Path:          apispec.NewOptString(entry.Path),
		ContentBase64: apispec.NewOptNilString(base64.StdEncoding.EncodeToString(content)),
		ContentSHA256: apispec.NewOptString(entry.SHA256),
		SizeBytes:     apispec.NewOptInt64(entry.Size),
	}, nil
}

func clearWorkspaceForBootstrap(root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(root, 0o755)
		}
		return err
	}
	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(root, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func extractBootstrapArchive(root string, archive []byte) error {
	gzr, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		relative, err := sanitizeArchivePath(header.Name)
		if err != nil {
			return err
		}
		if relative == "" {
			continue
		}
		target := filepath.Join(root, filepath.FromSlash(relative))
		switch header.Typeflag {
		case tar.TypeDir:
			mode := os.FileMode(header.Mode) & os.ModePerm
			if mode == 0 {
				mode = 0o755
			}
			if err := os.MkdirAll(target, mode); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode)&os.ModePerm)
			if err != nil {
				return err
			}
			if _, err := io.Copy(file, tr); err != nil {
				file.Close()
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
		default:
			return unsupportedBootstrapEntryError(relative, header.Typeflag)
		}
	}
}

func sanitizeArchivePath(name string) (string, error) {
	clean := path.Clean(strings.TrimPrefix(strings.ReplaceAll(name, "\\", "/"), "/"))
	if clean == "." || clean == "" {
		return "", nil
	}
	if strings.HasPrefix(clean, "../") || clean == ".." {
		return "", fmt.Errorf("invalid archive entry %q", name)
	}
	return clean, nil
}

func hashFile(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	hasher := sha256.New()
	size, err := io.Copy(hasher, file)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hasher.Sum(nil)), size, nil
}

func pathDepth(value string) int {
	if value == "" {
		return 0
	}
	return strings.Count(value, "/")
}
