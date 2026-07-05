package commands

import (
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestBuildCreateVolumeRequestSupportsSnapshotID(t *testing.T) {
	req, err := buildCreateVolumeRequest(createVolumeOptions{AccessMode: "RWO", SnapshotID: "snap_123"})
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
		Backend:     "s3",
		S3Provider:  "aws",
		S3Bucket:    "agent-data",
		S3Prefix:    "team-a",
		S3Region:    "us-east-1",
		S3Endpoint:  "https://s3.us-east-1.amazonaws.com",
		S3AccessKey: "access",
		S3SecretKey: "secret",
		S3Token:     "token",
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
	if s3.Bucket != "agent-data" {
		t.Fatalf("S3 bucket = %q, want agent-data", s3.Bucket)
	}
	if provider, ok := s3.Provider.Get(); !ok || provider != apispec.CreateSandboxVolumeS3ConfigProviderAWS {
		t.Fatalf("S3 provider = %q, %v; want aws, true", provider, ok)
	}
	if prefix, ok := s3.Prefix.Get(); !ok || prefix != "team-a" {
		t.Fatalf("S3 prefix = %q, %v; want team-a, true", prefix, ok)
	}
	if token, ok := s3.SessionToken.Get(); !ok || token != "token" {
		t.Fatalf("S3 token = %q, %v; want token, true", token, ok)
	}
}

func TestBuildCreateVolumeRequestInfersS3BackendFromS3Flags(t *testing.T) {
	req, err := buildCreateVolumeRequest(createVolumeOptions{S3Bucket: "agent-data"})
	if err != nil {
		t.Fatalf("buildCreateVolumeRequest() error = %v", err)
	}
	backend, ok := req.Backend.Get()
	if !ok || backend != apispec.VolumeBackendS3 {
		t.Fatalf("Backend = %q, %v; want s3, true", backend, ok)
	}
}

func TestBuildCreateVolumeRequestRejectsInvalidS3Options(t *testing.T) {
	tests := []struct {
		name string
		opts createVolumeOptions
	}{
		{name: "missing bucket", opts: createVolumeOptions{Backend: "s3"}},
		{name: "provider", opts: createVolumeOptions{Backend: "s3", S3Bucket: "bucket", S3Provider: "minio"}},
		{name: "partial credentials", opts: createVolumeOptions{Backend: "s3", S3Bucket: "bucket", S3AccessKey: "access"}},
		{name: "s0fs with s3 flags", opts: createVolumeOptions{Backend: "s0fs", S3Bucket: "bucket"}},
		{name: "unknown backend", opts: createVolumeOptions{Backend: "nfs"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := buildCreateVolumeRequest(tt.opts); err == nil {
				t.Fatal("buildCreateVolumeRequest() error = nil, want error")
			}
		})
	}
}

func TestVolumeCreateCommandRegistersSnapshotIDFlag(t *testing.T) {
	if flag := volumeCreateCmd.Flags().Lookup("snapshot-id"); flag == nil {
		t.Fatal("volume create --snapshot-id flag is not registered")
	}
}

func TestVolumeCreateCommandRegistersS3Flags(t *testing.T) {
	for _, name := range []string{
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
