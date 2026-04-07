package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sandbox0-ai/s0/internal/config"
)

func TestResolveTargetDefaultsToDirectWhenMetadataIsMissing(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	target, err := ResolveTarget(context.Background(), ResolveTargetOptions{
		BaseURL:   server.URL,
		Token:     "s0_api_key_1",
		Scope:     RouteScopeHomeRegion,
		UserAgent: "s0/test",
	})
	if err != nil {
		t.Fatalf("ResolveTarget() error = %v", err)
	}

	if target.BaseURL != server.URL {
		t.Fatalf("BaseURL = %q, want %q", target.BaseURL, server.URL)
	}
	if target.Token != "s0_api_key_1" {
		t.Fatalf("Token = %q, want s0_api_key_1", target.Token)
	}
	if target.GatewayMode != config.GatewayModeDirect {
		t.Fatalf("GatewayMode = %q, want %q", target.GatewayMode, config.GatewayModeDirect)
	}
}

func TestResolveTargetRequiresCurrentTeamForDirectHomeRegionRoutingWithUserToken(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	_, err := ResolveTarget(context.Background(), ResolveTargetOptions{
		BaseURL:   server.URL,
		Token:     "user-token",
		Scope:     RouteScopeHomeRegion,
		UserAgent: "s0/test",
	})
	if err != ErrCurrentTeamRequired {
		t.Fatalf("ResolveTarget() error = %v, want %v", err, ErrCurrentTeamRequired)
	}
}

func TestResolveTargetAllowsDirectHomeRegionRoutingWithAPIKeyWithoutCurrentTeam(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	target, err := ResolveTarget(context.Background(), ResolveTargetOptions{
		BaseURL:   server.URL,
		Token:     "s0_api_key_1",
		Scope:     RouteScopeHomeRegion,
		UserAgent: "s0/test",
	})
	if err != nil {
		t.Fatalf("ResolveTarget() error = %v", err)
	}
	if target.BaseURL != server.URL {
		t.Fatalf("BaseURL = %q, want %q", target.BaseURL, server.URL)
	}
}

func TestResolveTargetFallsBackToEntrypointForGlobalHomeRegionRoutingWithoutCurrentTeam(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata" {
			http.NotFound(w, r)
			return
		}

		writeJSON(t, w, http.StatusOK, map[string]any{
			"success": true,
			"data": map[string]any{
				"gateway_mode": "global",
				"service":      "global-gateway",
			},
		})
	}))
	defer server.Close()

	target, err := ResolveTarget(context.Background(), ResolveTargetOptions{
		BaseURL:   server.URL,
		Token:     "global-token",
		Scope:     RouteScopeHomeRegion,
		UserAgent: "s0/test",
	})
	if err != nil {
		t.Fatalf("ResolveTarget() error = %v", err)
	}
	if target.BaseURL != server.URL {
		t.Fatalf("BaseURL = %q, want %q", target.BaseURL, server.URL)
	}
	if target.Token != "global-token" {
		t.Fatalf("Token = %q, want global-token", target.Token)
	}
}

func TestResolveTargetUsesCachedCurrentTeamGatewayForGlobalHomeRegionRouting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/metadata":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"gateway_mode": "global",
					"service":      "global-gateway",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	target, err := ResolveTarget(context.Background(), ResolveTargetOptions{
		BaseURL:       server.URL,
		Token:         "global-token",
		CurrentTeamID: "team-1",
		CurrentTeamTarget: &config.CurrentTeamTarget{
			TeamID:     "team-1",
			GatewayURL: "https://regional.example.com",
			RegionID:   "aws/us-east-1",
		},
		Scope:     RouteScopeHomeRegion,
		UserAgent: "s0/test",
	})
	if err != nil {
		t.Fatalf("ResolveTarget() error = %v", err)
	}

	if target.BaseURL != "https://regional.example.com" {
		t.Fatalf("BaseURL = %q, want regional gateway URL", target.BaseURL)
	}
	if target.Token != "global-token" {
		t.Fatalf("Token = %q, want global-token", target.Token)
	}
}

func TestResolveTargetFallsBackToEntrypointForGlobalHomeRegionRoutingWithoutCachedTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/metadata":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"gateway_mode": "global",
					"service":      "global-gateway",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	target, err := ResolveTarget(context.Background(), ResolveTargetOptions{
		BaseURL:       server.URL,
		Token:         "global-token",
		CurrentTeamID: "team-1",
		Scope:         RouteScopeHomeRegion,
		UserAgent:     "s0/test",
	})
	if err != nil {
		t.Fatalf("ResolveTarget() error = %v", err)
	}
	if target.BaseURL != server.URL {
		t.Fatalf("BaseURL = %q, want %q", target.BaseURL, server.URL)
	}
	if target.Token != "global-token" {
		t.Fatalf("Token = %q, want global-token", target.Token)
	}
}

func TestResolveTargetHonorsConfiguredGlobalGatewayModeWithoutCurrentTeam(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	target, err := ResolveTarget(context.Background(), ResolveTargetOptions{
		BaseURL:               server.URL,
		Token:                 "global-token",
		ConfiguredGatewayMode: config.GatewayModeGlobal,
		Scope:                 RouteScopeHomeRegion,
		UserAgent:             "s0/test",
	})
	if err != nil {
		t.Fatalf("ResolveTarget() error = %v", err)
	}
	if target.BaseURL != server.URL {
		t.Fatalf("BaseURL = %q, want %q", target.BaseURL, server.URL)
	}
	if target.Token != "global-token" {
		t.Fatalf("Token = %q, want global-token", target.Token)
	}
}

func TestResolveTargetKeepsEntrypointCommandsOnGlobalGateway(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/metadata":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"gateway_mode": "global",
					"service":      "global-gateway",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	target, err := ResolveTarget(context.Background(), ResolveTargetOptions{
		BaseURL:   server.URL,
		Token:     "global-token",
		Scope:     RouteScopeEntrypoint,
		UserAgent: "s0/test",
	})
	if err != nil {
		t.Fatalf("ResolveTarget() error = %v", err)
	}

	if target.BaseURL != server.URL {
		t.Fatalf("BaseURL = %q, want %q", target.BaseURL, server.URL)
	}
	if target.Token != "global-token" {
		t.Fatalf("Token = %q, want global-token", target.Token)
	}
}

func TestResolveTargetHonorsConfiguredGatewayModeWhenMetadataIsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/metadata":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	target, err := ResolveTarget(context.Background(), ResolveTargetOptions{
		BaseURL:               server.URL,
		Token:                 "global-token",
		ConfiguredGatewayMode: config.GatewayModeGlobal,
		CurrentTeamID:         "team-1",
		CurrentTeamTarget: &config.CurrentTeamTarget{
			TeamID:     "team-1",
			GatewayURL: "https://regional.example.com",
			RegionID:   "aws/us-east-1",
		},
		Scope:     RouteScopeHomeRegion,
		UserAgent: "s0/test",
	})
	if err != nil {
		t.Fatalf("ResolveTarget() error = %v", err)
	}

	if target.BaseURL != "https://regional.example.com" {
		t.Fatalf("BaseURL = %q, want regional gateway URL", target.BaseURL)
	}
	if target.Token != "global-token" {
		t.Fatalf("Token = %q, want global-token", target.Token)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, payload any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode json: %v", err)
	}
}
