package output

import (
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// FormatSandboxAuditActor returns a compact actor kind and stable identity label.
func FormatSandboxAuditActor(actor apispec.SandboxAuditActor) string {
	identity := ""
	for _, candidate := range []apispec.OptString{actor.ID, actor.UserID, actor.APIKeyID} {
		if value, ok := candidate.Get(); ok && value != "" {
			identity = value
			break
		}
	}
	if identity == "" {
		return string(actor.Kind)
	}
	return string(actor.Kind) + ":" + identity
}

// FormatSandboxAuditResource returns a compact resource type, identity, and subresource label.
func FormatSandboxAuditResource(resource apispec.SandboxAuditResource) string {
	parts := []string{resource.Type, resource.ID}
	if subresource, ok := resource.Subresource.Get(); ok && subresource != "" {
		parts = append(parts, subresource)
	}
	return strings.Join(parts, ":")
}

// FormatSandboxAuditIntegrity returns the verification status and event-ID conflict state.
func FormatSandboxAuditIntegrity(integrity apispec.SandboxAuditIntegrity) string {
	status := string(integrity.SignatureStatus)
	if conflict, ok := integrity.EventIDConflict.Get(); ok && conflict {
		return status + "/conflict"
	}
	return status
}
