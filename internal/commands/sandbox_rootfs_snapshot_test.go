package commands

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
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

func TestBuildSandboxForkRequest(t *testing.T) {
	resetSandboxFlagsForTest()
	cmd := newSandboxForkFlagsForTest()

	if request := buildSandboxForkRequest(cmd); request != nil {
		t.Fatalf("request = %+v, want nil", request)
	}

	if err := cmd.Flags().Set("ttl", "0"); err != nil {
		t.Fatalf("set ttl flag: %v", err)
	}
	if err := cmd.Flags().Set("hard-ttl", "120"); err != nil {
		t.Fatalf("set hard-ttl flag: %v", err)
	}

	request := buildSandboxForkRequest(cmd)
	if request == nil {
		t.Fatal("request is nil")
	}
	config, ok := request.Config.Get()
	if !ok {
		t.Fatal("config not set")
	}
	ttl, ok := config.TTL.Get()
	if !ok || ttl != 0 {
		t.Fatalf("ttl = %d, %v; want 0, true", ttl, ok)
	}
	hardTTL, ok := config.HardTTL.Get()
	if !ok || hardTTL != 120 {
		t.Fatalf("hardTTL = %d, %v; want 120, true", hardTTL, ok)
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
	if flag := sandboxForkCmd.Flags().Lookup("ttl"); flag == nil {
		t.Fatal("sandbox fork --ttl flag is not registered")
	}
	if flag := sandboxForkCmd.Flags().Lookup("hard-ttl"); flag == nil {
		t.Fatal("sandbox fork --hard-ttl flag is not registered")
	}
}

func newSandboxForkFlagsForTest() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().Int32Var(&sandboxForkTTL, "ttl", 0, "soft TTL in seconds for the forked sandbox")
	cmd.Flags().Int32Var(&sandboxForkHardTTL, "hard-ttl", 0, "hard TTL in seconds for the forked sandbox")
	return cmd
}
