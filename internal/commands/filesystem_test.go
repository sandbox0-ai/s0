package commands

import "testing"

func TestBuildCreateFilesystemRequest(t *testing.T) {
	req := buildCreateFilesystemRequest("ubuntu-24", "snap_123", "sha256:base", "manifests/0001.json")

	if template, ok := req.Template.Get(); !ok || template != "ubuntu-24" {
		t.Fatalf("Template = %q, %v; want ubuntu-24, true", template, ok)
	}
	if snapshotID, ok := req.SnapshotID.Get(); !ok || snapshotID != "snap_123" {
		t.Fatalf("SnapshotID = %q, %v; want snap_123, true", snapshotID, ok)
	}
	if digest, ok := req.BaseImageDigest.Get(); !ok || digest != "sha256:base" {
		t.Fatalf("BaseImageDigest = %q, %v; want sha256:base, true", digest, ok)
	}
	if head, ok := req.S0fsHead.Get(); !ok || head != "manifests/0001.json" {
		t.Fatalf("S0fsHead = %q, %v; want manifests/0001.json, true", head, ok)
	}
}

func TestBuildCreateFilesystemRequestLeavesFieldsUnsetByDefault(t *testing.T) {
	req := buildCreateFilesystemRequest("", "", "", "")

	if _, ok := req.Template.Get(); ok {
		t.Fatal("Template is set; want unset")
	}
	if _, ok := req.SnapshotID.Get(); ok {
		t.Fatal("SnapshotID is set; want unset")
	}
	if _, ok := req.BaseImageDigest.Get(); ok {
		t.Fatal("BaseImageDigest is set; want unset")
	}
	if _, ok := req.S0fsHead.Get(); ok {
		t.Fatal("S0fsHead is set; want unset")
	}
}

func TestBuildForkFilesystemRequest(t *testing.T) {
	if req := buildForkFilesystemRequest(""); req != nil {
		t.Fatalf("buildForkFilesystemRequest(\"\") = %#v, want nil", req)
	}

	req := buildForkFilesystemRequest("ubuntu-24")
	if req == nil {
		t.Fatal("buildForkFilesystemRequest() = nil, want request")
	}
	if template, ok := req.Template.Get(); !ok || template != "ubuntu-24" {
		t.Fatalf("Template = %q, %v; want ubuntu-24, true", template, ok)
	}
}

func TestFilesystemCommandRegistration(t *testing.T) {
	subcommands := map[string]bool{}
	for _, cmd := range filesystemCmd.Commands() {
		subcommands[cmd.Name()] = true
	}

	for _, name := range []string{"list", "get", "create", "delete", "fork", "snapshot"} {
		if !subcommands[name] {
			t.Fatalf("expected filesystem subcommand %q to be registered", name)
		}
	}
}

func TestFilesystemCommandFlags(t *testing.T) {
	for _, name := range []string{"template", "snapshot-id", "base-image-digest", "s0fs-head"} {
		if flag := filesystemCreateCmd.Flags().Lookup(name); flag == nil {
			t.Fatalf("expected filesystem create --%s flag", name)
		}
	}
	if flag := filesystemForkCmd.Flags().Lookup("template"); flag == nil {
		t.Fatal("expected filesystem fork --template flag")
	}
}
