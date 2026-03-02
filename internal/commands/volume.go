package commands

import (
	"fmt"
	"os"
	"strconv"

	sdk "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

// Volume create flags.
var (
	volumeAccessMode  string
	volumeCacheSize   string
	volumePrefetch    string
	volumeBufferSize  string
	volumeWriteback   string
	volumeDeleteForce bool
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

		req := apispec.CreateSandboxVolumeRequest{}

		if volumeAccessMode != "" {
			req.AccessMode = apispec.NewOptVolumeAccessMode(apispec.VolumeAccessMode(volumeAccessMode))
		}
		if volumeCacheSize != "" {
			req.CacheSize = apispec.NewOptString(volumeCacheSize)
		}
		if volumePrefetch != "" {
			prefetch, err := strconv.Atoi(volumePrefetch)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing prefetch: %v\n", err)
				os.Exit(1)
			}
			req.Prefetch = apispec.NewOptInt(prefetch)
		}
		if volumeBufferSize != "" {
			req.BufferSize = apispec.NewOptString(volumeBufferSize)
		}
		if volumeWriteback != "" {
			writeback, err := strconv.ParseBool(volumeWriteback)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing writeback: %v\n", err)
				os.Exit(1)
			}
			req.Writeback = apispec.NewOptBool(writeback)
		}

		volume, err := client.CreateVolume(cmd.Context(), req)
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

		if volumeDeleteForce {
			_, err = client.DeleteVolumeWithOptions(cmd.Context(), volumeID, &sdk.DeleteVolumeOptions{Force: true})
		} else {
			_, err = client.DeleteVolume(cmd.Context(), volumeID)
		}
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

	// Volume create flags
	volumeCreateCmd.Flags().StringVar(&volumeAccessMode, "access-mode", "", "access mode (RWO or RWX)")
	volumeCreateCmd.Flags().StringVar(&volumeCacheSize, "cache-size", "", "cache size (e.g., 1Gi)")
	volumeCreateCmd.Flags().StringVar(&volumePrefetch, "prefetch", "", "prefetch count")
	volumeCreateCmd.Flags().StringVar(&volumeBufferSize, "buffer-size", "", "buffer size (e.g., 64Mi)")
	volumeCreateCmd.Flags().StringVar(&volumeWriteback, "writeback", "", "enable writeback (true/false)")
	volumeDeleteCmd.Flags().BoolVar(&volumeDeleteForce, "force", false, "force delete volume even if it has active mounts")
}
