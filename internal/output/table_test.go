package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/sandbox0-ai/s0/internal/syncstate"
	"github.com/sandbox0-ai/s0/internal/syncview"
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

func TestTableFormatterFormatRegionList(t *testing.T) {
	formatter := &TableFormatter{}
	regions := []apispec.Region{
		{
			ID:                 "aws/us-east-1",
			DisplayName:        apispec.NewOptString("US East 1"),
			RegionalGatewayURL: "https://use1.example.com",
			MeteringExportURL:  apispec.NewOptNilString("https://metering.use1.example.com"),
			Enabled:            true,
		},
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, regions); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"ID",
		"DISPLAY NAME",
		"REGIONAL GATEWAY URL",
		"METERING EXPORT URL",
		"ENABLED",
		"aws/us-east-1",
		"US East 1",
		"https://use1.example.com",
		"https://metering.use1.example.com",
		"true",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestTableFormatterFormatRegion(t *testing.T) {
	formatter := &TableFormatter{}
	region := &apispec.Region{
		ID:                 "aws/us-east-1",
		DisplayName:        apispec.NewOptString("US East 1"),
		RegionalGatewayURL: "https://use1.example.com",
		Enabled:            false,
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, region); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"ID:",
		"Display Name:",
		"Regional Gateway URL:",
		"Metering Export URL:",
		"Enabled:",
		"aws/us-east-1",
		"US East 1",
		"https://use1.example.com",
		"false",
		"-",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestTableFormatterFormatSandboxIncludesSSHConnection(t *testing.T) {
	formatter := &TableFormatter{}
	sandbox := &apispec.Sandbox{
		ID:         "sb_123",
		TemplateID: "default",
		TeamID:     "team_123",
		Status:     "running",
		Paused:     false,
		PowerState: apispec.SandboxPowerState{},
		AutoResume: true,
		PodName:    "sb-123-pod",
		SSH: apispec.NewOptSandboxSSHConnection(apispec.SandboxSSHConnection{
			Host:     "ssh.aws-us-east-1.sandbox0.app",
			Port:     30222,
			Username: "sb_123",
		}),
		ClaimedAt:     time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		ExpiresAt:     time.Date(2026, 4, 10, 13, 0, 0, 0, time.UTC),
		HardExpiresAt: time.Date(2026, 4, 10, 14, 0, 0, 0, time.UTC),
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, sandbox); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"SSH Host:",
		"ssh.aws-us-east-1.sandbox0.app",
		"SSH Port:",
		"30222",
		"SSH Username:",
		"sb_123",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestTableFormatterFormatSyncStatusView(t *testing.T) {
	formatter := &TableFormatter{}
	view := &syncview.StatusView{
		Attachment: &syncstate.Attachment{
			WorkspaceRoot: "/tmp/work",
			VolumeID:      "vol-123",
			ReplicaID:     "replica-mac",
			InitFrom:      "auto",
			Worker: syncstate.WorkerState{
				Status: "running",
				Mode:   "background",
			},
			Ignore: syncstate.IgnoreConfig{
				BuiltinPatterns: []string{".git/"},
			},
			CreatedAt: time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 3, 26, 12, 1, 0, 0, time.UTC),
			LastSync: &syncstate.SyncCheckpoint{
				HeadSeq:           10,
				LastAppliedSeq:    8,
				OpenConflictCount: 2,
			},
		},
		ConflictSummary: &syncview.ConflictListView{
			OpenCount: 2,
			UnmergedPaths: []syncview.ConflictListEntry{
				{Path: "src/main.go", Summary: `modified locally, conflicted with sandbox "sandbox-1"`},
				{Path: "docs/CON.txt", Summary: "namespace incompatible for Windows-capable replicas"},
			},
		},
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, view); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"Workspace:",
		"/tmp/work",
		"Open Conflicts: 2",
		"Unmerged sync paths:",
		`modified locally, conflicted with sandbox "sandbox-1": src/main.go`,
		"namespace incompatible for Windows-capable replicas: docs/CON.txt",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestTableFormatterFormatConflictDetailView(t *testing.T) {
	formatter := &TableFormatter{}
	now := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	view := &syncview.ConflictDetailView{
		Path:              "src/main.go",
		Summary:           `modified locally, conflicted with sandbox "sandbox-1"`,
		ReasonCode:        "concurrent_update",
		Status:            "open",
		RecordedFor:       `replica "replica-mac"`,
		ArtifactPath:      "src/main.sandbox0-conflict-replica-mac-seq-42.go",
		LatestRemoteActor: `sandbox "sandbox-1"`,
		LatestRemoteEvent: "write",
		SuggestedNextStep: "Inspect the artifact and repair the canonical path locally.",
		CreatedAt:         &now,
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, view); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"Path:",
		"Summary:",
		"Reason Code:",
		"Recorded For:",
		"Latest Remote Actor:",
		"Latest Remote Event:",
		"Suggested Next Step:",
		"src/main.go",
		"concurrent_update",
		`replica "replica-mac"`,
		`sandbox "sandbox-1"`,
		"write",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}
