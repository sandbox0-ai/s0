package commands

import (
	"fmt"
	"os"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	sandboxTemplate string
	sandboxTTL      int32
	sandboxHardTTL  int32
	// list flags
	sandboxListStatus     string
	sandboxListTemplateID string
	sandboxListPaused     string
	sandboxListLimit      int
	sandboxListOffset     int
	// update flags
	sandboxUpdateTTL        int32
	sandboxUpdateHardTTL    int32
	sandboxUpdateAutoResume string
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

		if err := getFormatter().Format(os.Stdout, sandbox); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
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

// sandboxUpdateCmd updates a sandbox's configuration.
var sandboxUpdateCmd = &cobra.Command{
	Use:   "update <sandbox-id>",
	Short: "Update sandbox configuration",
	Long:  `Update the configuration of a sandbox (TTL, env vars, auto-resume, etc.).`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		// Build the config based on provided flags
		var config apispec.SandboxUpdateConfig
		hasConfig := false

		if sandboxUpdateTTL > 0 {
			config.TTL = apispec.NewOptInt32(sandboxUpdateTTL)
			hasConfig = true
		}
		if sandboxUpdateHardTTL > 0 {
			config.HardTTL = apispec.NewOptInt32(sandboxUpdateHardTTL)
			hasConfig = true
		}
		if sandboxUpdateAutoResume != "" {
			autoResume := sandboxUpdateAutoResume == "true"
			config.AutoResume = apispec.NewOptBool(autoResume)
			hasConfig = true
		}

		if !hasConfig {
			fmt.Fprintln(os.Stderr, "Error: at least one update flag is required (--ttl, --hard-ttl, --env, --auto-resume)")
			os.Exit(1)
		}

		req := apispec.SandboxUpdateRequest{
			Config: apispec.NewOptSandboxUpdateConfig(config),
		}

		sandbox, err := client.UpdateSandbox(cmd.Context(), sandboxID, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating sandbox: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, sandbox); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxListCmd lists all sandboxes.
var sandboxListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sandboxes",
	Long:  `List all sandboxes for the authenticated team.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		opts := &sandbox0.ListSandboxesOptions{}
		if sandboxListStatus != "" {
			opts.Status = sandboxListStatus
		}
		if sandboxListTemplateID != "" {
			opts.TemplateID = sandboxListTemplateID
		}
		if sandboxListPaused != "" {
			paused := sandboxListPaused == "true"
			opts.Paused = &paused
		}
		if sandboxListLimit > 0 {
			opts.Limit = &sandboxListLimit
		}
		if sandboxListOffset > 0 {
			opts.Offset = &sandboxListOffset
		}

		resp, err := client.ListSandboxes(cmd.Context(), opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing sandboxes: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, resp); err != nil {
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
	sandboxCmd.AddCommand(sandboxUpdateCmd)

	// Update command flags
	sandboxUpdateCmd.Flags().Int32Var(&sandboxUpdateTTL, "ttl", 0, "soft TTL in seconds")
	sandboxUpdateCmd.Flags().Int32Var(&sandboxUpdateHardTTL, "hard-ttl", 0, "hard TTL in seconds")
	sandboxUpdateCmd.Flags().StringVar(&sandboxUpdateAutoResume, "auto-resume", "", "auto resume on access (true/false)")

	// List command flags
	sandboxListCmd.Flags().StringVar(&sandboxListStatus, "status", "", "filter by status (starting, running, failed, completed)")
	sandboxListCmd.Flags().StringVar(&sandboxListTemplateID, "template-id", "", "filter by template ID")
	sandboxListCmd.Flags().StringVar(&sandboxListPaused, "paused", "", "filter by paused state (true/false)")
	sandboxListCmd.Flags().IntVar(&sandboxListLimit, "limit", 50, "maximum number of results")
	sandboxListCmd.Flags().IntVar(&sandboxListOffset, "offset", 0, "pagination offset")
	sandboxCmd.AddCommand(sandboxListCmd)
}
