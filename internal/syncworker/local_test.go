package syncworker

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sandbox0-ai/s0/internal/syncapi"
	"github.com/sandbox0-ai/s0/internal/syncignore"
	"github.com/sandbox0-ai/s0/internal/syncstate"
	syncsdk "github.com/sandbox0-ai/sdk-go"
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

func TestAuditWorkspaceCompatibilityRejectsWindowsReservedName(t *testing.T) {
	manifest := &syncstate.Manifest{
		Entries: map[string]syncstate.ManifestEntry{
			"docs/CON.txt": {Path: "docs/CON.txt", Kind: "file"},
		},
	}

	err := auditWorkspaceCompatibility(manifest, syncstate.FilesystemCaps{
		CaseSensitive:                   false,
		UnicodeNormalizationInsensitive: true,
		WindowsCompatiblePaths:          true,
	})
	if err == nil {
		t.Fatal("auditWorkspaceCompatibility() error = nil, want compatibility error")
	}
	var compatibilityErr *LocalPathCompatibilityError
	if !errors.As(err, &compatibilityErr) {
		t.Fatalf("auditWorkspaceCompatibility() error = %T, want LocalPathCompatibilityError", err)
	}
	if len(compatibilityErr.Issues) == 0 || compatibilityErr.Issues[0].Code != issueCodeWindowsReservedName {
		t.Fatalf("compatibility issues = %+v, want first code %q", compatibilityErr.Issues, issueCodeWindowsReservedName)
	}
}

func TestAuditWorkspaceCompatibilityRejectsCasefoldCollision(t *testing.T) {
	manifest := &syncstate.Manifest{
		Entries: map[string]syncstate.ManifestEntry{
			"docs/Readme.md": {Path: "docs/Readme.md", Kind: "file"},
			"docs/README.md": {Path: "docs/README.md", Kind: "file"},
		},
	}

	err := auditWorkspaceCompatibility(manifest, syncstate.FilesystemCaps{
		CaseSensitive:                   false,
		UnicodeNormalizationInsensitive: true,
	})
	if err == nil {
		t.Fatal("auditWorkspaceCompatibility() error = nil, want compatibility error")
	}
	var compatibilityErr *LocalPathCompatibilityError
	if !errors.As(err, &compatibilityErr) {
		t.Fatalf("auditWorkspaceCompatibility() error = %T, want LocalPathCompatibilityError", err)
	}
	if len(compatibilityErr.Issues) == 0 || compatibilityErr.Issues[0].Code != issueCodeCasefoldCollision {
		t.Fatalf("compatibility issues = %+v, want first code %q", compatibilityErr.Issues, issueCodeCasefoldCollision)
	}
}

func TestScanWorkspaceRejectsSymlinkWithExplicitPolicy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on Windows")
	}

	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(target, []byte("hello\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(target) error = %v", err)
	}
	if err := os.Symlink("target.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	_, err := scanWorkspace(root)
	if err == nil {
		t.Fatal("scanWorkspace() error = nil, want unsupported workspace entry")
	}
	var unsupported *UnsupportedWorkspaceEntryError
	if !errors.As(err, &unsupported) {
		t.Fatalf("scanWorkspace() error = %T, want UnsupportedWorkspaceEntryError", err)
	}
	if unsupported.Path != "link.txt" {
		t.Fatalf("unsupported.Path = %q, want %q", unsupported.Path, "link.txt")
	}
	if unsupported.EntryType != "symlink" {
		t.Fatalf("unsupported.EntryType = %q, want %q", unsupported.EntryType, "symlink")
	}
}

func TestExtractBootstrapArchiveRejectsSymlinkWithExplicitPolicy(t *testing.T) {
	root := t.TempDir()
	var buffer bytes.Buffer
	gzw := gzip.NewWriter(&buffer)
	tw := tar.NewWriter(gzw)
	if err := tw.WriteHeader(&tar.Header{
		Name:     "latest",
		Typeflag: tar.TypeSymlink,
		Linkname: "target.txt",
		Mode:     0o777,
	}); err != nil {
		t.Fatalf("WriteHeader(symlink) error = %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar.Close() error = %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("gzip.Close() error = %v", err)
	}

	err := extractBootstrapArchive(root, buffer.Bytes())
	if err == nil {
		t.Fatal("extractBootstrapArchive() error = nil, want unsupported bootstrap entry")
	}
	var unsupported *UnsupportedBootstrapEntryError
	if !errors.As(err, &unsupported) {
		t.Fatalf("extractBootstrapArchive() error = %T, want UnsupportedBootstrapEntryError", err)
	}
	if unsupported.Path != "latest" {
		t.Fatalf("unsupported.Path = %q, want %q", unsupported.Path, "latest")
	}
	if unsupported.EntryType != "symlink" {
		t.Fatalf("unsupported.EntryType = %q, want %q", unsupported.EntryType, "symlink")
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

	changed, err := applyRemoteChange(context.Background(), nil, "", root, apispec.SyncJournalEntry{
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

func TestApplyRemoteCreateDirectory(t *testing.T) {
	root := t.TempDir()

	changed, err := applyRemoteChange(context.Background(), nil, "", root, apispec.SyncJournalEntry{
		EventType: apispec.NewOptSyncEventType(apispec.SyncEventTypeCreate),
		Path:      apispec.NewOptString("docs"),
		EntryKind: apispec.NewOptNilSyncJournalEntryEntryKind(apispec.SyncJournalEntryEntryKindDirectory),
		Mode:      apispec.NewOptNilInt64(0o755),
	})
	if err != nil {
		t.Fatalf("applyRemoteChange() error = %v", err)
	}
	if !changed {
		t.Fatal("applyRemoteChange() changed = false, want true")
	}
	info, err := os.Stat(filepath.Join(root, "docs"))
	if err != nil {
		t.Fatalf("Stat(docs) error = %v", err)
	}
	if !info.IsDir() {
		t.Fatal("docs is not a directory")
	}
}

func TestApplyRemoteWriteDownloadsReplayPayload(t *testing.T) {
	root := t.TempDir()
	payload := []byte("hello from replay\n")
	sum := sha256.Sum256(payload)
	server := newReplayPayloadTestServer(t, payload)
	defer server.Close()

	client, err := syncsdk.NewClient(syncsdk.WithBaseURL(server.URL), syncsdk.WithToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	api := syncapi.New(client)

	changed, err := applyRemoteChange(context.Background(), api, "vol-test", root, apispec.SyncJournalEntry{
		EventType:     apispec.NewOptSyncEventType(apispec.SyncEventTypeWrite),
		Path:          apispec.NewOptString("docs/note.txt"),
		EntryKind:     apispec.NewOptNilSyncJournalEntryEntryKind(apispec.SyncJournalEntryEntryKindFile),
		Mode:          apispec.NewOptNilInt64(0o600),
		ContentRef:    apispec.NewOptNilString("sha256:" + hex.EncodeToString(sum[:])),
		ContentSHA256: apispec.NewOptNilString(hex.EncodeToString(sum[:])),
	})
	if err != nil {
		t.Fatalf("applyRemoteChange() error = %v", err)
	}
	if !changed {
		t.Fatal("applyRemoteChange() changed = false, want true")
	}
	if got := readFileContent(t, filepath.Join(root, "docs", "note.txt")); got != string(payload) {
		t.Fatalf("file content = %q, want %q", got, string(payload))
	}
	info, err := os.Stat(filepath.Join(root, "docs", "note.txt"))
	if err != nil {
		t.Fatalf("Stat(note.txt) error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %#o, want %#o", info.Mode().Perm(), 0o600)
	}
}

func TestApplyRemoteChmodUpdatesMode(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "docs", "note.txt")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(target, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	changed, err := applyRemoteChange(context.Background(), nil, "", root, apispec.SyncJournalEntry{
		EventType: apispec.NewOptSyncEventType(apispec.SyncEventTypeChmod),
		Path:      apispec.NewOptString("docs/note.txt"),
		Mode:      apispec.NewOptNilInt64(0o600),
	})
	if err != nil {
		t.Fatalf("applyRemoteChange() error = %v", err)
	}
	if !changed {
		t.Fatal("applyRemoteChange() changed = false, want true")
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("Stat(note.txt) error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %#o, want %#o", info.Mode().Perm(), 0o600)
	}
}

func TestApplyRemoteCreateRejectsUnsupportedRemoteEntryKind(t *testing.T) {
	root := t.TempDir()

	changed, err := applyRemoteChange(context.Background(), nil, "", root, apispec.SyncJournalEntry{
		EventType: apispec.NewOptSyncEventType(apispec.SyncEventTypeCreate),
		Path:      apispec.NewOptString("latest"),
		EntryKind: apispec.NewOptNilSyncJournalEntryEntryKind(apispec.SyncJournalEntryEntryKind("symlink")),
	})
	if err == nil {
		t.Fatal("applyRemoteChange() error = nil, want unsupported remote change")
	}
	if changed {
		t.Fatal("applyRemoteChange() changed = true, want false")
	}
	var unsupported *UnsupportedRemoteChangeError
	if !errors.As(err, &unsupported) {
		t.Fatalf("applyRemoteChange() error = %T, want UnsupportedRemoteChangeError", err)
	}
	if unsupported.EntryKind != "symlink" {
		t.Fatalf("unsupported.EntryKind = %q, want %q", unsupported.EntryKind, "symlink")
	}
}

func TestCompatibilityTrackerRejectsRemoteWindowsReservedName(t *testing.T) {
	tracker := newCompatibilityTracker(&syncstate.Manifest{Entries: map[string]syncstate.ManifestEntry{}}, syncstate.FilesystemCaps{
		CaseSensitive:                   false,
		UnicodeNormalizationInsensitive: true,
		WindowsCompatiblePaths:          true,
	})

	err := tracker.ValidateRemoteChange("docs/CON.txt", "", "create")
	if err == nil {
		t.Fatal("ValidateRemoteChange() error = nil, want compatibility error")
	}
	var compatibilityErr *LocalPathCompatibilityError
	if !errors.As(err, &compatibilityErr) {
		t.Fatalf("ValidateRemoteChange() error = %T, want LocalPathCompatibilityError", err)
	}
	if len(compatibilityErr.Issues) == 0 || compatibilityErr.Issues[0].Code != issueCodeWindowsReservedName {
		t.Fatalf("compatibility issues = %+v, want first code %q", compatibilityErr.Issues, issueCodeWindowsReservedName)
	}
}

func readFileContent(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(data)
}

func newReplayPayloadTestServer(t *testing.T, payload []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/sync/replay-payload"):
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
}
