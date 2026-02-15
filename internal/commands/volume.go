package commands

import (
	"fmt"
	"os"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

// volumeCmd represents the volume command.
var volumeCmd = &cobra.Command{
	Use:   "volume",
	Short: "Manage volumes",
	Long:  `List, get, create, and delete sandbox volumes.`,
}

// volumeListCmd lists all volumes.
var volumeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List volumes",
	Long:  `List all sandbox volumes.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		volumes, err := client.ListVolume(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing volumes: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, volumes); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// volumeGetCmd gets a volume by ID.
var volumeGetCmd = &cobra.Command{
	Use:   "get <volume-id>",
	Short: "Get volume details",
	Long:  `Get details of a volume by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		volume, err := client.GetVolume(cmd.Context(), volumeID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting volume: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, volume); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// volumeCreateCmd creates a new volume.
var volumeCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a volume",
	Long:  `Create a new sandbox volume.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		volume, err := client.CreateVolume(cmd.Context(), apispec.CreateSandboxVolumeRequest{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating volume: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, volume); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// volumeDeleteCmd deletes a volume.
var volumeDeleteCmd = &cobra.Command{
	Use:   "delete <volume-id>",
	Short: "Delete a volume",
	Long:  `Delete a sandbox volume by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.DeleteVolume(cmd.Context(), volumeID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting volume: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Volume %s deleted successfully\n", volumeID)
	},
}

func init() {
	rootCmd.AddCommand(volumeCmd)

	volumeCmd.AddCommand(volumeListCmd)
	volumeCmd.AddCommand(volumeGetCmd)
	volumeCmd.AddCommand(volumeCreateCmd)
	volumeCmd.AddCommand(volumeDeleteCmd)
}
