package commands

import (
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestFunctionCommandsExposeAutoscalingFlags(t *testing.T) {
	for _, cmd := range []string{"create", "update"} {
		command := functionCreateCmd
		if cmd == "update" {
			command = functionUpdateCmd
		}
		for _, flag := range []string{"min-warm", "max-active", "target-concurrency", "scale-down-after-seconds"} {
			if command.Flags().Lookup(flag) == nil {
				t.Fatalf("function %s missing --%s flag", cmd, flag)
			}
		}
	}
}

func TestFunctionCommandsExposeSpecFileFlags(t *testing.T) {
	if functionCreateCmd.Flags().Lookup("spec-file") == nil {
		t.Fatal("function create missing --spec-file flag")
	}
	if functionRevisionCreateCmd.Flags().Lookup("spec-file") == nil {
		t.Fatal("function revision create missing --spec-file flag")
	}
}

func TestValidateFunctionAutoscalingFlags(t *testing.T) {
	tests := []struct {
		name              string
		minWarm           int32
		maxActive         int32
		targetConcurrency int32
		scaleDownSeconds  int32
		wantErr           bool
	}{
		{name: "default", minWarm: 0, maxActive: 20, targetConcurrency: 80, scaleDownSeconds: 300},
		{name: "warm pool", minWarm: 2, maxActive: 4, targetConcurrency: 10, scaleDownSeconds: 60},
		{name: "minimum scale down", minWarm: 0, maxActive: 20, targetConcurrency: 80, scaleDownSeconds: 30},
		{name: "negative min warm", minWarm: -1, maxActive: 20, targetConcurrency: 80, scaleDownSeconds: 300, wantErr: true},
		{name: "zero max active", minWarm: 0, maxActive: 0, targetConcurrency: 80, scaleDownSeconds: 300, wantErr: true},
		{name: "zero target concurrency", minWarm: 0, maxActive: 20, targetConcurrency: 0, scaleDownSeconds: 300, wantErr: true},
		{name: "zero scale down", minWarm: 0, maxActive: 20, targetConcurrency: 80, scaleDownSeconds: 0, wantErr: true},
		{name: "below minimum scale down", minWarm: 0, maxActive: 20, targetConcurrency: 80, scaleDownSeconds: 29, wantErr: true},
		{name: "min warm above max active", minWarm: 3, maxActive: 2, targetConcurrency: 80, scaleDownSeconds: 300, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFunctionAutoscalingFlags(tt.minWarm, tt.maxActive, tt.targetConcurrency, tt.scaleDownSeconds)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateFunctionAutoscalingFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseFunctionRevisionSpec(t *testing.T) {
	spec, err := parseFunctionRevisionSpec([]byte(`
template_id: default
runtime_service:
  id: web
  port: 8080
  ingress:
    public: true
  runtime:
    type: warm_process
mounts:
  - mount_point: /data
    source:
      type: sandbox_volume
      sandboxvolume_id: sv_123
`))
	if err != nil {
		t.Fatalf("parseFunctionRevisionSpec() error = %v", err)
	}
	if spec.TemplateID != "default" {
		t.Fatalf("template id = %q, want default", spec.TemplateID)
	}
	if spec.RuntimeService.ID != "web" || spec.RuntimeService.Port != 8080 {
		t.Fatalf("runtime service = %#v, want web:8080", spec.RuntimeService)
	}
	if len(spec.Mounts) != 1 {
		t.Fatalf("mounts count = %d, want 1", len(spec.Mounts))
	}
	if spec.Mounts[0].Source.Type != apispec.FunctionRevisionMountSourceTypeSandboxVolume {
		t.Fatalf("mount source type = %q, want sandbox_volume", spec.Mounts[0].Source.Type)
	}
}
