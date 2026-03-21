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
		if len(egress.AllowedDomains) != 1 || egress.AllowedDomains[0] != "github.com" {
			t.Fatalf("allowedDomains = %#v, want github.com", egress.AllowedDomains)
		}
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
		if len(egress.CredentialRules) != 1 {
			t.Fatalf("credentialRules count = %d, want 1", len(egress.CredentialRules))
		}
		if len(policy.CredentialBindings) != 1 {
			t.Fatalf("credentialBindings count = %d, want 1", len(policy.CredentialBindings))
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
	if len(egress.CredentialRules) != 1 {
		t.Fatalf("credentialRules count = %d, want 1", len(egress.CredentialRules))
	}
	if len(policy.CredentialBindings) != 1 {
		t.Fatalf("credentialBindings count = %d, want 1", len(policy.CredentialBindings))
	}
}
