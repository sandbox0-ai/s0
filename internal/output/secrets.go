package output

import (
	"strings"

	"github.com/sandbox0-ai/s0/internal/client"
)

const (
	visibleSecretPrefix = 3
	visibleSecretSuffix = 3
	minSecretMaskLength = 8
	fixedSecretMaskBody = "********"
)

func redactSensitiveData(data any, showSecrets bool) any {
	if showSecrets {
		return data
	}

	creds, ok := data.(*client.RegistryCredentials)
	if !ok || creds == nil {
		return data
	}

	redacted := *creds
	redacted.Password = maskSecret(redacted.Password)
	return &redacted
}

func maskSecret(raw string) string {
	if raw == "" {
		return ""
	}
	if len(raw) <= minSecretMaskLength {
		return strings.Repeat("*", len(raw))
	}

	prefix := raw[:visibleSecretPrefix]
	suffix := raw[len(raw)-visibleSecretSuffix:]
	return prefix + fixedSecretMaskBody + suffix
}
