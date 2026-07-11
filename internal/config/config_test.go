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

func TestGetConfiguredGatewayModeIgnoresProfileModeForEndpointOverride(t *testing.T) {
	originalURL := *GetAPIURLVar()
	t.Cleanup(func() { SetAPIURL(originalURL) })
	t.Setenv(EnvBaseURL, "")
	SetAPIURL("")

	profile := &Profile{
		GatewayMode:           string(GatewayModeGlobal),
		CurrentTeamID:         "team-profile",
		CurrentTeamRegionID:   "region-profile",
		CurrentTeamGatewayURL: "https://region-profile.example.com",
	}
	if mode, ok := profile.GetConfiguredGatewayMode(); !ok || mode != GatewayModeGlobal {
		t.Fatalf("GetConfiguredGatewayMode() = %q, %v, want global, true", mode, ok)
	}

	t.Setenv(EnvBaseURL, "http://127.0.0.1:30080")
	if mode, ok := profile.GetConfiguredGatewayMode(); ok || mode != "" {
		t.Fatalf("GetConfiguredGatewayMode() with env override = %q, %v, want empty, false", mode, ok)
	}
	if teamID := profile.GetCurrentTeamID(); teamID != "" {
		t.Fatalf("GetCurrentTeamID() with env override = %q, want empty", teamID)
	}
	if target, ok := profile.GetCurrentTeamTarget(); ok || target != nil {
		t.Fatalf("GetCurrentTeamTarget() with env override = %#v, %v, want nil, false", target, ok)
	}

	t.Setenv(EnvBaseURL, "")
	SetAPIURL("http://127.0.0.1:30080")
	if mode, ok := profile.GetConfiguredGatewayMode(); ok || mode != "" {
		t.Fatalf("GetConfiguredGatewayMode() with flag override = %q, %v, want empty, false", mode, ok)
	}
}
