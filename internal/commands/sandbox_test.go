package commands

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

func TestFormatSandboxCreateErrorClaimStartThrottled(t *testing.T) {
	err := &sandbox0.APIError{
		StatusCode:        429,
		Code:              sandbox0.CodeClaimStartThrottled,
		Message:           "claim start admission throttled",
		RetryAfterSeconds: 2,
	}

	got := formatSandboxCreateError(err)
	for _, want := range []string{
		"claim_start_throttled",
		"Hint: sandbox claim/start capacity is temporarily throttled.",
		"Retry after 2 seconds.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatSandboxCreateError() = %q, want substring %q", got, want)
		}
	}
}

func TestFormatSandboxCreateErrorNonThrottled(t *testing.T) {
	err := errors.New("request failed")

	got := formatSandboxCreateError(err)
	if got != "request failed" {
		t.Fatalf("formatSandboxCreateError() = %q, want original error", got)
	}
}

func TestBuildSandboxCreateConfig(t *testing.T) {
	resetSandboxFlagsForTest()

	t.Run("reads legacy sandbox config file", func(t *testing.T) {
		resetSandboxFlagsForTest()
		sandboxTemplate = "default"
		sandboxConfigFile = writeTempFile(t, `
network:
  mode: block-all
  egress:
    trafficRules:
      - name: allow-github-api
        action: allow
        domains: [api.github.com]
        ports:
          - port: 443
            protocol: tcp
  credentialBindings:
    - ref: gh-token
      sourceRef: github-source
      projection:
        type: http_headers
        httpHeaders:
          headers:
            - name: Authorization
              valueTemplate: "Bearer {{ .token }}"
`)

		request, err := buildSandboxCreateRequest()
		if err != nil {
			t.Fatalf("buildSandboxCreateRequest() error = %v", err)
		}
		template, ok := request.Template.Get()
		if !ok || template != "default" {
			t.Fatalf("template = %q, want default", template)
		}
		config, ok := request.Config.Get()
		if !ok {
			t.Fatal("config not set")
		}
		network, ok := config.Network.Get()
		if !ok {
			t.Fatal("network not set")
		}
		if network.Mode != apispec.SandboxNetworkPolicyModeBlockAll {
			t.Fatalf("mode = %q, want %q", network.Mode, apispec.SandboxNetworkPolicyModeBlockAll)
		}
		if len(network.CredentialBindings) != 1 {
			t.Fatalf("credentialBindings count = %d, want 1", len(network.CredentialBindings))
		}
	})

	t.Run("flags override file values", func(t *testing.T) {
		resetSandboxFlagsForTest()
		sandboxTemplate = "default"
		sandboxConfigFile = writeTempFile(t, `
ttl: 60
hard_ttl: 120
`)
		sandboxTTL = 300
		sandboxMemory = "512Mi"

		request, err := buildSandboxCreateRequest()
		if err != nil {
			t.Fatalf("buildSandboxCreateRequest() error = %v", err)
		}
		config, ok := request.Config.Get()
		if !ok {
			t.Fatal("config not set")
		}
		ttl, ok := config.TTL.Get()
		if !ok || ttl != 300 {
			t.Fatalf("ttl = %d, want 300", ttl)
		}
		hardTTL, ok := config.HardTTL.Get()
		if !ok || hardTTL != 120 {
			t.Fatalf("hardTTL = %d, want 120", hardTTL)
		}
		resources, ok := config.Resources.Get()
		if !ok {
			t.Fatal("resources not set")
		}
		memory, ok := resources.Memory.Get()
		if !ok || memory != "512Mi" {
			t.Fatalf("memory = %q, want 512Mi", memory)
		}
	})

	t.Run("reads full claim request file", func(t *testing.T) {
		resetSandboxFlagsForTest()
		sandboxConfigFile = writeTempFile(t, `
template: from-file
snapshot_id: snap_file
mounts:
  - sandboxvolume_id: vol_123
    mount_point: /workspace/data
config:
  ttl: 90
`)

		request, err := buildSandboxCreateRequest()
		if err != nil {
			t.Fatalf("buildSandboxCreateRequest() error = %v", err)
		}
		template, ok := request.Template.Get()
		if !ok || template != "from-file" {
			t.Fatalf("template = %q, want from-file", template)
		}
		snapshotID, ok := request.SnapshotID.Get()
		if !ok || snapshotID != "snap_file" {
			t.Fatalf("snapshot_id = %q, want snap_file", snapshotID)
		}
		if len(request.Mounts) != 1 {
			t.Fatalf("mount count = %d, want 1", len(request.Mounts))
		}
		if request.Mounts[0].SandboxvolumeID != "vol_123" || request.Mounts[0].MountPoint != "/workspace/data" {
			t.Fatalf("unexpected mount = %+v", request.Mounts[0])
		}
	})

	t.Run("mount flags extend request", func(t *testing.T) {
		resetSandboxFlagsForTest()
		sandboxTemplate = "default"
		sandboxMounts = []string{"vol_abc:/workspace/bootstrap-data"}

		request, err := buildSandboxCreateRequest()
		if err != nil {
			t.Fatalf("buildSandboxCreateRequest() error = %v", err)
		}
		if len(request.Mounts) != 1 {
			t.Fatalf("mount count = %d, want 1", len(request.Mounts))
		}
	})

	t.Run("mount flags append to request file mounts and template flag overrides", func(t *testing.T) {
		resetSandboxFlagsForTest()
		sandboxTemplate = "flag-template"
		sandboxSnapshotID = "snap_flag"
		sandboxConfigFile = writeTempFile(t, `
template: from-file
snapshot_id: snap_file
mounts:
  - sandboxvolume_id: vol_file
    mount_point: /workspace/from-file
`)
		sandboxMounts = []string{"vol_flag:/workspace/from-flag"}

		request, err := buildSandboxCreateRequest()
		if err != nil {
			t.Fatalf("buildSandboxCreateRequest() error = %v", err)
		}
		template, ok := request.Template.Get()
		if !ok || template != "flag-template" {
			t.Fatalf("template = %q, want flag-template", template)
		}
		snapshotID, ok := request.SnapshotID.Get()
		if !ok || snapshotID != "snap_flag" {
			t.Fatalf("snapshot_id = %q, want snap_flag", snapshotID)
		}
		if len(request.Mounts) != 2 {
			t.Fatalf("mount count = %d, want 2", len(request.Mounts))
		}
		if request.Mounts[0].SandboxvolumeID != "vol_file" || request.Mounts[1].SandboxvolumeID != "vol_flag" {
			t.Fatalf("unexpected mounts = %+v", request.Mounts)
		}
	})

	t.Run("invalid mount flag fails", func(t *testing.T) {
		resetSandboxFlagsForTest()
		sandboxTemplate = "default"
		sandboxMounts = []string{"missing-separator"}

		_, err := buildSandboxCreateRequest()
		if err == nil {
			t.Fatal("buildSandboxCreateRequest() error = nil, want error")
		}
	})

	t.Run("relative mount path fails", func(t *testing.T) {
		resetSandboxFlagsForTest()
		sandboxTemplate = "default"
		sandboxMounts = []string{"vol_abc:workspace/relative"}

		_, err := buildSandboxCreateRequest()
		if err == nil {
			t.Fatal("buildSandboxCreateRequest() error = nil, want error")
		}
	})
}

func TestBuildSandboxUpdateConfig(t *testing.T) {
	resetSandboxFlagsForTest()

	t.Run("reads update config file", func(t *testing.T) {
		resetSandboxFlagsForTest()
		sandboxUpdateConfigFile = writeTempFile(t, `
auto_resume: true
network:
  mode: allow-all
`)

		config, hasConfig, err := buildSandboxUpdateConfig()
		if err != nil {
			t.Fatalf("buildSandboxUpdateConfig() error = %v", err)
		}
		if !hasConfig {
			t.Fatal("hasConfig = false, want true")
		}
		autoResume, ok := config.AutoResume.Get()
		if !ok || !autoResume {
			t.Fatalf("autoResume = %v, want true", autoResume)
		}
		network, ok := config.Network.Get()
		if !ok {
			t.Fatal("network not set")
		}
		if network.Mode != apispec.SandboxNetworkPolicyModeAllowAll {
			t.Fatalf("mode = %q, want %q", network.Mode, apispec.SandboxNetworkPolicyModeAllowAll)
		}
	})

	t.Run("update flags override file values", func(t *testing.T) {
		resetSandboxFlagsForTest()
		sandboxUpdateConfigFile = writeTempFile(t, "ttl: 60\nauto_resume: false\n")
		sandboxUpdateTTL = 600
		sandboxUpdateAutoResume = "true"
		sandboxUpdateMemory = "2Gi"

		config, hasConfig, err := buildSandboxUpdateConfig()
		if err != nil {
			t.Fatalf("buildSandboxUpdateConfig() error = %v", err)
		}
		if !hasConfig {
			t.Fatal("hasConfig = false, want true")
		}
		ttl, ok := config.TTL.Get()
		if !ok || ttl != 600 {
			t.Fatalf("ttl = %d, want 600", ttl)
		}
		autoResume, ok := config.AutoResume.Get()
		if !ok || !autoResume {
			t.Fatalf("autoResume = %v, want true", autoResume)
		}
		resources, ok := config.Resources.Get()
		if !ok {
			t.Fatal("resources not set")
		}
		memory, ok := resources.Memory.Get()
		if !ok || memory != "2Gi" {
			t.Fatalf("memory = %q, want 2Gi", memory)
		}
	})
}

func TestBuildSandboxObservabilityOptions(t *testing.T) {
	t.Run("logs options use observability filters", func(t *testing.T) {
		resetSandboxFlagsForTest()
		cmd := newSandboxLogsOptionsTestCommand()
		if err := cmd.Flags().Set("limit", "25"); err != nil {
			t.Fatal(err)
		}
		if err := cmd.Flags().Set("context-id", "ctx_123"); err != nil {
			t.Fatal(err)
		}
		if err := cmd.Flags().Set("stream", "stderr"); err != nil {
			t.Fatal(err)
		}
		if err := cmd.Flags().Set("watch", "true"); err != nil {
			t.Fatal(err)
		}

		options, watch, err := buildSandboxLogObservabilityOptions(cmd)
		if err != nil {
			t.Fatalf("buildSandboxLogObservabilityOptions() error = %v", err)
		}
		if !watch {
			t.Fatal("watch = false, want true")
		}
		if options.Limit != 25 {
			t.Fatalf("limit = %d, want 25", options.Limit)
		}
		if options.ContextID != "ctx_123" {
			t.Fatalf("context_id = %q, want ctx_123", options.ContextID)
		}
		if options.Stream != apispec.SandboxObservabilityLogStreamStderr {
			t.Fatalf("stream = %q, want stderr", options.Stream)
		}
	})

	t.Run("watch rejects end time", func(t *testing.T) {
		resetSandboxFlagsForTest()
		cmd := newSandboxLogsOptionsTestCommand()
		if err := cmd.Flags().Set("watch", "true"); err != nil {
			t.Fatal(err)
		}
		if err := cmd.Flags().Set("end-time", "2026-07-03T00:00:00Z"); err != nil {
			t.Fatal(err)
		}

		_, _, err := buildSandboxLogObservabilityOptions(cmd)
		if err == nil {
			t.Fatal("buildSandboxLogObservabilityOptions() error = nil, want error")
		}
	})

	t.Run("metrics split repeated and comma separated names", func(t *testing.T) {
		resetSandboxFlagsForTest()
		cmd := newSandboxMetricsOptionsTestCommand()
		if err := cmd.Flags().Set("name", "cpu.percent,memory.rss"); err != nil {
			t.Fatal(err)
		}
		if err := cmd.Flags().Set("name", "io.read_bytes"); err != nil {
			t.Fatal(err)
		}

		options, _, err := buildSandboxMetricObservabilityOptions(cmd)
		if err != nil {
			t.Fatalf("buildSandboxMetricObservabilityOptions() error = %v", err)
		}
		want := []string{"cpu.percent", "memory.rss", "io.read_bytes"}
		if len(options.Names) != len(want) {
			t.Fatalf("names = %#v, want %#v", options.Names, want)
		}
		for i := range want {
			if options.Names[i] != want[i] {
				t.Fatalf("names = %#v, want %#v", options.Names, want)
			}
		}
	})
}

func TestSandboxAuditCommandIsNotRegistered(t *testing.T) {
	for _, cmd := range sandboxCmd.Commands() {
		if cmd.Name() == "audit" {
			t.Fatal("sandbox audit command should not be registered")
		}
	}
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func newSandboxLogsOptionsTestCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "logs"}
	addSandboxObservabilityFlags(cmd)
	cmd.Flags().StringVar(&sandboxObsContextID, "context-id", "", "")
	cmd.Flags().StringVar(&sandboxObsStream, "stream", "", "")
	cmd.Flags().BoolVarP(&sandboxLogsFollow, "follow", "f", false, "")
	cmd.Flags().IntVar(&sandboxLogsTailLines, "tail", 0, "")
	cmd.Flags().Int64Var(&sandboxLogsSinceSecs, "since-seconds", 0, "")
	return cmd
}

func newSandboxMetricsOptionsTestCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "metrics"}
	addSandboxObservabilityFlags(cmd)
	cmd.Flags().StringVar(&sandboxObsContextID, "context-id", "", "")
	cmd.Flags().StringArrayVar(&sandboxObsNames, "name", nil, "")
	return cmd
}

func resetSandboxFlagsForTest() {
	sandboxTemplate = ""
	sandboxTTL = 0
	sandboxHardTTL = 0
	sandboxMemory = ""
	sandboxConfigFile = ""
	sandboxMounts = nil
	sandboxSnapshotID = ""
	sandboxListStatus = ""
	sandboxListTemplateID = ""
	sandboxListPaused = ""
	sandboxListLimit = 0
	sandboxListOffset = 0
	sandboxObsLimit = 0
	sandboxObsCursor = ""
	sandboxObsStartTime = ""
	sandboxObsEndTime = ""
	sandboxObsSince = ""
	sandboxObsWatch = false
	sandboxObsContextID = ""
	sandboxObsStream = ""
	sandboxObsNames = nil
	sandboxObsSource = ""
	sandboxObsEventType = ""
	sandboxObsOutcome = ""
	sandboxLogsFollow = false
	sandboxLogsTailLines = 0
	sandboxLogsSinceSecs = 0
	sandboxUpdateTTL = 0
	sandboxUpdateHardTTL = 0
	sandboxUpdateMemory = ""
	sandboxUpdateAutoResume = ""
	sandboxUpdateConfigFile = ""
	sandboxRootFSSnapshotName = ""
	sandboxRootFSSnapshotDescription = ""
	sandboxRootFSSnapshotExpiresAt = ""
	sandboxForkTTL = 0
	sandboxForkHardTTL = 0
}
