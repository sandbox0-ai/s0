package syncworker

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-faster/jx"
	"github.com/sandbox0-ai/s0/internal/syncapi"
	"github.com/sandbox0-ai/s0/internal/syncstate"
	syncsdk "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestReconcileOnceEndToEndThroughSDKHTTP(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	service := newFakeSyncService()
	server := httptest.NewServer(service)
	defer server.Close()

	client, err := syncsdk.NewClient(
		syncsdk.WithBaseURL(server.URL),
		syncsdk.WithToken("test-token"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	api := syncapi.New(client)

	workspace := filepath.Join(home, "work")
	attachment, err := syncstate.NewAttachment(workspace, service.volumeID, "work", "volume")
	if err != nil {
		t.Fatalf("NewAttachment() error = %v", err)
	}
	if err := syncstate.SaveAttachment(attachment); err != nil {
		t.Fatalf("SaveAttachment() error = %v", err)
	}

	logger := log.New(io.Discard, "", 0)
	ctx := context.Background()

	if err := reconcileOnce(ctx, api, attachment.ID, logger, true); err != nil {
		t.Fatalf("initial reconcileOnce() error = %v", err)
	}
	readmePath := filepath.Join(workspace, "README.md")
	if got := readFile(t, readmePath); got != "hello from volume\n" {
		t.Fatalf("README after bootstrap = %q", got)
	}

	if err := os.WriteFile(readmePath, []byte("hello from local\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(local README) error = %v", err)
	}
	if err := reconcileOnce(ctx, api, attachment.ID, logger, false); err != nil {
		t.Fatalf("reconcileOnce(local upload) error = %v", err)
	}
	if got := service.fileContent("README.md"); got != "hello from local\n" {
		t.Fatalf("remote README after local upload = %q", got)
	}

	service.addSandboxWrite("README.md", "hello from sandbox\n")
	if err := reconcileOnce(ctx, api, attachment.ID, logger, false); err != nil {
		t.Fatalf("reconcileOnce(sandbox replay) error = %v", err)
	}
	if got := readFile(t, readmePath); got != "hello from sandbox\n" {
		t.Fatalf("README after sandbox replay = %q", got)
	}

	service.setConflictOnNextPath("README.md")
	if err := os.WriteFile(readmePath, []byte("conflicted locally\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(conflicted README) error = %v", err)
	}
	if err := reconcileOnce(ctx, api, attachment.ID, logger, false); err != nil {
		t.Fatalf("reconcileOnce(conflict) error = %v", err)
	}

	attachment, err = syncstate.LoadAttachmentByID(attachment.ID)
	if err != nil {
		t.Fatalf("LoadAttachmentByID() error = %v", err)
	}
	if attachment.LastSync == nil || attachment.LastSync.OpenConflictCount != 1 {
		t.Fatalf("OpenConflictCount = %+v, want 1", attachment.LastSync)
	}

	conflicts, err := api.ListConflicts(ctx, attachment.VolumeID, "open", 256)
	if err != nil {
		t.Fatalf("ListConflicts() error = %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("len(conflicts) = %d, want 1", len(conflicts))
	}
	conflictID, ok := conflicts[0].ID.Get()
	if !ok || conflictID == "" {
		t.Fatalf("conflict id missing: %+v", conflicts[0])
	}
	resolved, err := api.ResolveConflict(ctx, attachment.VolumeID, conflictID, false)
	if err != nil {
		t.Fatalf("ResolveConflict() error = %v", err)
	}
	status, _ := resolved.Status.Get()
	if status != "resolved" {
		t.Fatalf("resolved.Status = %q, want resolved", status)
	}
	if _, err := refreshOpenConflictCount(ctx, api, attachment.ID, attachment.VolumeID); err != nil {
		t.Fatalf("refreshOpenConflictCount() error = %v", err)
	}

	attachment, err = syncstate.LoadAttachmentByID(attachment.ID)
	if err != nil {
		t.Fatalf("LoadAttachmentByID() error = %v", err)
	}
	if attachment.LastSync.OpenConflictCount != 0 {
		t.Fatalf("OpenConflictCount after resolve = %d, want 0", attachment.LastSync.OpenConflictCount)
	}
}

func TestReconcileOnceReplaysSandboxCreateWriteChmodWithoutBootstrapFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	service := newFakeSyncService()
	server := httptest.NewServer(service)
	defer server.Close()

	client, err := syncsdk.NewClient(
		syncsdk.WithBaseURL(server.URL),
		syncsdk.WithToken("test-token"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	api := syncapi.New(client)

	workspace := filepath.Join(home, "work")
	attachment, err := syncstate.NewAttachment(workspace, service.volumeID, "work", "volume")
	if err != nil {
		t.Fatalf("NewAttachment() error = %v", err)
	}
	if err := syncstate.SaveAttachment(attachment); err != nil {
		t.Fatalf("SaveAttachment() error = %v", err)
	}

	logger := log.New(io.Discard, "", 0)
	ctx := context.Background()

	if err := reconcileOnce(ctx, api, attachment.ID, logger, true); err != nil {
		t.Fatalf("initial reconcileOnce() error = %v", err)
	}

	service.addSandboxCreateWriteChmod("docs/note.txt", "sandbox payload\n", 0o644, 0o600)
	if err := reconcileOnce(ctx, api, attachment.ID, logger, false); err != nil {
		t.Fatalf("reconcileOnce(sandbox create/write/chmod) error = %v", err)
	}

	target := filepath.Join(workspace, "docs", "note.txt")
	if got := readFile(t, target); got != "sandbox payload\n" {
		t.Fatalf("note.txt content = %q, want %q", got, "sandbox payload\n")
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("Stat(note.txt) error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("note.txt mode = %#o, want %#o", info.Mode().Perm(), 0o600)
	}

	attachment, err = syncstate.LoadAttachmentByID(attachment.ID)
	if err != nil {
		t.Fatalf("LoadAttachmentByID() error = %v", err)
	}
	if attachment.LastSync.LastAppliedSeq != service.headSeq {
		t.Fatalf("LastAppliedSeq = %d, want %d", attachment.LastSync.LastAppliedSeq, service.headSeq)
	}
}

func TestRunUploadsLocalChangesFromFilesystemWatch(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	service := newFakeSyncService()
	server := httptest.NewServer(service)
	defer server.Close()

	client, err := syncsdk.NewClient(
		syncsdk.WithBaseURL(server.URL),
		syncsdk.WithToken("test-token"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	workspace := filepath.Join(home, "work")
	attachment, err := syncstate.NewAttachment(workspace, service.volumeID, "work", "volume")
	if err != nil {
		t.Fatalf("NewAttachment() error = %v", err)
	}
	if err := syncstate.SaveAttachment(attachment); err != nil {
		t.Fatalf("SaveAttachment() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx, client, attachment.ID, "foreground", io.Discard)
	}()

	waitForCondition(t, 2*time.Second, func() bool {
		return service.fileContent("README.md") == "hello from volume\n"
	})

	readmePath := filepath.Join(workspace, "README.md")
	waitForCondition(t, 2*time.Second, func() bool {
		data, err := os.ReadFile(readmePath)
		return err == nil && string(data) == "hello from volume\n"
	})

	if err := os.WriteFile(readmePath, []byte("hello from watch\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(README.md) error = %v", err)
	}
	waitForCondition(t, 2*time.Second, func() bool {
		return service.fileContent("README.md") == "hello from watch\n"
	})

	nestedDir := filepath.Join(workspace, "notes")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(notes) error = %v", err)
	}
	nestedFile := filepath.Join(nestedDir, "todo.txt")
	if err := os.WriteFile(nestedFile, []byte("v1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(todo.txt) error = %v", err)
	}
	waitForCondition(t, 2*time.Second, func() bool {
		return service.fileContent("notes/todo.txt") == "v1\n"
	})

	if err := os.WriteFile(nestedFile, []byte("v2\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(todo.txt second update) error = %v", err)
	}
	waitForCondition(t, 2*time.Second, func() bool {
		return service.fileContent("notes/todo.txt") == "v2\n"
	})

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not stop after context cancellation")
	}
}

func TestReconcileOnceReturnsExplicitErrorForUnsupportedBootstrapEntry(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	service := newFakeSyncService()
	server := httptest.NewServer(service)
	defer server.Close()

	client, err := syncsdk.NewClient(
		syncsdk.WithBaseURL(server.URL),
		syncsdk.WithToken("test-token"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	api := syncapi.New(client)

	workspace := filepath.Join(home, "work")
	attachment, err := syncstate.NewAttachment(workspace, service.volumeID, "work", "volume")
	if err != nil {
		t.Fatalf("NewAttachment() error = %v", err)
	}
	if err := syncstate.SaveAttachment(attachment); err != nil {
		t.Fatalf("SaveAttachment() error = %v", err)
	}

	logger := log.New(io.Discard, "", 0)
	ctx := context.Background()

	service.setBootstrapArchive(buildUnsupportedSymlinkArchive(t))
	err = reconcileOnce(ctx, api, attachment.ID, logger, true)
	if err == nil {
		t.Fatal("reconcileOnce() error = nil, want unsupported bootstrap entry")
	}
	var unsupported *UnsupportedBootstrapEntryError
	if !errors.As(err, &unsupported) {
		t.Fatalf("reconcileOnce() error = %T, want UnsupportedBootstrapEntryError", err)
	}
	if unsupported.EntryType != "symlink" {
		t.Fatalf("unsupported.EntryType = %q, want %q", unsupported.EntryType, "symlink")
	}
	if _, statErr := os.Lstat(filepath.Join(workspace, "README.md")); !os.IsNotExist(statErr) {
		t.Fatalf("workspace README.md exists or unexpected error: %v", statErr)
	}
}

func TestReconcileOnceRejectsRemoteWindowsIncompatiblePathForWindowsReplica(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	service := newFakeSyncService()
	server := httptest.NewServer(service)
	defer server.Close()

	client, err := syncsdk.NewClient(
		syncsdk.WithBaseURL(server.URL),
		syncsdk.WithToken("test-token"),
	)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	api := syncapi.New(client)

	workspace := filepath.Join(home, "work")
	attachment, err := syncstate.NewAttachment(workspace, service.volumeID, "work", "volume")
	if err != nil {
		t.Fatalf("NewAttachment() error = %v", err)
	}
	attachment.Platform = "windows"
	attachment.Capabilities = syncstate.FilesystemCaps{
		CaseSensitive:                   false,
		UnicodeNormalizationInsensitive: true,
		WindowsCompatiblePaths:          true,
	}
	if err := syncstate.SaveAttachment(attachment); err != nil {
		t.Fatalf("SaveAttachment() error = %v", err)
	}

	logger := log.New(io.Discard, "", 0)
	ctx := context.Background()

	if err := reconcileOnce(ctx, api, attachment.ID, logger, true); err != nil {
		t.Fatalf("initial reconcileOnce() error = %v", err)
	}

	service.addSandboxWrite("docs/CON.txt", "sandbox payload\n")
	err = reconcileOnce(ctx, api, attachment.ID, logger, false)
	if err == nil {
		t.Fatal("reconcileOnce() error = nil, want compatibility error")
	}
	var compatibilityErr *LocalPathCompatibilityError
	if !errors.As(err, &compatibilityErr) {
		t.Fatalf("reconcileOnce() error = %T, want LocalPathCompatibilityError", err)
	}
	if len(compatibilityErr.Issues) == 0 || compatibilityErr.Issues[0].Code != issueCodeWindowsReservedName {
		t.Fatalf("compatibility issues = %+v, want first code %q", compatibilityErr.Issues, issueCodeWindowsReservedName)
	}
	if _, statErr := os.Lstat(filepath.Join(workspace, "docs", "CON.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("workspace incompatible path exists or unexpected error: %v", statErr)
	}
}

type fakeSyncService struct {
	mu               sync.Mutex
	volumeID         string
	teamID           string
	replicas         map[string]apispec.SyncReplica
	files            map[string]string
	journal          []apispec.SyncJournalEntry
	replayPayloads   map[string][]byte
	bootstrapArchive []byte
	conflicts        map[string]apispec.SyncConflict
	headSeq          int64
	conflictNextPath string
}

func newFakeSyncService() *fakeSyncService {
	return &fakeSyncService{
		volumeID: "vol-test",
		teamID:   "team-test",
		replicas: map[string]apispec.SyncReplica{},
		files: map[string]string{
			"README.md": "hello from volume\n",
		},
		replayPayloads: map[string][]byte{},
		conflicts:      map[string]apispec.SyncConflict{},
	}
}

func (s *fakeSyncService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/v1/sandboxvolumes/"+s.volumeID+"/sync/") {
		http.NotFound(w, r)
		return
	}

	switch {
	case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/sync/replicas/") && strings.HasSuffix(r.URL.Path, "/cursor"):
		s.handleUpdateCursor(w, r)
	case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/sync/replicas/") && !strings.HasSuffix(r.URL.Path, "/cursor"):
		s.handleUpsertReplica(w, r)
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/sync/bootstrap"):
		s.handleBootstrap(w, r)
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/sync/bootstrap/archive"):
		s.handleBootstrapArchive(w, r)
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/sync/changes"):
		s.handleListChanges(w, r)
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/sync/replay-payload"):
		s.handleReplayPayload(w, r)
	case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/sync/replicas/") && strings.HasSuffix(r.URL.Path, "/changes"):
		s.handleAppendChanges(w, r)
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/sync/conflicts"):
		s.handleListConflicts(w, r)
	case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/sync/conflicts/"):
		s.handleResolveConflict(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *fakeSyncService) handleUpsertReplica(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	replicaID := path.Base(r.URL.Path)
	var req apispec.UpsertSyncReplicaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	now := time.Now().UTC()
	replica := apispec.SyncReplica{
		ID:             apispec.NewOptString(replicaID),
		VolumeID:       apispec.NewOptString(s.volumeID),
		TeamID:         apispec.NewOptString(s.teamID),
		DisplayName:    req.DisplayName,
		Platform:       req.Platform,
		RootPath:       req.RootPath,
		CaseSensitive:  req.CaseSensitive,
		Capabilities:   req.Capabilities,
		LastSeenAt:     apispec.NewOptDateTime(now),
		LastAppliedSeq: apispec.NewOptInt64(0),
		CreatedAt:      apispec.NewOptDateTime(now),
		UpdatedAt:      apispec.NewOptDateTime(now),
	}
	if existing, ok := s.replicas[replicaID]; ok {
		replica.LastAppliedSeq = existing.LastAppliedSeq
		replica.CreatedAt = existing.CreatedAt
	}
	s.replicas[replicaID] = replica

	writeJSON(w, http.StatusOK, apispec.SuccessVolumeSyncReplicaResponse{
		Success: apispec.SuccessVolumeSyncReplicaResponseSuccessTrue,
		Data: apispec.NewOptVolumeSyncReplicaEnvelope(apispec.VolumeSyncReplicaEnvelope{
			Replica: replica,
			HeadSeq: s.headSeq,
		}),
	})
}

func (s *fakeSyncService) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	writeJSON(w, http.StatusCreated, apispec.SuccessVolumeSyncBootstrapResponse{
		Success: apispec.SuccessVolumeSyncBootstrapResponseSuccessTrue,
		Data: apispec.NewOptVolumeSyncBootstrap(apispec.VolumeSyncBootstrap{
			Snapshot: apispec.Snapshot{
				ID:        "snap-1",
				VolumeID:  s.volumeID,
				Name:      "bootstrap",
				SizeBytes: int64(len(s.files)),
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
			},
			ReplayAfterSeq:      s.headSeq,
			ArchiveDownloadPath: "/api/v1/sandboxvolumes/" + s.volumeID + "/sync/bootstrap/archive?snapshot_id=snap-1",
		}),
	})
}

func (s *fakeSyncService) handleBootstrapArchive(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.bootstrapArchive) > 0 {
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(s.bootstrapArchive)
		return
	}

	archive, err := buildArchive(s.files)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/gzip")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(archive)
}

func (s *fakeSyncService) handleListChanges(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 256
	}
	changes := make([]apispec.SyncJournalEntry, 0, limit)
	for _, entry := range s.journal {
		seq, _ := entry.Seq.Get()
		if seq <= after {
			continue
		}
		changes = append(changes, entry)
		if len(changes) >= limit {
			break
		}
	}

	writeJSON(w, http.StatusOK, apispec.SuccessVolumeSyncChangeListResponse{
		Success: apispec.SuccessVolumeSyncChangeListResponseSuccessTrue,
		Data: apispec.NewOptListVolumeSyncChangesResponse(apispec.ListVolumeSyncChangesResponse{
			HeadSeq:          s.headSeq,
			RetainedAfterSeq: 0,
			Changes:          changes,
		}),
	})
}

func (s *fakeSyncService) handleAppendChanges(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	replicaID := path.Base(path.Dir(r.URL.Path))
	var req apispec.AppendReplicaChangesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	accepted := make([]apispec.SyncJournalEntry, 0, len(req.Changes))
	conflicts := make([]apispec.SyncConflict, 0, 1)
	for _, change := range req.Changes {
		changePath, _ := change.Path.Get()
		if s.conflictNextPath != "" && changePath == s.conflictNextPath {
			conflict := apispec.SyncConflict{
				ID:             apispec.NewOptString("conflict-1"),
				VolumeID:       apispec.NewOptString(s.volumeID),
				TeamID:         apispec.NewOptString(s.teamID),
				ReplicaID:      apispec.NewOptNilString(replicaID),
				Path:           apispec.NewOptString(changePath),
				NormalizedPath: apispec.NewOptString(strings.ToLower(changePath)),
				ArtifactPath:   apispec.NewOptString(changePath + ".sandbox0-conflict-" + replicaID),
				IncomingPath:   apispec.NewOptNilString(changePath),
				Reason:         apispec.NewOptString("concurrent_update"),
				Status:         apispec.NewOptString("open"),
				ExistingSeq:    apispec.NewOptNilInt64(s.headSeq),
				Metadata: apispec.NewOptNilSyncConflictMetadata(apispec.SyncConflictMetadata{
					"latest_event":  jx.Raw(`"write"`),
					"latest_path":   jx.Raw(strconv.Quote(changePath)),
					"latest_source": jx.Raw(`"sandbox"`),
				}),
				CreatedAt: apispec.NewOptDateTime(time.Now().UTC()),
				UpdatedAt: apispec.NewOptDateTime(time.Now().UTC()),
			}
			s.conflicts["conflict-1"] = conflict
			s.conflictNextPath = ""
			conflicts = append(conflicts, conflict)
			continue
		}

		entry := s.appendJournalEntryLocked(replicaID, apispec.SyncJournalEntrySourceReplica, change)
		accepted = append(accepted, entry)
	}

	writeJSON(w, http.StatusOK, apispec.SuccessVolumeSyncAppendResponse{
		Success: apispec.SuccessVolumeSyncAppendResponseSuccessTrue,
		Data: apispec.NewOptAppendReplicaChangesResponse(apispec.AppendReplicaChangesResponse{
			HeadSeq:   s.headSeq,
			Accepted:  accepted,
			Conflicts: conflicts,
		}),
	})
}

func (s *fakeSyncService) handleReplayPayload(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	contentRef := strings.TrimSpace(r.URL.Query().Get("content_ref"))
	payload, ok := s.replayPayloads[contentRef]
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

func (s *fakeSyncService) handleUpdateCursor(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	replicaID := path.Base(path.Dir(r.URL.Path))
	var req apispec.UpdateSyncReplicaCursorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	replica := s.replicas[replicaID]
	replica.LastAppliedSeq = apispec.NewOptInt64(req.LastAppliedSeq)
	replica.LastSeenAt = apispec.NewOptDateTime(time.Now().UTC())
	s.replicas[replicaID] = replica

	writeJSON(w, http.StatusOK, apispec.SuccessVolumeSyncReplicaResponse{
		Success: apispec.SuccessVolumeSyncReplicaResponseSuccessTrue,
		Data: apispec.NewOptVolumeSyncReplicaEnvelope(apispec.VolumeSyncReplicaEnvelope{
			Replica: replica,
			HeadSeq: s.headSeq,
		}),
	})
}

func (s *fakeSyncService) handleListConflicts(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
	conflicts := make([]apispec.SyncConflict, 0, len(s.conflicts))
	for _, conflict := range s.conflicts {
		status, _ := conflict.Status.Get()
		if statusFilter != "" && status != statusFilter {
			continue
		}
		conflicts = append(conflicts, conflict)
	}
	writeJSON(w, http.StatusOK, apispec.SuccessVolumeSyncConflictListResponse{
		Success: apispec.SuccessVolumeSyncConflictListResponseSuccessTrue,
		Data: apispec.NewOptListVolumeSyncConflictsResponse(apispec.ListVolumeSyncConflictsResponse{
			Conflicts: conflicts,
		}),
	})
}

func (s *fakeSyncService) handleResolveConflict(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	conflictID := path.Base(r.URL.Path)
	conflict, ok := s.conflicts[conflictID]
	if !ok {
		http.NotFound(w, r)
		return
	}

	var req apispec.ResolveVolumeSyncConflictRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	conflict.Status = apispec.NewOptString(string(req.Status))
	conflict.UpdatedAt = apispec.NewOptDateTime(time.Now().UTC())
	metadata, _ := conflict.Metadata.Get()
	if metadata == nil {
		metadata = apispec.SyncConflictMetadata{}
	}
	if resolution, ok := req.Resolution.Get(); ok && strings.TrimSpace(resolution) != "" {
		metadata["resolution"] = jx.Raw(strconv.Quote(resolution))
	}
	if note, ok := req.Note.Get(); ok && strings.TrimSpace(note) != "" {
		metadata["note"] = jx.Raw(strconv.Quote(note))
	}
	metadata["resolved_at"] = jx.Raw(strconv.Quote(time.Now().UTC().Format(time.RFC3339)))
	conflict.Metadata = apispec.NewOptNilSyncConflictMetadata(metadata)
	s.conflicts[conflictID] = conflict

	writeJSON(w, http.StatusOK, apispec.SuccessVolumeSyncConflictResponse{
		Success: apispec.SuccessVolumeSyncConflictResponseSuccessTrue,
		Data:    apispec.NewOptSyncConflict(conflict),
	})
}

func (s *fakeSyncService) appendJournalEntryLocked(replicaID string, source apispec.SyncJournalEntrySource, change apispec.ChangeRequest) apispec.SyncJournalEntry {
	s.headSeq++
	eventType := change.EventType
	changePath, _ := change.Path.Get()
	oldPath, _ := change.OldPath.Get()
	content, _ := change.ContentBase64.Get()
	entryKind, _ := change.EntryKind.Get()
	mode, hasMode := change.Mode.Get()
	contentSHA256, _ := change.ContentSHA256.Get()
	sizeBytes, hasSize := change.SizeBytes.Get()
	var contentRef string

	switch string(eventType) {
	case string(apispec.SyncEventTypeCreate), string(apispec.SyncEventTypeWrite):
		payload, _ := base64.StdEncoding.DecodeString(content)
		s.files[changePath] = string(payload)
		if len(payload) > 0 {
			if contentSHA256 == "" {
				sum := sha256.Sum256(payload)
				contentSHA256 = hex.EncodeToString(sum[:])
			}
			if !hasSize {
				sizeBytes = int64(len(payload))
				hasSize = true
			}
			contentRef = "sha256:" + contentSHA256
			s.replayPayloads[contentRef] = append([]byte(nil), payload...)
		}
	case string(apispec.SyncEventTypeRemove):
		delete(s.files, changePath)
	case string(apispec.SyncEventTypeRename):
		s.files[changePath] = s.files[oldPath]
		delete(s.files, oldPath)
	}

	entry := apispec.SyncJournalEntry{
		Seq:            apispec.NewOptInt64(s.headSeq),
		VolumeID:       apispec.NewOptString(s.volumeID),
		TeamID:         apispec.NewOptString(s.teamID),
		Source:         apispec.NewOptSyncJournalEntrySource(source),
		ReplicaID:      apispec.NewOptNilString(replicaID),
		EventType:      apispec.NewOptSyncEventType(eventType),
		Path:           apispec.NewOptString(changePath),
		NormalizedPath: apispec.NewOptString(strings.ToLower(changePath)),
		Tombstone:      apispec.NewOptBool(eventType == apispec.SyncEventTypeRemove),
		CreatedAt:      apispec.NewOptDateTime(time.Now().UTC()),
	}
	if entryKind != "" {
		entry.EntryKind = apispec.NewOptNilSyncJournalEntryEntryKind(apispec.SyncJournalEntryEntryKind(entryKind))
	}
	if hasMode {
		entry.Mode = apispec.NewOptNilInt64(mode)
	}
	if contentRef != "" {
		entry.ContentRef = apispec.NewOptNilString(contentRef)
	}
	if contentSHA256 != "" {
		entry.ContentSHA256 = apispec.NewOptNilString(contentSHA256)
	}
	if hasSize {
		entry.SizeBytes = apispec.NewOptNilInt64(sizeBytes)
	}
	if oldPath != "" {
		entry.OldPath = apispec.NewOptNilString(oldPath)
		entry.NormalizedOldPath = apispec.NewOptNilString(strings.ToLower(oldPath))
	}
	s.journal = append(s.journal, entry)
	return entry
}

func (s *fakeSyncService) addSandboxWrite(pathValue, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	change := apispec.ChangeRequest{
		EventType:     apispec.SyncEventTypeWrite,
		Path:          apispec.NewOptString(pathValue),
		EntryKind:     apispec.NewOptChangeRequestEntryKind(apispec.ChangeRequestEntryKindFile),
		Mode:          apispec.NewOptNilInt64(0o644),
		ContentBase64: apispec.NewOptNilString(base64.StdEncoding.EncodeToString([]byte(content))),
	}
	s.appendJournalEntryLocked("", apispec.SyncJournalEntrySourceSandbox, change)
}

func (s *fakeSyncService) addSandboxCreateWriteChmod(pathValue, content string, createMode, chmodMode int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.appendJournalEntryLocked("", apispec.SyncJournalEntrySourceSandbox, apispec.ChangeRequest{
		EventType: apispec.SyncEventTypeCreate,
		Path:      apispec.NewOptString(pathValue),
		EntryKind: apispec.NewOptChangeRequestEntryKind(apispec.ChangeRequestEntryKindFile),
		Mode:      apispec.NewOptNilInt64(createMode),
	})
	s.appendJournalEntryLocked("", apispec.SyncJournalEntrySourceSandbox, apispec.ChangeRequest{
		EventType:     apispec.SyncEventTypeWrite,
		Path:          apispec.NewOptString(pathValue),
		EntryKind:     apispec.NewOptChangeRequestEntryKind(apispec.ChangeRequestEntryKindFile),
		Mode:          apispec.NewOptNilInt64(createMode),
		ContentBase64: apispec.NewOptNilString(base64.StdEncoding.EncodeToString([]byte(content))),
	})
	s.appendJournalEntryLocked("", apispec.SyncJournalEntrySourceSandbox, apispec.ChangeRequest{
		EventType: apispec.SyncEventTypeChmod,
		Path:      apispec.NewOptString(pathValue),
		Mode:      apispec.NewOptNilInt64(chmodMode),
	})
}

func (s *fakeSyncService) setBootstrapArchive(archive []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bootstrapArchive = append([]byte(nil), archive...)
}

func (s *fakeSyncService) setConflictOnNextPath(pathValue string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conflictNextPath = pathValue
}

func (s *fakeSyncService) fileContent(pathValue string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.files[pathValue]
}

func buildArchive(files map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	for name, content := range files {
		payload := []byte(content)
		if err := tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(payload)),
		}); err != nil {
			return nil, err
		}
		if _, err := tw.Write(payload); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gzw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(data)
}

func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	if condition() {
		return
	}
	t.Fatalf("condition not satisfied within %s", timeout)
}

func buildUnsupportedSymlinkArchive(t *testing.T) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzw := gzip.NewWriter(&buffer)
	tw := tar.NewWriter(gzw)
	if err := tw.WriteHeader(&tar.Header{
		Name:     "latest",
		Typeflag: tar.TypeSymlink,
		Linkname: "README.md",
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
	return buffer.Bytes()
}
