package commands

import (
	"bytes"
	"reflect"
	"testing"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
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

func TestShouldUseStreamingExec(t *testing.T) {
	t.Parallel()

	prevNoWait := execNoWait
	prevStream := execStream
	prevInteractive := execInteractive
	prevTTY := execTTY
	t.Cleanup(func() {
		execNoWait = prevNoWait
		execStream = prevStream
		execInteractive = prevInteractive
		execTTY = prevTTY
	})

	execNoWait = false
	execStream = true
	execInteractive = false
	execTTY = false
	if !shouldUseStreamingExec([]string{"python", "-c", "print(1)"}) {
		t.Fatal("shouldUseStreamingExec() = false, want true when --stream is enabled")
	}

	execNoWait = true
	if shouldUseStreamingExec([]string{"python", "-c", "print(1)"}) {
		t.Fatal("shouldUseStreamingExec() = true, want false when --no-wait is enabled")
	}
}

func TestIsTerminalDoneExecMessage(t *testing.T) {
	t.Parallel()

	code := 0
	if isTerminalDoneExecMessage(execWSMessage{Type: "done", RequestID: "req-1"}) {
		t.Fatal("request-scoped done should not terminate exec stream")
	}
	if !isTerminalDoneExecMessage(execWSMessage{Type: "done", ExitCode: &code, State: "stopped"}) {
		t.Fatal("process done should terminate exec stream")
	}
}

func TestRemoteExecFailureCode(t *testing.T) {
	t.Parallel()

	if code, failed := remoteExecFailureCode(nil); failed || code != 0 {
		t.Fatalf("nil exit code = (%d, %v), want (0, false)", code, failed)
	}

	zero := 0
	if code, failed := remoteExecFailureCode(&zero); failed || code != 0 {
		t.Fatalf("zero exit code = (%d, %v), want (0, false)", code, failed)
	}

	nonzero := 7
	if code, failed := remoteExecFailureCode(&nonzero); !failed || code != 7 {
		t.Fatalf("nonzero exit code = (%d, %v), want (7, true)", code, failed)
	}
}

func TestWriteCmdResultOutputSplitsStreams(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := writeCmdResultOutput(&stdout, &stderr, sandbox0.CmdResult{
		OutputRaw: "outerr",
		Stdout:    "out",
		Stderr:    "err",
	})
	if err != nil {
		t.Fatalf("writeCmdResultOutput() error = %v", err)
	}
	if got := stdout.String(); got != "out" {
		t.Fatalf("stdout = %q, want out", got)
	}
	if got := stderr.String(); got != "err" {
		t.Fatalf("stderr = %q, want err", got)
	}
}

func TestWriteCmdResultOutputFallsBackToOutputRaw(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := writeCmdResultOutput(&stdout, &stderr, sandbox0.CmdResult{
		OutputRaw: "legacy",
	})
	if err != nil {
		t.Fatalf("writeCmdResultOutput() error = %v", err)
	}
	if got := stdout.String(); got != "legacy" {
		t.Fatalf("stdout = %q, want legacy", got)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}
