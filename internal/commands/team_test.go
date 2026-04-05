package commands

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildCreateTeamRequest(t *testing.T) {
	req := buildCreateTeamRequest("Team One", " team-one ", " aws/us-east-1 ")

	if req.Name != "Team One" {
		t.Fatalf("Name = %q, want Team One", req.Name)
	}

	slug, ok := req.Slug.Get()
	if !ok || slug != "team-one" {
		t.Fatalf("Slug = %q, want team-one", slug)
	}

	homeRegion, ok := req.HomeRegionID.Get()
	if !ok || homeRegion != "aws/us-east-1" {
		t.Fatalf("HomeRegionID = %q, want aws/us-east-1", homeRegion)
	}
}

func TestBuildCreateTeamRequestOmitsOptionalFieldsWhenBlank(t *testing.T) {
	req := buildCreateTeamRequest("Team One", "   ", "   ")

	if _, ok := req.Slug.Get(); ok {
		t.Fatal("Slug should be unset")
	}
	if _, ok := req.HomeRegionID.Get(); ok {
		t.Fatal("HomeRegionID should be unset")
	}
}

func TestTeamCreateCommandDoesNotExposeActivateFlag(t *testing.T) {
	if flag := teamCreateCmd.Flags().Lookup("activate"); flag != nil {
		t.Fatalf("activate flag should be removed, got %v", flag)
	}
}

func TestResolveTeamHomeRegionID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/teams/team-1" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token-1" {
			t.Fatalf("Authorization = %q, want Bearer token-1", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"id":"team-1","name":"Team One","slug":"team-one","home_region_id":"aws/us-east-1","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}}`))
	}))
	defer server.Close()

	client, err := newSDKClientForBaseURL(server.URL, "token-1")
	if err != nil {
		t.Fatalf("newSDKClientForBaseURL() error = %v", err)
	}

	homeRegionID, err := resolveTeamHomeRegionID(context.Background(), client, "team-1")
	if err != nil {
		t.Fatalf("resolveTeamHomeRegionID() error = %v", err)
	}
	if homeRegionID != "aws/us-east-1" {
		t.Fatalf("resolveTeamHomeRegionID() = %q, want aws/us-east-1", homeRegionID)
	}
}

func TestResolveTeamHomeRegionIDRequiresConfiguredHomeRegion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/teams/team-1" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"id":"team-1","name":"Team One","slug":"team-one","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}}`))
	}))
	defer server.Close()

	client, err := newSDKClientForBaseURL(server.URL, "token-1")
	if err != nil {
		t.Fatalf("newSDKClientForBaseURL() error = %v", err)
	}

	_, err = resolveTeamHomeRegionID(context.Background(), client, "team-1")
	if err == nil || !strings.Contains(err.Error(), "team team-1 has no home region") {
		t.Fatalf("expected missing home region error, got %v", err)
	}
}

func TestResolveTeamGatewayURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/teams/team-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":"team-1","name":"Team One","slug":"team-one","home_region_id":"aws/us-east-1","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}}`))
		case "/regions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"data":{"regions":[{"id":"aws/us-east-1","display_name":"US East 1","regional_gateway_url":"https://use1.example.com","metering_export_url":"https://metering.use1.example.com","enabled":true}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := newSDKClientForBaseURL(server.URL, "token-1")
	if err != nil {
		t.Fatalf("newSDKClientForBaseURL() error = %v", err)
	}

	regionalGatewayURL, err := resolveTeamGatewayURL(context.Background(), client, "team-1")
	if err != nil {
		t.Fatalf("resolveTeamGatewayURL() error = %v", err)
	}
	if regionalGatewayURL != "https://use1.example.com" {
		t.Fatalf("resolveTeamGatewayURL() = %q, want https://use1.example.com", regionalGatewayURL)
	}
}
