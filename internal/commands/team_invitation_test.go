package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sandbox0-ai/s0/internal/config"
)

func TestBuildTeamInvitationCreateRequest(t *testing.T) {
	teamID, request, err := buildTeamInvitationCreateRequest(teamInvitationCreateOptions{
		teamID: " team-1 ",
		email:  " Invitee@Example.com ",
		role:   " BUILDER ",
	})
	if err != nil {
		t.Fatalf("buildTeamInvitationCreateRequest() error = %v", err)
	}
	if teamID != "team-1" {
		t.Fatalf("team ID = %q, want team-1", teamID)
	}
	if request.Email != "invitee@example.com" {
		t.Fatalf("email = %q, want invitee@example.com", request.Email)
	}
	if request.Role != "builder" {
		t.Fatalf("role = %q, want builder", request.Role)
	}
}

func TestBuildTeamInvitationCreateRequestRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		options teamInvitationCreateOptions
		want    string
	}{
		{name: "team", options: teamInvitationCreateOptions{email: "invitee@example.com", role: "developer"}, want: "--team-id is required"},
		{name: "email", options: teamInvitationCreateOptions{teamID: "team-1", role: "developer"}, want: "--email is required"},
		{name: "invalid email", options: teamInvitationCreateOptions{teamID: "team-1", email: "not-an-email", role: "developer"}, want: "--email must be a valid email address"},
		{name: "role", options: teamInvitationCreateOptions{teamID: "team-1", email: "invitee@example.com", role: "owner"}, want: "invalid --role"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := buildTeamInvitationCreateRequest(test.options)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestTeamInvitationCreateCommandUsesCloudEndpoint(t *testing.T) {
	resetTeamInvitationCommandConfig(t)
	cfgFormat = "json"
	t.Cleanup(func() {
		cfgFormat = "table"
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/metadata":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"data":{"gateway_mode":"global","service":"global-gateway"}}`))
		case "/cloud/v1/teams/team-1/invitations":
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer token-1" {
				t.Fatalf("Authorization = %q, want Bearer token-1", got)
			}
			if got := r.Header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
			var request teamInvitationCreateRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode invitation request: %v", err)
			}
			if request.Email != "invitee@example.com" || request.Role != "builder" {
				t.Fatalf("request = %#v", request)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":"invite-1","team_id":"team-1","invitee_email":"invitee@example.com","role":"builder","status":"pending","expires_at":"2026-08-17T00:00:00Z","last_send_requested_at":"2026-08-10T00:00:00Z","send_count":1,"delivery_state":"queued","created_at":"2026-08-10T00:00:00Z","updated_at":"2026-08-10T00:00:00Z"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv(config.EnvBaseURL, server.URL)
	t.Setenv(config.EnvToken, "token-1")

	command := newTeamInvitationCreateCommand()
	command.SetContext(context.Background())
	if err := command.Flags().Parse([]string{
		"--team-id", "team-1",
		"--email", "Invitee@Example.com",
		"--role", "builder",
	}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	var stdout bytes.Buffer
	command.SetOut(&stdout)

	if err := command.RunE(command, nil); err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
	for _, want := range []string{
		`"id": "invite-1"`,
		`"team_id": "team-1"`,
		`"invitee_email": "invitee@example.com"`,
		`"status": "pending"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestTeamInvitationCreateCommandRequiresGlobalGateway(t *testing.T) {
	resetTeamInvitationCommandConfig(t)

	invitationCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/metadata":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"data":{"gateway_mode":"direct","service":"cluster-gateway"}}`))
		case "/cloud/v1/teams/team-1/invitations":
			invitationCalled = true
			http.Error(w, "unexpected request", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv(config.EnvBaseURL, server.URL)
	t.Setenv(config.EnvToken, "token-1")

	command := newTeamInvitationCreateCommand()
	command.SetContext(context.Background())
	if err := command.Flags().Parse([]string{
		"--team-id", "team-1",
		"--email", "invitee@example.com",
	}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}

	err := command.RunE(command, nil)
	if err == nil || !strings.Contains(err.Error(), "require a Sandbox0 Cloud Global Gateway") {
		t.Fatalf("error = %v", err)
	}
	if invitationCalled {
		t.Fatal("invitation endpoint was called for a direct gateway")
	}
}

func resetTeamInvitationCommandConfig(t *testing.T) {
	t.Helper()
	config.SetConfigFile("")
	config.SetProfile("")
	config.SetAPIURL("")
	config.SetToken("")
	t.Cleanup(func() {
		config.SetConfigFile("")
		config.SetProfile("")
		config.SetAPIURL("")
		config.SetToken("")
	})
}
