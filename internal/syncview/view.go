package syncview

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sandbox0-ai/s0/internal/syncstate"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

const DefaultStatusSummaryLimit = 5

// StatusView renders attachment state plus a short Git-like conflict summary.
type StatusView struct {
	Attachment         *syncstate.Attachment `json:"attachment" yaml:"attachment"`
	ConflictSummary    *ConflictListView     `json:"conflict_summary,omitempty" yaml:"conflict_summary,omitempty"`
	ConflictQueryError string                `json:"conflict_query_error,omitempty" yaml:"conflict_query_error,omitempty"`
}

// ConflictListView renders unresolved conflicts grouped around logical paths.
type ConflictListView struct {
	WorkspaceRoot string              `json:"workspace_root,omitempty" yaml:"workspace_root,omitempty"`
	VolumeID      string              `json:"volume_id,omitempty" yaml:"volume_id,omitempty"`
	OpenCount     int                 `json:"open_count" yaml:"open_count"`
	UnmergedPaths []ConflictListEntry `json:"unmerged_paths,omitempty" yaml:"unmerged_paths,omitempty"`
	Remaining     int                 `json:"remaining,omitempty" yaml:"remaining,omitempty"`
	Truncated     bool                `json:"truncated,omitempty" yaml:"truncated,omitempty"`
}

// ConflictListEntry is one path-centric summary row.
type ConflictListEntry struct {
	Path         string `json:"path" yaml:"path"`
	Summary      string `json:"summary" yaml:"summary"`
	ReasonCode   string `json:"reason_code" yaml:"reason_code"`
	Status       string `json:"status" yaml:"status"`
	ArtifactPath string `json:"artifact_path,omitempty" yaml:"artifact_path,omitempty"`
}

// ConflictDetailView renders one conflict with operator-focused context.
type ConflictDetailView struct {
	ID                  string     `json:"id,omitempty" yaml:"id,omitempty"`
	Path                string     `json:"path" yaml:"path"`
	Summary             string     `json:"summary" yaml:"summary"`
	ReasonCode          string     `json:"reason_code" yaml:"reason_code"`
	Status              string     `json:"status" yaml:"status"`
	RecordedFor         string     `json:"recorded_for,omitempty" yaml:"recorded_for,omitempty"`
	CompatibilityImpact string     `json:"compatibility_impact,omitempty" yaml:"compatibility_impact,omitempty"`
	NormalizedPath      string     `json:"normalized_path,omitempty" yaml:"normalized_path,omitempty"`
	ArtifactPath        string     `json:"artifact_path,omitempty" yaml:"artifact_path,omitempty"`
	IncomingPath        string     `json:"incoming_path,omitempty" yaml:"incoming_path,omitempty"`
	IncomingOldPath     string     `json:"incoming_old_path,omitempty" yaml:"incoming_old_path,omitempty"`
	OtherPaths          []string   `json:"other_paths,omitempty" yaml:"other_paths,omitempty"`
	ExistingSeq         *int64     `json:"existing_seq,omitempty" yaml:"existing_seq,omitempty"`
	LatestRemoteActor   string     `json:"latest_remote_actor,omitempty" yaml:"latest_remote_actor,omitempty"`
	LatestRemotePath    string     `json:"latest_remote_path,omitempty" yaml:"latest_remote_path,omitempty"`
	LatestRemoteEvent   string     `json:"latest_remote_event,omitempty" yaml:"latest_remote_event,omitempty"`
	IssueMessage        string     `json:"issue_message,omitempty" yaml:"issue_message,omitempty"`
	Resolution          string     `json:"resolution,omitempty" yaml:"resolution,omitempty"`
	Note                string     `json:"note,omitempty" yaml:"note,omitempty"`
	ResolvedAt          string     `json:"resolved_at,omitempty" yaml:"resolved_at,omitempty"`
	SuggestedNextStep   string     `json:"suggested_next_step" yaml:"suggested_next_step"`
	CreatedAt           *time.Time `json:"created_at,omitempty" yaml:"created_at,omitempty"`
	UpdatedAt           *time.Time `json:"updated_at,omitempty" yaml:"updated_at,omitempty"`
}

type conflictMetadata struct {
	LatestSeq       *int64               `json:"latest_seq"`
	LatestPath      string               `json:"latest_path"`
	LatestEvent     string               `json:"latest_event"`
	LatestSource    string               `json:"latest_source"`
	LatestSandboxID string               `json:"latest_sandbox_id"`
	LatestReplicaID string               `json:"latest_replica_id"`
	LatestPlatform  string               `json:"latest_platform"`
	BaseSeq         *int64               `json:"base_seq"`
	Issues          []compatibilityIssue `json:"issues"`
	Capabilities    *filesystemCaps      `json:"capabilities"`
	Resolution      string               `json:"resolution"`
	Note            string               `json:"note"`
	ResolvedAt      string               `json:"resolved_at"`
}

type compatibilityIssue struct {
	Code           string   `json:"code"`
	Path           string   `json:"path"`
	NormalizedPath string   `json:"normalized_path"`
	Paths          []string `json:"paths"`
	Segment        string   `json:"segment"`
	Message        string   `json:"message"`
}

type filesystemCaps struct {
	CaseSensitive                   bool `json:"case_sensitive"`
	UnicodeNormalizationInsensitive bool `json:"unicode_normalization_insensitive"`
	WindowsCompatiblePaths          bool `json:"windows_compatible_paths"`
}

// BuildStatusView combines local attachment state with a short live conflict summary.
func BuildStatusView(attachment *syncstate.Attachment, conflicts []apispec.SyncConflict, queryErr error) *StatusView {
	view := &StatusView{
		Attachment: attachment,
	}
	if attachment == nil {
		if queryErr != nil {
			view.ConflictQueryError = queryErr.Error()
		}
		return view
	}

	if len(conflicts) > 0 {
		view.ConflictSummary = BuildConflictListView(attachment, conflicts, DefaultStatusSummaryLimit)
		return view
	}

	openCount := 0
	if attachment.LastSync != nil {
		openCount = attachment.LastSync.OpenConflictCount
	}
	if openCount > 0 {
		view.ConflictSummary = &ConflictListView{
			WorkspaceRoot: attachment.WorkspaceRoot,
			VolumeID:      attachment.VolumeID,
			OpenCount:     openCount,
		}
	}
	if queryErr != nil && openCount > 0 {
		view.ConflictQueryError = queryErr.Error()
	}
	return view
}

// BuildConflictListView creates a path-centric conflict listing. Set limit <= 0 to include all entries.
func BuildConflictListView(attachment *syncstate.Attachment, conflicts []apispec.SyncConflict, limit int) *ConflictListView {
	view := &ConflictListView{}
	if attachment != nil {
		view.WorkspaceRoot = attachment.WorkspaceRoot
		view.VolumeID = attachment.VolumeID
	}
	if len(conflicts) == 0 {
		return view
	}

	entries := make([]ConflictListEntry, 0, len(conflicts))
	for _, conflict := range conflicts {
		entries = append(entries, ConflictListEntry{
			Path:         conflictPrimaryPath(conflict),
			Summary:      conflictSummary(conflict, decodeConflictMetadata(conflict)),
			ReasonCode:   optString(conflict.Reason),
			Status:       optString(conflict.Status),
			ArtifactPath: optString(conflict.ArtifactPath),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})

	view.OpenCount = len(entries)
	if limit > 0 && len(entries) > limit {
		view.UnmergedPaths = entries[:limit]
		view.Remaining = len(entries) - limit
		view.Truncated = true
		return view
	}
	view.UnmergedPaths = entries
	return view
}

// BuildConflictDetailView creates an operator-focused detail view for one conflict.
func BuildConflictDetailView(conflict *apispec.SyncConflict) *ConflictDetailView {
	if conflict == nil {
		return &ConflictDetailView{}
	}

	path := conflictPrimaryPath(*conflict)
	metadata := decodeConflictMetadata(*conflict)
	otherPaths := conflictOtherPaths(*conflict, metadata)
	view := &ConflictDetailView{
		ID:                  optString(conflict.ID),
		Path:                path,
		Summary:             conflictSummary(*conflict, metadata),
		ReasonCode:          optString(conflict.Reason),
		Status:              optString(conflict.Status),
		RecordedFor:         recordedFor(*conflict),
		CompatibilityImpact: compatibilityImpact(*conflict, metadata),
		NormalizedPath:      optString(conflict.NormalizedPath),
		ArtifactPath:        optString(conflict.ArtifactPath),
		IncomingPath:        optNilString(conflict.IncomingPath),
		IncomingOldPath:     optNilString(conflict.IncomingOldPath),
		OtherPaths:          otherPaths,
		ExistingSeq:         optNilInt64Ptr(conflict.ExistingSeq),
		LatestRemoteActor:   remoteActorLabel(metadata),
		LatestRemotePath:    metadata.LatestPath,
		LatestRemoteEvent:   metadata.LatestEvent,
		IssueMessage:        issueMessage(metadata),
		Resolution:          strings.TrimSpace(metadata.Resolution),
		Note:                strings.TrimSpace(metadata.Note),
		ResolvedAt:          strings.TrimSpace(metadata.ResolvedAt),
		SuggestedNextStep:   suggestedNextStep(*conflict, metadata),
		CreatedAt:           optDateTimePtr(conflict.CreatedAt),
		UpdatedAt:           optDateTimePtr(conflict.UpdatedAt),
	}
	return view
}

func conflictSummary(conflict apispec.SyncConflict, metadata conflictMetadata) string {
	switch optString(conflict.Reason) {
	case "concurrent_update":
		return "modified locally, conflicted with " + remoteActorLabel(metadata)
	case "case_only_rename_conflict":
		return "case-only rename conflicted with " + remoteActorLabel(metadata)
	case "casefold_collision":
		return "casefold collision across portable replicas"
	case "windows_reserved_name":
		return "namespace incompatible for Windows-capable replicas"
	case "windows_trailing_dot_space":
		return "path ends with a dot or space and is incompatible for Windows-capable replicas"
	case "windows_forbidden_character":
		return "path contains a Windows-forbidden character"
	case "windows_control_character":
		return "path contains a Windows control character"
	default:
		return humanizeReason(optString(conflict.Reason))
	}
}

func recordedFor(conflict apispec.SyncConflict) string {
	replicaID := optNilString(conflict.ReplicaID)
	if replicaID == "" {
		return ""
	}
	return fmt.Sprintf("replica %q", replicaID)
}

func remoteActorLabel(metadata conflictMetadata) string {
	switch strings.TrimSpace(metadata.LatestSource) {
	case "sandbox":
		if id := strings.TrimSpace(metadata.LatestSandboxID); id != "" {
			return fmt.Sprintf("sandbox %q", id)
		}
		return "sandbox"
	case "replica":
		if id := strings.TrimSpace(metadata.LatestReplicaID); id != "" {
			if platform := strings.TrimSpace(metadata.LatestPlatform); platform != "" {
				return fmt.Sprintf("replica %q (%s)", id, platform)
			}
			return fmt.Sprintf("replica %q", id)
		}
		return "another replica"
	default:
		return "newer remote state"
	}
}

func compatibilityImpact(conflict apispec.SyncConflict, metadata conflictMetadata) string {
	switch optString(conflict.Reason) {
	case "windows_reserved_name", "windows_trailing_dot_space", "windows_forbidden_character", "windows_control_character":
		return "Windows-capable replicas cannot represent this path without repair."
	case "casefold_collision":
		return "Case-insensitive or Unicode-normalization-insensitive replicas may collapse multiple logical paths into one."
	default:
		if metadata.Capabilities != nil && metadata.Capabilities.WindowsCompatiblePaths {
			return "Portable replicas may require namespace repair before this path can sync cleanly."
		}
		return ""
	}
}

func conflictOtherPaths(conflict apispec.SyncConflict, metadata conflictMetadata) []string {
	seen := map[string]struct{}{}
	current := conflictPrimaryPath(conflict)
	appendPath := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" || path == current {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
	}

	if metadata.LatestPath != "" {
		appendPath(metadata.LatestPath)
	}
	for _, issue := range metadata.Issues {
		for _, path := range issue.Paths {
			appendPath(path)
		}
	}

	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func suggestedNextStep(conflict apispec.SyncConflict, metadata conflictMetadata) string {
	path := conflictPrimaryPath(conflict)
	switch optString(conflict.Status) {
	case "resolved", "ignored":
		return "No further conflict action is required. Continue syncing and verify the workspace state."
	}

	switch optString(conflict.Reason) {
	case "concurrent_update", "case_only_rename_conflict":
		if optString(conflict.ArtifactPath) != "" {
			return fmt.Sprintf("Inspect %s, repair the canonical path locally, then run `s0 sync conflicts mark %s`.", optString(conflict.ArtifactPath), path)
		}
		return fmt.Sprintf("Inspect the current workspace path, repair the canonical state locally, then run `s0 sync conflicts mark %s`.", path)
	case "casefold_collision":
		return fmt.Sprintf("Choose one canonical path spelling, remove or rename the colliding path locally, then run `s0 sync conflicts mark %s`.", path)
	case "windows_reserved_name", "windows_trailing_dot_space", "windows_forbidden_character", "windows_control_character":
		return fmt.Sprintf("Rename or remove the incompatible path locally, then run `s0 sync conflicts mark %s`.", path)
	default:
		if metadata.Resolution != "" {
			return "Conflict metadata already contains a recorded resolution state."
		}
		return fmt.Sprintf("Repair the path locally, then run `s0 sync conflicts mark %s`.", path)
	}
}

func issueMessage(metadata conflictMetadata) string {
	for _, issue := range metadata.Issues {
		if strings.TrimSpace(issue.Message) != "" {
			return strings.TrimSpace(issue.Message)
		}
	}
	return ""
}

func conflictPrimaryPath(conflict apispec.SyncConflict) string {
	for _, candidate := range []string{
		optString(conflict.Path),
		optNilString(conflict.IncomingPath),
		optNilString(conflict.IncomingOldPath),
	} {
		if strings.TrimSpace(candidate) != "" {
			return strings.TrimSpace(candidate)
		}
	}
	return "-"
}

func decodeConflictMetadata(conflict apispec.SyncConflict) conflictMetadata {
	value, ok := conflict.Metadata.Get()
	if !ok || value == nil {
		return conflictMetadata{}
	}
	raw := make(map[string]json.RawMessage, len(value))
	for key, payload := range value {
		if len(payload) == 0 {
			continue
		}
		raw[key] = json.RawMessage(payload)
	}
	if len(raw) == 0 {
		return conflictMetadata{}
	}

	var decoded conflictMetadata
	if err := json.Unmarshal(mustMarshal(raw), &decoded); err != nil {
		return conflictMetadata{}
	}
	return decoded
}

func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return data
}

func humanizeReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "unknown sync conflict"
	}
	reason = strings.ReplaceAll(reason, "_", " ")
	return strings.ToUpper(reason[:1]) + reason[1:]
}

func optString(value apispec.OptString) string {
	v, ok := value.Get()
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

func optNilString(value apispec.OptNilString) string {
	v, ok := value.Get()
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

func optNilInt64Ptr(value apispec.OptNilInt64) *int64 {
	v, ok := value.Get()
	if !ok {
		return nil
	}
	out := v
	return &out
}

func optDateTimePtr(value apispec.OptDateTime) *time.Time {
	v, ok := value.Get()
	if !ok {
		return nil
	}
	return &v
}
