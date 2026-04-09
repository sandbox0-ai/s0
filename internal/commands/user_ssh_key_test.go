package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildCreateSSHPublicKeyRequest(t *testing.T) {
	reset := func() {
		sshKeyName = ""
		sshPublicKey = ""
		sshKeyFile = ""
	}
	t.Cleanup(reset)

	t.Run("inline key requires name", func(t *testing.T) {
		reset()
		sshPublicKey = "ssh-ed25519 AAAA test@example"

		_, err := buildCreateSSHPublicKeyRequest()
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("inline key with name", func(t *testing.T) {
		reset()
		sshKeyName = "macbook"
		sshPublicKey = "ssh-ed25519 AAAA test@example"

		req, err := buildCreateSSHPublicKeyRequest()
		if err != nil {
			t.Fatalf("buildCreateSSHPublicKeyRequest() error = %v", err)
		}
		if req.Name != "macbook" {
			t.Fatalf("name = %q, want macbook", req.Name)
		}
	})

	t.Run("file derives name", func(t *testing.T) {
		reset()
		dir := t.TempDir()
		path := filepath.Join(dir, "id_ed25519.pub")
		if err := os.WriteFile(path, []byte("ssh-ed25519 AAAA file@example\n"), 0o644); err != nil {
			t.Fatalf("write public key: %v", err)
		}
		sshKeyFile = path

		req, err := buildCreateSSHPublicKeyRequest()
		if err != nil {
			t.Fatalf("buildCreateSSHPublicKeyRequest() error = %v", err)
		}
		if req.Name != "id_ed25519" {
			t.Fatalf("name = %q, want id_ed25519", req.Name)
		}
		if req.PublicKey != "ssh-ed25519 AAAA file@example" {
			t.Fatalf("public key = %q", req.PublicKey)
		}
	})

	t.Run("rejects both inline and file", func(t *testing.T) {
		reset()
		sshKeyName = "macbook"
		sshPublicKey = "ssh-ed25519 AAAA inline@example"
		sshKeyFile = "/tmp/id_ed25519.pub"

		_, err := buildCreateSSHPublicKeyRequest()
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
