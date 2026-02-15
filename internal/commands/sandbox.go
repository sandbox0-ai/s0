package commands

import (
	"fmt"
	"os"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/spf13/cobra"
)

var (
	sandboxTemplate   string
	sandboxTTL        int32
	sandboxHardTTL    int32
	sandboxRefreshTTL int32
)

// sandboxCmd represents the sandbox command.
var sandboxCmd = &cobra.Command{
	Use:   "sandbox",
	Short: "Manage sandboxes",
	Long:  `Create, get, delete, pause, resume, refresh, and check status of sandboxes.`,
}

// sandboxCreateCmd creates a new sandbox.
var sandboxCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create (claim) a new sandbox",
	Long:  `Create a new sandbox from a template.`,
	Run: func(cmd *cobra.Command, args []string) {
		if sandboxTemplate == "" {
			fmt.Fprintln(os.Stderr, "Error: --template is required")
			os.Exit(1)
		}

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		var opts []sandbox0.SandboxOption
		if sandboxTTL > 0 {
			opts = append(opts, sandbox0.WithSandboxTTL(sandboxTTL))
		}
		if sandboxHardTTL > 0 {
			opts = append(opts, sandbox0.WithSandboxHardTTL(sandboxHardTTL))
		}

		sandbox, err := client.ClaimSandbox(cmd.Context(), sandboxTemplate, opts...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating sandbox: %v\n", err)
			os.Exit(1)
		}

		// Print sandbox info
		fmt.Printf("ID:\t%s\n", sandbox.ID)
		fmt.Printf("Template:\t%s\n", sandbox.Template)
		fmt.Printf("Status:\t%s\n", sandbox.Status)
		if sandbox.ClusterID != nil {
			fmt.Printf("Cluster ID:\t%s\n", *sandbox.ClusterID)
		}
		if sandbox.PodName != "" {
			fmt.Printf("Pod Name:\t%s\n", sandbox.PodName)
		}
	},
}

// sandboxGetCmd gets a sandbox by ID.
var sandboxGetCmd = &cobra.Command{
	Use:   "get <sandbox-id>",
	Short: "Get sandbox details",
	Long:  `Get details of a sandbox by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		sandbox, err := client.GetSandbox(cmd.Context(), sandboxID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting sandbox: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, sandbox); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxDeleteCmd deletes a sandbox.
var sandboxDeleteCmd = &cobra.Command{
	Use:   "delete <sandbox-id>",
	Short: "Delete a sandbox",
	Long:  `Delete (terminate) a sandbox by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.DeleteSandbox(cmd.Context(), sandboxID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting sandbox: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Sandbox %s deleted successfully\n", sandboxID)
	},
}

// sandboxPauseCmd pauses a sandbox.
var sandboxPauseCmd = &cobra.Command{
	Use:   "pause <sandbox-id>",
	Short: "Pause a sandbox",
	Long:  `Pause (suspend) a sandbox by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.PauseSandbox(cmd.Context(), sandboxID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error pausing sandbox: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Sandbox %s paused successfully\n", sandboxID)
	},
}

// sandboxResumeCmd resumes a sandbox.
var sandboxResumeCmd = &cobra.Command{
	Use:   "resume <sandbox-id>",
	Short: "Resume a sandbox",
	Long:  `Resume a paused sandbox by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.ResumeSandbox(cmd.Context(), sandboxID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resuming sandbox: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Sandbox %s resumed successfully\n", sandboxID)
	},
}

// sandboxRefreshCmd refreshes a sandbox's TTL.
var sandboxRefreshCmd = &cobra.Command{
	Use:   "refresh <sandbox-id>",
	Short: "Refresh sandbox TTL",
	Long:  `Refresh the TTL of a sandbox by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		resp, err := client.RefreshSandbox(cmd.Context(), sandboxID, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error refreshing sandbox: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, resp); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxStatusCmd gets the status of a sandbox.
var sandboxStatusCmd = &cobra.Command{
	Use:   "status <sandbox-id>",
	Short: "Get sandbox status",
	Long:  `Get the status of a sandbox by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		status, err := client.StatusSandbox(cmd.Context(), sandboxID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting sandbox status: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, status); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(sandboxCmd)

	// Create command flags
	sandboxCreateCmd.Flags().StringVarP(&sandboxTemplate, "template", "t", "", "template ID (required)")
	sandboxCreateCmd.Flags().Int32Var(&sandboxTTL, "ttl", 0, "soft TTL in seconds")
	sandboxCreateCmd.Flags().Int32Var(&sandboxHardTTL, "hard-ttl", 0, "hard TTL in seconds")

	sandboxCmd.AddCommand(sandboxCreateCmd)
	sandboxCmd.AddCommand(sandboxGetCmd)
	sandboxCmd.AddCommand(sandboxDeleteCmd)
	sandboxCmd.AddCommand(sandboxPauseCmd)
	sandboxCmd.AddCommand(sandboxResumeCmd)
	sandboxCmd.AddCommand(sandboxRefreshCmd)
	sandboxCmd.AddCommand(sandboxStatusCmd)
}
