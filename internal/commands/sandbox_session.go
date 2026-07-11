package commands

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	sessionSpecFile          string
	sessionName              string
	sessionCWD               string
	sessionEnv               []string
	sessionPTY               bool
	sessionRows              int32
	sessionCols              int32
	sessionTerm              string
	sessionRestartPolicy     string
	sessionRuntimeRecovery   string
	sessionIdleTimeout       int64
	sessionMaxLifetime       int64
	sessionStopGrace         int32
	sessionReadiness         string
	sessionReadyDelay        int32
	sessionReadyOutput       string
	sessionReadyTimeout      int32
	sessionIdempotencyKey    string
	sessionReplaceAttempt    bool
	sessionInputData         string
	sessionInputBase64       string
	sessionInputFile         string
	sessionInputEOF          bool
	sessionInputID           string
	sessionExpectedAttemptID string
	sessionEventAfter        int64
	sessionEventLimit        int
	sessionEventFollow       bool
	sessionEventLastID       string
)

var sandboxSessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage durable execution sessions",
	Long:  `Create, inspect, control, and stream durable process-backed sessions in a sandbox.`,
}

var sandboxSessionListCmd = &cobra.Command{
	Use:   "list <sandbox-id>",
	Short: "List execution sessions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClientRaw(cmd)
		if err != nil {
			return err
		}
		values, err := client.Sandbox(args[0]).ListSessions(cmd.Context())
		if err != nil {
			return fmt.Errorf("list sessions: %w", err)
		}
		return getFormatter().Format(os.Stdout, values)
	},
}

var sandboxSessionCreateCmd = &cobra.Command{
	Use:   "create <sandbox-id> [--] <command> [args...]",
	Short: "Create an execution session",
	Args:  validateSessionCreateArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClientRaw(cmd)
		if err != nil {
			return err
		}
		spec, err := buildExecutionSessionSpec(args[1:])
		if err != nil {
			return err
		}
		options := &sandbox0.CreateSessionOptions{IdempotencyKey: sessionIdempotencyKey}
		value, err := client.Sandbox(args[0]).CreateSession(cmd.Context(), spec, options)
		if err != nil {
			return fmt.Errorf("create session: %w", err)
		}
		return getFormatter().Format(os.Stdout, value)
	},
}

var sandboxSessionGetCmd = &cobra.Command{
	Use:   "get <sandbox-id> <session-id>",
	Short: "Get an execution session",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClientRaw(cmd)
		if err != nil {
			return err
		}
		value, err := client.Sandbox(args[0]).GetSession(cmd.Context(), args[1])
		if err != nil {
			return fmt.Errorf("get session: %w", err)
		}
		return getFormatter().Format(os.Stdout, value)
	},
}

var sandboxSessionUpdateCmd = &cobra.Command{
	Use:   "update <sandbox-id> <session-id>",
	Short: "Replace an execution session specification",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(sessionSpecFile) == "" {
			return fmt.Errorf("--spec-file is required")
		}
		client, err := getClientRaw(cmd)
		if err != nil {
			return err
		}
		spec, err := readExecutionSessionSpec(sessionSpecFile)
		if err != nil {
			return err
		}
		value, err := client.Sandbox(args[0]).UpdateSession(cmd.Context(), args[1], spec)
		if err != nil {
			return fmt.Errorf("update session: %w", err)
		}
		return getFormatter().Format(os.Stdout, value)
	},
}

var sandboxSessionDeleteCmd = &cobra.Command{
	Use:   "delete <sandbox-id> <session-id>",
	Short: "Stop and delete an execution session",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClientRaw(cmd)
		if err != nil {
			return err
		}
		if _, err := client.Sandbox(args[0]).DeleteSession(cmd.Context(), args[1]); err != nil {
			return fmt.Errorf("delete session: %w", err)
		}
		_, err = fmt.Fprintf(os.Stdout, "Session %s deleted successfully\n", args[1])
		return err
	},
}

func newSandboxSessionStateCommand(name string, state apispec.ExecutionSessionDesiredState) *cobra.Command {
	return &cobra.Command{
		Use:   name + " <sandbox-id> <session-id>",
		Short: strings.ToUpper(name[:1]) + name[1:] + " an execution session",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClientRaw(cmd)
			if err != nil {
				return err
			}
			value, err := client.Sandbox(args[0]).SetSessionDesiredState(cmd.Context(), args[1], state)
			if err != nil {
				return fmt.Errorf("%s session: %w", name, err)
			}
			return getFormatter().Format(os.Stdout, value)
		},
	}
}

var sandboxSessionStartCmd = newSandboxSessionStateCommand("start", apispec.ExecutionSessionDesiredStateRunning)
var sandboxSessionStopCmd = newSandboxSessionStateCommand("stop", apispec.ExecutionSessionDesiredStateStopped)

var sandboxSessionAttemptCmd = &cobra.Command{
	Use:   "attempt <sandbox-id> <session-id>",
	Short: "Start a new process attempt",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClientRaw(cmd)
		if err != nil {
			return err
		}
		value, err := client.Sandbox(args[0]).CreateSessionAttempt(cmd.Context(), args[1], sessionReplaceAttempt)
		if err != nil {
			return fmt.Errorf("create session attempt: %w", err)
		}
		return getFormatter().Format(os.Stdout, value)
	},
}

var sandboxSessionInputCmd = &cobra.Command{
	Use:   "input <sandbox-id> <session-id>",
	Short: "Append binary-safe input or explicit EOF",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		request, err := buildExecutionSessionInputRequest()
		if err != nil {
			return err
		}
		client, err := getClientRaw(cmd)
		if err != nil {
			return err
		}
		value, err := client.Sandbox(args[0]).WriteSessionInput(cmd.Context(), args[1], request)
		if err != nil {
			return fmt.Errorf("write session input: %w", err)
		}
		return getFormatter().Format(os.Stdout, value)
	},
}

var sandboxSessionSignalCmd = &cobra.Command{
	Use:   "signal <sandbox-id> <session-id> <signal>",
	Short: "Send a signal to the current attempt",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		request := apispec.ExecutionSessionSignalRequest{Signal: args[2]}
		if sessionExpectedAttemptID != "" {
			request.ExpectedAttemptID = apispec.NewOptString(sessionExpectedAttemptID)
		}
		client, err := getClientRaw(cmd)
		if err != nil {
			return err
		}
		if _, err := client.Sandbox(args[0]).SendSessionSignal(cmd.Context(), args[1], request); err != nil {
			return fmt.Errorf("signal session: %w", err)
		}
		_, err = fmt.Fprintln(os.Stdout, "Signal accepted")
		return err
	},
}

var sandboxSessionResizeCmd = &cobra.Command{
	Use:   "resize <sandbox-id> <session-id> <rows> <cols>",
	Short: "Resize the current PTY attempt",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		rows, err := strconv.ParseInt(args[2], 10, 32)
		if err != nil || rows <= 0 {
			return fmt.Errorf("rows must be a positive integer")
		}
		cols, err := strconv.ParseInt(args[3], 10, 32)
		if err != nil || cols <= 0 {
			return fmt.Errorf("cols must be a positive integer")
		}
		request := apispec.ExecutionSessionTerminalResizeRequest{Rows: int32(rows), Cols: int32(cols)}
		if sessionExpectedAttemptID != "" {
			request.ExpectedAttemptID = apispec.NewOptString(sessionExpectedAttemptID)
		}
		client, err := getClientRaw(cmd)
		if err != nil {
			return err
		}
		if _, err := client.Sandbox(args[0]).ResizeSessionTerminal(cmd.Context(), args[1], request); err != nil {
			return fmt.Errorf("resize session terminal: %w", err)
		}
		_, err = fmt.Fprintln(os.Stdout, "Terminal resized")
		return err
	},
}

var sandboxSessionEventsCmd = &cobra.Command{
	Use:   "events <sandbox-id> <session-id>",
	Short: "Read or follow the durable event journal",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := getClientRaw(cmd)
		if err != nil {
			return err
		}
		sandbox := client.Sandbox(args[0])
		if sessionEventFollow {
			stream, err := sandbox.WatchSessionEvents(cmd.Context(), args[1], &sandbox0.SessionEventStreamOptions{
				After: sessionEventAfter, LastEventID: sessionEventLastID,
			})
			if err != nil {
				return fmt.Errorf("watch session events: %w", err)
			}
			defer stream.Close()
			for {
				event, err := stream.Recv()
				if err == io.EOF {
					return nil
				}
				if err != nil {
					return fmt.Errorf("read session event: %w", err)
				}
				if err := writeExecutionSessionEvent(os.Stdout, event); err != nil {
					return err
				}
			}
		}
		page, err := sandbox.ListSessionEvents(cmd.Context(), args[1], &sandbox0.SessionEventOptions{
			After: sessionEventAfter, Limit: sessionEventLimit,
		})
		if err != nil {
			return fmt.Errorf("list session events: %w", err)
		}
		return getFormatter().Format(os.Stdout, page)
	},
}

func init() {
	sandboxCmd.AddCommand(sandboxSessionCmd)
	sandboxSessionCmd.AddCommand(
		sandboxSessionListCmd,
		sandboxSessionCreateCmd,
		sandboxSessionGetCmd,
		sandboxSessionUpdateCmd,
		sandboxSessionDeleteCmd,
		sandboxSessionStartCmd,
		sandboxSessionStopCmd,
		sandboxSessionAttemptCmd,
		sandboxSessionInputCmd,
		sandboxSessionSignalCmd,
		sandboxSessionResizeCmd,
		sandboxSessionEventsCmd,
	)

	sandboxSessionCreateCmd.Flags().StringVarP(&sessionSpecFile, "spec-file", "f", "", "session spec YAML/JSON file, or - for stdin")
	sandboxSessionCreateCmd.Flags().StringVar(&sessionName, "name", "", "stable human-readable session name")
	sandboxSessionCreateCmd.Flags().StringVar(&sessionCWD, "cwd", "", "working directory")
	sandboxSessionCreateCmd.Flags().StringArrayVarP(&sessionEnv, "env", "e", nil, "environment variable KEY=VALUE (repeatable)")
	sandboxSessionCreateCmd.Flags().BoolVar(&sessionPTY, "pty", false, "allocate a PTY instead of separate pipes")
	sandboxSessionCreateCmd.Flags().Int32Var(&sessionRows, "rows", 24, "initial PTY rows")
	sandboxSessionCreateCmd.Flags().Int32Var(&sessionCols, "cols", 80, "initial PTY columns")
	sandboxSessionCreateCmd.Flags().StringVar(&sessionTerm, "term", "xterm-256color", "PTY TERM value")
	sandboxSessionCreateCmd.Flags().StringVar(&sessionRestartPolicy, "restart", "never", "restart policy (never, on_failure, always)")
	sandboxSessionCreateCmd.Flags().StringVar(&sessionRuntimeRecovery, "runtime-recovery", "restart", "runtime replacement policy (restart, stop)")
	sandboxSessionCreateCmd.Flags().Int64Var(&sessionIdleTimeout, "idle-timeout", 0, "idle timeout in seconds; 0 disables it")
	sandboxSessionCreateCmd.Flags().Int64Var(&sessionMaxLifetime, "max-lifetime", 0, "maximum lifetime in seconds; 0 disables it")
	sandboxSessionCreateCmd.Flags().Int32Var(&sessionStopGrace, "stop-grace", 10, "SIGTERM grace period in seconds")
	sandboxSessionCreateCmd.Flags().StringVar(&sessionReadiness, "readiness", "process", "readiness mode (process, delay, output)")
	sandboxSessionCreateCmd.Flags().Int32Var(&sessionReadyDelay, "ready-delay-ms", 0, "delay readiness threshold in milliseconds")
	sandboxSessionCreateCmd.Flags().StringVar(&sessionReadyOutput, "ready-output", "", "output token that marks the attempt ready")
	sandboxSessionCreateCmd.Flags().Int32Var(&sessionReadyTimeout, "ready-timeout-ms", 30000, "readiness timeout in milliseconds")
	sandboxSessionCreateCmd.Flags().StringVar(&sessionIdempotencyKey, "idempotency-key", "", "safe create retry key")
	sandboxSessionUpdateCmd.Flags().StringVarP(&sessionSpecFile, "spec-file", "f", "", "replacement session spec YAML/JSON file, or - for stdin")
	sandboxSessionAttemptCmd.Flags().BoolVar(&sessionReplaceAttempt, "replace", false, "stop and replace a running attempt")
	sandboxSessionInputCmd.Flags().StringVar(&sessionInputData, "data", "", "UTF-8 input data")
	sandboxSessionInputCmd.Flags().StringVar(&sessionInputBase64, "data-base64", "", "base64-encoded input bytes")
	sandboxSessionInputCmd.Flags().StringVarP(&sessionInputFile, "file", "f", "", "read input bytes from a file, or - for stdin")
	sandboxSessionInputCmd.Flags().BoolVar(&sessionInputEOF, "eof", false, "close stdin after queued input")
	sandboxSessionInputCmd.Flags().StringVar(&sessionInputID, "input-id", "", "idempotency key for this input operation")
	addSessionExpectedAttemptFlag(sandboxSessionInputCmd)
	addSessionExpectedAttemptFlag(sandboxSessionSignalCmd)
	addSessionExpectedAttemptFlag(sandboxSessionResizeCmd)
	sandboxSessionEventsCmd.Flags().Int64Var(&sessionEventAfter, "after", 0, "return events after this sequence")
	sandboxSessionEventsCmd.Flags().IntVar(&sessionEventLimit, "limit", 1000, "maximum retained events to return")
	sandboxSessionEventsCmd.Flags().BoolVarP(&sessionEventFollow, "follow", "f", false, "follow retained and live events over SSE")
	sandboxSessionEventsCmd.Flags().StringVar(&sessionEventLastID, "last-event-id", "", "SSE resume cursor; takes precedence over --after")
}

func addSessionExpectedAttemptFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&sessionExpectedAttemptID, "attempt-id", "", "reject the operation if the current attempt differs")
}

func validateSessionCreateArgs(_ *cobra.Command, args []string) error {
	if sessionSpecFile != "" {
		if len(args) != 1 {
			return fmt.Errorf("create with --spec-file expects only <sandbox-id>")
		}
		return nil
	}
	if len(args) < 2 {
		return fmt.Errorf("create requires <sandbox-id> and a command, or --spec-file")
	}
	return nil
}

func buildExecutionSessionSpec(command []string) (apispec.ExecutionSessionSpec, error) {
	if sessionSpecFile != "" {
		return readExecutionSessionSpec(sessionSpecFile)
	}
	if len(command) == 0 {
		return apispec.ExecutionSessionSpec{}, fmt.Errorf("command is required")
	}
	env, err := parseSessionEnv(sessionEnv)
	if err != nil {
		return apispec.ExecutionSessionSpec{}, err
	}
	var restart apispec.ExecutionSessionRestartPolicy
	if err := restart.UnmarshalText([]byte(sessionRestartPolicy)); err != nil {
		return apispec.ExecutionSessionSpec{}, fmt.Errorf("invalid --restart %q", sessionRestartPolicy)
	}
	var recovery apispec.ExecutionSessionRuntimeRecoveryPolicy
	if err := recovery.UnmarshalText([]byte(sessionRuntimeRecovery)); err != nil {
		return apispec.ExecutionSessionSpec{}, fmt.Errorf("invalid --runtime-recovery %q", sessionRuntimeRecovery)
	}
	var readinessType apispec.ExecutionSessionReadinessType
	if err := readinessType.UnmarshalText([]byte(sessionReadiness)); err != nil {
		return apispec.ExecutionSessionSpec{}, fmt.Errorf("invalid --readiness %q", sessionReadiness)
	}
	if readinessType == apispec.ExecutionSessionReadinessTypeOutput && sessionReadyOutput == "" {
		return apispec.ExecutionSessionSpec{}, fmt.Errorf("--ready-output is required when --readiness=output")
	}
	if sessionStopGrace < 0 || sessionIdleTimeout < 0 || sessionMaxLifetime < 0 || sessionReadyDelay < 0 || sessionReadyTimeout < 0 {
		return apispec.ExecutionSessionSpec{}, fmt.Errorf("session timing values cannot be negative")
	}

	spec := apispec.ExecutionSessionSpec{
		Command: append([]string(nil), command...),
		Lifecycle: apispec.NewOptExecutionSessionLifecycleSpec(apispec.ExecutionSessionLifecycleSpec{
			Restart: apispec.NewOptExecutionSessionRestartSpec(apispec.ExecutionSessionRestartSpec{
				Policy: apispec.NewOptExecutionSessionRestartPolicy(restart),
			}),
			RuntimeRecovery:        apispec.NewOptExecutionSessionRuntimeRecoveryPolicy(recovery),
			IdleTimeoutSeconds:     apispec.NewOptInt64(sessionIdleTimeout),
			MaxLifetimeSeconds:     apispec.NewOptInt64(sessionMaxLifetime),
			StopGracePeriodSeconds: apispec.NewOptInt32(sessionStopGrace),
		}),
		Readiness: apispec.NewOptExecutionSessionReadinessSpec(apispec.ExecutionSessionReadinessSpec{
			Type:      apispec.NewOptExecutionSessionReadinessType(readinessType),
			DelayMs:   apispec.NewOptInt32(sessionReadyDelay),
			Output:    apispec.NewOptString(sessionReadyOutput),
			TimeoutMs: apispec.NewOptInt32(sessionReadyTimeout),
		}),
	}
	if sessionName != "" {
		spec.Name = apispec.NewOptString(sessionName)
	}
	if sessionCWD != "" {
		spec.Cwd = apispec.NewOptString(sessionCWD)
	}
	if len(env) > 0 {
		spec.Env = apispec.NewOptExecutionSessionSpecEnv(apispec.ExecutionSessionSpecEnv(env))
	}
	mode := apispec.ExecutionSessionIOModePipes
	ioSpec := apispec.ExecutionSessionIOSpec{}
	if sessionPTY {
		if sessionRows <= 0 || sessionCols <= 0 {
			return apispec.ExecutionSessionSpec{}, fmt.Errorf("--rows and --cols must be positive")
		}
		mode = apispec.ExecutionSessionIOModePty
		ioSpec.Terminal = apispec.NewOptExecutionSessionTerminalSpec(apispec.ExecutionSessionTerminalSpec{
			Rows: apispec.NewOptInt32(sessionRows),
			Cols: apispec.NewOptInt32(sessionCols),
			Term: apispec.NewOptString(sessionTerm),
		})
	}
	ioSpec.Mode = apispec.NewOptExecutionSessionIOMode(mode)
	spec.Io = apispec.NewOptExecutionSessionIOSpec(ioSpec)
	return spec, nil
}

func readExecutionSessionSpec(path string) (apispec.ExecutionSessionSpec, error) {
	data, err := readConfigFile(path)
	if err != nil {
		return apispec.ExecutionSessionSpec{}, err
	}
	var spec apispec.ExecutionSessionSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return apispec.ExecutionSessionSpec{}, fmt.Errorf("parse session spec: %w", err)
	}
	if len(spec.Command) == 0 {
		return apispec.ExecutionSessionSpec{}, fmt.Errorf("session spec command is required")
	}
	return spec, nil
}

func parseSessionEnv(values []string) (map[string]string, error) {
	env := make(map[string]string, len(values))
	for _, value := range values {
		key, item, ok := strings.Cut(value, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid --env %q: expected KEY=VALUE", value)
		}
		env[key] = item
	}
	return env, nil
}

func buildExecutionSessionInputRequest() (apispec.ExecutionSessionInputRequest, error) {
	sources := 0
	if sessionInputData != "" {
		sources++
	}
	if sessionInputBase64 != "" {
		sources++
	}
	if sessionInputFile != "" {
		sources++
	}
	if sources > 1 {
		return apispec.ExecutionSessionInputRequest{}, fmt.Errorf("use only one of --data, --data-base64, or --file")
	}
	if sources == 0 && !sessionInputEOF {
		return apispec.ExecutionSessionInputRequest{}, fmt.Errorf("input data or --eof is required")
	}
	var data []byte
	var err error
	switch {
	case sessionInputData != "":
		data = []byte(sessionInputData)
	case sessionInputBase64 != "":
		data, err = base64.StdEncoding.DecodeString(sessionInputBase64)
		if err != nil {
			return apispec.ExecutionSessionInputRequest{}, fmt.Errorf("decode --data-base64: %w", err)
		}
	case sessionInputFile == "-":
		data, err = io.ReadAll(os.Stdin)
	case sessionInputFile != "":
		data, err = os.ReadFile(sessionInputFile)
	}
	if err != nil {
		return apispec.ExecutionSessionInputRequest{}, err
	}
	inputID := sessionInputID
	if inputID == "" {
		inputID = uuid.NewString()
	}
	request := apispec.ExecutionSessionInputRequest{
		InputID: inputID,
		EOF:     apispec.NewOptBool(sessionInputEOF),
	}
	if len(data) > 0 {
		request.DataBase64 = apispec.NewOptString(base64.StdEncoding.EncodeToString(data))
	}
	if sessionExpectedAttemptID != "" {
		request.ExpectedAttemptID = apispec.NewOptString(sessionExpectedAttemptID)
	}
	return request, nil
}

func writeExecutionSessionEvent(w io.Writer, event *apispec.ExecutionSessionEvent) error {
	if cfgFormat == "json" {
		return json.NewEncoder(w).Encode(event)
	}
	if cfgFormat == "yaml" {
		return getFormatter().Format(w, event)
	}
	attempt := "-"
	if value, ok := event.AttemptID.Get(); ok {
		attempt = value
	}
	stream := "-"
	if value, ok := event.Stream.Get(); ok {
		stream = string(value)
	}
	payload := ""
	if encoded, ok := event.DataBase64.Get(); ok {
		data, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return err
		}
		payload = strings.TrimRight(string(data), "\r\n")
	}
	_, err := fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n", event.Seq, event.OccurredAt.Format("2006-01-02 15:04:05"), attempt, event.Type, stream, payload)
	return err
}
