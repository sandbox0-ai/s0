package syncstate

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveAttachmentFromPathSelectsNearestAncestor(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := filepath.Join(home, "work", "repo")
	child := filepath.Join(root, "apps", "dashboard")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	parentAttachment, err := NewAttachment(root, "vol-parent", "repo", "auto")
	if err != nil {
		t.Fatalf("NewAttachment(parent) error = %v", err)
	}
	if err := SaveAttachment(parentAttachment); err != nil {
		t.Fatalf("SaveAttachment(parent) error = %v", err)
	}

	childAttachment, err := NewAttachment(child, "vol-child", "dashboard", "auto")
	if err != nil {
		t.Fatalf("NewAttachment(child) error = %v", err)
	}
	if err := SaveAttachment(childAttachment); err != nil {
		t.Fatalf("SaveAttachment(child) error = %v", err)
	}

	nested := filepath.Join(child, "src")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll(nested) error = %v", err)
	}

	resolved, err := ResolveAttachmentFromPath(nested)
	if err != nil {
		t.Fatalf("ResolveAttachmentFromPath() error = %v", err)
	}
	if resolved.ID != childAttachment.ID {
		t.Fatalf("ResolveAttachmentFromPath() id = %q, want %q", resolved.ID, childAttachment.ID)
	}
}

func TestResolveAttachmentTargetFallsBackToVolumeID(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := filepath.Join(home, "work", "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	attachment, err := NewAttachment(root, "vol-1", "repo", "auto")
	if err != nil {
		t.Fatalf("NewAttachment() error = %v", err)
	}
	if err := SaveAttachment(attachment); err != nil {
		t.Fatalf("SaveAttachment() error = %v", err)
	}

	resolved, err := ResolveAttachmentTarget("vol-1", home)
	if err != nil {
		t.Fatalf("ResolveAttachmentTarget() error = %v", err)
	}
	if resolved.ID != attachment.ID {
		t.Fatalf("ResolveAttachmentTarget() id = %q, want %q", resolved.ID, attachment.ID)
	}
}

func TestEffectiveStatusMarksStaleHeartbeat(t *testing.T) {
	attachment, err := NewAttachment(t.TempDir(), "vol-1", "", "auto")
	if err != nil {
		t.Fatalf("NewAttachment() error = %v", err)
	}

	attachment.Worker.Status = "running"
	if got := EffectiveStatus(attachment); got != "stale" {
		t.Fatalf("EffectiveStatus() = %q, want stale", got)
	}
}

func TestSyncHealthReportsReseedAndErrorStates(t *testing.T) {
	attachment, err := NewAttachment(t.TempDir(), "vol-1", "", "auto")
	if err != nil {
		t.Fatalf("NewAttachment() error = %v", err)
	}
	now := time.Now().UTC()
	attachment.Worker.Status = "running"
	attachment.Worker.LastHeartbeatAt = &now
	attachment.LastSync.ReseedRequired = true
	if got := SyncHealth(attachment); got != "reseed_required" {
		t.Fatalf("SyncHealth() = %q, want reseed_required", got)
	}

	attachment.LastSync.ReseedRequired = false
	attachment.LastSync.ConsecutiveErrors = 2
	attachment.LastError = "boom"
	if got := SyncHealth(attachment); got != "error" {
		t.Fatalf("SyncHealth() = %q, want error", got)
	}
}

func TestResetWorkerStateStopsRunningAttachment(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	attachment, err := NewAttachment(filepath.Join(home, "work"), "vol-1", "", "auto")
	if err != nil {
		t.Fatalf("NewAttachment() error = %v", err)
	}
	now := time.Now().UTC()
	attachment.Worker.Status = "running"
	attachment.Worker.Mode = "background"
	attachment.Worker.PID = 12345
	attachment.Worker.LastHeartbeatAt = &now
	if err := SaveAttachment(attachment); err != nil {
		t.Fatalf("SaveAttachment() error = %v", err)
	}

	reset, err := ResetWorkerState(attachment.ID)
	if err != nil {
		t.Fatalf("ResetWorkerState() error = %v", err)
	}
	if reset.Worker.Status != "stopped" {
		t.Fatalf("Worker.Status = %q, want stopped", reset.Worker.Status)
	}
	if reset.Worker.PID != 0 {
		t.Fatalf("Worker.PID = %d, want 0", reset.Worker.PID)
	}
	if reset.Worker.LastStoppedAt == nil {
		t.Fatalf("Worker.LastStoppedAt = nil, want value")
	}
}

func TestSQLiteStatePersistsAttachmentsAndManifests(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	attachment, err := NewAttachment(filepath.Join(home, "work"), "vol-1", "repo", "auto")
	if err != nil {
		t.Fatalf("NewAttachment() error = %v", err)
	}
	attachment.LastError = "db-state"
	if err := SaveAttachment(attachment); err != nil {
		t.Fatalf("SaveAttachment() error = %v", err)
	}

	manifest := &Manifest{
		Entries: map[string]ManifestEntry{
			"README.md": {
				Path:   "README.md",
				Kind:   "file",
				Mode:   0o644,
				Size:   12,
				SHA256: "abc123",
			},
		},
	}
	if err := SaveManifest(attachment.ID, manifest); err != nil {
		t.Fatalf("SaveManifest() error = %v", err)
	}

	loadedAttachment, err := LoadAttachmentByID(attachment.ID)
	if err != nil {
		t.Fatalf("LoadAttachmentByID() error = %v", err)
	}
	if loadedAttachment.VolumeID != "vol-1" {
		t.Fatalf("loaded attachment volume_id = %q, want %q", loadedAttachment.VolumeID, "vol-1")
	}
	if loadedAttachment.LastError != "db-state" {
		t.Fatalf("loaded attachment last_error = %q, want %q", loadedAttachment.LastError, "db-state")
	}

	loadedManifest, err := LoadManifest(attachment.ID)
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	entry, ok := loadedManifest.Entries["README.md"]
	if !ok {
		t.Fatalf("loaded manifest missing README.md entry")
	}
	if entry.SHA256 != "abc123" {
		t.Fatalf("loaded manifest sha256 = %q, want %q", entry.SHA256, "abc123")
	}

	if _, err := os.Stat(stateDatabasePath()); err != nil {
		t.Fatalf("Stat(state.db) error = %v", err)
	}
}
