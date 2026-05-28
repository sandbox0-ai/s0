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
	if auth.Mode != apispec.SandboxAppServiceRouteAuthModeBearer {
		t.Fatalf("auth mode = %q, want bearer", auth.Mode)
	}
}

func TestParseSandboxServicesSupportsFunctionRuntime(t *testing.T) {
	services, err := parseSandboxServices([]byte(`
services:
  - id: handler
    port: 49983
    runtime:
      type: function
      function:
        runtime: python
        handler: handler
        source:
          type: inline
          filename: main.py
          code: |
            def handler(request):
                return {"status": 200, "body": "ok"}
    ingress:
      public: true
      routes:
        - id: handler
          path_prefix: /
          resume: true
`))
	if err != nil {
		t.Fatalf("parseSandboxServices() error = %v", err)
	}

	if len(services.Services) != 1 {
		t.Fatalf("services count = %d, want 1", len(services.Services))
	}
	service := services.Services[0]
	if service.Port != 49983 {
		t.Fatalf("port = %d, want 49983", service.Port)
	}
	runtime, ok := service.Runtime.Get()
	if !ok {
		t.Fatal("runtime not set")
	}
	if runtime.Type != apispec.SandboxAppServiceRuntimeTypeFunction {
		t.Fatalf("runtime type = %q, want function", runtime.Type)
	}
	fn, ok := runtime.Function.Get()
	if !ok {
		t.Fatal("function config not set")
	}
	if fn.Runtime != apispec.SandboxFunctionRuntimePython {
		t.Fatalf("function runtime = %q, want python", fn.Runtime)
	}
	if fn.Source.Type != apispec.SandboxFunctionSourceTypeInline {
		t.Fatalf("source type = %q, want inline", fn.Source.Type)
	}
	if fn.Source.Code == "" {
		t.Fatal("source code is empty")
	}
}

func TestReadSandboxServicesFileRequiresPath(t *testing.T) {
	_, err := readSandboxServicesFile("")
	if err == nil {
		t.Fatal("readSandboxServicesFile() error = nil, want error")
	}
}

func TestDeleteSandboxServiceByID(t *testing.T) {
	runtime := apispec.SandboxAppServiceRuntime{
		Type: apispec.SandboxAppServiceRuntimeTypeFunction,
		Function: apispec.NewOptSandboxFunction(apispec.SandboxFunction{
			Runtime: apispec.SandboxFunctionRuntimePython,
			Handler: apispec.NewOptString("handler"),
			Source: apispec.SandboxFunctionSource{
				Type: apispec.SandboxFunctionSourceTypeInline,
				Code: "def handler(request):\n    return {'status': 200}\n",
			},
		}),
	}
	remaining, err := deleteSandboxServiceByID([]apispec.SandboxAppServiceView{
		{
			ID:      "api",
			Port:    8080,
			Ingress: apispec.SandboxAppServiceIngress{Public: true},
		},
		{
			ID:      "hello",
			Port:    49983,
			Runtime: apispec.NewOptSandboxAppServiceRuntime(runtime),
			Ingress: apispec.SandboxAppServiceIngress{
				Public: true,
				Routes: []apispec.SandboxAppServiceRoute{{
					ID:         "hello",
					PathPrefix: apispec.NewOptString("/hello"),
					Resume:     true,
				}},
			},
			Publishable: true,
			PublicURL:   apispec.NewOptString("https://example.sandbox0.app"),
		},
	}, "hello")
	if err != nil {
		t.Fatalf("deleteSandboxServiceByID() error = %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("remaining count = %d, want 1", len(remaining))
	}
	if remaining[0].ID != "api" || remaining[0].Port != 8080 {
		t.Fatalf("remaining service = %#v, want api service", remaining[0])
	}
}

func TestDeleteSandboxServiceByIDNotFound(t *testing.T) {
	_, err := deleteSandboxServiceByID([]apispec.SandboxAppServiceView{{ID: "api"}}, "missing")
	if err == nil {
		t.Fatal("deleteSandboxServiceByID() error = nil, want error")
	}
}
