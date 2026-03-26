package syncview

import (
	"testing"
	"time"

	"github.com/go-faster/jx"
	"github.com/sandbox0-ai/s0/internal/syncstate"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestBuildConflictListViewSummarizesPortableConflict(t *testing.T) {
	attachment := &syncstate.Attachment{
		WorkspaceRoot: "/tmp/work",
		VolumeID:      "vol-123",
	}
	conflicts := []apispec.SyncConflict{
		{
			Path:         apispec.NewOptString("docs/CON.txt"),
			Reason:       apispec.NewOptString("windows_reserved_name"),
			Status:       apispec.NewOptString("open"),
			ArtifactPath: apispec.NewOptString(""),
			Metadata: apispec.NewOptNilSyncConflictMetadata(apispec.SyncConflictMetadata{
				"issues": jx.Raw(`[{"code":"windows_reserved_name","path":"docs/CON.txt","message":"path segment uses a Windows reserved device name"}]`),
			}),
		},
	}

	view := BuildConflictListView(attachment, conflicts, 0)
	if view.OpenCount != 1 {
		t.Fatalf("OpenCount = %d, want 1", view.OpenCount)
	}
	if len(view.UnmergedPaths) != 1 {
		t.Fatalf("len(UnmergedPaths) = %d, want 1", len(view.UnmergedPaths))
	}
	if got := view.UnmergedPaths[0].Summary; got != "namespace incompatible for Windows-capable replicas" {
		t.Fatalf("summary = %q, want Windows compatibility wording", got)
	}
}

func TestBuildConflictListViewIncludesSandboxActorLabel(t *testing.T) {
	attachment := &syncstate.Attachment{
		WorkspaceRoot: "/tmp/work",
		VolumeID:      "vol-123",
	}
	conflicts := []apispec.SyncConflict{
		{
			Path:   apispec.NewOptString("src/main.go"),
			Reason: apispec.NewOptString("concurrent_update"),
			Status: apispec.NewOptString("open"),
			Metadata: apispec.NewOptNilSyncConflictMetadata(apispec.SyncConflictMetadata{
				"latest_source":     jx.Raw(`"sandbox"`),
				"latest_sandbox_id": jx.Raw(`"sandbox-1"`),
			}),
		},
	}

	view := BuildConflictListView(attachment, conflicts, 0)
	if len(view.UnmergedPaths) != 1 {
		t.Fatalf("len(UnmergedPaths) = %d, want 1", len(view.UnmergedPaths))
	}
	if got := view.UnmergedPaths[0].Path; got != "src/main.go" {
		t.Fatalf("path = %q, want src/main.go", got)
	}
	if got := view.UnmergedPaths[0].Summary; got != `modified locally, conflicted with sandbox "sandbox-1"` {
		t.Fatalf("summary = %q", got)
	}
}

func TestBuildConflictListViewNormalizesLeadingSlashPaths(t *testing.T) {
	attachment := &syncstate.Attachment{
		WorkspaceRoot: "/tmp/work",
		VolumeID:      "vol-123",
	}
	conflicts := []apispec.SyncConflict{
		{
			Path:   apispec.NewOptString("/conflict.txt"),
			Reason: apispec.NewOptString("concurrent_update"),
			Status: apispec.NewOptString("open"),
			Metadata: apispec.NewOptNilSyncConflictMetadata(apispec.SyncConflictMetadata{
				"latest_source": jx.Raw(`"sandbox"`),
			}),
		},
	}

	view := BuildConflictListView(attachment, conflicts, 0)
	if len(view.UnmergedPaths) != 1 {
		t.Fatalf("len(UnmergedPaths) = %d, want 1", len(view.UnmergedPaths))
	}
	if got := view.UnmergedPaths[0].Path; got != "conflict.txt" {
		t.Fatalf("path = %q, want conflict.txt", got)
	}
}

func TestBuildConflictDetailViewIncludesLatestRemoteContext(t *testing.T) {
	now := time.Date(2026, 3, 26, 10, 0, 0, 0, time.UTC)
	conflict := &apispec.SyncConflict{
		ID:             apispec.NewOptString("conflict-1"),
		Path:           apispec.NewOptString("src/main.go"),
		Reason:         apispec.NewOptString("concurrent_update"),
		Status:         apispec.NewOptString("open"),
		ReplicaID:      apispec.NewOptNilString("replica-mac"),
		ArtifactPath:   apispec.NewOptString("src/main.sandbox0-conflict-replica-mac-seq-42.go"),
		NormalizedPath: apispec.NewOptString("src/main.go"),
		ExistingSeq:    apispec.NewOptNilInt64(42),
		CreatedAt:      apispec.NewOptDateTime(now),
		UpdatedAt:      apispec.NewOptDateTime(now.Add(time.Minute)),
		Metadata: apispec.NewOptNilSyncConflictMetadata(apispec.SyncConflictMetadata{
			"latest_path":       jx.Raw(`"src/main.go"`),
			"latest_event":      jx.Raw(`"write"`),
			"latest_source":     jx.Raw(`"sandbox"`),
			"latest_sandbox_id": jx.Raw(`"sandbox-1"`),
			"base_seq":          jx.Raw(`41`),
		}),
	}

	view := BuildConflictDetailView(conflict)
	if got := view.Summary; got != `modified locally, conflicted with sandbox "sandbox-1"` {
		t.Fatalf("Summary = %q", got)
	}
	if got := view.RecordedFor; got != `replica "replica-mac"` {
		t.Fatalf("RecordedFor = %q", got)
	}
	if got := view.LatestRemoteActor; got != `sandbox "sandbox-1"` {
		t.Fatalf("LatestRemoteActor = %q", got)
	}
	if got := view.LatestRemoteEvent; got != "write" {
		t.Fatalf("LatestRemoteEvent = %q, want write", got)
	}
	if got := view.SuggestedNextStep; got == "" || got == "-" {
		t.Fatalf("SuggestedNextStep = %q, want non-empty", got)
	}
}

func TestBuildStatusViewTruncatesConflictSummary(t *testing.T) {
	attachment := &syncstate.Attachment{
		WorkspaceRoot: "/tmp/work",
		VolumeID:      "vol-123",
		LastSync:      &syncstate.SyncCheckpoint{OpenConflictCount: 7},
	}
	conflicts := make([]apispec.SyncConflict, 0, DefaultStatusSummaryLimit+2)
	for i := 0; i < DefaultStatusSummaryLimit+2; i++ {
		conflicts = append(conflicts, apispec.SyncConflict{
			Path:   apispec.NewOptString("file-" + string(rune('a'+i))),
			Reason: apispec.NewOptString("concurrent_update"),
			Status: apispec.NewOptString("open"),
		})
	}

	view := BuildStatusView(attachment, conflicts, nil)
	if view.ConflictSummary == nil {
		t.Fatalf("ConflictSummary is nil")
	}
	if !view.ConflictSummary.Truncated {
		t.Fatalf("Truncated = false, want true")
	}
	if got := view.ConflictSummary.Remaining; got != 2 {
		t.Fatalf("Remaining = %d, want 2", got)
	}
	if got := len(view.ConflictSummary.UnmergedPaths); got != DefaultStatusSummaryLimit {
		t.Fatalf("len(UnmergedPaths) = %d, want %d", got, DefaultStatusSummaryLimit)
	}
}
