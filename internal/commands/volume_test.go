package commands

import (
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestBuildCreateVolumeRequestSupportsSnapshotID(t *testing.T) {
	req := buildCreateVolumeRequest("RWO", "snap_123")

	snapshotID, ok := req.SnapshotID.Get()
	if !ok || snapshotID != "snap_123" {
		t.Fatalf("SnapshotID = %q, %v; want snap_123, true", snapshotID, ok)
	}

	accessMode, ok := req.AccessMode.Get()
	if !ok || accessMode != apispec.VolumeAccessModeRWO {
		t.Fatalf("AccessMode = %q, %v; want RWO, true", accessMode, ok)
	}
}

func TestBuildCreateVolumeRequestLeavesSnapshotIDUnsetByDefault(t *testing.T) {
	req := buildCreateVolumeRequest("", "")

	if _, ok := req.SnapshotID.Get(); ok {
		t.Fatal("SnapshotID is set; want unset")
	}
}

func TestVolumeCreateCommandRegistersSnapshotIDFlag(t *testing.T) {
	if flag := volumeCreateCmd.Flags().Lookup("snapshot-id"); flag == nil {
		t.Fatal("volume create --snapshot-id flag is not registered")
	}
}
