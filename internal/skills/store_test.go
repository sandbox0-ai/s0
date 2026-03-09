package skills

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreInstallActivateAndSync(t *testing.T) {
	archiveBytes := buildArchive(t)
	sum := sha256.Sum256(archiveBytes)
	checksum := hex.EncodeToString(sum[:])

	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/agent-skills/sandbox0":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"name":           "sandbox0",
					"releaseVersion": "0.1.0",
					"releaseTag":     "v0.1.0",
					"artifactPrefix": "sandbox0-agent-skill",
					"sourcePriority": []string{"source-code"},
					"downloadUrl":    baseURL + "/release/download",
					"checksumUrl":    baseURL + "/release/checksum",
					"manifestUrl":    baseURL + "/release/manifest",
				},
			})
		case "/api/v1/agent-skills/sandbox0/download", "/release/download":
			_, _ = w.Write(archiveBytes)
		case "/api/v1/agent-skills/sandbox0/checksum", "/release/checksum":
			_, _ = w.Write([]byte(checksum + "  sandbox0-agent-skill-0.1.0.tar.gz\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	baseURL = server.URL

	root := t.TempDir()
	store := NewStore(root)
	client := NewAPIClient(server.URL, "token", "s0/test")

	installed, err := store.Install(context.Background(), client, "sandbox0", false, true)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	if !installed.Active {
		t.Fatalf("expected active install")
	}

	active, err := store.ActiveVersion("sandbox0")
	if err != nil {
		t.Fatalf("active version: %v", err)
	}
	if active != "0.1.0" {
		t.Fatalf("unexpected active version %q", active)
	}

	list, err := store.List("sandbox0")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(list) != 1 || list[0].Version != "0.1.0" {
		t.Fatalf("unexpected list %+v", list)
	}

	target := filepath.Join(t.TempDir(), "codex-skill")
	if err := store.Sync("sandbox0", "", target, true); err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "SKILL.md")); err != nil {
		t.Fatalf("synced SKILL.md missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "bundled-docs", "manifest.json")); err != nil {
		t.Fatalf("synced bundled docs missing: %v", err)
	}
}

func buildArchive(t *testing.T) []byte {
	t.Helper()
	tmpDir := t.TempDir()
	root := filepath.Join(tmpDir, "sandbox0-agent-skill-0.1.0")
	if err := os.MkdirAll(filepath.Join(root, "skill", "references"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "bundled-docs"), 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "skill", "SKILL.md"), []byte("skill\n"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "bundled-docs", "manifest.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write docs manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "SHA256SUMS"), []byte("abc  manifest.json\n"), 0o644); err != nil {
		t.Fatalf("write checksums: %v", err)
	}

	archivePath := filepath.Join(tmpDir, "bundle.tar.gz")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(tmpDir, path)
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if info.IsDir() && header.Name[len(header.Name)-1] != '/' {
			header.Name += "/"
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		_, err = io.Copy(tarWriter, input)
		return err
	})
	if err != nil {
		t.Fatalf("walk archive: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close file: %v", err)
	}
	data, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	return data
}
