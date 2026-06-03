package commands

import (
	"fmt"
	"os"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	filesystemSnapshotName        string
	filesystemSnapshotDescription string
)

// filesystemSnapshotCmd represents the filesystem snapshot command group.
var filesystemSnapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Manage filesystem snapshots",
	Long:  `List, get, create, delete, and restore persistent sandbox filesystem snapshots.`,
}

// filesystemSnapshotListCmd lists all snapshots for a filesystem.
var filesystemSnapshotListCmd = &cobra.Command{
	Use:   "list <filesystem-id>",
	Short: "List snapshots",
	Long:  `List all snapshots for a persistent sandbox filesystem.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filesystemID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		snapshots, err := client.ListFilesystemSnapshots(cmd.Context(), filesystemID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing filesystem snapshots: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, snapshots); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// filesystemSnapshotGetCmd gets a filesystem snapshot by ID.
var filesystemSnapshotGetCmd = &cobra.Command{
	Use:   "get <filesystem-id> <snapshot-id>",
	Short: "Get snapshot details",
	Long:  `Get details of a persistent sandbox filesystem snapshot by its ID.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		filesystemID := args[0]
		snapshotID := args[1]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		snapshot, err := client.GetFilesystemSnapshot(cmd.Context(), filesystemID, snapshotID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting filesystem snapshot: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// filesystemSnapshotCreateCmd creates a new filesystem snapshot.
var filesystemSnapshotCreateCmd = &cobra.Command{
	Use:   "create <filesystem-id>",
	Short: "Create a snapshot",
	Long:  `Create a new snapshot for a persistent sandbox filesystem.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filesystemID := args[0]

		if filesystemSnapshotName == "" {
			fmt.Fprintln(os.Stderr, "Error: --name is required")
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		req := apispec.CreateSandboxFilesystemSnapshotRequest{
			Name: filesystemSnapshotName,
		}
		if filesystemSnapshotDescription != "" {
			req.Description = apispec.NewOptString(filesystemSnapshotDescription)
		}

		snapshot, err := client.CreateFilesystemSnapshot(cmd.Context(), filesystemID, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating filesystem snapshot: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// filesystemSnapshotDeleteCmd deletes a filesystem snapshot.
var filesystemSnapshotDeleteCmd = &cobra.Command{
	Use:   "delete <filesystem-id> <snapshot-id>",
	Short: "Delete a snapshot",
	Long:  `Delete a persistent sandbox filesystem snapshot by its ID.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		filesystemID := args[0]
		snapshotID := args[1]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.DeleteFilesystemSnapshot(cmd.Context(), filesystemID, snapshotID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting filesystem snapshot: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Filesystem snapshot %s deleted successfully\n", snapshotID)
	},
}

// filesystemSnapshotRestoreCmd restores a filesystem snapshot.
var filesystemSnapshotRestoreCmd = &cobra.Command{
	Use:   "restore <filesystem-id> <snapshot-id>",
	Short: "Restore a snapshot",
	Long:  `Restore a persistent sandbox filesystem to a specific snapshot.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		filesystemID := args[0]
		snapshotID := args[1]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		filesystem, err := client.RestoreFilesystemSnapshot(cmd.Context(), filesystemID, snapshotID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error restoring filesystem snapshot: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, filesystem); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	filesystemCmd.AddCommand(filesystemSnapshotCmd)

	filesystemSnapshotCreateCmd.Flags().StringVarP(&filesystemSnapshotName, "name", "n", "", "snapshot name (required)")
	filesystemSnapshotCreateCmd.Flags().StringVarP(&filesystemSnapshotDescription, "description", "d", "", "snapshot description")

	filesystemSnapshotCmd.AddCommand(filesystemSnapshotListCmd)
	filesystemSnapshotCmd.AddCommand(filesystemSnapshotGetCmd)
	filesystemSnapshotCmd.AddCommand(filesystemSnapshotCreateCmd)
	filesystemSnapshotCmd.AddCommand(filesystemSnapshotDeleteCmd)
	filesystemSnapshotCmd.AddCommand(filesystemSnapshotRestoreCmd)
}
