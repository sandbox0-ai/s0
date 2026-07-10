package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
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

func TestTableFormatterFormatVolumeShowsBackendAndS3Metadata(t *testing.T) {
	formatter := &TableFormatter{}
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	volume := &apispec.SandboxVolume{
		ID:        "vol_s3",
		TeamID:    "team-1",
		UserID:    "user-1",
		Backend:   apispec.VolumeBackendS3,
		CreatedAt: now,
		UpdatedAt: now,
		S3: apispec.NewOptSandboxVolumeS3Config(apispec.SandboxVolumeS3Config{
			Provider:    apispec.SandboxVolumeS3ConfigProviderR2,
			Bucket:      "agent-state",
			Prefix:      apispec.NewOptString("team-a/"),
			Region:      apispec.NewOptString("auto"),
			EndpointURL: apispec.NewOptString("https://account.r2.cloudflarestorage.com"),
		}),
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, volume); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"Backend:",
		"s3",
		"S3 Provider:",
		"r2",
		"S3 Bucket:",
		"agent-state",
		"S3 Prefix:",
		"team-a/",
		"S3 Region:",
		"auto",
		"S3 Endpoint URL:",
		"https://account.r2.cloudflarestorage.com",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestNewTeamListMarksCurrentTeam(t *testing.T) {
	teams := []apispec.Team{
		{ID: "team-1", Name: "Team One"},
		{ID: "team-2", Name: "Team Two"},
	}

	list := NewTeamList(teams, " team-2 ")

	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}
	if list[0].Current {
		t.Fatal("team-1 should not be current")
	}
	if !list[1].Current {
		t.Fatal("team-2 should be current")
	}
}

func TestTableFormatterFormatTeamListShowsCurrentMarker(t *testing.T) {
	formatter := &TableFormatter{}
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	teams := NewTeamList([]apispec.Team{
		{
			ID:        "team-1",
			Name:      "Team One",
			Slug:      "team-one",
			CreatedAt: now,
		},
		{
			ID:        "team-2",
			Name:      "Team Two",
			Slug:      "team-two",
			CreatedAt: now,
		},
	}, "team-2")

	var buf bytes.Buffer
	if err := formatter.Format(&buf, teams); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"CURRENT",
		"ID",
		"team-1",
		"team-2",
		"*",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "team-1") && strings.Contains(line, "*") {
			t.Fatalf("non-current team row contains current marker:\n%s", output)
		}
		if strings.Contains(line, "team-2") && !strings.Contains(line, "*") {
			t.Fatalf("current team row missing current marker:\n%s", output)
		}
	}
}

func TestJSONFormatterTeamListIncludesCurrentField(t *testing.T) {
	formatter := &JSONFormatter{}
	teams := NewTeamList([]apispec.Team{
		{ID: "team-1", Name: "Team One", Slug: "team-one"},
		{ID: "team-2", Name: "Team Two", Slug: "team-two"},
	}, "team-1")

	var buf bytes.Buffer
	if err := formatter.Format(&buf, teams); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	var got []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0]["id"] != "team-1" || got[0]["current"] != true {
		t.Fatalf("first team = %#v, want current team-1", got[0])
	}
	if got[1]["id"] != "team-2" || got[1]["current"] != false {
		t.Fatalf("second team = %#v, want non-current team-2", got[1])
	}
}

func TestTableFormatterFormatTeamMemberListIncludesProfileFields(t *testing.T) {
	formatter := &TableFormatter{}
	members := []apispec.TeamMember{
		{
			ID:        "tm_123",
			TeamID:    "team_123",
			UserID:    "user_123",
			Email:     apispec.NewOptString("dev@example.com"),
			Name:      apispec.NewOptString("Dev User"),
			AvatarURL: apispec.NewOptString("https://example.com/avatar.png"),
			Role:      "developer",
			JoinedAt:  time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC),
		},
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, members); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"EMAIL",
		"NAME",
		"dev@example.com",
		"Dev User",
		"developer",
		"2026-06-13 12:00:00",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestTableFormatterFormatTeamMemberIncludesProfileFields(t *testing.T) {
	formatter := &TableFormatter{}
	member := &apispec.TeamMember{
		ID:        "tm_123",
		TeamID:    "team_123",
		UserID:    "user_123",
		Email:     apispec.NewOptString("dev@example.com"),
		Name:      apispec.NewOptString("Dev User"),
		AvatarURL: apispec.NewOptString("https://example.com/avatar.png"),
		Role:      "developer",
		JoinedAt:  time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC),
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, member); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"Email:",
		"dev@example.com",
		"Name:",
		"Dev User",
		"Avatar URL:",
		"https://example.com/avatar.png",
		"Role:",
		"developer",
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
		AutoResume: true,
		Resources: apispec.NewOptSandboxResourceConfig(apispec.SandboxResourceConfig{
			Memory: apispec.NewOptString("2Gi"),
		}),
		PodName: "sb-123-pod",
		SSH: apispec.NewOptSandboxSSHConnection(apispec.SandboxSSHConnection{
			Host:     "aws-us-east-1.ssh.sandbox0.app",
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
		"aws-us-east-1.ssh.sandbox0.app",
		"SSH Port:",
		"30222",
		"SSH Username:",
		"sb_123",
		"Memory:",
		"2Gi",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestTableFormatterFormatSandboxRootFSSnapshotList(t *testing.T) {
	formatter := &TableFormatter{}
	snapshots := &apispec.SandboxRootFSSnapshotList{
		Snapshots: []apispec.SandboxRootFSSnapshot{
			{
				ID:        "snap_123",
				SandboxID: "sb_123",
				Name:      apispec.NewOptString("checkpoint"),
				CreatedAt: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
				ExpiresAt: apispec.NewOptDateTime(time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)),
			},
		},
		Count: 1,
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, snapshots); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"ID",
		"SANDBOX ID",
		"snap_123",
		"sb_123",
		"checkpoint",
		"2026-04-10 12:00:00",
		"2026-04-11 12:00:00",
		"Total: 1",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestTableFormatterFormatSandboxRootFSSnapshotActions(t *testing.T) {
	formatter := &TableFormatter{}

	restore := &apispec.RestoreSandboxRootFSResponse{
		SandboxID:  "sb_123",
		SnapshotID: "snap_123",
		Status:     apispec.SandboxLifecycleStatusPaused,
	}
	var restoreBuf bytes.Buffer
	if err := formatter.Format(&restoreBuf, restore); err != nil {
		t.Fatalf("Format() restore error = %v", err)
	}
	restoreOutput := restoreBuf.String()
	for _, want := range []string{"Sandbox ID:", "sb_123", "Snapshot ID:", "snap_123", "paused"} {
		if !strings.Contains(restoreOutput, want) {
			t.Fatalf("restore output missing %q:\n%s", want, restoreOutput)
		}
	}

	fork := &apispec.ForkSandboxResponse{
		SourceSandboxID: "sb_123",
		Sandbox: apispec.Sandbox{
			ID:         "sb_456",
			TemplateID: "default",
			Status:     apispec.SandboxLifecycleStatusPaused,
			Paused:     true,
		},
	}
	var forkBuf bytes.Buffer
	if err := formatter.Format(&forkBuf, fork); err != nil {
		t.Fatalf("Format() fork error = %v", err)
	}
	forkOutput := forkBuf.String()
	for _, want := range []string{"Source Sandbox ID:", "sb_123", "Fork Sandbox ID:", "sb_456", "default", "paused", "true"} {
		if !strings.Contains(forkOutput, want) {
			t.Fatalf("fork output missing %q:\n%s", want, forkOutput)
		}
	}
}

func TestTableFormatterFormatSandboxServicesIncludesPublicURL(t *testing.T) {
	formatter := &TableFormatter{}
	services := &sandbox0.SandboxServicesResponse{
		SandboxID: "rs-default-api-abcde",
		Services: []apispec.SandboxAppServiceView{
			{
				ID:   "api",
				Port: apispec.NewOptInt32(8080),
				Ingress: apispec.SandboxAppServiceIngress{
					Public: true,
					Routes: []apispec.SandboxAppServiceRoute{
						{
							ID:         "api",
							PathPrefix: apispec.NewOptString("/"),
							Resume:     true,
						},
					},
				},
				Publishable: true,
				PublicURL:   apispec.NewOptString("https://rs-default-api-abcde--p8080.us.sandbox0.app"),
			},
		},
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, services); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"URL",
		"https://rs-default-api-abcde--p8080.us.sandbox0.app",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestTableFormatterFormatExecutionSessions(t *testing.T) {
	formatter := &TableFormatter{}
	now := time.Date(2026, 7, 11, 4, 0, 0, 0, time.UTC)
	sessions := []apispec.ExecutionSession{{
		ID: "ses_123",
		Spec: apispec.ExecutionSessionSpec{
			Name:    apispec.NewOptString("worker"),
			Command: []string{"/bin/sleep", "30"},
		},
		Phase:             apispec.ExecutionSessionPhaseRunning,
		SpecVersion:       1,
		RuntimeGeneration: 2,
		Attempt: apispec.NewOptExecutionSessionAttempt(apispec.ExecutionSessionAttempt{
			ID: "att_1", Number: 1, RuntimeGeneration: 2, Pid: apispec.NewOptInt32(42),
		}),
		Cursor:         apispec.ExecutionSessionEventCursor{Earliest: 1, Latest: 7},
		CreatedAt:      now,
		UpdatedAt:      now,
		LastActivityAt: now,
	}}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, sessions); err != nil {
		t.Fatal(err)
	}
	output := buf.String()
	for _, want := range []string{"ses_123", "worker", "running", "att_1", "42"} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestTableFormatterFormatSSHPublicKeyList(t *testing.T) {
	formatter := &TableFormatter{}
	keys := []apispec.SSHPublicKey{
		{
			ID:                "key_123",
			Name:              "macbook",
			PublicKey:         "ssh-ed25519 AAAA test@example",
			KeyType:           "ssh-ed25519",
			FingerprintSHA256: "SHA256:abc",
			CreatedAt:         time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
			UpdatedAt:         time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		},
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, keys); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"ID",
		"NAME",
		"KEY TYPE",
		"FINGERPRINT",
		"macbook",
		"ssh-ed25519",
		"SHA256:abc",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestTableFormatterFormatSSHPublicKey(t *testing.T) {
	formatter := &TableFormatter{}
	key := &apispec.SSHPublicKey{
		ID:                "key_123",
		Name:              "macbook",
		PublicKey:         "ssh-ed25519 AAAA test@example",
		KeyType:           "ssh-ed25519",
		FingerprintSHA256: "SHA256:abc",
		Comment:           apispec.NewOptString("test@example"),
		CreatedAt:         time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
		UpdatedAt:         time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, key); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"Name:",
		"macbook",
		"Fingerprint:",
		"SHA256:abc",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
	for _, unwanted := range []string{
		"Public Key:",
		"ssh-ed25519 AAAA test@example",
	} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("output contains %q:\n%s", unwanted, output)
		}
	}
}
