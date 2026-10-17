package commands

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/signal"
	"regexp"
	"time"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

type sandboxRestartClient interface {
	RestartSandbox(context.Context, string, int64, string) (*apispec.SandboxRestartOperation, error)
	GetSandboxRestartOperation(context.Context, string, string) (*apispec.SandboxRestartOperation, error)
	FindSandboxRestartOperation(context.Context, string, string) (*apispec.SandboxRestartOperation, error)
}
type sandboxRestartClientFactory func(*cobra.Command) (sandboxRestartClient, error)

var _ sandboxRestartClient = (*sandbox0.Client)(nil)
var restartKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,255}$`)
var restartIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func init() {
	factory := func(cmd *cobra.Command) (sandboxRestartClient, error) { return getClientRaw(cmd) }
	sandboxCmd.AddCommand(newSandboxRestartCommand(factory, false), newSandboxRestartCommand(factory, true))
}

func newSandboxRestartCommand(factory sandboxRestartClientFactory, readOnly bool) *cobra.Command {
	var key, operationID string
	var generation int64
	var wait bool
	var timeout, interval time.Duration
	use, short := "restart <sandbox-id>", "End old processes and start a fresh runtime from the published RootFS"
	if readOnly {
		use, short = "restart-status <sandbox-id>", "Read an existing restart operation without restarting the sandbox"
	}
	cmd := &cobra.Command{Use: use, Short: short, Args: cobra.ExactArgs(1), SilenceUsage: true,
		Long: short + ".\nA restart discards process and memory state; it is not a reconnect, memory restore, or snapshot rollback.\nKeep the original generation and idempotency key after an uncertain response.\nWaiting or Ctrl-C only affects this client; accepted server work continues.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if timeout <= 0 || interval <= 0 {
				return fmt.Errorf("--timeout and --poll-interval must be positive")
			}
			if readOnly {
				if (key == "") == (operationID == "") {
					return fmt.Errorf("specify exactly one of --operation-id or --idempotency-key")
				}
				if operationID != "" && !restartIDPattern.MatchString(operationID) {
					return fmt.Errorf("invalid --operation-id")
				}
			} else if !cmd.Flags().Changed("expected-runtime-generation") || generation < 0 || generation == math.MaxInt64 {
				return fmt.Errorf("--expected-runtime-generation is required and must be between 0 and 9223372036854775806")
			}
			if (!readOnly || key != "") && !restartKeyPattern.MatchString(key) {
				return fmt.Errorf("--idempotency-key is required and must contain 1–255 ASCII letters, digits, dots, underscores, colons or hyphens")
			}
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
			defer stop()
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			cmd.SetContext(ctx)
			client, err := factory(cmd)
			if err != nil {
				return err
			}
			var operation *apispec.SandboxRestartOperation
			switch {
			case !readOnly:
				operation, err = client.RestartSandbox(ctx, args[0], generation, key)
			case operationID != "":
				operation, err = client.GetSandboxRestartOperation(ctx, args[0], operationID)
			default:
				operation, err = client.FindSandboxRestartOperation(ctx, args[0], key)
			}
			if err != nil {
				if !readOnly {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "No successful acknowledgement was received. Query the original request without restarting:\n  s0 sandbox restart-status %s --idempotency-key %s\nAny retry must keep --expected-runtime-generation %d and --idempotency-key %s.\n", args[0], key, generation, key)
				}
				return err
			}
			if operation == nil {
				return fmt.Errorf("restart returned no operation receipt")
			}
			if wait {
				operation, err = waitSandboxRestart(ctx, client, args[0], operation, interval)
			}
			if outputErr := getFormatter().Format(cmd.OutOrStdout(), operation); outputErr != nil {
				return outputErr
			}
			if err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Waiting stopped; accepted work was not canceled. Query with --operation-id %s.\n", operation.ID)
				return err
			}
			switch operation.State {
			case "failed", "superseded":
				return fmt.Errorf("restart operation %s ended as %s", operation.ID, operation.State)
			}
			return nil
		}}
	cmd.Flags().StringVar(&key, "idempotency-key", "", "Stable original request key; never change it to retry an uncertain submission")
	if readOnly {
		cmd.Flags().StringVar(&operationID, "operation-id", "", "Existing restart operation ID")
	} else {
		cmd.Flags().Int64Var(&generation, "expected-runtime-generation", 0, "Exact observed source generation (required; never refreshed automatically)")
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "Poll the same operation until terminal")
	cmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "Local request/wait timeout; does not cancel accepted work")
	cmd.Flags().DurationVar(&interval, "poll-interval", time.Second, "Delay between read-only polls")
	return cmd
}

func waitSandboxRestart(ctx context.Context, client sandboxRestartClient, sandbox string, operation *apispec.SandboxRestartOperation, interval time.Duration) (*apispec.SandboxRestartOperation, error) {
	for {
		switch operation.State {
		case "succeeded", "failed", "superseded":
			return operation, nil
		case "pausing", "paused", "resuming":
		default:
			return operation, fmt.Errorf("unknown restart operation state")
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return operation, ctx.Err()
		case <-timer.C:
		}
		next, err := client.GetSandboxRestartOperation(ctx, sandbox, operation.ID)
		if err != nil {
			return operation, err
		}
		if next == nil || next.ID != operation.ID || next.SandboxID != operation.SandboxID || next.RequestKey != operation.RequestKey || next.FromRuntimeGeneration != operation.FromRuntimeGeneration {
			return operation, fmt.Errorf("restart receipt identity changed")
		}
		operation = next
	}
}
