package commands

import (
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestParsePortSpec(t *testing.T) {
	t.Run("single port with protocol", func(t *testing.T) {
		spec, err := parsePortSpec("443/tcp")
		if err != nil {
			t.Fatalf("parsePortSpec() error = %v", err)
		}
		if spec.Port != 443 {
			t.Fatalf("port = %d, want 443", spec.Port)
		}
		proto, ok := spec.Protocol.Get()
		if !ok || proto != "tcp" {
			t.Fatalf("protocol = %q, want tcp", proto)
		}
		if _, ok := spec.EndPort.Get(); ok {
			t.Fatal("endPort set unexpectedly")
		}
	})

	t.Run("port range", func(t *testing.T) {
		spec, err := parsePortSpec("8000-8010/udp")
		if err != nil {
			t.Fatalf("parsePortSpec() error = %v", err)
		}
		if spec.Port != 8000 {
			t.Fatalf("port = %d, want 8000", spec.Port)
		}
		endPort, ok := spec.EndPort.Get()
		if !ok || endPort != 8010 {
			t.Fatalf("endPort = %d, want 8010", endPort)
		}
		proto, ok := spec.Protocol.Get()
		if !ok || proto != "udp" {
			t.Fatalf("protocol = %q, want udp", proto)
		}
	})

	t.Run("invalid protocol", func(t *testing.T) {
		if _, err := parsePortSpec("443/http"); err == nil {
			t.Fatal("parsePortSpec() error = nil, want error")
		}
	})

	t.Run("descending range", func(t *testing.T) {
		if _, err := parsePortSpec("9000-8000"); err == nil {
			t.Fatal("parsePortSpec() error = nil, want error")
		}
	})
}

func TestBuildNetworkPolicyFromUpdateOptions(t *testing.T) {
	t.Run("legacy flags with ports", func(t *testing.T) {
		policy, err := buildNetworkPolicyFromUpdateOptions(networkUpdateOptions{
			Mode:           "block-all",
			AllowedDomains: []string{"github.com"},
			AllowedPorts:   []string{"443/tcp"},
		})
		if err != nil {
			t.Fatalf("buildNetworkPolicyFromUpdateOptions() error = %v", err)
		}
		if policy.Mode != apispec.SandboxNetworkPolicyModeBlockAll {
			t.Fatalf("mode = %q, want %q", policy.Mode, apispec.SandboxNetworkPolicyModeBlockAll)
		}
		egress, ok := policy.Egress.Get()
		if !ok {
			t.Fatal("egress not set")
		}
		//nolint:staticcheck // Exercise legacy compatibility fields that the CLI still accepts.
		if len(egress.AllowedDomains) != 1 || egress.AllowedDomains[0] != "github.com" {
			t.Fatalf("allowedDomains = %#v, want github.com", egress.AllowedDomains)
		}
		//nolint:staticcheck // Exercise legacy compatibility fields that the CLI still accepts.
		if len(egress.AllowedPorts) != 1 || egress.AllowedPorts[0].Port != 443 {
			t.Fatalf("allowedPorts = %#v, want single 443", egress.AllowedPorts)
		}
	})

	t.Run("structured rule flags", func(t *testing.T) {
		policy, err := buildNetworkPolicyFromUpdateOptions(networkUpdateOptions{
			Mode: "block-all",
			TrafficRules: []string{
				`{"name":"allow-ssh","action":"allow","appProtocols":["ssh"],"ports":[{"port":22,"protocol":"tcp"}]}`,
			},
			ProtocolRules: []string{
				`{"name":"docs-mcp","protocol":"mcp","domains":["mcp.example.com"],"ports":[{"port":443,"protocol":"tcp"}],"tlsMode":"terminate-reoriginate","httpMatch":{"methods":["POST"],"paths":["/mcp"]},"mcp":{"tools":{"allowed":["read_file"],"denied":["run_command"]}}}`,
				`{"name":"api-http-readonly","protocol":"http","domains":["api.example.com"],"ports":[{"port":8080,"protocol":"tcp"}],"http":{"methods":{"allowed":["GET","HEAD"],"denied":["POST"]},"paths":{"allowed":["/healthz"],"allowedPrefixes":["/api/"],"deniedPrefixes":["/admin/"]}}}`,
			},
			CredentialBinds: []string{
				`{"ref":"gh-token","sourceRef":"github-source","projection":{"type":"http_headers","httpHeaders":{"headers":[{"name":"Authorization","valueTemplate":"Bearer {{token}}"}]}}}`,
			},
			CredentialRules: []string{
				`{"name":"github-auth","credentialRef":"gh-token","protocol":"https","domains":["api.github.com"],"ports":[{"port":443,"protocol":"tcp"}]}`,
			},
		})
		if err != nil {
			t.Fatalf("buildNetworkPolicyFromUpdateOptions() error = %v", err)
		}
		egress, ok := policy.Egress.Get()
		if !ok {
			t.Fatal("egress not set")
		}
		if len(egress.TrafficRules) != 1 {
			t.Fatalf("trafficRules count = %d, want 1", len(egress.TrafficRules))
		}
		if len(egress.ProtocolRules) != 2 {
			t.Fatalf("protocolRules count = %d, want 2", len(egress.ProtocolRules))
		}
		if egress.ProtocolRules[0].Protocol != apispec.ProtocolRuleProtocolMcp {
			t.Fatalf("protocol rule protocol = %q, want mcp", egress.ProtocolRules[0].Protocol)
		}
		mcp, ok := egress.ProtocolRules[0].Mcp.Get()
		if !ok {
			t.Fatal("mcp policy not set")
		}
		tools, ok := mcp.Tools.Get()
		if !ok || len(tools.Allowed) != 1 || tools.Allowed[0] != "read_file" {
			t.Fatalf("mcp tools = %#v, want read_file allowed", mcp.Tools)
		}
		if egress.ProtocolRules[1].Protocol != apispec.ProtocolRuleProtocolHTTP {
			t.Fatalf("protocol rule protocol = %q, want http", egress.ProtocolRules[1].Protocol)
		}
		httpRule, ok := egress.ProtocolRules[1].HTTP.Get()
		if !ok {
			t.Fatal("http policy not set")
		}
		methods, ok := httpRule.Methods.Get()
		if !ok || len(methods.Allowed) != 2 || methods.Allowed[0] != "GET" || methods.Allowed[1] != "HEAD" {
			t.Fatalf("http methods = %#v, want GET and HEAD allowed", httpRule.Methods)
		}
		paths, ok := httpRule.Paths.Get()
		if !ok || len(paths.DeniedPrefixes) != 1 || paths.DeniedPrefixes[0] != "/admin/" {
			t.Fatalf("http paths = %#v, want /admin/ denied prefix", httpRule.Paths)
		}
		if len(egress.CredentialRules) != 1 {
			t.Fatalf("credentialRules count = %d, want 1", len(egress.CredentialRules))
		}
		if len(policy.CredentialBindings) != 1 {
			t.Fatalf("credentialBindings count = %d, want 1", len(policy.CredentialBindings))
		}
	})

	t.Run("egress proxy flags create username password binding", func(t *testing.T) {
		policy, err := buildNetworkPolicyFromUpdateOptions(networkUpdateOptions{
			Mode:            "block-all",
			AllowedDomains:  []string{"api.internal.example.com"},
			AllowedPorts:    []string{"443/tcp"},
			Proxy:           "socks5://proxy.example.com:1080",
			ProxyCredRef:    "corp-proxy",
			ProxyCredSource: "corp-proxy-source",
		})
		if err != nil {
			t.Fatalf("buildNetworkPolicyFromUpdateOptions() error = %v", err)
		}
		egress, ok := policy.Egress.Get()
		if !ok {
			t.Fatal("egress not set")
		}
		proxy, ok := egress.Proxy.Get()
		if !ok {
			t.Fatal("proxy not set")
		}
		if proxy.Type != apispec.EgressProxyTypeSocks5 {
			t.Fatalf("proxy type = %q, want socks5", proxy.Type)
		}
		if proxy.Address != "proxy.example.com:1080" {
			t.Fatalf("proxy address = %q, want proxy.example.com:1080", proxy.Address)
		}
		credentialRef, ok := proxy.CredentialRef.Get()
		if !ok || credentialRef != "corp-proxy" {
			t.Fatalf("proxy credentialRef = %q, want corp-proxy", credentialRef)
		}
		if len(policy.CredentialBindings) != 1 {
			t.Fatalf("credentialBindings count = %d, want 1", len(policy.CredentialBindings))
		}
		binding := policy.CredentialBindings[0]
		if binding.Ref != "corp-proxy" || binding.SourceRef != "corp-proxy-source" {
			t.Fatalf("credential binding = %#v, want corp-proxy/corp-proxy-source", binding)
		}
		if binding.Projection.Type != apispec.CredentialProjectionTypeUsernamePassword || binding.Projection.UsernamePassword == nil {
			t.Fatalf("credential binding projection = %#v, want username_password", binding.Projection)
		}
	})

	t.Run("proxy credential source defaults ref", func(t *testing.T) {
		policy, err := buildNetworkPolicyFromUpdateOptions(networkUpdateOptions{
			Mode:            "block-all",
			AllowedCidrs:    []string{"10.0.0.0/8"},
			Proxy:           "proxy.example.com:1080",
			ProxyCredSource: "corp-proxy-source",
		})
		if err != nil {
			t.Fatalf("buildNetworkPolicyFromUpdateOptions() error = %v", err)
		}
		egress, ok := policy.Egress.Get()
		if !ok {
			t.Fatal("egress not set")
		}
		proxy, ok := egress.Proxy.Get()
		if !ok {
			t.Fatal("proxy not set")
		}
		credentialRef, ok := proxy.CredentialRef.Get()
		if !ok || credentialRef != "egress-proxy" {
			t.Fatalf("proxy credentialRef = %q, want egress-proxy", credentialRef)
		}
		if len(policy.CredentialBindings) != 1 || policy.CredentialBindings[0].Ref != "egress-proxy" {
			t.Fatalf("credentialBindings = %#v, want egress-proxy binding", policy.CredentialBindings)
		}
	})

	t.Run("policy file is exclusive", func(t *testing.T) {
		_, err := buildNetworkPolicyFromUpdateOptions(networkUpdateOptions{
			PolicyFile:     "network.yaml",
			Mode:           "allow-all",
			AllowedDomains: []string{"github.com"},
		})
		if err == nil {
			t.Fatal("buildNetworkPolicyFromUpdateOptions() error = nil, want error")
		}
	})

	t.Run("traffic rule cannot mix with legacy flags", func(t *testing.T) {
		_, err := buildNetworkPolicyFromUpdateOptions(networkUpdateOptions{
			Mode:          "allow-all",
			DeniedDomains: []string{"facebook.com"},
			TrafficRules:  []string{`{"action":"deny","domains":["example.com"]}`},
		})
		if err == nil {
			t.Fatal("buildNetworkPolicyFromUpdateOptions() error = nil, want error")
		}
	})

	t.Run("proxy credential flags require proxy", func(t *testing.T) {
		_, err := buildNetworkPolicyFromUpdateOptions(networkUpdateOptions{
			Mode:            "block-all",
			ProxyCredSource: "corp-proxy-source",
		})
		if err == nil {
			t.Fatal("buildNetworkPolicyFromUpdateOptions() error = nil, want error")
		}
	})

	t.Run("proxy rejects inline credentials", func(t *testing.T) {
		_, err := buildNetworkPolicyFromUpdateOptions(networkUpdateOptions{
			Mode:  "block-all",
			Proxy: "socks5://proxy-user:proxy-password@proxy.example.com:1080",
		})
		if err == nil {
			t.Fatal("buildNetworkPolicyFromUpdateOptions() error = nil, want error")
		}
	})
}

func TestParseNetworkPolicyUpdateFile(t *testing.T) {
	policy, err := parseNetworkPolicyUpdateFile([]byte(`
mode: block-all
egress:
  trafficRules:
    - name: allow-ssh
      action: allow
      appProtocols: [ssh]
      ports:
        - port: 22
          protocol: tcp
  protocolRules:
    - name: docs-mcp
      protocol: mcp
      domains: [mcp.example.com]
      ports:
        - port: 443
          protocol: tcp
      tlsMode: terminate-reoriginate
      httpMatch:
        methods: [POST]
        paths: [/mcp]
      mcp:
        tools:
          allowed: [read_file]
          denied: [run_command]
    - name: api-http-readonly
      protocol: http
      domains: [api.example.com]
      ports:
        - port: 8080
          protocol: tcp
      http:
        methods:
          allowed: [GET, HEAD]
          denied: [POST]
        paths:
          allowed: [/healthz]
          allowedPrefixes: [/api/]
          deniedPrefixes: [/admin/]
  credentialRules:
    - name: github-auth
      credentialRef: gh-token
      protocol: https
      domains: [api.github.com]
      ports:
        - port: 443
          protocol: tcp
credentialBindings:
  - ref: gh-token
    sourceRef: github-source
    projection:
      type: http_headers
      httpHeaders:
        headers:
          - name: Authorization
            valueTemplate: "Bearer {{token}}"
`))
	if err != nil {
		t.Fatalf("parseNetworkPolicyUpdateFile() error = %v", err)
	}
	if policy.Mode != apispec.SandboxNetworkPolicyModeBlockAll {
		t.Fatalf("mode = %q, want %q", policy.Mode, apispec.SandboxNetworkPolicyModeBlockAll)
	}
	egress, ok := policy.Egress.Get()
	if !ok {
		t.Fatal("egress not set")
	}
	if len(egress.TrafficRules) != 1 {
		t.Fatalf("trafficRules count = %d, want 1", len(egress.TrafficRules))
	}
	if len(egress.ProtocolRules) != 2 {
		t.Fatalf("protocolRules count = %d, want 2", len(egress.ProtocolRules))
	}
	if egress.ProtocolRules[1].Protocol != apispec.ProtocolRuleProtocolHTTP {
		t.Fatalf("protocol rule protocol = %q, want http", egress.ProtocolRules[1].Protocol)
	}
	httpRule, ok := egress.ProtocolRules[1].HTTP.Get()
	if !ok {
		t.Fatal("http policy not set")
	}
	paths, ok := httpRule.Paths.Get()
	if !ok || len(paths.AllowedPrefixes) != 1 || paths.AllowedPrefixes[0] != "/api/" {
		t.Fatalf("http paths = %#v, want /api/ allowed prefix", httpRule.Paths)
	}
	if len(egress.CredentialRules) != 1 {
		t.Fatalf("credentialRules count = %d, want 1", len(egress.CredentialRules))
	}
	if len(policy.CredentialBindings) != 1 {
		t.Fatalf("credentialBindings count = %d, want 1", len(policy.CredentialBindings))
	}
}
