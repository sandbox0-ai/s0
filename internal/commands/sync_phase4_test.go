package commands

import (
	"path/filepath"
	"testing"

	"github.com/sandbox0-ai/s0/internal/syncstate"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestNeedsWorkerRecovery(t *testing.T) {
	tests := map[string]bool{
		"running":         false,
		"stopped":         false,
		"stale":           true,
		"error":           false,
		"reseed_required": false,
	}
	for input, want := range tests {
		if got := needsWorkerRecovery(input); got != want {
			t.Fatalf("needsWorkerRecovery(%q) = %t, want %t", input, got, want)
		}
	}
}

func TestConflictMatchesPathNormalizesLogicalPaths(t *testing.T) {
	conflict := apispec.SyncConflict{
		Path:            apispec.NewOptString("/conflict.txt"),
		IncomingPath:    apispec.NewOptNilString("/conflict.txt"),
		IncomingOldPath: apispec.NewOptNilString("/old-conflict.txt"),
	}

	for _, candidate := range []string{"conflict.txt", "/conflict.txt", "./conflict.txt"} {
		if !conflictMatchesPath(conflict, normalizeConflictPath(candidate)) {
			t.Fatalf("conflictMatchesPath(%q) = false, want true", candidate)
		}
	}
}

func TestResolveWorkspaceRelativePathAcceptsLogicalAbsoluteConflictPath(t *testing.T) {
	attachment := &syncstate.Attachment{WorkspaceRoot: filepath.Clean("/tmp/work")}
	got, err := resolveWorkspaceRelativePath(attachment, "/conflict.txt")
	if err != nil {
		t.Fatalf("resolveWorkspaceRelativePath() error = %v", err)
	}
	if got != "conflict.txt" {
		t.Fatalf("resolveWorkspaceRelativePath() = %q, want conflict.txt", got)
	}
}
