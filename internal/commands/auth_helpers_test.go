package commands

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestBuildOIDCLoginURL(t *testing.T) {
	loginURL := buildOIDCLoginURL("https://global.example.com/", "google", "http://127.0.0.1:38123/callback")

	parsed, err := url.Parse(loginURL)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if parsed.Path != "/auth/oidc/google/login" {
		t.Fatalf("Path = %q, want /auth/oidc/google/login", parsed.Path)
	}
	if got := parsed.Query().Get("return_url"); got != "http://127.0.0.1:38123/callback" {
		t.Fatalf("return_url = %q, want callback URL", got)
	}
	if got := parsed.Query().Get("home_region_id"); got != "" {
		t.Fatalf("home_region_id = %q, want empty", got)
	}
}

func TestShouldShowFirstTeamOnboardingHint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"gateway_mode":"global","service":"global-gateway"}}`))
	}))
	defer server.Close()

	if !shouldShowFirstTeamOnboardingHint(context.Background(), server.URL, &authLoginData{}) {
		t.Fatal("shouldShowFirstTeamOnboardingHint() = false, want true")
	}
}

func TestShouldShowFirstTeamOnboardingHintSkipsWhenRegionalSessionExists(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	if shouldShowFirstTeamOnboardingHint(context.Background(), server.URL, &authLoginData{
		RegionalSession: &struct {
			RegionID           string `json:"region_id"`
			RegionalGatewayURL string `json:"regional_gateway_url"`
			Token              string `json:"token"`
			ExpiresAt          int64  `json:"expires_at"`
		}{
			RegionID:           "aws/us-east-1",
			RegionalGatewayURL: "https://regional.example.com",
			Token:              "region-token",
			ExpiresAt:          1893456000,
		},
	}) {
		t.Fatal("shouldShowFirstTeamOnboardingHint() = true, want false")
	}
}

func TestAuthLoginCommandDoesNotExposeHomeRegionFlag(t *testing.T) {
	if flag := authLoginCmd.Flags().Lookup("home-region"); flag != nil {
		t.Fatalf("home-region flag should be removed, got %v", flag)
	}
}

func TestSelectAuthProviderAutoPrefersDeviceOIDC(t *testing.T) {
	provider, mode, err := selectAuthProvider([]authProvider{
		{ID: "auth0", Type: "oidc", BrowserLoginEnabled: true, DeviceLoginEnabled: true},
		{ID: "builtin", Type: "builtin"},
	}, "auto")
	if err != nil {
		t.Fatalf("selectAuthProvider() error = %v", err)
	}
	if provider.ID != "auth0" {
		t.Fatalf("provider = %q, want auth0", provider.ID)
	}
	if mode != authLoginModeDevice {
		t.Fatalf("mode = %q, want %q", mode, authLoginModeDevice)
	}
}

func TestSelectAuthProviderBuiltinModeRequiresBuiltinProvider(t *testing.T) {
	_, _, err := selectAuthProvider([]authProvider{
		{ID: "auth0", Type: "oidc", BrowserLoginEnabled: true, DeviceLoginEnabled: true},
	}, "builtin")
	if err == nil {
		t.Fatal("expected error when builtin provider is absent")
	}
}
