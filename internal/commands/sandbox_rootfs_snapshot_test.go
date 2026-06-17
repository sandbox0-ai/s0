package commands

import (
	"testing"
	"time"
)

func TestBuildSandboxRootFSSnapshotCreateRequest(t *testing.T) {
	resetSandboxFlagsForTest()

	sandboxRootFSSnapshotName = "checkpoint"
	sandboxRootFSSnapshotDescription = "before upgrade"
	sandboxRootFSSnapshotExpiresAt = "2026-01-02T03:04:05Z"

	request, err := buildSandboxRootFSSnapshotCreateRequest()
	if err != nil {
		t.Fatalf("buildSandboxRootFSSnapshotCreateRequest() error = %v", err)
	}
	if request == nil {
		t.Fatal("request is nil")
	}

	name, ok := request.Name.Get()
	if !ok || name != "checkpoint" {
		t.Fatalf("Name = %q, %v; want checkpoint, true", name, ok)
	}
	description, ok := request.Description.Get()
	if !ok || description != "before upgrade" {
		t.Fatalf("Description = %q, %v; want before upgrade, true", description, ok)
	}
	expiresAt, ok := request.ExpiresAt.Get()
	if !ok {
		t.Fatal("ExpiresAt is not set")
	}
	wantExpiresAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if !expiresAt.Equal(wantExpiresAt) {
		t.Fatalf("ExpiresAt = %s, want %s", expiresAt, wantExpiresAt)
	}
}

func TestBuildSandboxRootFSSnapshotCreateRequestReturnsNilWithoutMetadata(t *testing.T) {
	resetSandboxFlagsForTest()

	request, err := buildSandboxRootFSSnapshotCreateRequest()
	if err != nil {
		t.Fatalf("buildSandboxRootFSSnapshotCreateRequest() error = %v", err)
	}
	if request != nil {
		t.Fatalf("request = %+v, want nil", request)
	}
}

func TestBuildSandboxRootFSSnapshotCreateRequestRejectsInvalidExpiresAt(t *testing.T) {
	resetSandboxFlagsForTest()

	sandboxRootFSSnapshotExpiresAt = "tomorrow"

	_, err := buildSandboxRootFSSnapshotCreateRequest()
	if err == nil {
		t.Fatal("buildSandboxRootFSSnapshotCreateRequest() error = nil, want error")
	}
}

func TestSandboxRootFSCommandsRegistered(t *testing.T) {
	if sandboxCmd.Commands() == nil {
		t.Fatal("sandbox commands are not registered")
	}
	if cmd, _, err := sandboxCmd.Find([]string{"snapshot", "create"}); err != nil || cmd != sandboxSnapshotCreateCmd {
		t.Fatalf("sandbox snapshot create command not registered: cmd=%v err=%v", cmd, err)
	}
	if cmd, _, err := sandboxCmd.Find([]string{"fork"}); err != nil || cmd != sandboxForkCmd {
		t.Fatalf("sandbox fork command not registered: cmd=%v err=%v", cmd, err)
	}
	if flag := sandboxSnapshotCreateCmd.Flags().Lookup("expires-at"); flag == nil {
		t.Fatal("sandbox snapshot create --expires-at flag is not registered")
	}
}
