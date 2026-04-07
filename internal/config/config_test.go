package config

import (
	"os"
	"strings"
	"testing"
)

func TestSetGatewayModePersistsToConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	originalPath := *GetConfigFile()
	t.Cleanup(func() {
		SetConfigFile(originalPath)
	})
	SetConfigFile(configPath)

	cfg := &Config{
		Profiles: map[string]Profile{},
	}
	cfg.SetGatewayMode("test-profile", GatewayModeGlobal)
	if err := cfg.Save(); err != nil {
		t.Fatalf("cfg.Save() error = %v", err)
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if !strings.Contains(string(raw), "gateway-mode: global") {
		t.Fatalf("saved config missing gateway-mode: %s", string(raw))
	}
}
