package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	processCreateSpecFile        string
	processCreateAlias           string
	processCreateCwd             string
	processCreateEnv             []string
	processCreateChannel         string
	processCreateKind            string
	processCreateFraming         string
	processCreatePTYRows         int32
	processCreatePTYCols         int32
	processCreateEventBufferSize int32
	processCreateInputBufferSize int32

	processInputChannel string
	processInputType    string
	processInputEventID string
	processInputData    string

	processEventsCursor int64

	processResizeChannel string
	processResizeRows    int32
	processResizeCols    int32
)

var sandboxProcessCmd = &cobra.Command{
	Use:   "process",
	Short: "Manage broker-owned sandbox processes",
	Long:  "Manage broker-owned process sessions inside a sandbox.",
}

var sandboxProcessCreateCmd = &cobra.Command{
	Use:   "create <sandbox-id> [-- command [args...]]",
	Short: "Create a sandbox process session",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runSandboxProcessCreate(cmd, args); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating process: %v\n", err)
			os.Exit(1)
		}
	},
}

var sandboxProcessListCmd = &cobra.Command{
	Use:   "list <sandbox-id>",
	Short: "List sandbox process sessions",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runSandboxProcessList(cmd, args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error listing processes: %v\n", err)
			os.Exit(1)
		}
	},
}

var sandboxProcessGetCmd = &cobra.Command{
	Use:   "get <sandbox-id> <process-id>",
	Short: "Get a sandbox process session",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runSandboxProcessGet(cmd, args[0], args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error getting process: %v\n", err)
			os.Exit(1)
		}
	},
}

var sandboxProcessDeleteCmd = &cobra.Command{
	Use:   "delete <sandbox-id> <process-id>",
	Short: "Delete a sandbox process session",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runSandboxProcessDelete(cmd, args[0], args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting process: %v\n", err)
			os.Exit(1)
		}
	},
}

var sandboxProcessInputCmd = &cobra.Command{
	Use:   "input <sandbox-id> <process-id>",
	Short: "Send an idempotent input event to a process",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runSandboxProcessInput(cmd, args[0], args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending process input: %v\n", err)
			os.Exit(1)
		}
	},
}

var sandboxProcessEventsCmd = &cobra.Command{
	Use:   "events <sandbox-id> <process-id>",
	Short: "Stream process events as JSON lines",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runSandboxProcessEvents(cmd, args[0], args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error streaming process events: %v\n", err)
			os.Exit(1)
		}
	},
}

var sandboxProcessSignalCmd = &cobra.Command{
	Use:   "signal <sandbox-id> <process-id> <signal>",
	Short: "Signal a sandbox process session",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runSandboxProcessSignal(cmd, args[0], args[1], args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "Error signaling process: %v\n", err)
			os.Exit(1)
		}
	},
}

var sandboxProcessResizeCmd = &cobra.Command{
	Use:   "resize <sandbox-id> <process-id>",
	Short: "Resize a PTY process channel",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runSandboxProcessResize(cmd, args[0], args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error resizing process PTY: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	sandboxProcessCreateCmd.Flags().StringVar(&processCreateSpecFile, "spec-file", "", "path to a process spec YAML/JSON file, or - for stdin")
	sandboxProcessCreateCmd.Flags().StringVar(&processCreateAlias, "alias", "", "process alias")
	sandboxProcessCreateCmd.Flags().StringVar(&processCreateCwd, "cwd", "", "working directory")
	sandboxProcessCreateCmd.Flags().StringArrayVar(&processCreateEnv, "env", nil, "environment variable in KEY=VALUE form (repeatable)")
	sandboxProcessCreateCmd.Flags().StringVar(&processCreateChannel, "channel", "", "channel name (default follows --kind)")
	sandboxProcessCreateCmd.Flags().StringVar(&processCreateKind, "kind", "stdio", "channel kind for command mode (stdio or pty)")
	sandboxProcessCreateCmd.Flags().StringVar(&processCreateFraming, "framing", "line", "channel framing (raw, line, jsonl, jsonrpc)")
	sandboxProcessCreateCmd.Flags().Int32Var(&processCreatePTYRows, "pty-rows", 0, "initial PTY rows when --kind=pty")
	sandboxProcessCreateCmd.Flags().Int32Var(&processCreatePTYCols, "pty-cols", 0, "initial PTY columns when --kind=pty")
	sandboxProcessCreateCmd.Flags().Int32Var(&processCreateEventBufferSize, "event-buffer-size", 0, "process event replay buffer size")
	sandboxProcessCreateCmd.Flags().Int32Var(&processCreateInputBufferSize, "input-buffer-size", 0, "process input de-duplication buffer size")

	sandboxProcessInputCmd.Flags().StringVar(&processInputChannel, "channel", "stdio", "target channel")
	sandboxProcessInputCmd.Flags().StringVar(&processInputType, "type", string(apispec.ProcessEventTypeStdinWrite), "input event type (stdin.write or pty.input)")
	sandboxProcessInputCmd.Flags().StringVar(&processInputEventID, "event-id", "", "idempotency key for this input event")
	sandboxProcessInputCmd.Flags().StringVar(&processInputData, "data", "", "input data payload")

	sandboxProcessEventsCmd.Flags().Int64Var(&processEventsCursor, "cursor", -1, "last observed event sequence to resume after")

	sandboxProcessResizeCmd.Flags().StringVar(&processResizeChannel, "channel", "pty", "PTY channel name")
	sandboxProcessResizeCmd.Flags().Int32Var(&processResizeRows, "rows", 0, "terminal rows")
	sandboxProcessResizeCmd.Flags().Int32Var(&processResizeCols, "cols", 0, "terminal columns")

	sandboxProcessCmd.AddCommand(sandboxProcessCreateCmd)
	sandboxProcessCmd.AddCommand(sandboxProcessListCmd)
	sandboxProcessCmd.AddCommand(sandboxProcessGetCmd)
	sandboxProcessCmd.AddCommand(sandboxProcessDeleteCmd)
	sandboxProcessCmd.AddCommand(sandboxProcessInputCmd)
	sandboxProcessCmd.AddCommand(sandboxProcessEventsCmd)
	sandboxProcessCmd.AddCommand(sandboxProcessSignalCmd)
	sandboxProcessCmd.AddCommand(sandboxProcessResizeCmd)
	sandboxCmd.AddCommand(sandboxProcessCmd)
}

func runSandboxProcessCreate(cmd *cobra.Command, args []string) error {
	client, err := getClientRaw(cmd)
	if err != nil {
		return err
	}
	spec, err := buildSandboxProcessCreateSpec(args[1:])
	if err != nil {
		return err
	}
	process, err := client.Sandbox(args[0]).CreateProcess(cmd.Context(), spec)
	if err != nil {
		return err
	}
	return getFormatter().Format(os.Stdout, process)
}

func runSandboxProcessList(cmd *cobra.Command, sandboxID string) error {
	client, err := getClientRaw(cmd)
	if err != nil {
		return err
	}
	processes, err := client.Sandbox(sandboxID).ListProcesses(cmd.Context())
	if err != nil {
		return err
	}
	return getFormatter().Format(os.Stdout, processes)
}

func runSandboxProcessGet(cmd *cobra.Command, sandboxID, processID string) error {
	client, err := getClientRaw(cmd)
	if err != nil {
		return err
	}
	process, err := client.Sandbox(sandboxID).GetProcess(cmd.Context(), processID)
	if err != nil {
		return err
	}
	return getFormatter().Format(os.Stdout, process)
}

func runSandboxProcessDelete(cmd *cobra.Command, sandboxID, processID string) error {
	client, err := getClientRaw(cmd)
	if err != nil {
		return err
	}
	resp, err := client.Sandbox(sandboxID).DeleteProcess(cmd.Context(), processID)
	if err != nil {
		return err
	}
	return getFormatter().Format(os.Stdout, resp)
}

func runSandboxProcessInput(cmd *cobra.Command, sandboxID, processID string) error {
	if !cmd.Flags().Changed("data") {
		return errors.New("--data is required")
	}
	eventType, err := parseProcessInputType(processInputType)
	if err != nil {
		return err
	}
	event, err := sandbox0.NewProcessInputEvent(processInputEventID, processInputChannel, eventType, map[string]any{"data": processInputData})
	if err != nil {
		return err
	}
	client, err := getClientRaw(cmd)
	if err != nil {
		return err
	}
	accepted, err := client.Sandbox(sandboxID).SendProcessEvent(cmd.Context(), processID, event)
	if err != nil {
		return err
	}
	return getFormatter().Format(os.Stdout, accepted)
}

func runSandboxProcessEvents(cmd *cobra.Command, sandboxID, processID string) error {
	client, err := getClientRaw(cmd)
	if err != nil {
		return err
	}
	var opts *sandbox0.ProcessEventWatchOptions
	if processEventsCursor >= 0 {
		cursor := processEventsCursor
		opts = &sandbox0.ProcessEventWatchOptions{Cursor: &cursor}
	}
	stream, err := client.Sandbox(sandboxID).WatchProcessEvents(cmd.Context(), processID, opts)
	if err != nil {
		return err
	}
	defer stream.Close()

	encoder := json.NewEncoder(os.Stdout)
	for {
		event, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := encoder.Encode(event); err != nil {
			return err
		}
	}
}

func runSandboxProcessSignal(cmd *cobra.Command, sandboxID, processID, signal string) error {
	client, err := getClientRaw(cmd)
	if err != nil {
		return err
	}
	resp, err := client.Sandbox(sandboxID).SignalProcess(cmd.Context(), processID, signal)
	if err != nil {
		return err
	}
	return getFormatter().Format(os.Stdout, resp)
}

func runSandboxProcessResize(cmd *cobra.Command, sandboxID, processID string) error {
	if processResizeRows <= 0 || processResizeCols <= 0 {
		return errors.New("--rows and --cols must be > 0")
	}
	client, err := getClientRaw(cmd)
	if err != nil {
		return err
	}
	resp, err := client.Sandbox(sandboxID).ResizeProcessPTY(cmd.Context(), processID, processResizeChannel, uint16(processResizeRows), uint16(processResizeCols))
	if err != nil {
		return err
	}
	return getFormatter().Format(os.Stdout, resp)
}

func buildSandboxProcessCreateSpec(command []string) (apispec.ProcessSpec, error) {
	if processCreateSpecFile != "" {
		return readSandboxProcessSpecFile(processCreateSpecFile)
	}
	if len(command) == 0 {
		return apispec.ProcessSpec{}, errors.New("command is required unless --spec-file is set")
	}

	kind, err := parseProcessChannelKind(processCreateKind)
	if err != nil {
		return apispec.ProcessSpec{}, err
	}
	if kind != apispec.ProcessChannelKindStdio && kind != apispec.ProcessChannelKindPty {
		return apispec.ProcessSpec{}, fmt.Errorf("command mode supports stdio and pty channels; use --spec-file for %s", kind)
	}
	framing, err := parseProcessChannelFraming(processCreateFraming)
	if err != nil {
		return apispec.ProcessSpec{}, err
	}

	channelName := processCreateChannel
	if channelName == "" {
		channelName = string(kind)
	}
	spec := apispec.ProcessSpec{
		Command: command,
		Channels: []apispec.ProcessChannelSpec{
			{
				Name:    channelName,
				Kind:    kind,
				Framing: apispec.NewOptProcessChannelFraming(framing),
				Stdin:   apispec.NewOptBool(true),
				Stdout:  apispec.NewOptBool(true),
				Stderr:  apispec.NewOptBool(true),
			},
		},
	}
	if kind == apispec.ProcessChannelKindPty && (processCreatePTYRows > 0 || processCreatePTYCols > 0) {
		spec.Channels[0].PtySize = apispec.NewOptPTYSize(apispec.PTYSize{
			Rows: apispec.NewOptInt32(processCreatePTYRows),
			Cols: apispec.NewOptInt32(processCreatePTYCols),
		})
	}
	if err := applySandboxProcessCreateOptions(&spec); err != nil {
		return apispec.ProcessSpec{}, err
	}
	return spec, nil
}

func applySandboxProcessCreateOptions(spec *apispec.ProcessSpec) error {
	if processCreateAlias != "" {
		spec.Alias = apispec.NewOptString(processCreateAlias)
	}
	if processCreateCwd != "" {
		spec.Cwd = apispec.NewOptString(processCreateCwd)
	}
	if len(processCreateEnv) > 0 {
		env := make(apispec.ProcessSpecEnvVars, len(processCreateEnv))
		for _, entry := range processCreateEnv {
			key, value, ok := strings.Cut(entry, "=")
			if !ok || key == "" {
				return fmt.Errorf("invalid --env %q, expected KEY=VALUE", entry)
			}
			env[key] = value
		}
		if len(env) > 0 {
			spec.EnvVars = apispec.NewOptProcessSpecEnvVars(env)
		}
	}
	if processCreateEventBufferSize > 0 {
		spec.EventBufferSize = apispec.NewOptInt32(processCreateEventBufferSize)
	}
	if processCreateInputBufferSize > 0 {
		spec.InputBufferSize = apispec.NewOptInt32(processCreateInputBufferSize)
	}
	return nil
}

func readSandboxProcessSpecFile(path string) (apispec.ProcessSpec, error) {
	var r io.Reader
	if path == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return apispec.ProcessSpec{}, err
		}
		defer f.Close()
		r = f
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return apispec.ProcessSpec{}, err
	}
	var spec apispec.ProcessSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return apispec.ProcessSpec{}, err
	}
	return spec, nil
}

func parseProcessChannelKind(value string) (apispec.ProcessChannelKind, error) {
	var kind apispec.ProcessChannelKind
	if err := kind.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(value)))); err != nil {
		return "", err
	}
	return kind, nil
}

func parseProcessChannelFraming(value string) (apispec.ProcessChannelFraming, error) {
	var framing apispec.ProcessChannelFraming
	if err := framing.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(value)))); err != nil {
		return "", err
	}
	return framing, nil
}

func parseProcessInputType(value string) (apispec.ProcessEventType, error) {
	var eventType apispec.ProcessEventType
	if err := eventType.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(value)))); err != nil {
		return "", err
	}
	switch eventType {
	case apispec.ProcessEventTypeStdinWrite, apispec.ProcessEventTypePtyInput:
		return eventType, nil
	default:
		return "", fmt.Errorf("unsupported input event type %q", value)
	}
}
