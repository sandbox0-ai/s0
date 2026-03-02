package commands

import (
	"fmt"
	"os"
	"strconv"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	// sandbox volume flags (persistent)
	volumeSandboxID string
	// mount flags
	mountVolumeID   string
	mountPoint      string
	mountCacheSize  string
	mountBufferSize string
	mountPrefetch   int
	mountWriteback  string
	// unmount flags
	unmountVolumeID       string
	unmountMountSessionID string
)

// sandboxVolumeCmd represents the sandbox volume command group.
var sandboxVolumeCmd = &cobra.Command{
	Use:   "volume",
	Short: "Manage volume mounts",
	Long:  `Mount, unmount, and query volume mount status in a sandbox.`,
}

// sandboxVolumeMountCmd mounts a volume to a sandbox.
var sandboxVolumeMountCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount a volume",
	Long:  `Mount a volume to the sandbox at the specified path.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		// Build optional volume config
		var config *apispec.VolumeConfig
		if mountCacheSize != "" || mountBufferSize != "" || mountPrefetch > 0 || mountWriteback != "" {
			config = &apispec.VolumeConfig{}
			if mountCacheSize != "" {
				config.CacheSize = apispec.NewOptString(mountCacheSize)
			}
			if mountBufferSize != "" {
				config.BufferSize = apispec.NewOptString(mountBufferSize)
			}
			if mountPrefetch > 0 {
				config.Prefetch = apispec.NewOptInt32(int32(mountPrefetch))
			}
			if mountWriteback != "" {
				writeback, err := strconv.ParseBool(mountWriteback)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error parsing writeback: %v\n", err)
					os.Exit(1)
				}
				config.Writeback = apispec.NewOptBool(writeback)
			}
		}

		result, err := client.Sandbox(volumeSandboxID).Mount(cmd.Context(), mountVolumeID, mountPoint, config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error mounting volume: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxVolumeUnmountCmd unmounts a volume from a sandbox.
var sandboxVolumeUnmountCmd = &cobra.Command{
	Use:   "unmount",
	Short: "Unmount a volume",
	Long:  `Unmount a volume from the sandbox.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.Sandbox(volumeSandboxID).Unmount(cmd.Context(), unmountVolumeID, unmountMountSessionID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error unmounting volume: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Volume %s unmounted successfully\n", unmountVolumeID)
	},
}

// sandboxVolumeMountStatusCmd shows mount status for a sandbox.
var sandboxVolumeMountStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show mount status",
	Long:  `Show volume mount status for the sandbox.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(volumeSandboxID).MountStatus(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting mount status: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	// Add volume subcommand group to sandbox command
	sandboxCmd.AddCommand(sandboxVolumeCmd)

	sandboxVolumeCmd.AddCommand(sandboxVolumeMountCmd)
	sandboxVolumeCmd.AddCommand(sandboxVolumeUnmountCmd)
	sandboxVolumeCmd.AddCommand(sandboxVolumeMountStatusCmd)

	// Sandbox ID flag (required for all subcommands)
	sandboxVolumeCmd.PersistentFlags().StringVarP(&volumeSandboxID, "sandbox-id", "s", "", "sandbox ID (required)")
	_ = sandboxVolumeCmd.MarkPersistentFlagRequired("sandbox-id")

	// Mount command flags
	sandboxVolumeMountCmd.Flags().StringVar(&mountVolumeID, "volume-id", "", "volume ID to mount (required)")
	sandboxVolumeMountCmd.Flags().StringVar(&mountPoint, "path", "", "mount path inside sandbox (required)")
	sandboxVolumeMountCmd.Flags().StringVar(&mountCacheSize, "cache-size", "", "cache size override (e.g., 1G)")
	sandboxVolumeMountCmd.Flags().StringVar(&mountBufferSize, "buffer-size", "", "buffer size override (e.g., 128M)")
	sandboxVolumeMountCmd.Flags().IntVar(&mountPrefetch, "prefetch", 0, "prefetch count override")
	sandboxVolumeMountCmd.Flags().StringVar(&mountWriteback, "writeback", "", "enable writeback (true/false)")
	_ = sandboxVolumeMountCmd.MarkFlagRequired("volume-id")
	_ = sandboxVolumeMountCmd.MarkFlagRequired("path")

	// Unmount command flags
	sandboxVolumeUnmountCmd.Flags().StringVar(&unmountVolumeID, "volume-id", "", "volume ID to unmount (required)")
	sandboxVolumeUnmountCmd.Flags().StringVar(&unmountMountSessionID, "session-id", "", "mount session ID (required)")
	_ = sandboxVolumeUnmountCmd.MarkFlagRequired("volume-id")
	_ = sandboxVolumeUnmountCmd.MarkFlagRequired("session-id")
}
