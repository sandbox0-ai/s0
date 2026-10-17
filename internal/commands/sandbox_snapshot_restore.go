package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

type sandboxSnapshotRestoreClient interface {
	RestoreSandboxSnapshot(context.Context, string, apispec.SandboxSnapshotRestoreRequest, string) (*apispec.SandboxSnapshotRestoreOperation, error)
	GetSandboxSnapshotRestoreOperation(context.Context, string, string) (*apispec.SandboxSnapshotRestoreOperation, error)
	FindSandboxSnapshotRestoreOperation(context.Context, string, string) (*apispec.SandboxSnapshotRestoreOperation, error)
}
type sandboxSnapshotRestoreClientFactory func(*cobra.Command) (sandboxSnapshotRestoreClient, error)

var _ sandboxSnapshotRestoreClient = (*sandbox0.Client)(nil)

func init() {
	factory := func(cmd *cobra.Command) (sandboxSnapshotRestoreClient, error) { return getClientRaw(cmd) }
	sandboxCmd.AddCommand(newSandboxSnapshotRestoreCommand(factory, false), newSandboxSnapshotRestoreCommand(factory, true))
}

func newSandboxSnapshotRestoreCommand(factory sandboxSnapshotRestoreClientFactory, readOnly bool) *cobra.Command {
	var key, operationID, snapshotID, snapshotGeneration string
	var discard bool
	var generation int64
	var wait bool
	var timeout, interval time.Duration
	use, short := "restore-snapshot <sandbox-id>", "Replace writable RootFS with a selected snapshot and end old processes"
	if readOnly {
		use, short = "restore-snapshot-status <sandbox-id>", "Read an existing snapshot restore operation"
	}
	cmd := &cobra.Command{Use: use, Short: short, Args: cobra.ExactArgs(1), SilenceUsage: true,
		Long: short + ".\nA snapshot restore discards process and memory state and replaces writable RootFS with the selected snapshot.\nKeep the original generation, snapshot identity and idempotency key after an uncertain response.\nWaiting or Ctrl-C only affects this client; accepted server work continues.",
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
			} else if !cmd.Flags().Changed("expected-runtime-generation") || generation < 1 || generation >= 9007199254740991 {
				return fmt.Errorf("--expected-runtime-generation is required and must be between 1 and 9007199254740990")
			}
			if !readOnly && (!discard || snapshotID == "" || snapshotGeneration == "" || strings.TrimSpace(snapshotID) != snapshotID || strings.TrimSpace(snapshotGeneration) != snapshotGeneration || len(snapshotID) > 512 || len(snapshotGeneration) > 512 || strings.ContainsRune(snapshotID, 0) || strings.ContainsRune(snapshotGeneration, 0)) {
				return fmt.Errorf("--discard-process-state, --snapshot-id and --snapshot-generation-id are required")
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
			var operation *apispec.SandboxSnapshotRestoreOperation
			switch {
			case !readOnly:
				operation, err = client.RestoreSandboxSnapshot(ctx, args[0], apispec.SandboxSnapshotRestoreRequest{ExpectedRuntimeGeneration: generation, SnapshotID: snapshotID, SnapshotGenerationID: snapshotGeneration, DiscardProcessState: true}, key)
			case operationID != "":
				operation, err = client.GetSandboxSnapshotRestoreOperation(ctx, args[0], operationID)
			default:
				operation, err = client.FindSandboxSnapshotRestoreOperation(ctx, args[0], key)
			}
			if err != nil {
				if !readOnly {
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "No successful acknowledgement was received. Query the original request without restoring again:\n  s0 sandbox restore-snapshot-status %s --idempotency-key %s\nAny retry must keep --expected-runtime-generation %d, --idempotency-key %s and the original snapshot ID/generation and discard acknowledgement.\n", args[0], key, generation, key)
				}
				return err
			}
			if operation == nil {
				return fmt.Errorf("snapshot restore returned no operation receipt")
			}
			if wait {
				operation, err = waitSandboxSnapshotRestore(ctx, client, args[0], operation, interval)
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
				return fmt.Errorf("snapshot restore operation %s ended as %s", operation.ID, operation.State)
			}
			return nil
		}}
	cmd.Flags().StringVar(&key, "idempotency-key", "", "Stable original request key; never change it to retry an uncertain submission")
	if readOnly {
		cmd.Flags().StringVar(&operationID, "operation-id", "", "Existing snapshot restore operation ID")
	} else {
		cmd.Flags().StringVar(&snapshotID, "snapshot-id", "", "Exact selected snapshot ID (required)")
		cmd.Flags().StringVar(&snapshotGeneration, "snapshot-generation-id", "", "Exact selected snapshot generation (required)")
		cmd.Flags().BoolVar(&discard, "discard-process-state", false, "Explicitly acknowledge ending old processes and replacing writable RootFS")
		cmd.Flags().Int64Var(&generation, "expected-runtime-generation", 0, "Exact observed source generation (required; never refreshed automatically)")
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "Poll the same operation until terminal")
	cmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "Local request/wait timeout; does not cancel accepted work")
	cmd.Flags().DurationVar(&interval, "poll-interval", time.Second, "Delay between read-only polls")
	return cmd
}

func waitSandboxSnapshotRestore(ctx context.Context, client sandboxSnapshotRestoreClient, sandbox string, operation *apispec.SandboxSnapshotRestoreOperation, interval time.Duration) (*apispec.SandboxSnapshotRestoreOperation, error) {
	for {
		switch operation.State {
		case "succeeded", "failed", "superseded":
			return operation, nil
		case "pausing", "paused", "resuming":
		default:
			return operation, fmt.Errorf("unknown snapshot restore operation state")
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return operation, ctx.Err()
		case <-timer.C:
		}
		next, err := client.GetSandboxSnapshotRestoreOperation(ctx, sandbox, operation.ID)
		if err != nil {
			return operation, err
		}
		if next == nil || next.ID != operation.ID || next.SandboxID != operation.SandboxID || next.RequestKey != operation.RequestKey || next.FromRuntimeGeneration != operation.FromRuntimeGeneration || next.SnapshotID != operation.SnapshotID || next.SnapshotGenerationID != operation.SnapshotGenerationID || !bool(next.DiscardProcessState) || (operation.RestoredRootfsGenerationID.Set && next.RestoredRootfsGenerationID != operation.RestoredRootfsGenerationID) {
			return operation, fmt.Errorf("snapshot restore receipt identity changed")
		}
		operation = next
	}
}
