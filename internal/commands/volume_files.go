package commands

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
)

var (
	volumeFilesRecursive bool
	volumeFilesParents   bool
	volumeFilesStdin     bool
	volumeFilesData      string
)

var volumeFilesCmd = &cobra.Command{
	Use:   "files",
	Short: "Manage files in a volume",
	Long:  `List, read, write, upload, download, and manage files in a volume directly by volume ID.`,
}

var volumeFilesLsCmd = &cobra.Command{
	Use:   "ls <volume-id> [path]",
	Short: "List directory contents",
	Long:  `List files and directories in the specified volume path (default: /).`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]
		path := "/"
		if len(args) == 2 {
			path = args[1]
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.ListVolumeFiles(cmd.Context(), volumeID, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing files: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var volumeFilesCatCmd = &cobra.Command{
	Use:   "cat <volume-id> <path>",
	Short: "Read file content",
	Long:  `Read file content from a volume and write to stdout.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]
		path := args[1]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.ReadVolumeFile(cmd.Context(), volumeID, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}

		if _, err := os.Stdout.Write(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
	},
}

var volumeFilesStatCmd = &cobra.Command{
	Use:   "stat <volume-id> <path>",
	Short: "Get file metadata",
	Long:  `Get file or directory metadata from a volume.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]
		path := args[1]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.StatVolumeFile(cmd.Context(), volumeID, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting file info: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var volumeFilesMkdirCmd = &cobra.Command{
	Use:   "mkdir <volume-id> <path>",
	Short: "Create a directory",
	Long:  `Create a directory in a volume. Use --parents to create parent directories as needed.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]
		path := args[1]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.MkdirVolumeFile(cmd.Context(), volumeID, path, volumeFilesParents)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Directory %s created successfully\n", path)
	},
}

var volumeFilesRmCmd = &cobra.Command{
	Use:   "rm <volume-id> <path>",
	Short: "Delete a file or directory",
	Long:  `Delete a file or directory from a volume.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]
		path := args[1]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.DeleteVolumeFile(cmd.Context(), volumeID, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("File %s deleted successfully\n", path)
	},
}

var volumeFilesMvCmd = &cobra.Command{
	Use:   "mv <volume-id> <source> <destination>",
	Short: "Move or rename a file",
	Long:  `Move or rename a file or directory inside a volume.`,
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]
		source := args[1]
		destination := args[2]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.MoveVolumeFile(cmd.Context(), volumeID, source, destination)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error moving file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("File moved from %s to %s\n", source, destination)
	},
}

var volumeFilesUploadCmd = &cobra.Command{
	Use:   "upload <volume-id> <local-path> <remote-path>",
	Short: "Upload a local file",
	Long:  `Upload a local file to a volume path.`,
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]
		localPath := args[1]
		remotePath := args[2]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		data, err := os.ReadFile(localPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading local file: %v\n", err)
			os.Exit(1)
		}

		_, err = client.WriteVolumeFile(cmd.Context(), volumeID, remotePath, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error uploading file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("File uploaded to %s\n", remotePath)
	},
}

var volumeFilesDownloadCmd = &cobra.Command{
	Use:   "download <volume-id> <remote-path> <local-path>",
	Short: "Download a file",
	Long:  `Download a file from a volume to the local filesystem.`,
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]
		remotePath := args[1]
		localPath := args[2]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		data, err := client.ReadVolumeFile(cmd.Context(), volumeID, remotePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error downloading file: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(localPath, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing local file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("File downloaded to %s\n", localPath)
	},
}

var volumeFilesWriteCmd = &cobra.Command{
	Use:   "write <volume-id> <path>",
	Short: "Write content to a file",
	Long:  `Write content to a volume file. Use --stdin to read from stdin or --data to provide content directly.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]
		path := args[1]

		data, err := readCommandContent(volumeFilesStdin, volumeFilesData, os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.WriteVolumeFile(cmd.Context(), volumeID, path, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("File %s written successfully\n", path)
	},
}

var volumeFilesWatchCmd = &cobra.Command{
	Use:   "watch <volume-id> <path>",
	Short: "Watch for file changes",
	Long:  `Watch for file changes in the specified volume path. Use -r for recursive watching.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]
		path := args[1]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		ctx, cancel := signal.NotifyContext(cmd.Context(), forwardingSignals()...)
		defer cancel()

		events, errs, unsubscribe, err := client.WatchVolumeFiles(ctx, volumeID, path, volumeFilesRecursive)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error watching files: %v\n", err)
			os.Exit(1)
		}
		defer func() { _ = unsubscribe() }()

		fmt.Printf("Watching %s (recursive: %v). Press Ctrl+C to stop.\n", path, volumeFilesRecursive)

		for {
			select {
			case event, ok := <-events:
				if !ok {
					return
				}
				switch event.Type {
				case "event":
					fmt.Printf("%s: %s\n", event.Event, event.Path)
				case "error":
					fmt.Fprintf(os.Stderr, "Error: %s\n", event.Error)
				}
			case err, ok := <-errs:
				if !ok {
					return
				}
				fmt.Fprintf(os.Stderr, "Watch error: %v\n", err)
			case <-ctx.Done():
				fmt.Println("\nWatch stopped.")
				return
			}
		}
	},
}

func init() {
	volumeFilesCmd.AddCommand(volumeFilesLsCmd)
	volumeFilesCmd.AddCommand(volumeFilesCatCmd)
	volumeFilesCmd.AddCommand(volumeFilesStatCmd)
	volumeFilesCmd.AddCommand(volumeFilesMkdirCmd)
	volumeFilesCmd.AddCommand(volumeFilesRmCmd)
	volumeFilesCmd.AddCommand(volumeFilesMvCmd)
	volumeFilesCmd.AddCommand(volumeFilesUploadCmd)
	volumeFilesCmd.AddCommand(volumeFilesDownloadCmd)
	volumeFilesCmd.AddCommand(volumeFilesWriteCmd)
	volumeFilesCmd.AddCommand(volumeFilesWatchCmd)

	volumeFilesMkdirCmd.Flags().BoolVar(&volumeFilesParents, "parents", false, "create parent directories as needed")
	volumeFilesWatchCmd.Flags().BoolVarP(&volumeFilesRecursive, "recursive", "r", false, "watch recursively")
	volumeFilesWriteCmd.Flags().BoolVar(&volumeFilesStdin, "stdin", false, "read content from stdin")
	volumeFilesWriteCmd.Flags().StringVar(&volumeFilesData, "data", "", "content to write directly")

	volumeCmd.AddCommand(volumeFilesCmd)
}
