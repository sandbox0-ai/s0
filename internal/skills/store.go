package skills

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Store struct {
	root string
}

type InstallMetadata struct {
	Name           string    `json:"name"`
	ReleaseVersion string    `json:"releaseVersion"`
	ReleaseTag     string    `json:"releaseTag"`
	ArtifactURL    string    `json:"artifactUrl"`
	Checksum       string    `json:"checksum"`
	InstalledAt    time.Time `json:"installedAt"`
}

type InstalledVersion struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Active      bool      `json:"active"`
	InstalledAt time.Time `json:"installedAt"`
	Path        string    `json:"path"`
}

func DefaultRootDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".s0", "skills"), nil
}

func NewStore(root string) *Store {
	return &Store{root: root}
}

func (s *Store) Root() string {
	return s.root
}

func (s *Store) Install(ctx context.Context, api *APIClient, name string, force, activate bool) (*InstalledVersion, error) {
	release, err := api.GetRelease(ctx, name)
	if err != nil {
		return nil, err
	}

	versionDir := s.versionDir(release.Name, release.ReleaseVersion)
	if !force {
		if existing, err := s.GetInstalledVersion(release.Name, release.ReleaseVersion); err == nil {
			if activate {
				if err := s.Activate(release.Name, release.ReleaseVersion); err != nil {
					return nil, err
				}
				existing.Active = true
			}
			return existing, nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(versionDir), 0o755); err != nil {
		return nil, fmt.Errorf("create version parent dir: %w", err)
	}

	tempRoot := filepath.Join(s.root, ".tmp")
	if err := os.MkdirAll(tempRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create temp root: %w", err)
	}
	tmpDir, err := os.MkdirTemp(tempRoot, "install-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, "skill.tar.gz")
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return nil, fmt.Errorf("create artifact file: %w", err)
	}
	artifactURL, downloadErr := api.DownloadArtifact(ctx, release.Name, archiveFile)
	closeErr := archiveFile.Close()
	if downloadErr != nil {
		return nil, downloadErr
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close artifact file: %w", closeErr)
	}

	expectedChecksum, err := api.DownloadChecksum(ctx, release.Name)
	if err != nil {
		return nil, err
	}
	actualChecksum, err := SHA256File(archivePath)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(expectedChecksum, actualChecksum) {
		return nil, fmt.Errorf("checksum mismatch: expected %s got %s", expectedChecksum, actualChecksum)
	}

	extractRoot := filepath.Join(tmpDir, "extract")
	if err := os.MkdirAll(extractRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create extract dir: %w", err)
	}
	if err := extractTarGz(archivePath, extractRoot); err != nil {
		return nil, err
	}
	bundleDir, err := singleRootDir(extractRoot)
	if err != nil {
		return nil, err
	}

	if force {
		if err := os.RemoveAll(versionDir); err != nil {
			return nil, fmt.Errorf("remove existing version dir: %w", err)
		}
	}
	if err := os.Rename(bundleDir, versionDir); err != nil {
		return nil, fmt.Errorf("move installed bundle: %w", err)
	}

	installedAt := time.Now().UTC().Truncate(time.Second)
	installMetadata := InstallMetadata{
		Name:           release.Name,
		ReleaseVersion: release.ReleaseVersion,
		ReleaseTag:     release.ReleaseTag,
		ArtifactURL:    artifactURL,
		Checksum:       actualChecksum,
		InstalledAt:    installedAt,
	}
	if err := writeJSON(filepath.Join(versionDir, ".install.json"), installMetadata); err != nil {
		return nil, err
	}

	if activate {
		if err := s.Activate(release.Name, release.ReleaseVersion); err != nil {
			return nil, err
		}
	}

	return &InstalledVersion{
		Name:        release.Name,
		Version:     release.ReleaseVersion,
		Active:      activate,
		InstalledAt: installedAt,
		Path:        versionDir,
	}, nil
}

func (s *Store) List(name string) ([]InstalledVersion, error) {
	if strings.TrimSpace(name) != "" {
		return s.listOne(name)
	}

	entries, err := os.ReadDir(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read skill root: %w", err)
	}
	var installed []InstalledVersion
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		items, err := s.listOne(entry.Name())
		if err != nil {
			return nil, err
		}
		installed = append(installed, items...)
	}
	sort.Slice(installed, func(i, j int) bool {
		if installed[i].Name == installed[j].Name {
			return installed[i].Version < installed[j].Version
		}
		return installed[i].Name < installed[j].Name
	})
	return installed, nil
}

func (s *Store) GetInstalledVersion(name, version string) (*InstalledVersion, error) {
	path := s.versionDir(name, version)
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", path)
	}

	activeVersion, _ := s.ActiveVersion(name)
	metadata, err := s.readInstallMetadata(path)
	if err != nil {
		return nil, err
	}
	return &InstalledVersion{
		Name:        name,
		Version:     version,
		Active:      activeVersion == version,
		InstalledAt: metadata.InstalledAt,
		Path:        path,
	}, nil
}

func (s *Store) ActiveVersion(name string) (string, error) {
	data, err := os.ReadFile(filepath.Join(s.skillDir(name), "active_version"))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (s *Store) Activate(name, version string) error {
	if _, err := s.GetInstalledVersion(name, version); err != nil {
		return fmt.Errorf("activate skill: %w", err)
	}
	if err := os.MkdirAll(s.skillDir(name), 0o755); err != nil {
		return fmt.Errorf("create skill dir: %w", err)
	}
	return os.WriteFile(filepath.Join(s.skillDir(name), "active_version"), []byte(version+"\n"), 0o644)
}

func (s *Store) Sync(name, version, targetDir string, force bool) error {
	if strings.TrimSpace(targetDir) == "" {
		return fmt.Errorf("target directory is required")
	}
	resolvedVersion := strings.TrimSpace(version)
	if resolvedVersion == "" {
		active, err := s.ActiveVersion(name)
		if err != nil {
			return fmt.Errorf("resolve active version: %w", err)
		}
		resolvedVersion = active
	}

	sourceDir := s.versionDir(name, resolvedVersion)
	if _, err := os.Stat(sourceDir); err != nil {
		return fmt.Errorf("stat source skill version: %w", err)
	}

	if !force {
		if entries, err := os.ReadDir(targetDir); err == nil && len(entries) > 0 {
			return fmt.Errorf("target directory %s is not empty; use --force to replace it", targetDir)
		}
	}
	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("remove target dir: %w", err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("create target dir: %w", err)
	}

	if err := copyDir(filepath.Join(sourceDir, "skill"), targetDir); err != nil {
		return fmt.Errorf("copy skill tree: %w", err)
	}
	if err := copyDir(filepath.Join(sourceDir, "bundled-docs"), filepath.Join(targetDir, "bundled-docs")); err != nil {
		return fmt.Errorf("copy bundled docs: %w", err)
	}
	for _, fileName := range []string{"manifest.json", "SHA256SUMS", ".install.json"} {
		sourcePath := filepath.Join(sourceDir, fileName)
		if _, err := os.Stat(sourcePath); err == nil {
			if err := copyFile(sourcePath, filepath.Join(targetDir, fileName), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) skillDir(name string) string {
	return filepath.Join(s.root, name)
}

func (s *Store) versionDir(name, version string) string {
	return filepath.Join(s.root, name, "versions", version)
}

func (s *Store) listOne(name string) ([]InstalledVersion, error) {
	versionRoot := filepath.Join(s.skillDir(name), "versions")
	entries, err := os.ReadDir(versionRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read versions for %s: %w", name, err)
	}
	activeVersion, _ := s.ActiveVersion(name)
	var installed []InstalledVersion
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metadata, err := s.readInstallMetadata(filepath.Join(versionRoot, entry.Name()))
		if err != nil {
			return nil, err
		}
		installed = append(installed, InstalledVersion{
			Name:        name,
			Version:     entry.Name(),
			Active:      activeVersion == entry.Name(),
			InstalledAt: metadata.InstalledAt,
			Path:        filepath.Join(versionRoot, entry.Name()),
		})
	}
	sort.Slice(installed, func(i, j int) bool {
		return installed[i].Version < installed[j].Version
	})
	return installed, nil
}

func (s *Store) readInstallMetadata(versionDir string) (*InstallMetadata, error) {
	data, err := os.ReadFile(filepath.Join(versionDir, ".install.json"))
	if err != nil {
		return nil, fmt.Errorf("read install metadata: %w", err)
	}
	var metadata InstallMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("decode install metadata: %w", err)
	}
	return &metadata, nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func singleRootDir(root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", fmt.Errorf("read extract root: %w", err)
	}
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(root, entry.Name()))
		}
	}
	if len(dirs) != 1 {
		return "", fmt.Errorf("expected a single extracted root directory, got %d", len(dirs))
	}
	return dirs[0], nil
}

func extractTarGz(archivePath, dest string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read archive entry: %w", err)
		}
		target := filepath.Join(dest, filepath.Clean(header.Name))
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) && filepath.Clean(target) != filepath.Clean(dest) {
			return fmt.Errorf("archive entry escapes destination: %s", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("mkdir %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("mkdir %s: %w", filepath.Dir(target), err)
			}
			file, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fs.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("create %s: %w", target, err)
			}
			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return fmt.Errorf("write %s: %w", target, err)
			}
			if err := file.Close(); err != nil {
				return fmt.Errorf("close %s: %w", target, err)
			}
		}
	}
}

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

func copyFile(src, dest string, perm fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(dest), err)
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copy %s: %w", dest, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close %s: %w", dest, err)
	}
	return nil
}
