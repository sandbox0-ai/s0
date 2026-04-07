package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildTemplateCreateRequestPreservesSidecarSpec(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	specFile := filepath.Join(dir, "template.yaml")
	specYAML := `spec:
  displayName: "Claude Code Sidecar Kind Test"
  description: "Minimal template for verifying sidecar startup and direct exposed-port routing"
  mainContainer:
    image: nginx:1.27-alpine
    resources:
      cpu: "250m"
      memory: 256Mi
  sharedVolumes:
    - name: workspace
      sandboxVolumeId: vol_123
      mountPath: /workspace/shared
  sidecars:
    - name: claude-code
      image: cc-demo:test
      resources:
        cpu: "500m"
        memory: 512Mi
      mounts:
        - name: workspace
          mountPath: /shared
      env:
        - name: PORT
          value: "8081"
        - name: WORKSPACE_DIR
          value: /workspace
      readinessProbe:
        exec:
          command: ["test", "-f", "/tmp/cc-sidecar-ready"]
        initialDelaySeconds: 1
        periodSeconds: 2
        failureThreshold: 30
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
	if len(req.Spec.Sidecars) != 1 {
		t.Fatalf("len(Spec.Sidecars) = %d, want 1", len(req.Spec.Sidecars))
	}
	if len(req.Spec.SharedVolumes) != 1 {
		t.Fatalf("len(Spec.SharedVolumes) = %d, want 1", len(req.Spec.SharedVolumes))
	}
	if req.Spec.SharedVolumes[0].Name != "workspace" {
		t.Fatalf("SharedVolumes[0].Name = %q, want workspace", req.Spec.SharedVolumes[0].Name)
	}
	if req.Spec.SharedVolumes[0].MountPath != "/workspace/shared" {
		t.Fatalf("SharedVolumes[0].MountPath = %q, want /workspace/shared", req.Spec.SharedVolumes[0].MountPath)
	}

	sidecar := req.Spec.Sidecars[0]
	if sidecar.Name != "claude-code" {
		t.Fatalf("Sidecars[0].Name = %q, want claude-code", sidecar.Name)
	}
	if sidecar.Image != "cc-demo:test" {
		t.Fatalf("Sidecars[0].Image = %q, want cc-demo:test", sidecar.Image)
	}
	if len(sidecar.Env) != 2 {
		t.Fatalf("len(Sidecars[0].Env) = %d, want 2", len(sidecar.Env))
	}
	if len(sidecar.Mounts) != 1 {
		t.Fatalf("len(Sidecars[0].Mounts) = %d, want 1", len(sidecar.Mounts))
	}
	if sidecar.Mounts[0].Name != "workspace" || sidecar.Mounts[0].MountPath != "/shared" {
		t.Fatalf("Sidecars[0].Mounts[0] = %+v, want workspace:/shared", sidecar.Mounts[0])
	}
	if !sidecar.ReadinessProbe.IsSet() {
		t.Fatal("Sidecars[0].ReadinessProbe should be set")
	}
	probe, ok := sidecar.ReadinessProbe.Get()
	if !ok {
		t.Fatal("Sidecars[0].ReadinessProbe.Get() should return value")
	}
	if !probe.Exec.IsSet() {
		t.Fatal("Sidecars[0].ReadinessProbe.Exec should be set")
	}
	execAction, ok := probe.Exec.Get()
	if !ok {
		t.Fatal("Sidecars[0].ReadinessProbe.Exec.Get() should return value")
	}
	if len(execAction.Command) != 3 {
		t.Fatalf("len(Sidecars[0].ReadinessProbe.Exec.Command) = %d, want 3", len(execAction.Command))
	}

	cpu, ok := sidecar.Resources.CPU.Get()
	if !ok || cpu != "500m" {
		t.Fatalf("Sidecars[0].Resources.CPU = %q, want 500m", cpu)
	}
	memory, ok := sidecar.Resources.Memory.Get()
	if !ok || memory != "512Mi" {
		t.Fatalf("Sidecars[0].Resources.Memory = %q, want 512Mi", memory)
	}
}
