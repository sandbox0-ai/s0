package commands

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sandbox0-ai/s0/internal/config"
)

func TestConditionalSessionLookupCommandUsesOnlyOriginalReceipt(t *testing.T) {
	config.SetConfigFile("")
	config.SetProfile("")
	config.SetAPIURL("")
	config.SetToken("")
	oldFormat, oldGeneration := cfgFormat, sessionLookupGeneration
	cfgFormat = "json"
	sessionLookupGeneration = 7
	t.Cleanup(func() {
		cfgFormat = oldFormat
		sessionLookupGeneration = oldGeneration
		sandboxSessionFindConditionalCmd.SetOut(nil)
	})
	calls, status := 0, 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected mutation %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/metadata" {
			_, _ = w.Write([]byte(`{"success":true,"data":{"gateway_mode":"direct","service":"regional-gateway"}}`))
			return
		}
		calls++
		if r.URL.Path != "/api/v1/sandboxes/sandbox/sessions/conditional/lookup" || r.URL.Query().Get("runtime_generation") != "7" || r.Header.Get("Idempotency-Key") != "original-key" {
			t.Errorf("unexpected query %s", r.URL.String())
		}
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(status)
		if status != 200 {
			_, _ = w.Write([]byte(`{"success":false,"error":{"code":"session_not_found","message":"missing"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"success":true,"data":{"id":"session","spec":{"command":["true"]},"spec_version":1,"phase":"failed","runtime_generation":7,"restart_count":0,"cursor":{"earliest":0,"latest":0},"created_at":"2026-10-15T00:00:00Z","updated_at":"2026-10-15T00:00:00Z","last_activity_at":"2026-10-15T00:00:00Z"}}`))
	}))
	defer server.Close()
	t.Setenv(config.EnvBaseURL, server.URL)
	t.Setenv(config.EnvToken, "fixture")
	cmd := sandboxSessionFindConditionalCmd
	cmd.SetContext(context.Background())
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.RunE(cmd, []string{"sandbox", "original-key"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"phase": "failed"`) {
		t.Fatalf("invented running receipt: %s", out.String())
	}
	out.Reset()
	status = 404
	if err := cmd.RunE(cmd, []string{"sandbox", "original-key"}); err == nil {
		t.Fatal("missing receipt accepted")
	}
	if out.Len() != 0 || calls != 2 {
		t.Fatalf("unexpected output/retry: %s, %d", out.String(), calls)
	}
	sessionLookupGeneration = 0
	if err := cmd.RunE(cmd, []string{"sandbox", "original-key"}); err == nil {
		t.Fatal("generation required")
	}
	if calls != 2 {
		t.Fatal("invalid input reached network")
	}
}
