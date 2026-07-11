package commands

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestBuildExecutionSessionSpec(t *testing.T) {
	resetSessionFlagsForTest()
	sessionName = "worker"
	sessionCWD = "/workspace"
	sessionEnv = []string{"A=one", "B=two=three"}
	sessionPTY = true
	sessionRows = 30
	sessionCols = 100
	sessionRestartPolicy = "on_failure"
	sessionRuntimeRecovery = "restart"
	sessionReadiness = "output"
	sessionReadyOutput = "READY"

	spec, err := buildExecutionSessionSpec([]string{"/bin/sh", "-c", "run"})
	if err != nil {
		t.Fatal(err)
	}
	if spec.Name.Or("") != "worker" || spec.Cwd.Or("") != "/workspace" {
		t.Fatalf("spec identity = %#v", spec)
	}
	env, ok := spec.Env.Get()
	if !ok || env["A"] != "one" || env["B"] != "two=three" {
		t.Fatalf("env = %#v", env)
	}
	ioSpec, ok := spec.Io.Get()
	if !ok || ioSpec.Mode.Or("") != apispec.ExecutionSessionIOModePty {
		t.Fatalf("io = %#v", ioSpec)
	}
	terminal, ok := ioSpec.Terminal.Get()
	if !ok || terminal.Rows.Or(0) != 30 || terminal.Cols.Or(0) != 100 {
		t.Fatalf("terminal = %#v", terminal)
	}
	lifecycle, ok := spec.Lifecycle.Get()
	if !ok {
		t.Fatal("lifecycle missing")
	}
	restart, ok := lifecycle.Restart.Get()
	if !ok || restart.Policy.Or("") != apispec.ExecutionSessionRestartPolicyOnFailure {
		t.Fatalf("restart = %#v", restart)
	}
	readiness, ok := spec.Readiness.Get()
	if !ok || readiness.Type.Or("") != apispec.ExecutionSessionReadinessTypeOutput || readiness.Output.Or("") != "READY" {
		t.Fatalf("readiness = %#v", readiness)
	}
}

func TestReadExecutionSessionSpec(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.yaml")
	if err := os.WriteFile(path, []byte("command:\n  - /bin/echo\n  - hello\nname: docs\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	spec, err := readExecutionSessionSpec(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Command) != 2 || spec.Command[1] != "hello" || spec.Name.Or("") != "docs" {
		t.Fatalf("spec = %#v", spec)
	}
}

func TestBuildExecutionSessionInputRequest(t *testing.T) {
	resetSessionFlagsForTest()
	sessionInputData = "hello\n"
	sessionInputEOF = true
	sessionInputID = "input-1"
	sessionExpectedAttemptID = "att-1"

	request, err := buildExecutionSessionInputRequest()
	if err != nil {
		t.Fatal(err)
	}
	encoded, ok := request.DataBase64.Get()
	if !ok {
		t.Fatal("data_base64 missing")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != "hello\n" || !request.EOF.Or(false) || request.ExpectedAttemptID.Or("") != "att-1" {
		t.Fatalf("request = %#v", request)
	}
}

func TestSandboxSessionCommandIsRegistered(t *testing.T) {
	for _, command := range sandboxCmd.Commands() {
		if command.Name() == "session" {
			return
		}
	}
	t.Fatal("sandbox session command is not registered")
}

func resetSessionFlagsForTest() {
	sessionSpecFile = ""
	sessionName = ""
	sessionCWD = ""
	sessionEnv = nil
	sessionPTY = false
	sessionRows = 24
	sessionCols = 80
	sessionTerm = "xterm-256color"
	sessionRestartPolicy = "never"
	sessionRuntimeRecovery = "restart"
	sessionIdleTimeout = 0
	sessionMaxLifetime = 0
	sessionStopGrace = 10
	sessionReadiness = "process"
	sessionReadyDelay = 0
	sessionReadyOutput = ""
	sessionReadyTimeout = 30000
	sessionIdempotencyKey = ""
	sessionReplaceAttempt = false
	sessionInputData = ""
	sessionInputBase64 = ""
	sessionInputFile = ""
	sessionInputEOF = false
	sessionInputID = ""
	sessionExpectedAttemptID = ""
	sessionEventAfter = 0
	sessionEventLimit = 1000
	sessionEventFollow = false
	sessionEventLastID = ""
}
