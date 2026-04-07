package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

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
              valueTemplate: "Bearer {{token}}"
`)

		request, err := buildSandboxCreateRequest(false)
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

		request, err := buildSandboxCreateRequest(false)
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
	})

	t.Run("reads full claim request file", func(t *testing.T) {
		resetSandboxFlagsForTest()
		sandboxConfigFile = writeTempFile(t, `
template: from-file
mounts:
  - sandboxvolume_id: vol_123
    mount_point: /workspace/data
wait_for_mounts: true
mount_wait_timeout_ms: 45000
config:
  ttl: 90
`)

		request, err := buildSandboxCreateRequest(false)
		if err != nil {
			t.Fatalf("buildSandboxCreateRequest() error = %v", err)
		}
		template, ok := request.Template.Get()
		if !ok || template != "from-file" {
			t.Fatalf("template = %q, want from-file", template)
		}
		if len(request.Mounts) != 1 {
			t.Fatalf("mount count = %d, want 1", len(request.Mounts))
		}
		if request.Mounts[0].SandboxvolumeID != "vol_123" || request.Mounts[0].MountPoint != "/workspace/data" {
			t.Fatalf("unexpected mount = %+v", request.Mounts[0])
		}
		waitForMounts, ok := request.WaitForMounts.Get()
		if !ok || !waitForMounts {
			t.Fatalf("wait_for_mounts = %v, want true", waitForMounts)
		}
		timeout, ok := request.MountWaitTimeoutMs.Get()
		if !ok || timeout != 45000 {
			t.Fatalf("mount_wait_timeout_ms = %d, want 45000", timeout)
		}
	})

	t.Run("mount flags extend request and timeout implies wait", func(t *testing.T) {
		resetSandboxFlagsForTest()
		sandboxTemplate = "default"
		sandboxMounts = []string{"vol_abc:/workspace/bootstrap-data"}
		sandboxMountWaitTimeoutMS = 30000

		request, err := buildSandboxCreateRequest(false)
		if err != nil {
			t.Fatalf("buildSandboxCreateRequest() error = %v", err)
		}
		if len(request.Mounts) != 1 {
			t.Fatalf("mount count = %d, want 1", len(request.Mounts))
		}
		waitForMounts, ok := request.WaitForMounts.Get()
		if !ok || !waitForMounts {
			t.Fatalf("wait_for_mounts = %v, want true", waitForMounts)
		}
	})

	t.Run("mount flags append to request file mounts and template flag overrides", func(t *testing.T) {
		resetSandboxFlagsForTest()
		sandboxTemplate = "flag-template"
		sandboxConfigFile = writeTempFile(t, `
template: from-file
mounts:
  - sandboxvolume_id: vol_file
    mount_point: /workspace/from-file
wait_for_mounts: false
`)
		sandboxMounts = []string{"vol_flag:/workspace/from-flag"}
		sandboxWaitForMounts = true

		request, err := buildSandboxCreateRequest(true)
		if err != nil {
			t.Fatalf("buildSandboxCreateRequest() error = %v", err)
		}
		template, ok := request.Template.Get()
		if !ok || template != "flag-template" {
			t.Fatalf("template = %q, want flag-template", template)
		}
		if len(request.Mounts) != 2 {
			t.Fatalf("mount count = %d, want 2", len(request.Mounts))
		}
		if request.Mounts[0].SandboxvolumeID != "vol_file" || request.Mounts[1].SandboxvolumeID != "vol_flag" {
			t.Fatalf("unexpected mounts = %+v", request.Mounts)
		}
		waitForMounts, ok := request.WaitForMounts.Get()
		if !ok || !waitForMounts {
			t.Fatalf("wait_for_mounts = %v, want true", waitForMounts)
		}
	})

	t.Run("invalid mount flag fails", func(t *testing.T) {
		resetSandboxFlagsForTest()
		sandboxTemplate = "default"
		sandboxMounts = []string{"missing-separator"}

		_, err := buildSandboxCreateRequest(false)
		if err == nil {
			t.Fatal("buildSandboxCreateRequest() error = nil, want error")
		}
	})

	t.Run("relative mount path fails", func(t *testing.T) {
		resetSandboxFlagsForTest()
		sandboxTemplate = "default"
		sandboxMounts = []string{"vol_abc:workspace/relative"}

		_, err := buildSandboxCreateRequest(false)
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
		sandboxUpdateConfigFile = writeTempFile(t, `
ttl: 60
auto_resume: false
`)
		sandboxUpdateTTL = 600
		sandboxUpdateAutoResume = "true"

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
	})
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func resetSandboxFlagsForTest() {
	sandboxTemplate = ""
	sandboxTTL = 0
	sandboxHardTTL = 0
	sandboxConfigFile = ""
	sandboxMounts = nil
	sandboxWaitForMounts = false
	sandboxMountWaitTimeoutMS = 0
	sandboxListStatus = ""
	sandboxListTemplateID = ""
	sandboxListPaused = ""
	sandboxListLimit = 0
	sandboxListOffset = 0
	sandboxUpdateTTL = 0
	sandboxUpdateHardTTL = 0
	sandboxUpdateAutoResume = ""
	sandboxUpdateConfigFile = ""
}
