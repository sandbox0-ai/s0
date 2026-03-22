package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestBuildSandboxCreateConfig(t *testing.T) {
	resetSandboxFlagsForTest()

	t.Run("reads network policy and bindings from config file", func(t *testing.T) {
		resetSandboxFlagsForTest()
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

		config, hasConfig, err := buildSandboxCreateConfig()
		if err != nil {
			t.Fatalf("buildSandboxCreateConfig() error = %v", err)
		}
		if !hasConfig {
			t.Fatal("hasConfig = false, want true")
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
		sandboxConfigFile = writeTempFile(t, `
ttl: 60
hard_ttl: 120
`)
		sandboxTTL = 300

		config, hasConfig, err := buildSandboxCreateConfig()
		if err != nil {
			t.Fatalf("buildSandboxCreateConfig() error = %v", err)
		}
		if !hasConfig {
			t.Fatal("hasConfig = false, want true")
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
