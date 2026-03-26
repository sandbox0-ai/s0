package syncworker

import (
	"fmt"
	"path"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/sandbox0-ai/s0/internal/syncstate"
	"golang.org/x/text/unicode/norm"
)

const (
	issueCodeCasefoldCollision       = "casefold_collision"
	issueCodeWindowsReservedName     = "windows_reserved_name"
	issueCodeWindowsTrailingDotSpace = "windows_trailing_dot_space"
	issueCodeWindowsForbiddenRune    = "windows_forbidden_character"
	issueCodeWindowsControlCharacter = "windows_control_character"
)

type pathCompatibilityIssue struct {
	Code           string
	Path           string
	NormalizedPath string
	Paths          []string
	Segment        string
	Message        string
}

type LocalPathCompatibilityError struct {
	Scope  string
	Issues []pathCompatibilityIssue
}

func (e *LocalPathCompatibilityError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "workspace path compatibility error"
	}
	scope := strings.TrimSpace(e.Scope)
	if scope == "" {
		scope = "workspace"
	}
	first := e.Issues[0]
	switch {
	case len(first.Paths) > 1:
		return fmt.Sprintf("%s contains paths incompatible with replica filesystem capabilities: %s (%s)", scope, first.Code, strings.Join(first.Paths, ", "))
	case first.Path != "":
		return fmt.Sprintf("%s contains paths incompatible with replica filesystem capabilities: %s at %s", scope, first.Code, first.Path)
	default:
		return fmt.Sprintf("%s contains paths incompatible with replica filesystem capabilities: %s", scope, first.Code)
	}
}

type compatibilityTracker struct {
	caps     syncstate.FilesystemCaps
	pathKeys map[string]string
}

func newCompatibilityTracker(manifest *syncstate.Manifest, caps syncstate.FilesystemCaps) *compatibilityTracker {
	if !requiresPathCompatibilityAudit(caps) {
		return nil
	}
	tracker := &compatibilityTracker{
		caps:     caps,
		pathKeys: map[string]string{},
	}
	if manifest == nil {
		return tracker
	}
	for _, entry := range manifest.Entries {
		tracker.pathKeys[entry.Path] = compatibilityPathKey(entry.Path, caps)
	}
	return tracker
}

func auditWorkspaceCompatibility(manifest *syncstate.Manifest, caps syncstate.FilesystemCaps) error {
	if !requiresPathCompatibilityAudit(caps) || manifest == nil {
		return nil
	}
	issues := make([]pathCompatibilityIssue, 0)
	byKey := map[string][]string{}
	paths := make([]string, 0, len(manifest.Entries))
	for relativePath := range manifest.Entries {
		paths = append(paths, relativePath)
	}
	slices.Sort(paths)
	for _, relativePath := range paths {
		issues = append(issues, validateCompatibilityPath(relativePath, caps)...)
		key := compatibilityPathKey(relativePath, caps)
		if key != "" {
			byKey[key] = append(byKey[key], relativePath)
		}
	}
	issues = append(issues, buildCasefoldCollisionIssues(byKey)...)
	issues = deduplicateCompatibilityIssues(issues)
	if len(issues) == 0 {
		return nil
	}
	return &LocalPathCompatibilityError{
		Scope:  "workspace",
		Issues: issues,
	}
}

func (t *compatibilityTracker) ValidateRemoteChange(changePath, oldPath, eventType string) error {
	if t == nil {
		return nil
	}
	issues := make([]pathCompatibilityIssue, 0)
	if changePath != "" {
		issues = append(issues, validateCompatibilityPath(changePath, t.caps)...)
	}
	if oldPath != "" {
		issues = append(issues, validateCompatibilityPath(oldPath, t.caps)...)
	}
	if len(issues) == 0 && compatibilityChangeCreatesPath(eventType) {
		key := compatibilityPathKey(changePath, t.caps)
		if key != "" {
			for existingPath, existingKey := range t.pathKeys {
				if existingKey != key {
					continue
				}
				if existingPath == changePath || (eventType == "rename" && existingPath == oldPath) {
					continue
				}
				issues = append(issues, pathCompatibilityIssue{
					Code:           issueCodeCasefoldCollision,
					NormalizedPath: key,
					Paths:          []string{existingPath, changePath},
					Message:        "logical paths collide under the replica filesystem capabilities",
				})
				break
			}
		}
	}
	issues = deduplicateCompatibilityIssues(issues)
	if len(issues) == 0 {
		return nil
	}
	return &LocalPathCompatibilityError{
		Scope:  "remote change",
		Issues: issues,
	}
}

func (t *compatibilityTracker) ApplyRemoteChange(changePath, oldPath, eventType string) {
	if t == nil {
		return
	}
	switch strings.TrimSpace(eventType) {
	case "create", "write", "chmod":
		if changePath == "" {
			return
		}
		t.pathKeys[changePath] = compatibilityPathKey(changePath, t.caps)
	case "remove":
		delete(t.pathKeys, changePath)
	case "rename":
		delete(t.pathKeys, oldPath)
		if changePath != "" {
			t.pathKeys[changePath] = compatibilityPathKey(changePath, t.caps)
		}
	}
}

func requiresPathCompatibilityAudit(caps syncstate.FilesystemCaps) bool {
	return !caps.CaseSensitive || caps.UnicodeNormalizationInsensitive || caps.WindowsCompatiblePaths
}

func compatibilityChangeCreatesPath(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case "create", "write", "rename":
		return true
	default:
		return false
	}
}

func compatibilityPathKey(raw string, caps syncstate.FilesystemCaps) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	cleaned := path.Clean("/" + strings.TrimSpace(raw))
	if cleaned == "." {
		return "/"
	}
	parts := strings.Split(cleaned, "/")
	for i, part := range parts {
		if i == 0 || part == "" {
			continue
		}
		if caps.UnicodeNormalizationInsensitive {
			part = norm.NFD.String(part)
		}
		if !caps.CaseSensitive {
			part = strings.ToLower(part)
		}
		parts[i] = part
	}
	return strings.Join(parts, "/")
}

func validateCompatibilityPath(raw string, caps syncstate.FilesystemCaps) []pathCompatibilityIssue {
	cleaned := path.Clean("/" + strings.TrimSpace(raw))
	if cleaned == "." || cleaned == "/" {
		return nil
	}
	if !caps.WindowsCompatiblePaths {
		return nil
	}

	issues := make([]pathCompatibilityIssue, 0)
	for _, segment := range strings.Split(strings.TrimPrefix(cleaned, "/"), "/") {
		if segment == "" {
			continue
		}
		if hasControlCharacter(segment) {
			issues = append(issues, pathCompatibilityIssue{
				Code:    issueCodeWindowsControlCharacter,
				Path:    strings.TrimPrefix(cleaned, "/"),
				Segment: segment,
				Message: "path segment contains a Windows control character",
			})
		}
		if strings.ContainsAny(segment, "<>:\"\\|?*") {
			issues = append(issues, pathCompatibilityIssue{
				Code:    issueCodeWindowsForbiddenRune,
				Path:    strings.TrimPrefix(cleaned, "/"),
				Segment: segment,
				Message: "path segment contains a Windows-forbidden character",
			})
		}
		if strings.TrimRight(segment, ". ") != segment {
			issues = append(issues, pathCompatibilityIssue{
				Code:    issueCodeWindowsTrailingDotSpace,
				Path:    strings.TrimPrefix(cleaned, "/"),
				Segment: segment,
				Message: "path segment ends with a dot or space and is not portable to Windows",
			})
		}
		if isWindowsReservedName(segment) {
			issues = append(issues, pathCompatibilityIssue{
				Code:    issueCodeWindowsReservedName,
				Path:    strings.TrimPrefix(cleaned, "/"),
				Segment: segment,
				Message: "path segment uses a Windows reserved device name",
			})
		}
	}
	return issues
}

func buildCasefoldCollisionIssues(byKey map[string][]string) []pathCompatibilityIssue {
	issues := make([]pathCompatibilityIssue, 0)
	for key, paths := range byKey {
		if len(paths) <= 1 {
			continue
		}
		clone := slices.Clone(paths)
		slices.Sort(clone)
		issues = append(issues, pathCompatibilityIssue{
			Code:           issueCodeCasefoldCollision,
			NormalizedPath: key,
			Paths:          clone,
			Message:        "logical paths collide under the replica filesystem capabilities",
		})
	}
	return issues
}

func deduplicateCompatibilityIssues(issues []pathCompatibilityIssue) []pathCompatibilityIssue {
	if len(issues) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(issues))
	out := make([]pathCompatibilityIssue, 0, len(issues))
	for _, issue := range issues {
		key := issue.Code + "\x00" + issue.Path + "\x00" + issue.NormalizedPath + "\x00" + issue.Segment + "\x00" + strings.Join(issue.Paths, "\x00")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, issue)
	}
	return out
}

func hasControlCharacter(v string) bool {
	for len(v) > 0 {
		r, size := utf8.DecodeRuneInString(v)
		if r < 0x20 {
			return true
		}
		v = v[size:]
	}
	return false
}

func isWindowsReservedName(segment string) bool {
	base := segment
	if idx := strings.IndexRune(base, '.'); idx >= 0 {
		base = base[:idx]
	}
	switch strings.ToUpper(base) {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return true
	default:
		return false
	}
}
