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

func TestBuildCreateRegionRequest(t *testing.T) {
	var opts adminRegionOptions
	cmd := &cobra.Command{Use: "create"}
	opts.addCreateFlags(cmd)

	if err := cmd.Flags().Parse([]string{
		"--id", " aws/us-east-1 ",
		"--display-name", " US East 1 ",
		"--regional-gateway-url", " https://use1.example.com ",
		"--metering-export-url", " https://metering.use1.example.com ",
		"--enabled=false",
	}); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	req, err := buildCreateRegionRequest(cmd, opts)
	if err != nil {
		t.Fatalf("buildCreateRegionRequest() error = %v", err)
	}

	if req.ID != "aws/us-east-1" {
		t.Fatalf("ID = %q, want aws/us-east-1", req.ID)
	}
	if req.RegionalGatewayURL != "https://use1.example.com" {
		t.Fatalf("RegionalGatewayURL = %q, want https://use1.example.com", req.RegionalGatewayURL)
	}
	if displayName, ok := req.DisplayName.Get(); !ok || displayName != "US East 1" {
		t.Fatalf("DisplayName = %q, want US East 1", displayName)
	}
	if meteringURL, ok := req.MeteringExportURL.Get(); !ok || meteringURL != "https://metering.use1.example.com" {
		t.Fatalf("MeteringExportURL = %q, want https://metering.use1.example.com", meteringURL)
	}
	if enabled, ok := req.Enabled.Get(); !ok || enabled {
		t.Fatalf("Enabled = %v, want false", enabled)
	}
}

func TestBuildCreateRegionRequestRequiresGatewayURL(t *testing.T) {
	var opts adminRegionOptions
	cmd := &cobra.Command{Use: "create"}
	opts.addCreateFlags(cmd)

	if err := cmd.Flags().Parse([]string{"--id", "aws/us-east-1"}); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	_, err := buildCreateRegionRequest(cmd, opts)
	if err == nil || !strings.Contains(err.Error(), "--regional-gateway-url is required") {
		t.Fatalf("expected missing gateway URL error, got %v", err)
	}
}

func TestBuildUpdateRegionRequestSupportsClearingMeteringExportURLAndEnabledFalse(t *testing.T) {
	var opts adminRegionOptions
	cmd := &cobra.Command{Use: "update"}
	opts.addUpdateFlags(cmd)

	if err := cmd.Flags().Parse([]string{
		"--metering-export-url=",
		"--enabled=false",
		"--display-name", " US East 1 ",
	}); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	req, err := buildUpdateRegionRequest(cmd, opts)
	if err != nil {
		t.Fatalf("buildUpdateRegionRequest() error = %v", err)
	}

	if displayName, ok := req.DisplayName.Get(); !ok || displayName != "US East 1" {
		t.Fatalf("DisplayName = %q, want US East 1", displayName)
	}
	if meteringURL, ok := req.MeteringExportURL.Get(); !ok || meteringURL != "" {
		t.Fatalf("MeteringExportURL = %q, want empty string", meteringURL)
	}
	if enabled, ok := req.Enabled.Get(); !ok || enabled {
		t.Fatalf("Enabled = %v, want false", enabled)
	}
}

func TestBuildUpdateRegionRequestRejectsBlankRegionalGatewayURL(t *testing.T) {
	var opts adminRegionOptions
	cmd := &cobra.Command{Use: "update"}
	opts.addUpdateFlags(cmd)

	if err := cmd.Flags().Parse([]string{"--edge-gateway-url", "   "}); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	_, err := buildUpdateRegionRequest(cmd, opts)
	if err == nil || !strings.Contains(err.Error(), "--regional-gateway-url is required") {
		t.Fatalf("expected blank gateway URL error, got %v", err)
	}
}

func TestAdminRegionListCommand(t *testing.T) {
	config.SetConfigFile("")
	config.SetProfile("")
	config.SetAPIURL("")
	config.SetToken("")
	cfgFormat = "table"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/metadata":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"data":{"gateway_mode":"global","service":"global-gateway"}}`))
		case "/regions":
			if got := r.Header.Get("Authorization"); got != "Bearer token-1" {
				t.Fatalf("Authorization = %q, want Bearer token-1", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"data":{"regions":[{"id":"aws/us-east-1","display_name":"US East 1","regional_gateway_url":"https://use1.example.com","metering_export_url":"https://metering.use1.example.com","enabled":true}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv(config.EnvBaseURL, server.URL)
	t.Setenv(config.EnvToken, "token-1")

	cmd := newAdminRegionListCommand()
	cmd.SetContext(context.Background())
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE() error = %v", err)
	}

	output := stdout.String()
	for _, want := range []string{
		"aws/us-east-1",
		"US East 1",
		"https://use1.example.com",
		"https://metering.use1.example.com",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}
