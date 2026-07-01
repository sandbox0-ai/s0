package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	sandboxRootFSSnapshotName        string
	sandboxRootFSSnapshotDescription string
	sandboxRootFSSnapshotExpiresAt   string
	sandboxForkTTL                   int32
	sandboxForkHardTTL               int32
)

// sandboxSnapshotCmd represents the sandbox rootfs snapshot command group.
var sandboxSnapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Manage sandbox rootfs snapshots",
	Long:  `List, get, create, delete, and restore sandbox rootfs snapshots.`,
}

// sandboxSnapshotListCmd lists all rootfs snapshots for a sandbox.
var sandboxSnapshotListCmd = &cobra.Command{
	Use:   "list <sandbox-id>",
	Short: "List sandbox rootfs snapshots",
	Long:  `List all rootfs snapshots for a sandbox.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		snapshots, err := client.ListSandboxRootFSSnapshots(cmd.Context(), sandboxID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing sandbox rootfs snapshots: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, snapshots); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxSnapshotGetCmd gets a rootfs snapshot by ID.
var sandboxSnapshotGetCmd = &cobra.Command{
	Use:   "get <snapshot-id>",
	Short: "Get sandbox rootfs snapshot details",
	Long:  `Get details of a sandbox rootfs snapshot by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		snapshotID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		snapshot, err := client.GetSandboxRootFSSnapshot(cmd.Context(), snapshotID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting sandbox rootfs snapshot: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxSnapshotCreateCmd creates a rootfs snapshot for a paused sandbox.
var sandboxSnapshotCreateCmd = &cobra.Command{
	Use:   "create <sandbox-id>",
	Short: "Create a sandbox rootfs snapshot",
	Long:  `Create a rootfs snapshot for a paused sandbox.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		request, err := buildSandboxRootFSSnapshotCreateRequest()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building sandbox rootfs snapshot request: %v\n", err)
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		snapshot, err := client.CreateSandboxRootFSSnapshot(cmd.Context(), sandboxID, request)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating sandbox rootfs snapshot: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxSnapshotDeleteCmd deletes a rootfs snapshot by ID.
var sandboxSnapshotDeleteCmd = &cobra.Command{
	Use:   "delete <snapshot-id>",
	Short: "Delete a sandbox rootfs snapshot",
	Long:  `Delete a sandbox rootfs snapshot by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		snapshotID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.DeleteSandboxRootFSSnapshot(cmd.Context(), snapshotID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting sandbox rootfs snapshot: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Sandbox rootfs snapshot %s deleted successfully\n", snapshotID)
	},
}

// sandboxSnapshotRestoreCmd restores a paused sandbox from a rootfs snapshot.
var sandboxSnapshotRestoreCmd = &cobra.Command{
	Use:   "restore <sandbox-id> <snapshot-id>",
	Short: "Restore sandbox rootfs from a snapshot",
	Long:  `Restore a paused sandbox rootfs from a rootfs snapshot.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]
		snapshotID := args[1]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		response, err := client.RestoreSandboxRootFS(cmd.Context(), sandboxID, apispec.RestoreSandboxRootFSRequest{
			SnapshotID: snapshotID,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error restoring sandbox rootfs snapshot: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, response); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxForkCmd creates a paused sandbox fork from a paused source sandbox.
var sandboxForkCmd = &cobra.Command{
	Use:   "fork <sandbox-id>",
	Short: "Fork a paused sandbox",
	Long:  `Create a paused sandbox fork from a paused source sandbox rootfs.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		response, err := client.ForkSandbox(cmd.Context(), sandboxID, buildSandboxForkRequest(cmd))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error forking sandbox: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, response); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	sandboxSnapshotCreateCmd.Flags().StringVarP(&sandboxRootFSSnapshotName, "name", "n", "", "snapshot name")
	sandboxSnapshotCreateCmd.Flags().StringVarP(&sandboxRootFSSnapshotDescription, "description", "d", "", "snapshot description")
	sandboxSnapshotCreateCmd.Flags().StringVar(&sandboxRootFSSnapshotExpiresAt, "expires-at", "", "snapshot expiration timestamp (RFC3339)")
	sandboxForkCmd.Flags().Int32Var(&sandboxForkTTL, "ttl", 0, "soft TTL in seconds for the forked sandbox")
	sandboxForkCmd.Flags().Int32Var(&sandboxForkHardTTL, "hard-ttl", 0, "hard TTL in seconds for the forked sandbox")

	sandboxSnapshotCmd.AddCommand(sandboxSnapshotListCmd)
	sandboxSnapshotCmd.AddCommand(sandboxSnapshotGetCmd)
	sandboxSnapshotCmd.AddCommand(sandboxSnapshotCreateCmd)
	sandboxSnapshotCmd.AddCommand(sandboxSnapshotDeleteCmd)
	sandboxSnapshotCmd.AddCommand(sandboxSnapshotRestoreCmd)

	sandboxCmd.AddCommand(sandboxSnapshotCmd)
	sandboxCmd.AddCommand(sandboxForkCmd)
}

func buildSandboxRootFSSnapshotCreateRequest() (*apispec.CreateSandboxRootFSSnapshotRequest, error) {
	var request apispec.CreateSandboxRootFSSnapshotRequest
	hasRequest := false

	if sandboxRootFSSnapshotName != "" {
		request.Name = apispec.NewOptString(sandboxRootFSSnapshotName)
		hasRequest = true
	}
	if sandboxRootFSSnapshotDescription != "" {
		request.Description = apispec.NewOptString(sandboxRootFSSnapshotDescription)
		hasRequest = true
	}
	if sandboxRootFSSnapshotExpiresAt != "" {
		expiresAt, err := time.Parse(time.RFC3339, sandboxRootFSSnapshotExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("invalid --expires-at %q: expected RFC3339 timestamp", sandboxRootFSSnapshotExpiresAt)
		}
		request.ExpiresAt = apispec.NewOptDateTime(expiresAt)
		hasRequest = true
	}
	if !hasRequest {
		return nil, nil
	}
	return &request, nil
}

func buildSandboxForkRequest(cmd *cobra.Command) *apispec.ForkSandboxRequest {
	config := apispec.ForkSandboxConfig{}
	hasConfig := false

	if cmd.Flags().Changed("ttl") {
		config.TTL = apispec.NewOptInt32(sandboxForkTTL)
		hasConfig = true
	}
	if cmd.Flags().Changed("hard-ttl") {
		config.HardTTL = apispec.NewOptInt32(sandboxForkHardTTL)
		hasConfig = true
	}
	if !hasConfig {
		return nil
	}
	return &apispec.ForkSandboxRequest{
		Config: apispec.NewOptForkSandboxConfig(config),
	}
}
