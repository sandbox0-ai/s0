package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestBuildTemplateCreateRequestPreservesWarmProcessSpec(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	specFile := filepath.Join(dir, "template.yaml")
	specYAML := `spec:
  displayName: "Claude Code Warm Process Kind Test"
  description: "Minimal template for verifying warm process startup"
  mainContainer:
    image: cc-demo:test
    resources:
      cpu: "250m"
      memory: 256Mi
  warmProcesses:
    - type: cmd
      alias: claude-code
      command: ["sh", "-lc", "touch /tmp/cc-warm-ready; tail -f /dev/null"]
      cwd: /workspace
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
	if len(req.Spec.WarmProcesses) != 1 {
		t.Fatalf("len(Spec.WarmProcesses) = %d, want 1", len(req.Spec.WarmProcesses))
	}

	process := req.Spec.WarmProcesses[0]
	if process.Type != apispec.WarmProcessSpecTypeCmd {
		t.Fatalf("WarmProcesses[0].Type = %q, want cmd", process.Type)
	}
	alias, ok := process.Alias.Get()
	if !ok || alias != "claude-code" {
		t.Fatalf("WarmProcesses[0].Alias = %q, want claude-code", alias)
	}
	if len(process.Command) != 3 {
		t.Fatalf("len(WarmProcesses[0].Command) = %d, want 3", len(process.Command))
	}
	if process.Command[2] != "touch /tmp/cc-warm-ready; tail -f /dev/null" {
		t.Fatalf("WarmProcesses[0].Command[2] = %q, want warm command", process.Command[2])
	}
	cwd, ok := process.Cwd.Get()
	if !ok || cwd != "/workspace" {
		t.Fatalf("WarmProcesses[0].Cwd = %q, want /workspace", cwd)
	}
	envVars, ok := process.EnvVars.Get()
	if !ok {
		t.Fatal("WarmProcesses[0].EnvVars should be set")
	}
	if envVars["PORT"] != "8081" || envVars["WORKSPACE_DIR"] != "/workspace" {
		t.Fatalf("WarmProcesses[0].EnvVars = %+v, want PORT and WORKSPACE_DIR", envVars)
	}
}
