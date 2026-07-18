package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildTemplateCreateRequestPreservesTemplateEnvVars(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	specFile := filepath.Join(dir, "template.yaml")
	specYAML := `spec:
  displayName: "Template Env Vars Test"
  description: "Minimal template for verifying template env vars"
  mainContainer:
    image: cc-demo:test
    resources:
      memory: 256Mi
  envVars:
    PORT: "8081"
    WORKSPACE_DIR: /workspace
  network:
    mode: allow-all
  pool:
    minIdle: 0
    maxIdle: 1
`
	if err := os.WriteFile(specFile, []byte(specYAML), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	req, err := buildTemplateCreateRequest("cc-demo-kind-test", specFile)
	if err != nil {
		t.Fatalf("buildTemplateCreateRequest() error = %v", err)
	}

	if req.TemplateID != "cc-demo-kind-test" {
		t.Fatalf("TemplateID = %q, want cc-demo-kind-test", req.TemplateID)
	}
	displayName, ok := req.Spec.DisplayName.Get()
	if !ok || displayName != "Template Env Vars Test" {
		t.Fatalf("DisplayName = %q, want Template Env Vars Test", displayName)
	}
	envVars, ok := req.Spec.EnvVars.Get()
	if !ok {
		t.Fatal("Spec.EnvVars should be set")
	}
	if envVars["PORT"] != "8081" || envVars["WORKSPACE_DIR"] != "/workspace" {
		t.Fatalf("Spec.EnvVars = %+v, want PORT and WORKSPACE_DIR", envVars)
	}
}

func TestBuildTemplateFromSandboxCreateRequest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	specFile := filepath.Join(dir, "overrides.yaml")
	specYAML := `displayName: Python ready
tags:
  - python
pool:
  minIdle: 1
  maxIdle: 2
`
	if err := os.WriteFile(specFile, []byte(specYAML), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	req, err := buildTemplateFromSandboxCreateRequest("python-ready", "sb_source", specFile)
	if err != nil {
		t.Fatalf("buildTemplateFromSandboxCreateRequest() error = %v", err)
	}
	if req.TemplateID != "python-ready" || req.SandboxID != "sb_source" {
		t.Fatalf("request = %+v", req)
	}
	overrides, ok := req.SpecOverrides.Get()
	if !ok {
		t.Fatal("SpecOverrides should be set")
	}
	if overrides.DisplayName.Or("") != "Python ready" || len(overrides.Tags) != 1 {
		t.Fatalf("SpecOverrides = %+v", overrides)
	}
	pool, ok := overrides.Pool.Get()
	if !ok || pool.MinIdle != 1 || pool.MaxIdle != 2 {
		t.Fatalf("pool = %+v, set = %v", pool, ok)
	}
}

func TestBuildTemplateFromSandboxCreateRequestWithoutOverrides(t *testing.T) {
	t.Parallel()

	req, err := buildTemplateFromSandboxCreateRequest("python-ready", "sb_source", "")
	if err != nil {
		t.Fatalf("buildTemplateFromSandboxCreateRequest() error = %v", err)
	}
	if req.SpecOverrides.IsSet() {
		t.Fatal("SpecOverrides should not be set")
	}
}

func TestBuildTemplateFromSandboxCreateRequestRejectsUnsupportedOverrides(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name: "unsupported override",
			content: `mainContainer:
  image: python:3.13
`,
			wantErr: "mainContainer is not supported in an overrides file",
		},
		{
			name: "wrapped API request field",
			content: `spec_overrides:
  displayName: ignored
`,
			wantErr: "spec_overrides is not supported in an overrides file",
		},
		{
			name: "unsupported pool field",
			content: `pool:
  minIdle: 0
  maxIdle: 1
  warmup: 2
`,
			wantErr: "pool.warmup is not supported",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			specFile := filepath.Join(t.TempDir(), "overrides.yaml")
			if err := os.WriteFile(specFile, []byte(test.content), 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
			_, err := buildTemplateFromSandboxCreateRequest("python-ready", "sb_source", specFile)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestValidateTemplateCreateMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    templateCreateModeOptions
		wantErr string
	}{
		{name: "image mode", opts: templateCreateModeOptions{templateID: "image", specFile: "template.yaml"}},
		{name: "from sandbox", opts: templateCreateModeOptions{templateID: "ready", fromSandbox: "sb_source"}},
		{name: "from sandbox with overrides", opts: templateCreateModeOptions{templateID: "ready", fromSandbox: "sb_source", overridesFile: "overrides.yaml"}},
		{name: "from sandbox wait", opts: templateCreateModeOptions{templateID: "ready", fromSandbox: "sb_source", wait: true, waitTimeout: time.Minute, pollInterval: time.Second}},
		{name: "missing id", opts: templateCreateModeOptions{specFile: "template.yaml"}, wantErr: "--id is required"},
		{name: "missing image spec", opts: templateCreateModeOptions{templateID: "image"}, wantErr: "--spec-file is required"},
		{name: "overrides without source", opts: templateCreateModeOptions{templateID: "image", specFile: "template.yaml", overridesFile: "overrides.yaml"}, wantErr: "--overrides-file requires --from-sandbox"},
		{name: "spec with source", opts: templateCreateModeOptions{templateID: "ready", fromSandbox: "sb_source", specFile: "template.yaml"}, wantErr: "--spec-file cannot be used with --from-sandbox"},
		{name: "wait without source", opts: templateCreateModeOptions{templateID: "image", specFile: "template.yaml", wait: true}, wantErr: "require --from-sandbox"},
		{name: "timeout without wait", opts: templateCreateModeOptions{templateID: "ready", fromSandbox: "sb_source", waitTimeoutChanged: true}, wantErr: "--wait-timeout requires --wait"},
		{name: "invalid timeout", opts: templateCreateModeOptions{templateID: "ready", fromSandbox: "sb_source", wait: true, waitTimeout: 0, pollInterval: time.Second}, wantErr: "--wait-timeout must be greater than zero"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := validateTemplateCreateMode(test.opts)
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("validateTemplateCreateMode() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}

func TestBuildTemplateCreateRequestRejectsCPU(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	specFile := filepath.Join(dir, "template.yaml")
	specYAML := `spec:
  mainContainer:
    image: cc-demo:test
    resources:
      cpu: "250m"
      memory: 256Mi
`
	if err := os.WriteFile(specFile, []byte(specYAML), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := buildTemplateCreateRequest("cc-demo-kind-test", specFile)
	if err == nil {
		t.Fatal("buildTemplateCreateRequest() error = nil, want unsupported CPU error")
	}
	if !strings.Contains(err.Error(), "resources.cpu is not supported") {
		t.Fatalf("buildTemplateCreateRequest() error = %q, want unsupported CPU error", err)
	}
}

func TestBuildTemplateCreateRequestPreservesSystemTemplateFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	specFile := filepath.Join(dir, "dins-template.yaml")
	specYAML := `spec:
  displayName: Docker in Sandbox
  mainContainer:
    image: sandbox0ai/otemplates:default-v0.1.0
    imagePullPolicy: IfNotPresent
    resources:
      memory: 4Gi
      ephemeralStorage: 20Gi
    securityContext:
      privileged: true
      allowPrivilegeEscalation: true
  pod:
    emptyDirMounts:
      - mountPath: /var/lib/docker
        sizeLimit: 20Gi
  network:
    mode: allow-all
  pool:
    minIdle: 1
    maxIdle: 5
`
	if err := os.WriteFile(specFile, []byte(specYAML), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	req, err := buildTemplateCreateRequest("dins", specFile)
	if err != nil {
		t.Fatalf("buildTemplateCreateRequest() error = %v", err)
	}

	main, ok := req.Spec.MainContainer.Get()
	if !ok {
		t.Fatal("mainContainer should be set")
	}
	if main.Image != "sandbox0ai/otemplates:default-v0.1.0" {
		t.Fatalf("mainContainer.image = %q, want default image", main.Image)
	}
	if policy, ok := main.ImagePullPolicy.Get(); !ok || policy != "IfNotPresent" {
		t.Fatalf("mainContainer.imagePullPolicy = %q, want IfNotPresent", policy)
	}
	securityContext, ok := main.SecurityContext.Get()
	if !ok {
		t.Fatal("mainContainer.securityContext should be set")
	}
	privileged, ok := securityContext.Privileged.Get()
	if !ok || !privileged {
		t.Fatalf("securityContext.privileged = %v, want true", privileged)
	}
	allowPrivilegeEscalation, ok := securityContext.AllowPrivilegeEscalation.Get()
	if !ok || !allowPrivilegeEscalation {
		t.Fatalf("securityContext.allowPrivilegeEscalation = %v, want true", allowPrivilegeEscalation)
	}
	pod, ok := req.Spec.Pod.Get()
	if !ok {
		t.Fatal("pod should be set")
	}
	if len(pod.EmptyDirMounts) != 1 {
		t.Fatalf("len(pod.emptyDirMounts) = %d, want 1", len(pod.EmptyDirMounts))
	}
	if got := pod.EmptyDirMounts[0].MountPath; got != "/var/lib/docker" {
		t.Fatalf("pod.emptyDirMounts[0].mountPath = %q, want /var/lib/docker", got)
	}
	if got, ok := pod.EmptyDirMounts[0].SizeLimit.Get(); !ok || got != "20Gi" {
		t.Fatalf("pod.emptyDirMounts[0].sizeLimit = %q, want 20Gi", got)
	}
}
