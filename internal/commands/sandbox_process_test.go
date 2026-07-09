package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestSandboxProcessCommandsRegistered(t *testing.T) {
	if cmd, _, err := sandboxCmd.Find([]string{"process", "create"}); err != nil || cmd != sandboxProcessCreateCmd {
		t.Fatalf("sandbox process create command not registered: cmd=%v err=%v", cmd, err)
	}
	if cmd, _, err := sandboxCmd.Find([]string{"process", "events"}); err != nil || cmd != sandboxProcessEventsCmd {
		t.Fatalf("sandbox process events command not registered: cmd=%v err=%v", cmd, err)
	}
	if cmd, _, err := sandboxCmd.Find([]string{"process", "input"}); err != nil || cmd != sandboxProcessInputCmd {
		t.Fatalf("sandbox process input command not registered: cmd=%v err=%v", cmd, err)
	}
}

func TestBuildSandboxProcessCreateSpecForStdio(t *testing.T) {
	resetSandboxProcessFlagsForTest()
	processCreateAlias = "worker"
	processCreateCwd = "/workspace"
	processCreateEnv = []string{"MODE=test", "EMPTY="}
	processCreateChannel = "stdio"
	processCreateKind = "stdio"
	processCreateFraming = "line"
	processCreateEventBufferSize = 32
	processCreateInputBufferSize = 16

	spec, err := buildSandboxProcessCreateSpec([]string{"python", "-u", "worker.py"})
	if err != nil {
		t.Fatal(err)
	}
	if got := spec.Command; len(got) != 3 || got[0] != "python" || got[2] != "worker.py" {
		t.Fatalf("command = %#v", got)
	}
	if alias, ok := spec.Alias.Get(); !ok || alias != "worker" {
		t.Fatalf("alias = %q %v, want worker", alias, ok)
	}
	if cwd, ok := spec.Cwd.Get(); !ok || cwd != "/workspace" {
		t.Fatalf("cwd = %q %v, want /workspace", cwd, ok)
	}
	env, ok := spec.EnvVars.Get()
	if !ok || env["MODE"] != "test" || env["EMPTY"] != "" {
		t.Fatalf("env = %#v, want MODE and EMPTY", env)
	}
	if len(spec.Channels) != 1 {
		t.Fatalf("channels = %#v, want one channel", spec.Channels)
	}
	channel := spec.Channels[0]
	if channel.Name != "stdio" || channel.Kind != apispec.ProcessChannelKindStdio {
		t.Fatalf("channel = %#v, want stdio", channel)
	}
	if framing, ok := channel.Framing.Get(); !ok || framing != apispec.ProcessChannelFramingLine {
		t.Fatalf("framing = %q %v, want line", framing, ok)
	}
	if size, ok := spec.EventBufferSize.Get(); !ok || size != 32 {
		t.Fatalf("event buffer size = %d %v, want 32", size, ok)
	}
	if size, ok := spec.InputBufferSize.Get(); !ok || size != 16 {
		t.Fatalf("input buffer size = %d %v, want 16", size, ok)
	}
}

func TestBuildSandboxProcessCreateSpecForPTY(t *testing.T) {
	resetSandboxProcessFlagsForTest()
	processCreateKind = "pty"
	processCreateFraming = "raw"
	processCreatePTYRows = 24
	processCreatePTYCols = 80

	spec, err := buildSandboxProcessCreateSpec([]string{"bash"})
	if err != nil {
		t.Fatal(err)
	}
	channel := spec.Channels[0]
	if channel.Name != "pty" || channel.Kind != apispec.ProcessChannelKindPty {
		t.Fatalf("channel = %#v, want pty", channel)
	}
	size, ok := channel.PtySize.Get()
	if !ok {
		t.Fatal("pty size was not set")
	}
	if rows, ok := size.Rows.Get(); !ok || rows != 24 {
		t.Fatalf("rows = %d %v, want 24", rows, ok)
	}
	if cols, ok := size.Cols.Get(); !ok || cols != 80 {
		t.Fatalf("cols = %d %v, want 80", cols, ok)
	}
}

func TestBuildSandboxProcessCreateSpecRejectsInvalidEnv(t *testing.T) {
	resetSandboxProcessFlagsForTest()
	processCreateEnv = []string{"NO_EQUALS"}

	if _, err := buildSandboxProcessCreateSpec([]string{"echo", "ok"}); err == nil {
		t.Fatal("expected invalid env error")
	}
}

func TestReadSandboxProcessSpecFile(t *testing.T) {
	resetSandboxProcessFlagsForTest()
	path := filepath.Join(t.TempDir(), "process.yaml")
	if err := os.WriteFile(path, []byte("command: [python, -u]\nchannels:\n- name: stdio\n  kind: stdio\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	processCreateSpecFile = path

	spec, err := buildSandboxProcessCreateSpec(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Command) != 2 || spec.Command[0] != "python" {
		t.Fatalf("command = %#v, want python -u", spec.Command)
	}
	if len(spec.Channels) != 1 || spec.Channels[0].Kind != apispec.ProcessChannelKindStdio {
		t.Fatalf("channels = %#v, want stdio", spec.Channels)
	}
}

func TestParseProcessInputType(t *testing.T) {
	if eventType, err := parseProcessInputType("stdin.write"); err != nil || eventType != apispec.ProcessEventTypeStdinWrite {
		t.Fatalf("stdin.write = %q err=%v", eventType, err)
	}
	if eventType, err := parseProcessInputType("pty.input"); err != nil || eventType != apispec.ProcessEventTypePtyInput {
		t.Fatalf("pty.input = %q err=%v", eventType, err)
	}
	if _, err := parseProcessInputType("stdout.line"); err == nil {
		t.Fatal("expected unsupported input event type error")
	}
}

func resetSandboxProcessFlagsForTest() {
	processCreateSpecFile = ""
	processCreateAlias = ""
	processCreateCwd = ""
	processCreateEnv = nil
	processCreateChannel = ""
	processCreateKind = "stdio"
	processCreateFraming = "line"
	processCreatePTYRows = 0
	processCreatePTYCols = 0
	processCreateEventBufferSize = 0
	processCreateInputBufferSize = 0
	processInputChannel = "stdio"
	processInputType = string(apispec.ProcessEventTypeStdinWrite)
	processInputEventID = ""
	processInputData = ""
	processEventsCursor = -1
	processResizeChannel = "pty"
	processResizeRows = 0
	processResizeCols = 0
}
