package commands

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestExternalMountCreateConfig(t *testing.T) {
	for _, content := range []string{
		"external_mounts:\n  - source_ref: customer-data\n    mount_path: /data\n    prefix: runs/\n",
		`{"template":"default","config":{"external_mounts":[{"source_ref":"customer-data","mount_path":"/data","prefix":"runs/"}]}}`,
	} {
		resetSandboxFlagsForTest()
		sandboxTemplate = "default"
		sandboxMemory = "1Gi"
		sandboxConfigFile = writeTempFile(t, content)
		request, err := buildSandboxCreateRequest()
		if err != nil {
			t.Fatal(err)
		}
		config, ok := request.Config.Get()
		if !ok || len(config.ExternalMounts) != 1 {
			t.Fatal("external binding lost")
		}
		mount := config.ExternalMounts[0]
		if mount.SourceRef != "customer-data" || mount.MountPath != "/data" || mount.Prefix.Or("") != "runs/" || mount.Access.Or("") != apispec.ExternalMountBindingAccessReadOnly {
			t.Fatal("binding changed or read-only default lost")
		}
		resources, ok := config.Resources.Get()
		if !ok || resources.Memory.Or("") != "1Gi" {
			t.Fatal("memory override lost")
		}
	}
	resetSandboxFlagsForTest()
}

func TestExternalMountInlineAuthorityRejected(t *testing.T) {
	for _, field := range []string{"secret_key", "bucket", "worker_container_id"} {
		content := `{"external_mounts":[{"source_ref":"customer-data","mount_path":"/data","` + field + `":"must-not-be-echoed"}]}`
		_, err := readSandboxConfigFile(writeTempFile(t, content))
		if err == nil {
			t.Fatal("inline authority accepted")
		}
		if strings.Contains(err.Error(), "must-not-be-echoed") {
			t.Fatal("inline secret echoed in error")
		}
	}
}

func TestExternalMountUpdateRejected(t *testing.T) {
	for _, content := range []string{"external_mounts: []", "external_mounts: null", "external_mounts:\n  - source_ref: other\n    mount_path: /data"} {
		_, err := readSandboxUpdateConfigFile(writeTempFile(t, content))
		if err == nil || !strings.Contains(err.Error(), "external_mounts") {
			t.Fatalf("expected immutable binding rejection: %v", err)
		}
	}
}

func TestStaticS3CredentialSourceConfig(t *testing.T) {
	request, err := parseCredentialSourceWriteRequest([]byte(`
name: customer-data
resolverKind: static_s3
spec:
  staticS3:
    bucket: customer-bucket
    region: us-east-1
    prefix: agents/
    endpointUrl: https://s3.example.test
    allowWrite: true
    accessKey: test-access
    secretKey: test-secret
    sessionToken: test-session
`))
	if err != nil {
		t.Fatal(err)
	}
	spec, ok := request.Spec.StaticS3.Get()
	if !ok || request.ResolverKind != apispec.CredentialSourceResolverKindStaticS3 {
		t.Fatal("S3 source not selected")
	}
	if spec.AccessKey != "test-access" || spec.SecretKey != "test-secret" || spec.SessionToken.Or("") != "test-session" || !spec.AllowWrite.Or(false) || spec.EndpointUrl.Or("") != "https://s3.example.test" || spec.Prefix.Or("") != "agents/" {
		t.Fatal("S3 source fields lost")
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["resolverKind"] != "static_s3" || wire["spec"].(map[string]any)["staticS3"] == nil {
		t.Fatal("S3 source wire shape incorrect")
	}
	for _, content := range []string{
		"name: source\nresolverKind: static_s3\nspec: {}",
		"name: source\nresolverKind: static_s3\nspec:\n  staticS3:\n    bucket: bucket\n    region: us-east-1\n    accessKey: test-access\n",
	} {
		if _, err := parseCredentialSourceWriteRequest([]byte(content)); err == nil {
			t.Fatal("incomplete source accepted")
		}
	}
}
