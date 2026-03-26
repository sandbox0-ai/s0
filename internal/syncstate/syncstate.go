package syncstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"
)

const (
	workerHeartbeatTTL = 8 * time.Second
)

// Attachment represents one local workspace bound to one Sandbox0 volume.
type Attachment struct {
	ID            string          `json:"id" yaml:"id"`
	DisplayName   string          `json:"display_name,omitempty" yaml:"display_name,omitempty"`
	WorkspaceRoot string          `json:"workspace_root" yaml:"workspace_root"`
	VolumeID      string          `json:"volume_id" yaml:"volume_id"`
	ReplicaID     string          `json:"replica_id" yaml:"replica_id"`
	Platform      string          `json:"platform" yaml:"platform"`
	Capabilities  FilesystemCaps  `json:"capabilities" yaml:"capabilities"`
	InitFrom      string          `json:"init_from" yaml:"init_from"`
	Ignore        IgnoreConfig    `json:"ignore" yaml:"ignore"`
	Worker        WorkerState     `json:"worker" yaml:"worker"`
	CreatedAt     time.Time       `json:"created_at" yaml:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" yaml:"updated_at"`
	LastError     string          `json:"last_error,omitempty" yaml:"last_error,omitempty"`
	LastSync      *SyncCheckpoint `json:"last_sync,omitempty" yaml:"last_sync,omitempty"`
}

// IgnoreConfig stores built-in local ignore defaults for future sync worker phases.
type IgnoreConfig struct {
	BuiltinPatterns []string `json:"builtin_patterns" yaml:"builtin_patterns"`
}

// FilesystemCaps mirrors the server-side capability contract used during replica registration.
type FilesystemCaps struct {
	CaseSensitive                   bool `json:"case_sensitive" yaml:"case_sensitive"`
	UnicodeNormalizationInsensitive bool `json:"unicode_normalization_insensitive" yaml:"unicode_normalization_insensitive"`
	WindowsCompatiblePaths          bool `json:"windows_compatible_paths" yaml:"windows_compatible_paths"`
}

// WorkerState tracks the local worker process.
type WorkerState struct {
	Mode            string     `json:"mode,omitempty" yaml:"mode,omitempty"`
	Status          string     `json:"status,omitempty" yaml:"status,omitempty"`
	PID             int        `json:"pid,omitempty" yaml:"pid,omitempty"`
	LogPath         string     `json:"log_path,omitempty" yaml:"log_path,omitempty"`
	LastStartedAt   *time.Time `json:"last_started_at,omitempty" yaml:"last_started_at,omitempty"`
	LastStoppedAt   *time.Time `json:"last_stopped_at,omitempty" yaml:"last_stopped_at,omitempty"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at,omitempty" yaml:"last_heartbeat_at,omitempty"`
}

// SyncCheckpoint reserves the persisted shape for later sync phases.
type SyncCheckpoint struct {
	HeadSeq           int64      `json:"head_seq" yaml:"head_seq"`
	LastAppliedSeq    int64      `json:"last_applied_seq" yaml:"last_applied_seq"`
	LastSuccessAt     *time.Time `json:"last_success_at,omitempty" yaml:"last_success_at,omitempty"`
	LastFailureAt     *time.Time `json:"last_failure_at,omitempty" yaml:"last_failure_at,omitempty"`
	LastReplicaSyncAt *time.Time `json:"last_replica_sync_at,omitempty" yaml:"last_replica_sync_at,omitempty"`
	ConsecutiveErrors int        `json:"consecutive_errors" yaml:"consecutive_errors"`
	ReseedRequired    bool       `json:"reseed_required" yaml:"reseed_required"`
	OpenConflictCount int        `json:"open_conflict_count" yaml:"open_conflict_count"`
}

// Manifest stores the last uploaded local filesystem snapshot for delta generation.
type Manifest struct {
	Entries map[string]ManifestEntry `json:"entries" yaml:"entries"`
}

// ManifestEntry stores one logical workspace entry in slash-separated form.
type ManifestEntry struct {
	Path   string `json:"path" yaml:"path"`
	Kind   string `json:"kind" yaml:"kind"`
	Mode   uint32 `json:"mode" yaml:"mode"`
	Size   int64  `json:"size" yaml:"size"`
	SHA256 string `json:"sha256,omitempty" yaml:"sha256,omitempty"`
}

// NewAttachment builds the initial persisted attachment record.
func NewAttachment(workspaceRoot, volumeID, displayName, initFrom string) (*Attachment, error) {
	root, err := canonicalPath(workspaceRoot)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	id := attachmentID(root)
	return &Attachment{
		ID:            id,
		DisplayName:   strings.TrimSpace(displayName),
		WorkspaceRoot: root,
		VolumeID:      strings.TrimSpace(volumeID),
		ReplicaID:     replicaID(root),
		Platform:      runtime.GOOS,
		Capabilities:  defaultCapabilities(runtime.GOOS),
		InitFrom:      normalizeInitFrom(initFrom),
		Ignore: IgnoreConfig{
			BuiltinPatterns: DefaultIgnorePatterns(),
		},
		Worker: WorkerState{
			Status:  "stopped",
			LogPath: logPath(id),
		},
		CreatedAt: now,
		UpdatedAt: now,
		LastSync:  &SyncCheckpoint{},
	}, nil
}

// DefaultIgnorePatterns returns built-in local-only patterns reserved for future phases.
func DefaultIgnorePatterns() []string {
	return []string{
		".git/",
	}
}

// ListAttachments returns all persisted attachments sorted by workspace root.
func ListAttachments() ([]Attachment, error) {
	dir, err := attachmentsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	attachments := make([]Attachment, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		attachment, err := LoadAttachmentByID(strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, *attachment)
	}

	slices.SortFunc(attachments, func(a, b Attachment) int {
		return strings.Compare(a.WorkspaceRoot, b.WorkspaceRoot)
	})
	return attachments, nil
}

// LoadAttachmentByID loads one attachment record by ID.
func LoadAttachmentByID(id string) (*Attachment, error) {
	data, err := os.ReadFile(attachmentPath(id))
	if err != nil {
		return nil, err
	}
	var attachment Attachment
	if err := json.Unmarshal(data, &attachment); err != nil {
		return nil, err
	}
	return &attachment, nil
}

// SaveAttachment persists one attachment atomically.
func SaveAttachment(attachment *Attachment) error {
	if attachment == nil {
		return errors.New("attachment is nil")
	}
	dir, err := attachmentsDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	attachment.UpdatedAt = time.Now().UTC()
	payload, err := json.MarshalIndent(attachment, "", "  ")
	if err != nil {
		return err
	}
	path := attachmentPath(attachment.ID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// UpdateAttachment loads, mutates, and persists one attachment atomically enough for the local worker.
func UpdateAttachment(id string, update func(*Attachment) error) (*Attachment, error) {
	if update == nil {
		return nil, errors.New("update function is nil")
	}
	attachment, err := LoadAttachmentByID(id)
	if err != nil {
		return nil, err
	}
	if attachment.LastSync == nil {
		attachment.LastSync = &SyncCheckpoint{}
	}
	if err := update(attachment); err != nil {
		return nil, err
	}
	if err := SaveAttachment(attachment); err != nil {
		return nil, err
	}
	return attachment, nil
}

// DeleteAttachment removes one attachment record.
func DeleteAttachment(id string) error {
	err := os.Remove(attachmentPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return DeleteManifest(id)
	}
	if err != nil {
		return err
	}
	return DeleteManifest(id)
}

// ResolveAttachmentTarget resolves either an explicit target or the current directory.
func ResolveAttachmentTarget(target, cwd string) (*Attachment, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return ResolveAttachmentFromPath(cwd)
	}

	if attachment, err := ResolveAttachmentFromPath(target); err == nil {
		return attachment, nil
	}

	matches, err := findByVolumeID(target)
	if err != nil {
		return nil, err
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no attached workspace found for %q", target)
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("volume %q is attached to multiple local workspaces; use an explicit path", target)
	}
}

// ResolveAttachmentFromPath finds the nearest attached ancestor for the given path.
func ResolveAttachmentFromPath(path string) (*Attachment, error) {
	root, err := canonicalPath(path)
	if err != nil {
		return nil, err
	}
	attachments, err := ListAttachments()
	if err != nil {
		return nil, err
	}
	var selected *Attachment
	for _, attachment := range attachments {
		if !isAncestorPath(attachment.WorkspaceRoot, root) {
			continue
		}
		clone := attachment
		if selected == nil || len(clone.WorkspaceRoot) > len(selected.WorkspaceRoot) {
			selected = &clone
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("current directory is not inside an attached workspace")
	}
	return selected, nil
}

// EffectiveStatus returns the worker state label after stale-heartbeat detection.
func EffectiveStatus(attachment *Attachment) string {
	if attachment == nil {
		return "unknown"
	}
	status := strings.TrimSpace(attachment.Worker.Status)
	if status == "" {
		status = "stopped"
	}
	if status == "running" && heartbeatExpired(attachment.Worker.LastHeartbeatAt) {
		return "stale"
	}
	return status
}

// SyncHealth returns the latest persisted sync health independent from worker liveness.
func SyncHealth(attachment *Attachment) string {
	if attachment == nil {
		return "unknown"
	}
	if attachment.LastSync == nil {
		return "idle"
	}
	switch {
	case attachment.LastSync.ReseedRequired:
		return "reseed_required"
	case attachment.LastSync.ConsecutiveErrors > 0 && strings.TrimSpace(attachment.LastError) != "":
		return "error"
	case attachment.LastSync.LastSuccessAt != nil:
		return "ok"
	default:
		return "idle"
	}
}

// TouchWorkerHeartbeat updates the worker heartbeat fields on one attachment.
func TouchWorkerHeartbeat(attachmentID, mode string, pid int) (*Attachment, error) {
	attachment, err := LoadAttachmentByID(attachmentID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	attachment.Worker.Mode = strings.TrimSpace(mode)
	attachment.Worker.Status = "running"
	attachment.Worker.PID = pid
	if attachment.Worker.LastStartedAt == nil {
		attachment.Worker.LastStartedAt = &now
	}
	attachment.Worker.LastHeartbeatAt = &now
	attachment.LastError = ""
	if err := SaveAttachment(attachment); err != nil {
		return nil, err
	}
	return attachment, nil
}

// MarkWorkerStarted sets the worker start metadata.
func MarkWorkerStarted(attachmentID, mode string, pid int) (*Attachment, error) {
	attachment, err := LoadAttachmentByID(attachmentID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	attachment.Worker.Mode = strings.TrimSpace(mode)
	attachment.Worker.Status = "running"
	attachment.Worker.PID = pid
	attachment.Worker.LastStartedAt = &now
	attachment.Worker.LastHeartbeatAt = &now
	attachment.LastError = ""
	if err := SaveAttachment(attachment); err != nil {
		return nil, err
	}
	return attachment, nil
}

// MarkWorkerStopped updates the worker state after a clean shutdown.
func MarkWorkerStopped(attachmentID string, workerErr error) (*Attachment, error) {
	attachment, err := LoadAttachmentByID(attachmentID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	attachment.Worker.PID = 0
	attachment.Worker.Status = "stopped"
	attachment.Worker.LastStoppedAt = &now
	if workerErr != nil {
		attachment.LastError = workerErr.Error()
	}
	if err := SaveAttachment(attachment); err != nil {
		return nil, err
	}
	return attachment, nil
}

// ResetWorkerState prepares one attachment for local worker takeover after stale or crashed state.
func ResetWorkerState(attachmentID string) (*Attachment, error) {
	return UpdateAttachment(attachmentID, func(attachment *Attachment) error {
		now := time.Now().UTC()
		attachment.Worker.Status = "stopped"
		attachment.Worker.PID = 0
		attachment.Worker.LastStoppedAt = &now
		return nil
	})
}

// attachmentPath returns the JSON record path for one attachment.
func attachmentPath(id string) string {
	return filepath.Join(rootDir(), "attachments", id+".json")
}

// LogPath returns the persisted log file path for one attachment.
func LogPath(id string) string {
	return logPath(id)
}

// LoadManifest loads the last local manifest snapshot for one attachment.
func LoadManifest(id string) (*Manifest, error) {
	data, err := os.ReadFile(manifestPath(id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Manifest{Entries: map[string]ManifestEntry{}}, nil
		}
		return nil, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	if manifest.Entries == nil {
		manifest.Entries = map[string]ManifestEntry{}
	}
	return &manifest, nil
}

// SaveManifest persists one manifest atomically.
func SaveManifest(id string, manifest *Manifest) error {
	if manifest == nil {
		return errors.New("manifest is nil")
	}
	if manifest.Entries == nil {
		manifest.Entries = map[string]ManifestEntry{}
	}
	dir := filepath.Join(rootDir(), "manifests")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	path := manifestPath(id)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// DeleteManifest removes one manifest file.
func DeleteManifest(id string) error {
	err := os.Remove(manifestPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// EnsureLogDir prepares the log directory on disk.
func EnsureLogDir() error {
	return os.MkdirAll(filepath.Join(rootDir(), "logs"), 0o755)
}

func logPath(id string) string {
	return filepath.Join(rootDir(), "logs", id+".log")
}

func manifestPath(id string) string {
	return filepath.Join(rootDir(), "manifests", id+".json")
}

func attachmentsDir() (string, error) {
	dir := filepath.Join(rootDir(), "attachments")
	return dir, nil
}

func rootDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".s0", "sync")
	}
	return filepath.Join(home, ".s0", "sync")
}

func findByVolumeID(volumeID string) ([]Attachment, error) {
	attachments, err := ListAttachments()
	if err != nil {
		return nil, err
	}
	matches := make([]Attachment, 0, 1)
	for _, attachment := range attachments {
		if attachment.VolumeID == strings.TrimSpace(volumeID) {
			matches = append(matches, attachment)
		}
	}
	return matches, nil
}

func canonicalPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path is empty")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return absolute, nil
		}
		return "", err
	}
	return resolved, nil
}

func attachmentID(workspaceRoot string) string {
	sum := sha256.Sum256([]byte(workspaceRoot))
	return hex.EncodeToString(sum[:8])
}

func replicaID(workspaceRoot string) string {
	host, _ := os.Hostname()
	host = sanitizeLabel(host)
	if host == "" {
		host = "local"
	}
	return fmt.Sprintf("s0-%s-%s", host, attachmentID(workspaceRoot))
}

func normalizeInitFrom(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "volume", "local":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "auto"
	}
}

func sanitizeLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-' || r == '_':
			builder.WriteRune('-')
		}
	}
	return strings.Trim(builder.String(), "-")
}

func defaultCapabilities(platform string) FilesystemCaps {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "windows":
		return FilesystemCaps{
			CaseSensitive:                   false,
			UnicodeNormalizationInsensitive: true,
			WindowsCompatiblePaths:          true,
		}
	case "darwin", "macos":
		return FilesystemCaps{
			CaseSensitive:                   false,
			UnicodeNormalizationInsensitive: true,
			WindowsCompatiblePaths:          false,
		}
	default:
		return FilesystemCaps{
			CaseSensitive:                   true,
			UnicodeNormalizationInsensitive: false,
			WindowsCompatiblePaths:          false,
		}
	}
}

func isAncestorPath(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	if root == path {
		return true
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func heartbeatExpired(lastHeartbeat *time.Time) bool {
	if lastHeartbeat == nil {
		return true
	}
	return time.Since(*lastHeartbeat) > workerHeartbeatTTL
}
