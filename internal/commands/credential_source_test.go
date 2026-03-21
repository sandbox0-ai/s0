package commands

import (
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestParseCredentialSourceWriteRequest(t *testing.T) {
	t.Run("static headers", func(t *testing.T) {
		req, err := parseCredentialSourceWriteRequest([]byte(`
name: source-a
resolverKind: static_headers
spec:
  staticHeaders:
    values:
      Authorization: Bearer token
      X-Trace: trace
`))
		if err != nil {
			t.Fatalf("parseCredentialSourceWriteRequest() error = %v", err)
		}
		if req.Name != "source-a" {
			t.Fatalf("name = %q, want source-a", req.Name)
		}
		if req.ResolverKind != apispec.CredentialSourceResolverKindStaticHeaders {
			t.Fatalf("resolverKind = %q, want %q", req.ResolverKind, apispec.CredentialSourceResolverKindStaticHeaders)
		}
		spec, ok := req.Spec.StaticHeaders.Get()
		if !ok {
			t.Fatal("staticHeaders not set")
		}
		values, ok := spec.Values.Get()
		if !ok {
			t.Fatal("staticHeaders.values not set")
		}
		if values["Authorization"] != "Bearer token" {
			t.Fatalf("Authorization = %q, want Bearer token", values["Authorization"])
		}
	})

	t.Run("static tls client certificate", func(t *testing.T) {
		req, err := parseCredentialSourceWriteRequest([]byte(`
name: mtls-source
resolverKind: static_tls_client_certificate
spec:
  staticTLSClientCertificate:
    certificatePem: cert
    privateKeyPem: key
    caPem: ca
`))
		if err != nil {
			t.Fatalf("parseCredentialSourceWriteRequest() error = %v", err)
		}
		spec, ok := req.Spec.StaticTLSClientCertificate.Get()
		if !ok {
			t.Fatal("staticTLSClientCertificate not set")
		}
		if spec.CertificatePem != "cert" {
			t.Fatalf("certificatePem = %q, want cert", spec.CertificatePem)
		}
		if spec.PrivateKeyPem != "key" {
			t.Fatalf("privateKeyPem = %q, want key", spec.PrivateKeyPem)
		}
		caPem, ok := spec.CaPem.Get()
		if !ok || caPem != "ca" {
			t.Fatalf("caPem = %q, want ca", caPem)
		}
	})

	t.Run("static username password", func(t *testing.T) {
		req, err := parseCredentialSourceWriteRequest([]byte(`
name: basic-auth
resolverKind: static_username_password
spec:
  staticUsernamePassword:
    username: alice
    password: secret
`))
		if err != nil {
			t.Fatalf("parseCredentialSourceWriteRequest() error = %v", err)
		}
		spec, ok := req.Spec.StaticUsernamePassword.Get()
		if !ok {
			t.Fatal("staticUsernamePassword not set")
		}
		if spec.Username != "alice" {
			t.Fatalf("username = %q, want alice", spec.Username)
		}
		if spec.Password != "secret" {
			t.Fatalf("password = %q, want secret", spec.Password)
		}
	})

	t.Run("invalid resolver kind", func(t *testing.T) {
		_, err := parseCredentialSourceWriteRequest([]byte(`
name: source-a
resolverKind: vault
spec: {}
`))
		if err == nil {
			t.Fatal("parseCredentialSourceWriteRequest() error = nil, want error")
		}
	})

	t.Run("missing resolver spec", func(t *testing.T) {
		_, err := parseCredentialSourceWriteRequest([]byte(`
name: source-a
resolverKind: static_username_password
spec: {}
`))
		if err == nil {
			t.Fatal("parseCredentialSourceWriteRequest() error = nil, want error")
		}
	})
}
