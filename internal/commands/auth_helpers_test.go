package commands

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sandbox0-ai/s0/internal/config"
	sandbox0 "github.com/sandbox0-ai/sdk-go"
)

func TestShouldShowCurrentTeamSelectionHint(t *testing.T) {
	if !shouldShowCurrentTeamSelectionHint(config.GatewayModeGlobal, "") {
		t.Fatal("shouldShowCurrentTeamSelectionHint() = false, want true")
	}
}

func TestShouldShowCurrentTeamSelectionHintSkipsWhenCurrentTeamExists(t *testing.T) {
	if shouldShowCurrentTeamSelectionHint(config.GatewayModeGlobal, "team-1") {
		t.Fatal("shouldShowCurrentTeamSelectionHint() = true, want false")
	}
}

func TestShouldShowCurrentTeamSelectionHintSkipsInDirectMode(t *testing.T) {
	if shouldShowCurrentTeamSelectionHint(config.GatewayModeDirect, "") {
		t.Fatal("shouldShowCurrentTeamSelectionHint() = true, want false")
	}
}

func TestMaybeAutoSelectCurrentTeamSelectsOnlyTeamInDirectMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/teams":
			writeAuthJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"teams": []map[string]any{{
						"id":         "team-1",
						"name":       "Personal Team",
						"slug":       "personal-team",
						"created_at": "2026-01-01T00:00:00Z",
						"updated_at": "2026-01-01T00:00:00Z",
					}},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		CurrentProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				APIURL: server.URL,
				Token:  "user-token",
			},
		},
	}

	team, ok, err := maybeAutoSelectCurrentTeam(context.Background(), cfg, "default")
	if err != nil {
		t.Fatalf("maybeAutoSelectCurrentTeam() error = %v", err)
	}
	if !ok {
		t.Fatal("maybeAutoSelectCurrentTeam() did not auto-select team")
	}
	if team.ID != "team-1" {
		t.Fatalf("team.ID = %q, want team-1", team.ID)
	}

	profile, err := cfg.GetProfile("default")
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if got := profile.GetCurrentTeamID(); got != "team-1" {
		t.Fatalf("CurrentTeamID = %q, want team-1", got)
	}
	if target, ok := profile.GetCurrentTeamTarget(); ok {
		t.Fatalf("CurrentTeamTarget should be unset in direct mode, got %+v", target)
	}
}

func TestMaybeAutoSelectCurrentTeamDoesNotSelectWhenMultipleTeamsExist(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/teams":
			writeAuthJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"teams": []map[string]any{
						{
							"id": "team-1", "name": "One", "slug": "one", "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
						},
						{
							"id": "team-2", "name": "Two", "slug": "two", "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		CurrentProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				APIURL: server.URL,
				Token:  "user-token",
			},
		},
	}

	team, ok, err := maybeAutoSelectCurrentTeam(context.Background(), cfg, "default")
	if err != nil {
		t.Fatalf("maybeAutoSelectCurrentTeam() error = %v", err)
	}
	if ok {
		t.Fatalf("maybeAutoSelectCurrentTeam() selected unexpected team %+v", team)
	}

	profile, err := cfg.GetProfile("default")
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if got := profile.GetCurrentTeamID(); got != "" {
		t.Fatalf("CurrentTeamID = %q, want empty", got)
	}
}

func TestMaybeAutoSelectCurrentTeamReplacesStaleTeamWhenOnlyOneTeamExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/teams":
			writeAuthJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"teams": []map[string]any{{
						"id":         "team-2",
						"name":       "New Team",
						"slug":       "new-team",
						"created_at": "2026-01-01T00:00:00Z",
						"updated_at": "2026-01-01T00:00:00Z",
					}},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		CurrentProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				APIURL:        server.URL,
				Token:         "user-token",
				CurrentTeamID: "team-1",
			},
		},
	}

	team, ok, err := maybeAutoSelectCurrentTeam(context.Background(), cfg, "default")
	if err != nil {
		t.Fatalf("maybeAutoSelectCurrentTeam() error = %v", err)
	}
	if !ok {
		t.Fatal("maybeAutoSelectCurrentTeam() did not replace stale team")
	}
	if team == nil || team.ID != "team-2" {
		t.Fatalf("team = %+v, want team-2", team)
	}

	profile, err := cfg.GetProfile("default")
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if got := profile.GetCurrentTeamID(); got != "team-2" {
		t.Fatalf("CurrentTeamID = %q, want team-2", got)
	}
}

func TestMaybeAutoSelectCurrentTeamClearsStaleTeamWhenMultipleTeamsExist(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/teams":
			writeAuthJSON(t, w, http.StatusOK, map[string]any{
				"success": true,
				"data": map[string]any{
					"teams": []map[string]any{
						{
							"id":         "team-2",
							"name":       "Two",
							"slug":       "two",
							"created_at": "2026-01-01T00:00:00Z",
							"updated_at": "2026-01-01T00:00:00Z",
						},
						{
							"id":         "team-3",
							"name":       "Three",
							"slug":       "three",
							"created_at": "2026-01-01T00:00:00Z",
							"updated_at": "2026-01-01T00:00:00Z",
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		CurrentProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				APIURL:              server.URL,
				Token:               "user-token",
				CurrentTeamID:       "team-1",
				CurrentTeamRegionID: "aws-us-east-1",
			},
		},
	}

	team, ok, err := maybeAutoSelectCurrentTeam(context.Background(), cfg, "default")
	if err != nil {
		t.Fatalf("maybeAutoSelectCurrentTeam() error = %v", err)
	}
	if ok || team != nil {
		t.Fatalf("maybeAutoSelectCurrentTeam() = (%+v, %v), want no auto-selection", team, ok)
	}

	profile, err := cfg.GetProfile("default")
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if got := profile.GetCurrentTeamID(); got != "" {
		t.Fatalf("CurrentTeamID = %q, want empty", got)
	}
	if _, ok := profile.GetCurrentTeamTarget(); ok {
		t.Fatal("CurrentTeamTarget should be cleared when current team is stale")
	}
}

func TestRefreshProfileTeamGrantsRefreshesToken(t *testing.T) {
	oldToken := *config.GetTokenVar()
	*config.GetTokenVar() = ""
	t.Cleanup(func() { *config.GetTokenVar() = oldToken })
	t.Setenv(config.EnvToken, "")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/refresh" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		writeAuthJSON(t, w, http.StatusOK, map[string]any{
			"success": true,
			"data": map[string]any{
				"access_token":  "new-access-token",
				"refresh_token": "new-refresh-token",
				"expires_at":    int64(1893456000),
			},
		})
	}))
	defer server.Close()

	cfg := &config.Config{
		CurrentProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				APIURL:       server.URL,
				Token:        "old-access-token",
				RefreshToken: "old-refresh-token",
				ExpiresAt:    100,
			},
		},
	}

	refreshed, err := refreshProfileTeamGrants(context.Background(), cfg, "default")
	if err != nil {
		t.Fatalf("refreshProfileTeamGrants() error = %v", err)
	}
	if !refreshed {
		t.Fatal("refreshProfileTeamGrants() did not refresh")
	}
	profile, err := cfg.GetProfile("default")
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if profile.Token != "new-access-token" {
		t.Fatalf("Token = %q, want new-access-token", profile.Token)
	}
	if profile.RefreshToken != "new-refresh-token" {
		t.Fatalf("RefreshToken = %q, want new-refresh-token", profile.RefreshToken)
	}
	if profile.ExpiresAt != 1893456000 {
		t.Fatalf("ExpiresAt = %d, want 1893456000", profile.ExpiresAt)
	}
}

func TestRefreshProfileTeamGrantsSkipsWithoutRefreshToken(t *testing.T) {
	oldToken := *config.GetTokenVar()
	*config.GetTokenVar() = ""
	t.Cleanup(func() { *config.GetTokenVar() = oldToken })
	t.Setenv(config.EnvToken, "")

	cfg := &config.Config{
		CurrentProfile: "default",
		Profiles: map[string]config.Profile{
			"default": {
				APIURL: "https://api.example.test",
				Token:  "access-token",
			},
		},
	}

	refreshed, err := refreshProfileTeamGrants(context.Background(), cfg, "default")
	if err != nil {
		t.Fatalf("refreshProfileTeamGrants() error = %v", err)
	}
	if refreshed {
		t.Fatal("refreshProfileTeamGrants() refreshed without a refresh token")
	}
}

func TestWithSelectedTeamAuthHintAddsTokenStaleHint(t *testing.T) {
	err := withSelectedTeamAuthHint(&sandbox0.APIError{
		StatusCode: http.StatusUnauthorized,
		Message:    "not a member of selected team",
	})
	if err == nil {
		t.Fatal("withSelectedTeamAuthHint() returned nil")
	}
	got := err.Error()
	if !strings.Contains(got, "not a member of selected team") {
		t.Fatalf("error = %q, want original message", got)
	}
	if !strings.Contains(got, "token stale, run `s0 auth login` or refresh token") {
		t.Fatalf("error = %q, want stale token hint", got)
	}
}

func TestWithSelectedTeamAuthHintHandlesCompactJSONError(t *testing.T) {
	err := withSelectedTeamAuthHint(&sandbox0.APIError{
		StatusCode: http.StatusUnauthorized,
		Message:    `{"error":"not a member of selected team"}`,
	})
	if err == nil {
		t.Fatal("withSelectedTeamAuthHint() returned nil")
	}
	if got := err.Error(); !strings.Contains(got, "token stale, run `s0 auth login` or refresh token") {
		t.Fatalf("error = %q, want stale token hint", got)
	}
}

func writeAuthJSON(t *testing.T, w http.ResponseWriter, status int, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode response: %v", err)
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

func TestSelectAuthProviderRejectsBrowserMode(t *testing.T) {
	_, _, err := selectAuthProvider([]authProvider{{ID: "auth0", Type: "oidc", BrowserLoginEnabled: true, DeviceLoginEnabled: true}}, "browser")
	if err == nil {
		t.Fatal("expected browser mode to be rejected")
	}
	if got := err.Error(); got != "browser auth mode is no longer supported; use --mode device or --mode builtin" {
		t.Fatalf("error = %q, want browser mode rejection", got)
	}
}
