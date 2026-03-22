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
		Token:     "token-1",
		Scope:     RouteScopeHomeRegion,
		UserAgent: "s0/test",
	})
	if err != nil {
		t.Fatalf("ResolveTarget() error = %v", err)
	}

	if target.BaseURL != server.URL {
		t.Fatalf("BaseURL = %q, want %q", target.BaseURL, server.URL)
	}
	if target.Token != "token-1" {
		t.Fatalf("Token = %q, want token-1", target.Token)
	}
	if target.GatewayMode != config.GatewayModeDirect {
		t.Fatalf("GatewayMode = %q, want %q", target.GatewayMode, config.GatewayModeDirect)
	}
}

func TestResolveTargetUsesGlobalHomeRegionRouting(t *testing.T) {
	var tenantActiveCalls int
	var regionTokenCalls int

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
		case "/tenant/active":
			tenantActiveCalls++
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"user_id":              "user-1",
					"team_id":              "team-1",
					"team_role":            "admin",
					"home_region_id":       "aws/us-east-1",
					"default_team":         true,
					"regional_gateway_url": "https://regional.example.com",
				},
			})
		case "/auth/region-token":
			regionTokenCalls++
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"region_id":            "aws/us-east-1",
					"regional_gateway_url": "https://regional.example.com",
					"token":                "region-token",
					"expires_at":           int64(1893456000),
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
		Scope:     RouteScopeHomeRegion,
		UserAgent: "s0/test",
	})
	if err != nil {
		t.Fatalf("ResolveTarget() error = %v", err)
	}

	if target.BaseURL != "https://regional.example.com" {
		t.Fatalf("BaseURL = %q, want regional gateway URL", target.BaseURL)
	}
	if target.Token != "region-token" {
		t.Fatalf("Token = %q, want region-token", target.Token)
	}
	if tenantActiveCalls != 1 {
		t.Fatalf("tenant/active calls = %d, want 1", tenantActiveCalls)
	}
	if regionTokenCalls != 1 {
		t.Fatalf("auth/region-token calls = %d, want 1", regionTokenCalls)
	}
}

func TestResolveTargetUsesStoredRegionalSessionBeforeGlobalExchange(t *testing.T) {
	var tenantActiveCalls int
	var regionTokenCalls int

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
		case "/tenant/active":
			tenantActiveCalls++
			http.Error(w, "unexpected", http.StatusInternalServerError)
		case "/auth/region-token":
			regionTokenCalls++
			http.Error(w, "unexpected", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	target, err := ResolveTarget(context.Background(), ResolveTargetOptions{
		BaseURL: server.URL,
		Token:   "global-token",
		RegionalSession: &config.RegionalSession{
			Token:      "regional-token",
			GatewayURL: "https://regional.example.com",
			RegionID:   "aws/us-east-1",
			ExpiresAt:  1893456000,
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
	if target.Token != "regional-token" {
		t.Fatalf("Token = %q, want regional-token", target.Token)
	}
	if tenantActiveCalls != 0 {
		t.Fatalf("tenant/active calls = %d, want 0", tenantActiveCalls)
	}
	if regionTokenCalls != 0 {
		t.Fatalf("auth/region-token calls = %d, want 0", regionTokenCalls)
	}
}

func TestResolveTargetKeepsEntrypointCommandsOnGlobalGateway(t *testing.T) {
	var tenantActiveCalls int
	var regionTokenCalls int

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
		case "/tenant/active":
			tenantActiveCalls++
			http.Error(w, "unexpected", http.StatusInternalServerError)
		case "/auth/region-token":
			regionTokenCalls++
			http.Error(w, "unexpected", http.StatusInternalServerError)
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
	if tenantActiveCalls != 0 {
		t.Fatalf("tenant/active calls = %d, want 0", tenantActiveCalls)
	}
	if regionTokenCalls != 0 {
		t.Fatalf("auth/region-token calls = %d, want 0", regionTokenCalls)
	}
}

func TestResolveTargetHonorsConfiguredGatewayModeWhenMetadataIsUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/metadata":
			http.NotFound(w, r)
		case "/tenant/active":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"user_id":              "user-1",
					"team_id":              "team-1",
					"team_role":            "admin",
					"home_region_id":       "aws/us-east-1",
					"default_team":         true,
					"regional_gateway_url": "https://regional.example.com",
				},
			})
		case "/auth/region-token":
			writeJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"region_id":            "aws/us-east-1",
					"regional_gateway_url": "https://regional.example.com",
					"token":                "region-token",
					"expires_at":           int64(1893456000),
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
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

	if target.BaseURL != "https://regional.example.com" {
		t.Fatalf("BaseURL = %q, want regional gateway URL", target.BaseURL)
	}
	if target.Token != "region-token" {
		t.Fatalf("Token = %q, want region-token", target.Token)
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
