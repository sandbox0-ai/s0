package commands

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	execCwd         string
	execEnv         []string
	execNoWait      bool
	execTTL         int32
	execInteractive bool
	execTTY         bool
)

type execWSMessage struct {
	Type   string `json:"type,omitempty"`
	Source string `json:"source,omitempty"`
	Data   string `json:"data,omitempty"`
	Rows   int    `json:"rows,omitempty"`
	Cols   int    `json:"cols,omitempty"`
	Signal string `json:"signal,omitempty"`
}

// sandboxExecCmd executes a command in a sandbox.
var sandboxExecCmd = &cobra.Command{
	Use:   "exec <sandbox-id> -- <command> [args...]",
	Short: "Execute a command in a sandbox",
	Long: `Execute a command in a sandbox and wait for completion.

The command must be preceded by '--' to separate it from flags.

Examples:
  s0 sandbox exec sb_abc123 -- echo "Hello"
  s0 sandbox exec sb_abc123 --cwd /app -- python script.py
  s0 sandbox exec sb_abc123 -it -- bash`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]
		command := extractExecCommand(args)
		if len(command) == 0 {
			fmt.Fprintln(os.Stderr, "Error: command is required (use '--' followed by command)")
			os.Exit(1)
		}

		envMap, err := parseExecEnvVars(execEnv)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		if execNoWait && (execInteractive || execTTY) {
			fmt.Fprintln(os.Stderr, "Error: --no-wait cannot be used with interactive or TTY exec")
			os.Exit(1)
		}

		if shouldUseStreamingExec(command) {
			if err := runStreamingExec(cmd.Context(), client, sandboxID, command, envMap); err != nil {
				fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Pass command as string for SDK validation, WithCommand overrides the actual command array.
		opts := []sandbox0.CmdOption{
			sandbox0.WithCommand(command),
		}
		if execCwd != "" {
			opts = append(opts, sandbox0.WithCmdCWD(execCwd))
		}
		if len(envMap) > 0 {
			opts = append(opts, sandbox0.WithCmdEnvVars(envMap))
		}
		if execNoWait {
			opts = append(opts, sandbox0.WithCmdWait(false))
		}
		if execTTL > 0 {
			opts = append(opts, sandbox0.WithCmdTTL(execTTL))
		}

		cmdStr := strings.Join(command, " ")
		result, err := client.Sandbox(sandboxID).Cmd(cmd.Context(), cmdStr, opts...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(result.OutputRaw)
	},
}

func extractExecCommand(args []string) []string {
	separatorIndex := -1
	for i, arg := range args {
		if arg == "--" {
			separatorIndex = i
			break
		}
	}

	if separatorIndex >= 0 {
		return args[separatorIndex+1:]
	}
	if len(args) > 1 {
		return args[1:]
	}
	return nil
}

func parseExecEnvVars(values []string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	envMap := make(map[string]string, len(values))
	for _, e := range values {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid env format %q, expected KEY=VALUE", e)
		}
		envMap[parts[0]] = parts[1]
	}
	return envMap, nil
}

func shouldUseStreamingExec(command []string) bool {
	if execNoWait {
		return false
	}
	if execInteractive || execTTY {
		return true
	}
	return shouldAutoInteractiveExec(command)
}

func shouldAutoInteractiveExec(command []string) bool {
	return isTerminalFile(os.Stdin) && isTerminalFile(os.Stdout) && isInteractiveCommand(command)
}

func isInteractiveCommand(command []string) bool {
	if len(command) == 0 {
		return false
	}
	switch filepath.Base(command[0]) {
	case "bash", "sh", "zsh", "fish", "ash", "dash", "ksh", "tcsh", "csh":
		return true
	default:
		return false
	}
}

func runStreamingExec(ctx context.Context, client *sandbox0.Client, sandboxID string, command []string, envMap map[string]string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	req := apispec.CreateContextRequest{
		Type:          apispec.NewOptProcessType(apispec.ProcessTypeCmd),
		Cmd:           apispec.NewOptCreateCMDContextRequest(apispec.CreateCMDContextRequest{Command: command}),
		WaitUntilDone: apispec.NewOptBool(false),
	}
	if execCwd != "" {
		req.Cwd = apispec.NewOptString(execCwd)
	}
	if len(envMap) > 0 {
		req.EnvVars = apispec.NewOptCreateContextRequestEnvVars(apispec.CreateContextRequestEnvVars(envMap))
	}
	if execTTL > 0 {
		req.TTLSec = apispec.NewOptInt32(execTTL)
	}

	allocateTTY := execTTY || execInteractive || shouldAutoInteractiveExec(command)
	if allocateTTY {
		rows, cols := currentTerminalSize()
		req.PtySize = apispec.NewOptPTYSize(apispec.PTYSize{
			Rows: apispec.NewOptInt32(rows),
			Cols: apispec.NewOptInt32(cols),
		})
	}

	contextResp, err := client.Sandbox(sandboxID).CreateContext(ctx, req)
	if err != nil {
		return err
	}

	conn, _, err := client.Sandbox(sandboxID).ConnectWSContext(ctx, contextResp.ID)
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	restoreTerminal, err := prepareTerminalForStreaming(allocateTTY)
	if err != nil {
		return err
	}
	defer restoreTerminal()

	var writeMu sync.Mutex
	writeJSON := func(msg execWSMessage) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteJSON(msg)
	}
	writeControl := func(messageType int, data []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteControl(messageType, data, time.Now().Add(time.Second))
	}

	if allocateTTY && isTerminalFile(os.Stdin) {
		go forwardResizeEvents(ctx, writeJSON)
	}
	go forwardSignals(ctx, writeJSON)
	streamInput := execInteractive || shouldAutoInteractiveExec(command)
	go forwardInput(ctx, cancel, writeJSON, writeControl, streamInput)

	for {
		var msg execWSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			if isNormalWSClose(err) || ctx.Err() != nil {
				return nil
			}
			return err
		}
		switch msg.Source {
		case "stderr":
			if _, err := io.WriteString(os.Stderr, msg.Data); err != nil {
				return err
			}
		default:
			if _, err := io.WriteString(os.Stdout, msg.Data); err != nil {
				return err
			}
		}
	}
}

func prepareTerminalForStreaming(allocateTTY bool) (func(), error) {
	if !allocateTTY || !isTerminalFile(os.Stdin) {
		return func() {}, nil
	}

	stateCmd := exec.Command("stty", "-g")
	stateCmd.Stdin = os.Stdin
	state, err := stateCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("capture terminal state: %w", err)
	}

	rawCmd := exec.Command("stty", "raw", "-echo")
	rawCmd.Stdin = os.Stdin
	if err := rawCmd.Run(); err != nil {
		return nil, fmt.Errorf("enable raw terminal mode: %w", err)
	}

	restore := strings.TrimSpace(string(state))
	return func() {
		if restore == "" {
			return
		}
		restoreCmd := exec.Command("stty", restore)
		restoreCmd.Stdin = os.Stdin
		_ = restoreCmd.Run()
	}, nil
}

func forwardInput(ctx context.Context, cancel context.CancelFunc, writeJSON func(execWSMessage) error, writeControl func(int, []byte) error, streamInput bool) {
	if !streamInput {
		return
	}

	buf := make([]byte, 4096)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			if writeErr := writeJSON(execWSMessage{
				Type: "input",
				Data: string(buf[:n]),
			}); writeErr != nil {
				cancel()
				return
			}
		}
		if err != nil {
			if err == io.EOF {
				writeControlClose(writeControl)
			}
			cancel()
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func forwardResizeEvents(ctx context.Context, writeJSON func(execWSMessage) error) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-ctx.Done():
			return
		case <-sigCh:
			rows, cols := currentTerminalSize()
			_ = writeJSON(execWSMessage{
				Type: "resize",
				Rows: int(rows),
				Cols: int(cols),
			})
		}
	}
}

func forwardSignals(ctx context.Context, writeJSON func(execWSMessage) error) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-ctx.Done():
			return
		case sig := <-sigCh:
			name := signalName(sig)
			if name == "" {
				continue
			}
			_ = writeJSON(execWSMessage{
				Type:   "signal",
				Signal: name,
			})
		}
	}
}

func signalName(sig os.Signal) string {
	switch sig {
	case os.Interrupt:
		return "INT"
	case syscall.SIGTERM:
		return "TERM"
	default:
		return ""
	}
}

func currentTerminalSize() (int32, int32) {
	if !isTerminalFile(os.Stdin) {
		return 24, 80
	}
	sizeCmd := exec.Command("stty", "size")
	sizeCmd.Stdin = os.Stdin
	out, err := sizeCmd.Output()
	if err != nil {
		return 24, 80
	}
	var rows, cols int32
	if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%d %d", &rows, &cols); err != nil {
		return 24, 80
	}
	if rows <= 0 || cols <= 0 {
		return 24, 80
	}
	return rows, cols
}

func isTerminalFile(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func isNormalWSClose(err error) bool {
	if err == nil {
		return false
	}
	if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		return true
	}
	if errorsAsNetClosed(err) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "context exited")
}

func errorsAsNetClosed(err error) bool {
	return err == net.ErrClosed || strings.Contains(strings.ToLower(err.Error()), "use of closed network connection")
}

func writeControlClose(writeControl func(int, []byte) error) {
	_ = writeControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	)
}

func init() {
	sandboxExecCmd.Flags().StringVar(&execCwd, "cwd", "", "working directory")
	sandboxExecCmd.Flags().StringArrayVar(&execEnv, "env", nil, "environment variables (KEY=VALUE, can be repeated)")
	sandboxExecCmd.Flags().BoolVar(&execNoWait, "no-wait", false, "don't wait for command completion")
	sandboxExecCmd.Flags().Int32Var(&execTTL, "ttl", 0, "context TTL in seconds")
	sandboxExecCmd.Flags().BoolVarP(&execInteractive, "interactive", "i", false, "keep stdin attached and stream exec I/O (implies TTY)")
	sandboxExecCmd.Flags().BoolVarP(&execTTY, "tty", "t", false, "allocate a TTY and stream exec I/O")

	sandboxCmd.AddCommand(sandboxExecCmd)
}
