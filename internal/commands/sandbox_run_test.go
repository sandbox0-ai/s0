package commands

import (
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestFindReusableRunContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		contexts []apispec.ContextResponse
		alias    string
		wantID   string
		wantErr  bool
	}{
		{
			name:  "defaults alias to python",
			alias: "",
			contexts: []apispec.ContextResponse{
				{
					ID:      "ctx_python",
					Type:    apispec.ProcessTypeRepl,
					Alias:   apispec.NewOptString("python"),
					Running: true,
				},
			},
			wantID: "ctx_python",
		},
		{
			name:  "ignores cmd contexts",
			alias: "python",
			contexts: []apispec.ContextResponse{
				{
					ID:      "ctx_cmd",
					Type:    apispec.ProcessTypeCmd,
					Alias:   apispec.NewOptString("python"),
					Running: true,
				},
			},
		},
		{
			name:  "ignores non-running contexts",
			alias: "python",
			contexts: []apispec.ContextResponse{
				{
					ID:      "ctx_stopped",
					Type:    apispec.ProcessTypeRepl,
					Alias:   apispec.NewOptString("python"),
					Running: false,
				},
			},
		},
		{
			name:  "ignores paused contexts",
			alias: "python",
			contexts: []apispec.ContextResponse{
				{
					ID:      "ctx_paused",
					Type:    apispec.ProcessTypeRepl,
					Alias:   apispec.NewOptString("python"),
					Running: true,
					Paused:  true,
				},
			},
		},
		{
			name:  "returns error on ambiguous alias",
			alias: "python",
			contexts: []apispec.ContextResponse{
				{
					ID:      "ctx_1",
					Type:    apispec.ProcessTypeRepl,
					Alias:   apispec.NewOptString("python"),
					Running: true,
				},
				{
					ID:      "ctx_2",
					Type:    apispec.ProcessTypeRepl,
					Alias:   apispec.NewOptString("python"),
					Running: true,
				},
			},
			wantErr: true,
		},
		{
			name:  "returns match for requested alias",
			alias: "bash",
			contexts: []apispec.ContextResponse{
				{
					ID:      "ctx_python",
					Type:    apispec.ProcessTypeRepl,
					Alias:   apispec.NewOptString("python"),
					Running: true,
				},
				{
					ID:      "ctx_bash",
					Type:    apispec.ProcessTypeRepl,
					Alias:   apispec.NewOptString("bash"),
					Running: true,
				},
			},
			wantID: "ctx_bash",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotID, err := findReusableRunContext(tt.contexts, tt.alias)
			if tt.wantErr {
				if err == nil {
					t.Fatal("findReusableRunContext() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("findReusableRunContext() error = %v", err)
			}
			if gotID != tt.wantID {
				t.Fatalf("findReusableRunContext() = %q, want %q", gotID, tt.wantID)
			}
		})
	}
}
