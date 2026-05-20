package commands

import "testing"

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
		{name: "negative min warm", minWarm: -1, maxActive: 20, targetConcurrency: 80, scaleDownSeconds: 300, wantErr: true},
		{name: "zero max active", minWarm: 0, maxActive: 0, targetConcurrency: 80, scaleDownSeconds: 300, wantErr: true},
		{name: "zero target concurrency", minWarm: 0, maxActive: 20, targetConcurrency: 0, scaleDownSeconds: 300, wantErr: true},
		{name: "zero scale down", minWarm: 0, maxActive: 20, targetConcurrency: 80, scaleDownSeconds: 0, wantErr: true},
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
