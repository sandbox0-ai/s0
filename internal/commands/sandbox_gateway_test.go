package commands

import (
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestParsePublicGatewayPolicy(t *testing.T) {
	policy, err := parsePublicGatewayPolicy([]byte(`
enabled: true
routes:
  - id: app
    port: 8080
    path_prefix: /api
    methods: [GET, POST]
    auth:
      mode: bearer
      bearer_token_sha256: abc123
    rate_limit:
      rps: 10
      burst: 20
    timeout_seconds: 30
    resume: true
`))
	if err != nil {
		t.Fatalf("parsePublicGatewayPolicy() error = %v", err)
	}

	if !policy.Enabled {
		t.Fatal("enabled = false, want true")
	}
	if len(policy.Routes) != 1 {
		t.Fatalf("routes count = %d, want 1", len(policy.Routes))
	}
	route := policy.Routes[0]
	if route.ID != "app" || route.Port != 8080 {
		t.Fatalf("route = %#v, want id app and port 8080", route)
	}
	if route.PathPrefix.Or("") != "/api" {
		t.Fatalf("path_prefix = %q, want /api", route.PathPrefix.Or(""))
	}
	auth, ok := route.Auth.Get()
	if !ok {
		t.Fatal("auth not set")
	}
	if auth.Mode != apispec.PublicGatewayAuthModeBearer {
		t.Fatalf("auth mode = %q, want bearer", auth.Mode)
	}
}

func TestReadPublicGatewayPolicyFileRequiresPath(t *testing.T) {
	_, err := readPublicGatewayPolicyFile("")
	if err == nil {
		t.Fatal("readPublicGatewayPolicyFile() error = nil, want error")
	}
}
