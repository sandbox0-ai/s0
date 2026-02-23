package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	contextSandboxID string
	contextType      string
	contextLanguage  string
	contextCommand   []string
	contextCwd       string
	contextEnv       []string
	contextWait      bool
)

// sandboxContextCmd represents the sandbox context command group.
var sandboxContextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage contexts",
	Long:  `List, create, get, delete, restart, and manage contexts in a sandbox.`,
}

// sandboxContextListCmd lists all contexts.
var sandboxContextListCmd = &cobra.Command{
	Use:   "list",
	Short: "List contexts",
	Long:  `List all contexts in the sandbox.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(contextSandboxID).ListContext(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing contexts: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxContextGetCmd gets a context by ID.
var sandboxContextGetCmd = &cobra.Command{
	Use:   "get <context-id>",
	Short: "Get context details",
	Long:  `Get details of a specific context.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contextID := args[0]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(contextSandboxID).GetContext(cmd.Context(), contextID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting context: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxContextCreateCmd creates a new context.
var sandboxContextCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a context",
	Long:  `Create a new REPL or CMD context in the sandbox.`,
	Run: func(cmd *cobra.Command, args []string) {
		if contextType == "" {
			fmt.Fprintln(os.Stderr, "Error: --type is required (repl or cmd)")
			os.Exit(1)
		}

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		req := apispec.CreateContextRequest{}
		switch contextType {
		case "repl":
			req.Type = apispec.NewOptProcessType(apispec.ProcessTypeRepl)
			if contextLanguage != "" {
				req.Repl = apispec.NewOptCreateREPLContextRequest(apispec.CreateREPLContextRequest{
					Language: apispec.NewOptString(contextLanguage),
				})
			}
		case "cmd":
			req.Type = apispec.NewOptProcessType(apispec.ProcessTypeCmd)
			if len(contextCommand) > 0 {
				req.Cmd = apispec.NewOptCreateCMDContextRequest(apispec.CreateCMDContextRequest{
					Command: contextCommand,
				})
			}
		default:
			fmt.Fprintf(os.Stderr, "Error: invalid type %q, must be 'repl' or 'cmd'\n", contextType)
			os.Exit(1)
		}

		if contextCwd != "" {
			req.Cwd = apispec.NewOptString(contextCwd)
		}

		if len(contextEnv) > 0 {
			envMap := make(apispec.CreateContextRequestEnvVars)
			for _, e := range contextEnv {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) != 2 {
					fmt.Fprintf(os.Stderr, "Error: invalid env format %q, expected KEY=VALUE\n", e)
					os.Exit(1)
				}
				envMap[parts[0]] = parts[1]
			}
			req.EnvVars = apispec.NewOptCreateContextRequestEnvVars(envMap)
		}

		if contextWait {
			req.WaitUntilDone = apispec.NewOptBool(true)
		}

		result, err := client.Sandbox(contextSandboxID).CreateContext(cmd.Context(), req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating context: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxContextDeleteCmd deletes a context.
var sandboxContextDeleteCmd = &cobra.Command{
	Use:   "delete <context-id>",
	Short: "Delete a context",
	Long:  `Delete a context from the sandbox.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contextID := args[0]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.Sandbox(contextSandboxID).DeleteContext(cmd.Context(), contextID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting context: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Context %s deleted successfully\n", contextID)
	},
}

// sandboxContextRestartCmd restarts a context.
var sandboxContextRestartCmd = &cobra.Command{
	Use:   "restart <context-id>",
	Short: "Restart a context",
	Long:  `Restart a context in the sandbox.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contextID := args[0]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(contextSandboxID).RestartContext(cmd.Context(), contextID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error restarting context: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxContextInputCmd sends input to a context.
var sandboxContextInputCmd = &cobra.Command{
	Use:   "input <context-id> <input>",
	Short: "Send input to a context",
	Long:  `Send input to a context (for REPL or interactive CMD).`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		contextID := args[0]
		input := args[1]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.Sandbox(contextSandboxID).ContextInput(cmd.Context(), contextID, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error sending input: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Input sent to context %s\n", contextID)
	},
}

// sandboxContextSignalCmd sends a signal to a context.
var sandboxContextSignalCmd = &cobra.Command{
	Use:   "signal <context-id> <signal>",
	Short: "Send signal to a context",
	Long:  `Send a signal (e.g., SIGTERM, SIGKILL) to a context.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		contextID := args[0]
		signal := args[1]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.Sandbox(contextSandboxID).ContextSignal(cmd.Context(), contextID, signal)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error sending signal: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Signal %s sent to context %s\n", signal, contextID)
	},
}

// sandboxContextStatsCmd gets resource stats for a context.
var sandboxContextStatsCmd = &cobra.Command{
	Use:   "stats <context-id>",
	Short: "Get context resource stats",
	Long:  `Get resource usage statistics for a context.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contextID := args[0]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(contextSandboxID).ContextStats(cmd.Context(), contextID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting context stats: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	sandboxContextCmd.AddCommand(sandboxContextListCmd)
	sandboxContextCmd.AddCommand(sandboxContextGetCmd)
	sandboxContextCmd.AddCommand(sandboxContextCreateCmd)
	sandboxContextCmd.AddCommand(sandboxContextDeleteCmd)
	sandboxContextCmd.AddCommand(sandboxContextRestartCmd)
	sandboxContextCmd.AddCommand(sandboxContextInputCmd)
	sandboxContextCmd.AddCommand(sandboxContextSignalCmd)
	sandboxContextCmd.AddCommand(sandboxContextStatsCmd)

	// Sandbox ID flag (required for all subcommands)
	sandboxContextCmd.PersistentFlags().StringVarP(&contextSandboxID, "sandbox-id", "s", "", "sandbox ID (required)")
	_ = sandboxContextCmd.MarkPersistentFlagRequired("sandbox-id")

	// Create command flags
	sandboxContextCreateCmd.Flags().StringVar(&contextType, "type", "", "context type (repl or cmd) (required)")
	sandboxContextCreateCmd.Flags().StringVar(&contextLanguage, "language", "", "REPL language (e.g., python, node)")
	sandboxContextCreateCmd.Flags().StringArrayVar(&contextCommand, "command", nil, "CMD command (can be repeated)")
	sandboxContextCreateCmd.Flags().StringVar(&contextCwd, "cwd", "", "working directory")
	sandboxContextCreateCmd.Flags().StringArrayVar(&contextEnv, "env", nil, "environment variables (KEY=VALUE, can be repeated)")
	sandboxContextCreateCmd.Flags().BoolVar(&contextWait, "wait", false, "wait for CMD to complete")

	sandboxCmd.AddCommand(sandboxContextCmd)
}
