package commands

import (
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestParseSandboxServices(t *testing.T) {
	services, err := parseSandboxServices([]byte(`
services:
  - id: app
    port: 8080
    ingress:
      public: true
      routes:
        - id: app
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
		t.Fatalf("parseSandboxServices() error = %v", err)
	}

	if len(services.Services) != 1 {
		t.Fatalf("services count = %d, want 1", len(services.Services))
	}
	service := services.Services[0]
	if service.ID != "app" || service.Port != 8080 {
		t.Fatalf("service = %#v, want id app and port 8080", service)
	}
	if !service.Ingress.Public || len(service.Ingress.Routes) != 1 {
		t.Fatalf("ingress = %#v, want one public route", service.Ingress)
	}
	route := service.Ingress.Routes[0]
	if route.ID != "app" {
		t.Fatalf("route = %#v, want id app", route)
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

func TestReadSandboxServicesFileRequiresPath(t *testing.T) {
	_, err := readSandboxServicesFile("")
	if err == nil {
		t.Fatal("readSandboxServicesFile() error = nil, want error")
	}
}
