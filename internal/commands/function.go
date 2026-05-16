package commands

import (
	"fmt"
	"os"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/spf13/cobra"
)

var (
	functionName            string
	functionUpdateName      string
	functionUpdateEnabled   bool
	functionRevisionPromote bool
)

// functionCmd represents the function command.
var functionCmd = &cobra.Command{
	Use:   "function",
	Short: "Manage functions",
	Long:  `List, get, create, revise, and promote sandbox0 functions.`,
}

var functionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List functions",
	Long:  `List functions for the current team.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		functions, err := client.ListFunctions(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing functions: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, functions); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionGetCmd = &cobra.Command{
	Use:   "get <function-id-or-slug>",
	Short: "Get function details",
	Long:  `Get function details by ID or slug.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		function, err := client.GetFunction(cmd.Context(), args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting function: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, function); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionCreateCmd = &cobra.Command{
	Use:   "create <sandbox-id> <service-id>",
	Short: "Create a function from a sandbox service",
	Long:  `Create a function from an existing publishable sandbox service.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		opts := make([]sandbox0.FunctionCreateOption, 0, 1)
		if functionName != "" {
			opts = append(opts, sandbox0.WithFunctionName(functionName))
		}

		result, err := client.CreateFunctionFromSandbox(cmd.Context(), args[0], args[1], opts...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating function: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionUpdateCmd = &cobra.Command{
	Use:   "update <function-id-or-slug>",
	Short: "Update a function",
	Long:  `Update mutable function metadata and serving state.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		opts := make([]sandbox0.FunctionUpdateOption, 0, 2)
		if functionUpdateName != "" {
			opts = append(opts, sandbox0.WithFunctionUpdateName(functionUpdateName))
		}
		if cmd.Flags().Changed("enabled") {
			opts = append(opts, sandbox0.WithFunctionEnabled(functionUpdateEnabled))
		}
		if len(opts) == 0 {
			fmt.Fprintln(os.Stderr, "Error updating function: at least one of --name or --enabled is required")
			os.Exit(1)
		}

		function, err := client.UpdateFunctionWithOptions(cmd.Context(), args[0], opts...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating function: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, function); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionDeleteCmd = &cobra.Command{
	Use:   "delete <function-id-or-slug>",
	Short: "Delete a function",
	Long:  `Soft-delete a function and remove it from normal list, get, and host traffic.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		function, err := client.DeleteFunction(cmd.Context(), args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting function: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, function); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionRevisionCmd = &cobra.Command{
	Use:   "revision",
	Short: "Manage function revisions",
	Long:  `List, get, and create function revisions.`,
}

var functionRevisionListCmd = &cobra.Command{
	Use:   "list <function-id-or-slug>",
	Short: "List function revisions",
	Long:  `List revisions for a function.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		revisions, err := client.ListFunctionRevisions(cmd.Context(), args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing function revisions: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, revisions); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionRevisionGetCmd = &cobra.Command{
	Use:   "get <function-id-or-slug> <revision-number>",
	Short: "Get a function revision",
	Long:  `Get one function revision by revision number.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		revisionNumber := parseInt32(args[1], "revision number")
		revision, err := client.GetFunctionRevision(cmd.Context(), args[0], revisionNumber)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting function revision: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, revision); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionRevisionCreateCmd = &cobra.Command{
	Use:   "create <function-id-or-slug> <sandbox-id> <service-id>",
	Short: "Create a function revision",
	Long:  `Create a new function revision from an existing publishable sandbox service.`,
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.CreateFunctionRevisionFromSandbox(
			cmd.Context(),
			args[0],
			args[1],
			args[2],
			sandbox0.WithFunctionRevisionPromote(functionRevisionPromote),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating function revision: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionAliasCmd = &cobra.Command{
	Use:   "alias",
	Short: "Manage function aliases",
	Long:  `List, get, and promote function aliases.`,
}

var functionAliasListCmd = &cobra.Command{
	Use:   "list <function-id-or-slug>",
	Short: "List function aliases",
	Long:  `List aliases for a function.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		aliases, err := client.ListFunctionAliases(cmd.Context(), args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing function aliases: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, aliases); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionAliasGetCmd = &cobra.Command{
	Use:   "get <function-id-or-slug> <alias>",
	Short: "Get a function alias",
	Long:  `Get one alias for a function.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		alias, err := client.GetFunctionAlias(cmd.Context(), args[0], args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting function alias: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, alias); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionAliasSetCmd = &cobra.Command{
	Use:   "set <function-id-or-slug> <alias> <revision-number>",
	Short: "Set a function alias",
	Long:  `Point a function alias at a revision number.`,
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		revisionNumber := parseInt32(args[2], "revision number")
		alias, err := client.SetFunctionAlias(cmd.Context(), args[0], args[1], revisionNumber)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error setting function alias: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, alias); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionRuntimeCmd = &cobra.Command{
	Use:   "runtime",
	Short: "Manage function runtime",
	Long:  `Inspect and reset the restored runtime sandbox for a function.`,
}

var functionRuntimeGetCmd = &cobra.Command{
	Use:   "get <function-id-or-slug>",
	Short: "Get function runtime status",
	Long:  `Get the currently restored runtime status for a function.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		runtime, err := client.GetFunctionRuntime(cmd.Context(), args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting function runtime: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, runtime); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionRuntimeRestartCmd = &cobra.Command{
	Use:   "restart <function-id-or-slug>",
	Short: "Restart function runtime",
	Long:  `Delete the current runtime sandbox and leave the function idle until the next host request.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		runtime, err := client.RestartFunctionRuntime(cmd.Context(), args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error restarting function runtime: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, runtime); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionRuntimeRecycleCmd = &cobra.Command{
	Use:   "recycle <function-id-or-slug>",
	Short: "Recycle function runtime",
	Long:  `Recycle the current runtime sandbox and leave the function idle until the next host request.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		runtime, err := client.RecycleFunctionRuntime(cmd.Context(), args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error recycling function runtime: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, runtime); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(functionCmd)

	functionCreateCmd.Flags().StringVar(&functionName, "name", "", "function display name")
	functionUpdateCmd.Flags().StringVar(&functionUpdateName, "name", "", "function display name")
	functionUpdateCmd.Flags().BoolVar(&functionUpdateEnabled, "enabled", true, "whether the function should serve traffic")
	functionRevisionCreateCmd.Flags().BoolVar(&functionRevisionPromote, "promote", true, "move the production alias to the new revision")

	functionRevisionCmd.AddCommand(functionRevisionListCmd)
	functionRevisionCmd.AddCommand(functionRevisionGetCmd)
	functionRevisionCmd.AddCommand(functionRevisionCreateCmd)
	functionAliasCmd.AddCommand(functionAliasListCmd)
	functionAliasCmd.AddCommand(functionAliasGetCmd)
	functionAliasCmd.AddCommand(functionAliasSetCmd)
	functionRuntimeCmd.AddCommand(functionRuntimeGetCmd)
	functionRuntimeCmd.AddCommand(functionRuntimeRestartCmd)
	functionRuntimeCmd.AddCommand(functionRuntimeRecycleCmd)

	functionCmd.AddCommand(functionListCmd)
	functionCmd.AddCommand(functionGetCmd)
	functionCmd.AddCommand(functionCreateCmd)
	functionCmd.AddCommand(functionUpdateCmd)
	functionCmd.AddCommand(functionDeleteCmd)
	functionCmd.AddCommand(functionRevisionCmd)
	functionCmd.AddCommand(functionAliasCmd)
	functionCmd.AddCommand(functionRuntimeCmd)
}
