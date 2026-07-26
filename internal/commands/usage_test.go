package commands

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sandbox0-ai/s0/internal/config"
	"github.com/spf13/cobra"
)

func TestUsageCommandUsesHomeRegionRouting(t *testing.T) {
	root := &cobra.Command{Use: "s0"}
	usage := newUsageCommand()
	root.AddCommand(usage)

	list, _, err := usage.Find([]string{"list"})
	if err != nil {
		t.Fatalf("find usage list: %v", err)
	}
	if got := commandRouteScope(list); got != "home-region" {
		t.Fatalf("commandRouteScope(usage list) = %q, want home-region", got)
	}
}

func TestUsageListRejectsInvalidLimitBeforeCreatingClient(t *testing.T) {
	for _, limit := range []string{"0", "1001"} {
		cmd := newUsageListCommand()
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
		cmd.SetArgs([]string{"--limit", limit})

		err := cmd.Execute()
		if err == nil {
			t.Fatalf("usage list --limit %s unexpectedly succeeded", limit)
		}
		if !strings.Contains(err.Error(), "limit must be between 1 and 1000") {
			t.Fatalf("error = %q, want limit validation", err)
		}
	}
}

func TestUsageListCommandUsesSDKFilters(t *testing.T) {
	config.SetConfigFile("")
	config.SetProfile("")
	config.SetAPIURL("")
	config.SetToken("")
	cfgFormat = "json"
	t.Cleanup(func() {
		cfgFormat = "table"
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/metadata":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"data":{"gateway_mode":"direct","service":"regional-gateway"}}`))
		case "/api/v1/usage/windows":
			if got := r.Header.Get("Authorization"); got != "Bearer token-1" {
				t.Fatalf("Authorization = %q, want Bearer token-1", got)
			}
			query := r.URL.Query()
			if got := query.Get("cursor"); got != "page-1" {
				t.Fatalf("cursor = %q, want page-1", got)
			}
			if got := query.Get("limit"); got != "250" {
				t.Fatalf("limit = %q, want 250", got)
			}
			if got := query.Get("window_type"); got != "sandbox.runtime_mib_milliseconds" {
				t.Fatalf("window_type = %q, want sandbox.runtime_mib_milliseconds", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"data":{"windows":[],"next_cursor":"page-2"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv(config.EnvBaseURL, server.URL)
	t.Setenv(config.EnvToken, "token-1")

	cmd := newUsageListCommand()
	cmd.SetContext(context.Background())
	if err := cmd.Flags().Parse([]string{
		"--cursor", "page-1",
		"--limit", "250",
		"--window-type", "sandbox.runtime_mib_milliseconds",
	}); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE() error = %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, `"next_cursor": "page-2"`) {
		t.Fatalf("output missing next cursor:\n%s", got)
	}
}
