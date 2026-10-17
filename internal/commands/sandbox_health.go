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

type sandboxHealthClient interface {
	GetSandboxHealth(context.Context, string) (*apispec.SandboxHealth, error)
}

var _ sandboxHealthClient = (*sandbox0.Client)(nil)

func init() {
	sandboxCmd.AddCommand(newSandboxHealthCommand(func(cmd *cobra.Command) (sandboxHealthClient, error) { return getClientRaw(cmd) }))
}

func newSandboxHealthCommand(factory func(*cobra.Command) (sandboxHealthClient, error)) *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use: "health <sandbox-id>", Short: "Read independent runtime, execution, terminal and storage observations",
		Long: "Read independent runtime, execution, terminal and storage observations.\nMay run bounded guest diagnostics. Does not resume or restart the sandbox.\nGuest observations are advisory; unknown is not available or unavailable.\nExit status reports whether the query succeeded, not overall sandbox health.",
		Args: cobra.ExactArgs(1), SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if timeout <= 0 {
				return fmt.Errorf("--timeout must be positive")
			}
			if args[0] == "" || strings.TrimSpace(args[0]) != args[0] {
				return fmt.Errorf("invalid sandbox identity")
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
			health, err := client.GetSandboxHealth(ctx, args[0])
			if err != nil {
				return err
			}
			if health == nil {
				return fmt.Errorf("health returned no observation")
			}
			return getFormatter().Format(cmd.OutOrStdout(), health)
		},
	}
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Second, "Local query timeout")
	return cmd
}
