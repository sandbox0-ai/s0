package commands

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseRunMounts(t *testing.T) {
	t.Parallel()

	got, err := parseRunMounts([]string{"snap_123:/workspace", "snap_456:/data"})
	if err != nil {
		t.Fatalf("parseRunMounts() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(parseRunMounts()) = %d, want 2", len(got))
	}
	if got[0].SnapshotID != "snap_123" || got[0].MountPath != "/workspace" {
		t.Fatalf("first mount = %+v, want snap_123:/workspace", got[0])
	}
}

func TestParseRunMountsRejectsRelativePath(t *testing.T) {
	t.Parallel()

	if _, err := parseRunMounts([]string{"snap_123:workspace"}); err == nil {
		t.Fatal("parseRunMounts() error = nil, want relative path error")
	}
}

func TestParseRunCommand(t *testing.T) {
	t.Parallel()

	got, err := parseRunCommand(`python -m http.server "8080"`, nil)
	if err != nil {
		t.Fatalf("parseRunCommand() error = %v", err)
	}
	want := []string{"python", "-m", "http.server", "8080"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseRunCommand() = %#v, want %#v", got, want)
	}
}

func TestReadRunDeploySpecFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "run.yaml")
	input := `name: api
slug: api
template: python
service:
  id: app
  port: 8080
  command: ["python", "-m", "http.server", "8080"]
  cwd: /workspace
  envVars:
    APP_ENV: production
  healthPath: /healthz
mounts:
  - snapshotID: snap_123
    mountPath: /workspace
scale:
  maxInstances: 4
  idleTimeoutSeconds: 120
activate: false
`
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	spec, err := readRunDeploySpecFile(path)
	if err != nil {
		t.Fatalf("readRunDeploySpecFile() error = %v", err)
	}
	if spec.Name != "api" || spec.Slug != "api" || spec.Template != "python" {
		t.Fatalf("run identity = %#v, want api/python", spec)
	}
	if spec.Service.Port != 8080 || spec.Service.CWD != "/workspace" {
		t.Fatalf("service = %+v, want port 8080 cwd /workspace", spec.Service)
	}
	if !reflect.DeepEqual(spec.Service.Command, []string{"python", "-m", "http.server", "8080"}) {
		t.Fatalf("service command = %#v", spec.Service.Command)
	}
	if spec.Service.EnvVars["APP_ENV"] != "production" {
		t.Fatalf("service env = %#v, want APP_ENV", spec.Service.EnvVars)
	}
	if len(spec.Mounts) != 1 || spec.Mounts[0].SnapshotID != "snap_123" || spec.Mounts[0].MountPath != "/workspace" {
		t.Fatalf("mounts = %#v, want snap_123:/workspace", spec.Mounts)
	}
	if spec.Scale == nil {
		t.Fatal("Scale = nil, want policy")
	}
	maxInstances, ok := spec.Scale.MaxInstances.Get()
	if !ok || maxInstances != 4 {
		t.Fatalf("Scale.MaxInstances = %d, %v; want 4, true", maxInstances, ok)
	}
	idleTimeout, ok := spec.Scale.IdleTimeoutSeconds.Get()
	if !ok || idleTimeout != 120 {
		t.Fatalf("Scale.IdleTimeoutSeconds = %d, %v; want 120, true", idleTimeout, ok)
	}
	if spec.Activate == nil || *spec.Activate {
		t.Fatalf("Activate = %#v, want false pointer", spec.Activate)
	}
}
