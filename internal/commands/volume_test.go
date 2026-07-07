package commands

import (
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestBuildCreateVolumeRequestSupportsSnapshotID(t *testing.T) {
	req, err := buildCreateVolumeRequest(createVolumeOptions{
		accessMode: "RWO",
		snapshotID: "snap_123",
	})
	if err != nil {
		t.Fatalf("buildCreateVolumeRequest() error = %v", err)
	}

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
	req, err := buildCreateVolumeRequest(createVolumeOptions{})
	if err != nil {
		t.Fatalf("buildCreateVolumeRequest() error = %v", err)
	}

	if _, ok := req.SnapshotID.Get(); ok {
		t.Fatal("SnapshotID is set; want unset")
	}
}

func TestBuildCreateVolumeRequestSupportsS3Backend(t *testing.T) {
	req, err := buildCreateVolumeRequest(createVolumeOptions{
		backend:       "s3",
		s3Provider:    "r2",
		s3Bucket:      "agent-state",
		s3Prefix:      "team-a/",
		s3Region:      "auto",
		s3EndpointURL: "https://account.r2.cloudflarestorage.com",
		s3AccessKey:   "access",
		s3SecretKey:   "secret",
	})
	if err != nil {
		t.Fatalf("buildCreateVolumeRequest() error = %v", err)
	}

	backend, ok := req.Backend.Get()
	if !ok || backend != apispec.VolumeBackendS3 {
		t.Fatalf("Backend = %q, %v; want s3, true", backend, ok)
	}
	s3, ok := req.S3.Get()
	if !ok {
		t.Fatal("S3 config is unset")
	}
	if s3.Bucket != "agent-state" {
		t.Fatalf("Bucket = %q, want agent-state", s3.Bucket)
	}
	if provider, ok := s3.Provider.Get(); !ok || provider != apispec.CreateSandboxVolumeS3ConfigProviderR2 {
		t.Fatalf("Provider = %q, %v; want r2, true", provider, ok)
	}
	if prefix, ok := s3.Prefix.Get(); !ok || prefix != "team-a/" {
		t.Fatalf("Prefix = %q, %v; want team-a/, true", prefix, ok)
	}
	if accessKey, ok := s3.AccessKey.Get(); !ok || accessKey != "access" {
		t.Fatalf("AccessKey = %q, %v; want access, true", accessKey, ok)
	}
	if secretKey, ok := s3.SecretKey.Get(); !ok || secretKey != "secret" {
		t.Fatalf("SecretKey = %q, %v; want secret, true", secretKey, ok)
	}
}

func TestBuildCreateVolumeRequestInfersS3BackendFromBucket(t *testing.T) {
	req, err := buildCreateVolumeRequest(createVolumeOptions{s3Bucket: "agent-state"})
	if err != nil {
		t.Fatalf("buildCreateVolumeRequest() error = %v", err)
	}

	backend, ok := req.Backend.Get()
	if !ok || backend != apispec.VolumeBackendS3 {
		t.Fatalf("Backend = %q, %v; want s3, true", backend, ok)
	}
}

func TestBuildCreateVolumeRequestRejectsS3Snapshot(t *testing.T) {
	_, err := buildCreateVolumeRequest(createVolumeOptions{
		backend:    "s3",
		s3Bucket:   "agent-state",
		snapshotID: "snap_123",
	})
	if err == nil {
		t.Fatal("buildCreateVolumeRequest() error is nil; want error")
	}
}

func TestBuildCreateVolumeRequestRejectsPartialS3Credentials(t *testing.T) {
	_, err := buildCreateVolumeRequest(createVolumeOptions{
		backend:     "s3",
		s3Bucket:    "agent-state",
		s3AccessKey: "access",
	})
	if err == nil {
		t.Fatal("buildCreateVolumeRequest() error is nil; want error")
	}
}

func TestVolumeCreateCommandRegistersBackendFlags(t *testing.T) {
	for _, name := range []string{
		"snapshot-id",
		"backend",
		"s3-provider",
		"s3-bucket",
		"s3-prefix",
		"s3-region",
		"s3-endpoint-url",
		"s3-access-key",
		"s3-secret-key",
		"s3-session-token",
	} {
		if flag := volumeCreateCmd.Flags().Lookup(name); flag == nil {
			t.Fatalf("volume create --%s flag is not registered", name)
		}
	}
}
