package syncworker

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	syncsdk "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"

	"github.com/sandbox0-ai/s0/internal/syncapi"
	"github.com/sandbox0-ai/s0/internal/syncignore"
	"github.com/sandbox0-ai/s0/internal/syncstate"
)

const (
	workerHeartbeatInterval = 2 * time.Second
	reconcileInterval       = 5 * time.Second
	replicaRefreshInterval  = 30 * time.Second
	watchDebounceInterval   = 250 * time.Millisecond
	changeBatchLimit        = 256
	maxReplayBatches        = 8
)

// Run executes the phase-2 sync worker loop for one attachment.
func Run(ctx context.Context, client *syncsdk.Client, attachmentID, mode string, out io.Writer) error {
	attachment, err := syncstate.MarkWorkerStarted(attachmentID, mode, os.Getpid())
	if err != nil {
		return err
	}

	logger := log.New(out, "", log.LstdFlags)
	api := syncapi.New(client)

	logger.Printf("Sync worker started for workspace %s (volume %s, replica %s, mode %s)", attachment.WorkspaceRoot, attachment.VolumeID, attachment.ReplicaID, mode)

	heartbeatTicker := time.NewTicker(workerHeartbeatInterval)
	defer heartbeatTicker.Stop()

	reconcileTicker := time.NewTicker(reconcileInterval)
	defer reconcileTicker.Stop()

	replicaTicker := time.NewTicker(replicaRefreshInterval)
	defer replicaTicker.Stop()

	var loopErr error
	defer func() {
		if _, err := syncstate.MarkWorkerStopped(attachmentID, loopErr); err != nil {
			logger.Printf("Failed to persist worker shutdown state: %v", err)
		}
		logger.Printf("Sync worker stopped for attachment %s", attachmentID)
	}()

	var workspaceWatcher *workspaceWatcher
	syncWorkspaceWatch := func() {
		if workspaceWatcher == nil {
			return
		}
		if err := workspaceWatcher.Sync(); err != nil {
			_ = recordSyncError(attachmentID, err)
			logger.Printf("Filesystem watch refresh failed: %v", err)
		}
	}
	runReconcile := func(forceReplicaRefresh bool) {
		syncWorkspaceWatch()
		if err := reconcileOnce(ctx, api, attachmentID, logger, forceReplicaRefresh); err != nil {
			_ = recordSyncError(attachmentID, err)
			if forceReplicaRefresh {
				logger.Printf("Initial reconcile failed: %v", err)
				return
			}
			logger.Printf("Reconcile failed: %v", err)
			syncWorkspaceWatch()
			return
		}
		syncWorkspaceWatch()
	}

	runReconcile(true)

	workspaceWatcher, err = newWorkspaceWatcher(attachment.WorkspaceRoot)
	if err != nil {
		logger.Printf("Filesystem watch unavailable for workspace %s: %v", attachment.WorkspaceRoot, err)
	}
	if workspaceWatcher != nil {
		defer workspaceWatcher.Close()
	}

	var watchEvents <-chan fsnotify.Event
	var watchErrors <-chan error
	if workspaceWatcher != nil {
		watchEvents = workspaceWatcher.Events()
		watchErrors = workspaceWatcher.Errors()
	}

	var watchDebounceTimer *time.Timer
	var watchDebounceC <-chan time.Time
	stopWatchDebounce := func() {
		if watchDebounceTimer == nil {
			return
		}
		if !watchDebounceTimer.Stop() {
			select {
			case <-watchDebounceTimer.C:
			default:
			}
		}
		watchDebounceTimer = nil
		watchDebounceC = nil
	}
	defer stopWatchDebounce()

	scheduleWatchReconcile := func() {
		if watchDebounceTimer == nil {
			watchDebounceTimer = time.NewTimer(watchDebounceInterval)
			watchDebounceC = watchDebounceTimer.C
			return
		}
		if !watchDebounceTimer.Stop() {
			select {
			case <-watchDebounceTimer.C:
			default:
			}
		}
		watchDebounceTimer.Reset(watchDebounceInterval)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-heartbeatTicker.C:
			if _, err := syncstate.TouchWorkerHeartbeat(attachmentID, mode, os.Getpid()); err != nil {
				loopErr = err
				return err
			}
		case <-replicaTicker.C:
			if err := refreshReplica(ctx, api, attachmentID); err != nil {
				_ = recordSyncError(attachmentID, err)
				logger.Printf("Replica refresh failed: %v", err)
			}
		case event, ok := <-watchEvents:
			if !ok {
				watchEvents = nil
				continue
			}
			trigger, err := workspaceWatcher.HandleEvent(event)
			if err != nil {
				_ = recordSyncError(attachmentID, err)
				logger.Printf("Filesystem watch refresh failed: %v", err)
				continue
			}
			if trigger {
				scheduleWatchReconcile()
			}
		case err, ok := <-watchErrors:
			if !ok {
				watchErrors = nil
				continue
			}
			logger.Printf("Filesystem watch error: %v", err)
		case <-watchDebounceC:
			stopWatchDebounce()
			runReconcile(false)
		case <-reconcileTicker.C:
			runReconcile(false)
		}
	}
}

func reconcileOnce(ctx context.Context, api *syncapi.Client, attachmentID string, logger *log.Logger, forceReplicaRefresh bool) error {
	attachment, err := syncstate.LoadAttachmentByID(attachmentID)
	if err != nil {
		return err
	}
	if forceReplicaRefresh {
		if _, err := upsertReplica(ctx, api, attachmentID); err != nil {
			return err
		}
	}

	matcher, err := syncignore.Load(attachment.WorkspaceRoot)
	if err != nil {
		return err
	}
	manifest, err := syncstate.LoadManifest(attachmentID)
	if err != nil {
		return err
	}
	if err := ensureInitialized(ctx, api, attachment, manifest, matcher, logger); err != nil {
		return err
	}

	if _, err := refreshOpenConflictCount(ctx, api, attachmentID, attachment.VolumeID); err != nil {
		logger.Printf("Conflict refresh failed: %v", err)
	}

	attachment, err = syncstate.LoadAttachmentByID(attachmentID)
	if err != nil {
		return err
	}
	manifest, err = syncstate.LoadManifest(attachmentID)
	if err != nil {
		return err
	}

	if err := uploadLocalChanges(ctx, api, attachment, manifest, matcher, logger); err != nil {
		return err
	}
	if err := replayRemoteChanges(ctx, api, attachmentID, logger); err != nil {
		return err
	}

	if _, err := refreshOpenConflictCount(ctx, api, attachmentID, attachment.VolumeID); err != nil {
		logger.Printf("Conflict refresh failed: %v", err)
	}
	return markSyncSuccess(attachmentID)
}

func ensureInitialized(ctx context.Context, api *syncapi.Client, attachment *syncstate.Attachment, manifest *syncstate.Manifest, matcher *syncignore.Matcher, logger *log.Logger) error {
	if attachment.LastSync != nil && (attachment.LastSync.LastSuccessAt != nil || attachment.LastSync.LastAppliedSeq > 0 || len(manifest.Entries) > 0) {
		return nil
	}

	current, err := scanWorkspace(attachment.WorkspaceRoot)
	if err != nil {
		return err
	}
	useVolume := attachment.InitFrom == "volume" || (attachment.InitFrom == "auto" && len(current.Entries) == 0)
	if useVolume {
		logger.Printf("Bootstrapping workspace from volume %s", attachment.VolumeID)
		return bootstrapFromVolume(ctx, api, attachment.ID, logger)
	}

	logger.Printf("Seeding volume %s from local workspace %s", attachment.VolumeID, attachment.WorkspaceRoot)
	if err := auditWorkspaceCompatibility(current, attachment.Capabilities); err != nil {
		return err
	}
	return seedFromLocal(ctx, api, attachment.ID, current, matcher, logger)
}

func uploadLocalChanges(ctx context.Context, api *syncapi.Client, attachment *syncstate.Attachment, manifest *syncstate.Manifest, matcher *syncignore.Matcher, logger *log.Logger) error {
	current, err := scanWorkspace(attachment.WorkspaceRoot)
	if err != nil {
		return err
	}
	if err := auditWorkspaceCompatibility(current, attachment.Capabilities); err != nil {
		return err
	}

	changes, err := buildUploadChanges(attachment.WorkspaceRoot, manifest, current, matcher)
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		return syncstate.SaveManifest(attachment.ID, current)
	}
	if attachment.LastSync != nil && attachment.LastSync.OpenConflictCount > 0 {
		logger.Printf("Skipping local upload while %d sync conflicts remain open", attachment.LastSync.OpenConflictCount)
		return nil
	}

	resp, err := api.AppendChanges(ctx, attachment, attachment.LastSync.LastAppliedSeq, newRequestID(), changes)
	if err != nil {
		var reseed *syncapi.ReseedRequiredError
		if errors.As(err, &reseed) {
			logger.Printf("Local state fell behind retained journal floor (retained_after_seq=%d head_seq=%d), bootstrapping again", reseed.RetainedAfterSeq, reseed.HeadSeq)
			return bootstrapFromVolume(ctx, api, attachment.ID, logger)
		}
		return err
	}

	if err := updateCheckpoint(attachment.ID, func(checkpoint *syncstate.SyncCheckpoint) {
		checkpoint.HeadSeq = resp.HeadSeq
		checkpoint.OpenConflictCount = len(resp.Conflicts)
	}); err != nil {
		return err
	}
	if len(resp.Conflicts) > 0 {
		logger.Printf("Local upload produced %d sync conflicts; waiting for manual resolution", len(resp.Conflicts))
		return nil
	}

	if err := syncstate.SaveManifest(attachment.ID, current); err != nil {
		return err
	}
	if _, err := api.UpdateCursor(ctx, attachment, resp.HeadSeq); err != nil {
		return err
	}
	return updateCheckpoint(attachment.ID, func(checkpoint *syncstate.SyncCheckpoint) {
		checkpoint.HeadSeq = resp.HeadSeq
		checkpoint.LastAppliedSeq = resp.HeadSeq
		checkpoint.ReseedRequired = false
	})
}

func replayRemoteChanges(ctx context.Context, api *syncapi.Client, attachmentID string, logger *log.Logger) error {
	for batch := 0; batch < maxReplayBatches; batch++ {
		attachment, err := syncstate.LoadAttachmentByID(attachmentID)
		if err != nil {
			return err
		}
		resp, err := api.ListChanges(ctx, attachment.VolumeID, attachment.LastSync.LastAppliedSeq, changeBatchLimit)
		if err != nil {
			var reseed *syncapi.ReseedRequiredError
			if errors.As(err, &reseed) {
				logger.Printf("Remote replay requires reseed (retained_after_seq=%d head_seq=%d), bootstrapping again", reseed.RetainedAfterSeq, reseed.HeadSeq)
				return bootstrapFromVolume(ctx, api, attachmentID, logger)
			}
			return err
		}

		if len(resp.Changes) == 0 {
			return updateCheckpoint(attachmentID, func(checkpoint *syncstate.SyncCheckpoint) {
				checkpoint.HeadSeq = resp.HeadSeq
			})
		}

		manifest, err := syncstate.LoadManifest(attachmentID)
		if err != nil {
			return err
		}
		compatibilityTracker := newCompatibilityTracker(manifest, attachment.Capabilities)

		mutated := false
		lastApplied := attachment.LastSync.LastAppliedSeq
		for _, change := range resp.Changes {
			seq, ok := change.Seq.Get()
			if !ok {
				return fmt.Errorf("sync journal entry missing seq")
			}
			if isOwnReplicaEcho(change, attachment.ReplicaID) {
				lastApplied = seq
				continue
			}
			if compatibilityTracker != nil {
				if err := compatibilityTracker.ValidateRemoteChange(optString(change.Path), optNilString(change.OldPath), optEvent(change.EventType)); err != nil {
					return err
				}
			}
			applied, err := applyRemoteChange(ctx, api, attachment.VolumeID, attachment.WorkspaceRoot, change)
			if err != nil {
				if errors.Is(err, errRemoteBootstrapRequired) {
					logger.Printf("Remote change requires bootstrap fallback (seq=%d path=%s event=%s)", seq, optString(change.Path), optEvent(change.EventType))
					return bootstrapFromVolume(ctx, api, attachmentID, logger)
				}
				return err
			}
			if applied {
				mutated = true
				if compatibilityTracker != nil {
					compatibilityTracker.ApplyRemoteChange(optString(change.Path), optNilString(change.OldPath), optEvent(change.EventType))
				}
			}
			lastApplied = seq
		}

		if mutated {
			current, err := scanWorkspace(attachment.WorkspaceRoot)
			if err != nil {
				return err
			}
			if err := syncstate.SaveManifest(attachmentID, current); err != nil {
				return err
			}
		}
		if _, err := api.UpdateCursor(ctx, attachment, lastApplied); err != nil {
			var reseed *syncapi.ReseedRequiredError
			if errors.As(err, &reseed) {
				return bootstrapFromVolume(ctx, api, attachmentID, logger)
			}
			return err
		}
		if err := updateCheckpoint(attachmentID, func(checkpoint *syncstate.SyncCheckpoint) {
			checkpoint.HeadSeq = resp.HeadSeq
			checkpoint.LastAppliedSeq = lastApplied
			checkpoint.ReseedRequired = false
		}); err != nil {
			return err
		}
		if lastApplied >= resp.HeadSeq {
			return nil
		}
	}

	return nil
}

func upsertReplica(ctx context.Context, api *syncapi.Client, attachmentID string) (*apispec.VolumeSyncReplicaEnvelope, error) {
	attachment, err := syncstate.LoadAttachmentByID(attachmentID)
	if err != nil {
		return nil, err
	}
	replica, err := api.UpsertReplica(ctx, attachment)
	if err != nil {
		return nil, err
	}
	if err := updateCheckpoint(attachmentID, func(checkpoint *syncstate.SyncCheckpoint) {
		now := time.Now().UTC()
		checkpoint.HeadSeq = replica.HeadSeq
		checkpoint.LastReplicaSyncAt = &now
		if checkpoint.LastAppliedSeq == 0 {
			if lastApplied, ok := replica.Replica.LastAppliedSeq.Get(); ok {
				checkpoint.LastAppliedSeq = lastApplied
			}
		}
	}); err != nil {
		return nil, err
	}
	return replica, nil
}

func refreshReplica(ctx context.Context, api *syncapi.Client, attachmentID string) error {
	_, err := upsertReplica(ctx, api, attachmentID)
	return err
}

func bootstrapFromVolume(ctx context.Context, api *syncapi.Client, attachmentID string, logger *log.Logger) error {
	attachment, err := syncstate.LoadAttachmentByID(attachmentID)
	if err != nil {
		return err
	}
	bootstrap, err := api.Bootstrap(ctx, attachment)
	if err != nil {
		var compatibility *syncapi.BootstrapCompatibilityError
		if errors.As(err, &compatibility) {
			if compatibility.Details != nil && len(compatibility.Details.Issues) > 0 {
				logger.Printf("Bootstrap compatibility conflict: %s", compatibility.Details.Issues[0].Code)
			}
		}
		return err
	}
	archive, err := api.DownloadBootstrapArchive(ctx, attachment.VolumeID, bootstrap.Snapshot.ID)
	if err != nil {
		return err
	}
	if err := clearWorkspaceForBootstrap(attachment.WorkspaceRoot); err != nil {
		return err
	}
	if err := extractBootstrapArchive(attachment.WorkspaceRoot, archive); err != nil {
		return err
	}
	current, err := scanWorkspace(attachment.WorkspaceRoot)
	if err != nil {
		return err
	}
	if err := syncstate.SaveManifest(attachmentID, current); err != nil {
		return err
	}
	if _, err := api.UpdateCursor(ctx, attachment, bootstrap.ReplayAfterSeq); err != nil {
		return err
	}
	if err := updateCheckpoint(attachmentID, func(checkpoint *syncstate.SyncCheckpoint) {
		checkpoint.HeadSeq = bootstrap.ReplayAfterSeq
		checkpoint.LastAppliedSeq = bootstrap.ReplayAfterSeq
		checkpoint.ReseedRequired = false
	}); err != nil {
		return err
	}
	logger.Printf("Workspace bootstrapped from snapshot %s at replay_after_seq=%d", bootstrap.Snapshot.ID, bootstrap.ReplayAfterSeq)
	return nil
}

func seedFromLocal(ctx context.Context, api *syncapi.Client, attachmentID string, current *syncstate.Manifest, matcher *syncignore.Matcher, logger *log.Logger) error {
	if _, err := upsertReplica(ctx, api, attachmentID); err != nil {
		return err
	}
	attachment, err := syncstate.LoadAttachmentByID(attachmentID)
	if err != nil {
		return err
	}

	changes, err := buildUploadChanges(attachment.WorkspaceRoot, &syncstate.Manifest{Entries: map[string]syncstate.ManifestEntry{}}, current, matcher)
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		if err := syncstate.SaveManifest(attachmentID, current); err != nil {
			return err
		}
		return updateCheckpoint(attachmentID, func(checkpoint *syncstate.SyncCheckpoint) {
			if checkpoint.LastAppliedSeq < checkpoint.HeadSeq {
				checkpoint.LastAppliedSeq = checkpoint.HeadSeq
			}
			checkpoint.ReseedRequired = false
		})
	}

	resp, err := api.AppendChanges(ctx, attachment, attachment.LastSync.LastAppliedSeq, newRequestID(), changes)
	if err != nil {
		var reseed *syncapi.ReseedRequiredError
		if errors.As(err, &reseed) {
			logger.Printf("Local seed rejected because the remote journal was compacted; bootstrapping instead")
			return bootstrapFromVolume(ctx, api, attachmentID, logger)
		}
		return err
	}
	if len(resp.Conflicts) > 0 {
		if err := updateCheckpoint(attachmentID, func(checkpoint *syncstate.SyncCheckpoint) {
			checkpoint.HeadSeq = resp.HeadSeq
			checkpoint.OpenConflictCount = len(resp.Conflicts)
		}); err != nil {
			return err
		}
		return nil
	}
	if err := syncstate.SaveManifest(attachmentID, current); err != nil {
		return err
	}
	if _, err := api.UpdateCursor(ctx, attachment, resp.HeadSeq); err != nil {
		return err
	}
	return updateCheckpoint(attachmentID, func(checkpoint *syncstate.SyncCheckpoint) {
		checkpoint.HeadSeq = resp.HeadSeq
		checkpoint.LastAppliedSeq = resp.HeadSeq
		checkpoint.OpenConflictCount = 0
		checkpoint.ReseedRequired = false
	})
}

func refreshOpenConflictCount(ctx context.Context, api *syncapi.Client, attachmentID, volumeID string) (int, error) {
	conflicts, err := api.ListConflicts(ctx, volumeID, "open", 256)
	if err != nil {
		return 0, err
	}
	if err := updateCheckpoint(attachmentID, func(checkpoint *syncstate.SyncCheckpoint) {
		checkpoint.OpenConflictCount = len(conflicts)
	}); err != nil {
		return 0, err
	}
	return len(conflicts), nil
}

func markSyncSuccess(attachmentID string) error {
	now := time.Now().UTC()
	_, err := syncstate.UpdateAttachment(attachmentID, func(attachment *syncstate.Attachment) error {
		attachment.LastError = ""
		if attachment.LastSync == nil {
			attachment.LastSync = &syncstate.SyncCheckpoint{}
		}
		attachment.LastSync.LastSuccessAt = &now
		attachment.LastSync.ConsecutiveErrors = 0
		attachment.LastSync.ReseedRequired = false
		return nil
	})
	return err
}

func recordSyncError(attachmentID string, workerErr error) error {
	_, err := syncstate.UpdateAttachment(attachmentID, func(attachment *syncstate.Attachment) error {
		if workerErr == nil {
			attachment.LastError = ""
			return nil
		}
		now := time.Now().UTC()
		attachment.LastError = workerErr.Error()
		if attachment.LastSync == nil {
			attachment.LastSync = &syncstate.SyncCheckpoint{}
		}
		var reseed *syncapi.ReseedRequiredError
		attachment.LastSync.LastFailureAt = &now
		attachment.LastSync.ConsecutiveErrors++
		attachment.LastSync.ReseedRequired = errors.As(workerErr, &reseed)
		return nil
	})
	return err
}

func updateCheckpoint(attachmentID string, update func(*syncstate.SyncCheckpoint)) error {
	_, err := syncstate.UpdateAttachment(attachmentID, func(attachment *syncstate.Attachment) error {
		if attachment.LastSync == nil {
			attachment.LastSync = &syncstate.SyncCheckpoint{}
		}
		update(attachment.LastSync)
		return nil
	})
	return err
}

var errRemoteBootstrapRequired = errors.New("remote change requires bootstrap")

func applyRemoteChange(ctx context.Context, api *syncapi.Client, volumeID, root string, change apispec.SyncJournalEntry) (bool, error) {
	event := optEvent(change.EventType)
	switch event {
	case string(apispec.SyncEventTypeRemove):
		target := filepath.Join(root, filepath.FromSlash(optString(change.Path)))
		if err := os.RemoveAll(target); err != nil {
			return false, err
		}
		return true, nil
	case string(apispec.SyncEventTypeRename):
		oldPath := optNilString(change.OldPath)
		newPath := optString(change.Path)
		if oldPath == "" || newPath == "" {
			return false, errRemoteBootstrapRequired
		}
		source := filepath.Join(root, filepath.FromSlash(oldPath))
		destination := filepath.Join(root, filepath.FromSlash(newPath))
		if _, err := os.Stat(source); err != nil {
			if os.IsNotExist(err) {
				return false, errRemoteBootstrapRequired
			}
			return false, err
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return false, err
		}
		if _, err := os.Stat(destination); err == nil {
			return false, errRemoteBootstrapRequired
		}
		if err := os.Rename(source, destination); err != nil {
			return false, err
		}
		return true, nil
	case string(apispec.SyncEventTypeCreate):
		return applyRemoteCreate(root, change)
	case string(apispec.SyncEventTypeWrite):
		return applyRemoteWrite(ctx, api, volumeID, root, change)
	case string(apispec.SyncEventTypeChmod):
		return applyRemoteChmod(root, change)
	default:
		return false, unsupportedRemoteChangeError(optString(change.Path), event, optEntryKind(change.EntryKind))
	}
}

func applyRemoteCreate(root string, change apispec.SyncJournalEntry) (bool, error) {
	relativePath := optString(change.Path)
	if strings.TrimSpace(relativePath) == "" {
		return false, errRemoteBootstrapRequired
	}
	target := filepath.Join(root, filepath.FromSlash(relativePath))
	mode := os.FileMode(optMode(change.Mode, 0))
	entryKind := optEntryKind(change.EntryKind)
	switch entryKind {
	case string(apispec.SyncJournalEntryEntryKindDirectory):
		if err := os.MkdirAll(target, defaultDirMode(mode)); err != nil {
			return false, err
		}
		if mode != 0 {
			if err := os.Chmod(target, mode); err != nil {
				return false, err
			}
		}
		return true, nil
	case "", string(apispec.SyncJournalEntryEntryKindFile):
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return false, err
		}
		info, err := os.Stat(target)
		switch {
		case err == nil && info.IsDir():
			return false, errRemoteBootstrapRequired
		case err == nil:
			if mode != 0 {
				if err := os.Chmod(target, mode); err != nil {
					return false, err
				}
			}
			return true, nil
		case !os.IsNotExist(err):
			return false, err
		}

		file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, defaultFileMode(mode))
		if err != nil {
			return false, err
		}
		if err := file.Close(); err != nil {
			return false, err
		}
		if mode != 0 {
			if err := os.Chmod(target, mode); err != nil {
				return false, err
			}
		}
		return true, nil
	default:
		return false, unsupportedRemoteChangeError(relativePath, optEvent(change.EventType), entryKind)
	}
}

func applyRemoteWrite(ctx context.Context, api *syncapi.Client, volumeID, root string, change apispec.SyncJournalEntry) (bool, error) {
	if api == nil || strings.TrimSpace(volumeID) == "" {
		return false, errRemoteBootstrapRequired
	}
	relativePath := optString(change.Path)
	contentRef := optNilString(change.ContentRef)
	if strings.TrimSpace(relativePath) == "" || strings.TrimSpace(contentRef) == "" {
		return false, errRemoteBootstrapRequired
	}
	entryKind := optEntryKind(change.EntryKind)
	if entryKind != "" && entryKind != string(apispec.SyncJournalEntryEntryKindFile) {
		return false, unsupportedRemoteChangeError(relativePath, optEvent(change.EventType), entryKind)
	}

	payload, err := api.DownloadReplayPayload(ctx, volumeID, contentRef)
	if err != nil {
		return false, err
	}
	if expected := optNilString(change.ContentSHA256); expected != "" {
		sum := sha256.Sum256(payload)
		if hex.EncodeToString(sum[:]) != expected {
			return false, fmt.Errorf("replay payload sha256 mismatch for %s", relativePath)
		}
	}

	target := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return false, err
	}
	if info, err := os.Stat(target); err == nil && info.IsDir() {
		return false, errRemoteBootstrapRequired
	} else if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	mode := os.FileMode(optMode(change.Mode, 0))
	file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, defaultFileMode(mode))
	if err != nil {
		return false, err
	}
	if _, err := file.Write(payload); err != nil {
		file.Close()
		return false, err
	}
	if err := file.Close(); err != nil {
		return false, err
	}
	if mode != 0 {
		if err := os.Chmod(target, mode); err != nil {
			return false, err
		}
	}
	return true, nil
}

func applyRemoteChmod(root string, change apispec.SyncJournalEntry) (bool, error) {
	relativePath := optString(change.Path)
	modeValue, ok := change.Mode.Get()
	if strings.TrimSpace(relativePath) == "" || !ok {
		return false, errRemoteBootstrapRequired
	}
	target := filepath.Join(root, filepath.FromSlash(relativePath))
	if _, err := os.Stat(target); err != nil {
		if os.IsNotExist(err) {
			return false, errRemoteBootstrapRequired
		}
		return false, err
	}
	if err := os.Chmod(target, os.FileMode(modeValue)); err != nil {
		return false, err
	}
	return true, nil
}

func optEntryKind(value apispec.OptNilSyncJournalEntryEntryKind) string {
	v, _ := value.Get()
	return string(v)
}

func optMode(value apispec.OptNilInt64, fallback int64) int64 {
	if v, ok := value.Get(); ok {
		return v
	}
	return fallback
}

func defaultFileMode(mode os.FileMode) os.FileMode {
	if mode != 0 {
		return mode
	}
	return 0o644
}

func defaultDirMode(mode os.FileMode) os.FileMode {
	if mode != 0 {
		return mode
	}
	return 0o755
}

func isOwnReplicaEcho(change apispec.SyncJournalEntry, replicaID string) bool {
	source, ok := change.Source.Get()
	if !ok || source != apispec.SyncJournalEntrySourceReplica {
		return false
	}
	remoteReplicaID, ok := change.ReplicaID.Get()
	return ok && remoteReplicaID == replicaID
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return "req-" + hex.EncodeToString(bytes[:])
}

func optString(value apispec.OptString) string {
	v, _ := value.Get()
	return v
}

func optNilString(value apispec.OptNilString) string {
	v, _ := value.Get()
	return v
}

func optEvent(value apispec.OptSyncEventType) string {
	v, _ := value.Get()
	return string(v)
}
