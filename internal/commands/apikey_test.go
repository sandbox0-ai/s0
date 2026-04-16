package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestPrintCreatedAPIKeyRaw(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var out bytes.Buffer
		data := &apispec.CreateAPIKeyResponse{
			Key: apispec.NewOptString("s0_test_secret"),
		}

		if err := printCreatedAPIKeyRaw(&out, data); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got := out.String(); got != "s0_test_secret\n" {
			t.Fatalf("unexpected output %q", got)
		}
	})

	t.Run("missing key", func(t *testing.T) {
		var out bytes.Buffer
		data := &apispec.CreateAPIKeyResponse{}

		err := printCreatedAPIKeyRaw(&out, data)
		if err == nil || !strings.Contains(err.Error(), "not returned") {
			t.Fatalf("expected missing key error, got %v", err)
		}
	})

	t.Run("blank key", func(t *testing.T) {
		var out bytes.Buffer
		data := &apispec.CreateAPIKeyResponse{
			Key: apispec.NewOptString("   "),
		}

		err := printCreatedAPIKeyRaw(&out, data)
		if err == nil || !strings.Contains(err.Error(), "not returned") {
			t.Fatalf("expected blank key error, got %v", err)
		}
	})

	t.Run("nil data", func(t *testing.T) {
		var out bytes.Buffer
		err := printCreatedAPIKeyRaw(&out, nil)
		if err == nil || !strings.Contains(err.Error(), "missing API key data") {
			t.Fatalf("expected nil data error, got %v", err)
		}
	})
}

func TestNormalizeAPIKeyScope(t *testing.T) {
	tests := []struct {
		name    string
		scope   string
		want    string
		wantErr bool
	}{
		{name: "defaults to team", want: apiKeyScopeTeam},
		{name: "team", scope: "team", want: apiKeyScopeTeam},
		{name: "platform", scope: "platform", want: apiKeyScopePlatform},
		{name: "rejects unknown", scope: "system", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeAPIKeyScope(tt.scope)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeAPIKeyScope() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("scope = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateAPIKeyCreateOptions(t *testing.T) {
	tests := []struct {
		name    string
		scope   string
		roles   []string
		wantErr bool
	}{
		{name: "team requires role", scope: apiKeyScopeTeam, wantErr: true},
		{name: "team accepts roles", scope: apiKeyScopeTeam, roles: []string{"developer"}},
		{name: "platform forbids roles", scope: apiKeyScopePlatform, roles: []string{"admin"}, wantErr: true},
		{name: "platform accepts no roles", scope: apiKeyScopePlatform},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAPIKeyCreateOptions(tt.scope, tt.roles)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("validateAPIKeyCreateOptions() error = %v", err)
			}
		})
	}
}
