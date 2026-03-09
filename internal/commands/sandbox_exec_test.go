package commands

import (
	"reflect"
	"testing"
)

func TestExtractExecCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "separator kept by cobra",
			args: []string{"sb_123", "--", "bash", "-lc", "echo hi"},
			want: []string{"bash", "-lc", "echo hi"},
		},
		{
			name: "separator consumed by cobra",
			args: []string{"sb_123", "bash", "-lc", "echo hi"},
			want: []string{"bash", "-lc", "echo hi"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractExecCommand(tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("extractExecCommand() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseExecEnvVars(t *testing.T) {
	t.Parallel()

	got, err := parseExecEnvVars([]string{"FOO=bar", "HELLO=world=again"})
	if err != nil {
		t.Fatalf("parseExecEnvVars() error = %v", err)
	}

	want := map[string]string{
		"FOO":   "bar",
		"HELLO": "world=again",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseExecEnvVars() = %#v, want %#v", got, want)
	}
}

func TestParseExecEnvVarsRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	if _, err := parseExecEnvVars([]string{"BROKEN"}); err == nil {
		t.Fatal("parseExecEnvVars() error = nil, want invalid env error")
	}
}

func TestIsInteractiveCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command []string
		want    bool
	}{
		{name: "bash", command: []string{"bash"}, want: true},
		{name: "zsh path", command: []string{"/bin/zsh"}, want: true},
		{name: "python", command: []string{"python"}, want: false},
		{name: "empty", command: nil, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isInteractiveCommand(tt.command); got != tt.want {
				t.Fatalf("isInteractiveCommand(%#v) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}
