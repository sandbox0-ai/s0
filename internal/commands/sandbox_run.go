package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	runAlias     string
	runContextID string
)

// sandboxRunCmd executes input in a REPL context.
var sandboxRunCmd = &cobra.Command{
	Use:   "run <sandbox-id> <input>",
	Short: "Execute input in a REPL context",
	Long: `Execute input in a REPL context and wait for completion.

Unlike 'sandbox exec', this command targets a REPL context and preserves state.
By default it reuses the only running REPL context with the same alias in the
sandbox, or creates one when none exists.

Examples:
  s0 sandbox run sb_abc123 "x = 2"
  s0 sandbox run sb_abc123 "print(x)"
  s0 sandbox run sb_abc123 --alias bash "echo hello"
  s0 sandbox run sb_abc123 --context-id ctx_abc123 "print(x)"`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]
		input := args[1]
		if strings.TrimSpace(input) == "" {
			fmt.Fprintln(os.Stderr, "Error: input cannot be empty")
			os.Exit(1)
		}

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		contextID, err := resolveRunContextID(cmd.Context(), client, sandboxID, runAlias, runContextID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving run context: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(sandboxID).ContextExec(cmd.Context(), contextID, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing input: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(result.OutputRaw)
	},
}

func resolveRunContextID(ctx context.Context, client *sandbox0.Client, sandboxID, alias, explicitContextID string) (string, error) {
	if strings.TrimSpace(explicitContextID) != "" {
		return explicitContextID, nil
	}

	normalizedAlias := strings.TrimSpace(alias)
	if normalizedAlias == "" {
		normalizedAlias = "python"
	}

	contexts, err := client.Sandbox(sandboxID).ListContext(ctx)
	if err != nil {
		return "", err
	}

	contextID, err := findReusableRunContext(contexts, normalizedAlias)
	if err != nil {
		return "", fmt.Errorf("select reusable context: %w", err)
	}
	if contextID != "" {
		return contextID, nil
	}

	contextResp, err := client.Sandbox(sandboxID).CreateContext(ctx, apispec.CreateContextRequest{
		Type: apispec.NewOptProcessType(apispec.ProcessTypeRepl),
		Repl: apispec.NewOptCreateREPLContextRequest(apispec.CreateREPLContextRequest{
			Alias: apispec.NewOptString(normalizedAlias),
		}),
	})
	if err != nil {
		return "", err
	}
	if contextResp == nil || strings.TrimSpace(contextResp.ID) == "" {
		return "", fmt.Errorf("create context returned empty ID")
	}
	return contextResp.ID, nil
}

func findReusableRunContext(contexts []apispec.ContextResponse, alias string) (string, error) {
	normalizedAlias := strings.TrimSpace(alias)
	if normalizedAlias == "" {
		normalizedAlias = "python"
	}

	match := ""
	for _, ctx := range contexts {
		if ctx.Type != apispec.ProcessTypeRepl || !ctx.Running || ctx.Paused {
			continue
		}

		ctxAlias, ok := ctx.Alias.Get()
		if !ok || ctxAlias != normalizedAlias {
			continue
		}

		if match != "" {
			return "", fmt.Errorf("multiple running REPL contexts found for alias %q; use --context-id", normalizedAlias)
		}
		match = ctx.ID
	}
	return match, nil
}

func init() {
	sandboxRunCmd.Flags().StringVar(&runAlias, "alias", "python", "REPL alias to use when resolving or creating a context")
	sandboxRunCmd.Flags().StringVar(&runContextID, "context-id", "", "explicit REPL context ID to run against")

	sandboxCmd.AddCommand(sandboxRunCmd)
}
