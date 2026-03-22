package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestTableFormatterFormatCredentialSourceList(t *testing.T) {
	formatter := &TableFormatter{}
	now := time.Date(2026, 3, 23, 12, 34, 56, 0, time.UTC)
	sources := []apispec.CredentialSourceMetadata{
		{
			Name:           "gh-token",
			ResolverKind:   apispec.CredentialSourceResolverKindStaticHeaders,
			CurrentVersion: apispec.NewOptInt64(3),
			Status:         apispec.NewOptString("ready"),
			CreatedAt:      apispec.NewOptNilDateTime(now),
			UpdatedAt:      apispec.NewOptNilDateTime(now.Add(2 * time.Hour)),
		},
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, sources); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"NAME",
		"RESOLVER KIND",
		"gh-token",
		"static_headers",
		"ready",
		"2026-03-23 12:34:56",
		"2026-03-23 14:34:56",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "{3 true}") {
		t.Fatalf("output still contains raw OptInt64 formatting:\n%s", output)
	}
}

func TestTableFormatterFormatCredentialSource(t *testing.T) {
	formatter := &TableFormatter{}
	source := &apispec.CredentialSourceMetadata{
		Name:         "db-auth",
		ResolverKind: apispec.CredentialSourceResolverKindStaticUsernamePassword,
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, source); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"Name:",
		"db-auth",
		"Resolver Kind:",
		"static_username_password",
		"Current Version:",
		"Status:",
		"Created At:",
		"Updated At:",
		"-",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestTableFormatterFormatCredentialSourceListEmpty(t *testing.T) {
	formatter := &TableFormatter{}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, []apispec.CredentialSourceMetadata{}); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	if got := strings.TrimSpace(buf.String()); got != "No credential sources found." {
		t.Fatalf("empty output = %q, want %q", got, "No credential sources found.")
	}
}
