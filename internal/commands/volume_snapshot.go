package commands

import (
	"fmt"
	"os"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	snapshotName        string
	snapshotDescription string
)

// volumeSnapshotCmd represents the volume snapshot command group.
var volumeSnapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Manage volume snapshots",
	Long:  `List, get, create, delete, and restore volume snapshots.`,
}

// volumeSnapshotListCmd lists all snapshots for a volume.
var volumeSnapshotListCmd = &cobra.Command{
	Use:   "list <volume-id>",
	Short: "List snapshots",
	Long:  `List all snapshots for a volume.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		snapshots, err := client.ListVolumeSnapshots(cmd.Context(), volumeID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing snapshots: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, snapshots); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// volumeSnapshotGetCmd gets a snapshot by ID.
var volumeSnapshotGetCmd = &cobra.Command{
	Use:   "get <volume-id> <snapshot-id>",
	Short: "Get snapshot details",
	Long:  `Get details of a snapshot by its ID.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]
		snapshotID := args[1]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		snapshot, err := client.GetVolumeSnapshot(cmd.Context(), volumeID, snapshotID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting snapshot: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// volumeSnapshotCreateCmd creates a new snapshot.
var volumeSnapshotCreateCmd = &cobra.Command{
	Use:   "create <volume-id>",
	Short: "Create a snapshot",
	Long:  `Create a new snapshot for a volume.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]

		if snapshotName == "" {
			fmt.Fprintln(os.Stderr, "Error: --name is required")
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		req := apispec.CreateSnapshotRequest{
			Name: snapshotName,
		}
		if snapshotDescription != "" {
			req.Description = apispec.NewOptString(snapshotDescription)
		}

		snapshot, err := client.CreateVolumeSnapshot(cmd.Context(), volumeID, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating snapshot: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// volumeSnapshotDeleteCmd deletes a snapshot.
var volumeSnapshotDeleteCmd = &cobra.Command{
	Use:   "delete <volume-id> <snapshot-id>",
	Short: "Delete a snapshot",
	Long:  `Delete a snapshot by its ID.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]
		snapshotID := args[1]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.DeleteVolumeSnapshot(cmd.Context(), volumeID, snapshotID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting snapshot: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Snapshot %s deleted successfully\n", snapshotID)
	},
}

// volumeSnapshotRestoreCmd restores a snapshot.
var volumeSnapshotRestoreCmd = &cobra.Command{
	Use:   "restore <volume-id> <snapshot-id>",
	Short: "Restore a snapshot",
	Long:  `Restore a volume to a specific snapshot.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]
		snapshotID := args[1]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.RestoreVolumeSnapshot(cmd.Context(), volumeID, snapshotID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error restoring snapshot: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Snapshot %s restored successfully\n", snapshotID)
	},
}

func init() {
	// Add snapshot as subcommand of volume
	volumeCmd.AddCommand(volumeSnapshotCmd)

	// Create command flags
	volumeSnapshotCreateCmd.Flags().StringVarP(&snapshotName, "name", "n", "", "snapshot name (required)")
	volumeSnapshotCreateCmd.Flags().StringVarP(&snapshotDescription, "description", "d", "", "snapshot description")

	volumeSnapshotCmd.AddCommand(volumeSnapshotListCmd)
	volumeSnapshotCmd.AddCommand(volumeSnapshotGetCmd)
	volumeSnapshotCmd.AddCommand(volumeSnapshotCreateCmd)
	volumeSnapshotCmd.AddCommand(volumeSnapshotDeleteCmd)
	volumeSnapshotCmd.AddCommand(volumeSnapshotRestoreCmd)
}
