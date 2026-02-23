package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	filesSandboxID   string
	filesRecursive   bool
	filesParents     bool
	filesStdin       bool
	filesData        string
)

// sandboxFilesCmd represents the sandbox files command group.
var sandboxFilesCmd = &cobra.Command{
	Use:   "files",
	Short: "Manage files",
	Long:  `List, read, write, upload, download, and manage files in a sandbox.`,
}

// sandboxFilesLsCmd lists directory contents.
var sandboxFilesLsCmd = &cobra.Command{
	Use:   "ls [path]",
	Short: "List directory contents",
	Long:  `List files and directories in the specified path (default: /).`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := "/"
		if len(args) > 0 {
			path = args[0]
		}

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(filesSandboxID).ListFiles(cmd.Context(), path)
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

// sandboxFilesCatCmd reads a file to stdout.
var sandboxFilesCatCmd = &cobra.Command{
	Use:   "cat <path>",
	Short: "Read file content",
	Long:  `Read file content and write to stdout.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(filesSandboxID).ReadFile(cmd.Context(), path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}

		os.Stdout.Write(result)
	},
}

// sandboxFilesStatCmd gets file metadata.
var sandboxFilesStatCmd = &cobra.Command{
	Use:   "stat <path>",
	Short: "Get file metadata",
	Long:  `Get file or directory metadata.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(filesSandboxID).StatFile(cmd.Context(), path)
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

// sandboxFilesMkdirCmd creates a directory.
var sandboxFilesMkdirCmd = &cobra.Command{
	Use:   "mkdir <path>",
	Short: "Create a directory",
	Long:  `Create a directory. Use --parents to create parent directories as needed.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.Sandbox(filesSandboxID).Mkdir(cmd.Context(), path, filesParents)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Directory %s created successfully\n", path)
	},
}

// sandboxFilesRmCmd deletes a file or directory.
var sandboxFilesRmCmd = &cobra.Command{
	Use:   "rm <path>",
	Short: "Delete a file or directory",
	Long:  `Delete a file or directory.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.Sandbox(filesSandboxID).DeleteFile(cmd.Context(), path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("File %s deleted successfully\n", path)
	},
}

// sandboxFilesMvCmd moves or renames a file.
var sandboxFilesMvCmd = &cobra.Command{
	Use:   "mv <source> <destination>",
	Short: "Move or rename a file",
	Long:  `Move or rename a file or directory.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		source := args[0]
		destination := args[1]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.Sandbox(filesSandboxID).MoveFile(cmd.Context(), source, destination)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error moving file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("File moved from %s to %s\n", source, destination)
	},
}

// sandboxFilesUploadCmd uploads a local file.
var sandboxFilesUploadCmd = &cobra.Command{
	Use:   "upload <local-path> <remote-path>",
	Short: "Upload a local file",
	Long:  `Upload a local file to the sandbox.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		localPath := args[0]
		remotePath := args[1]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		data, err := os.ReadFile(localPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading local file: %v\n", err)
			os.Exit(1)
		}

		_, err = client.Sandbox(filesSandboxID).WriteFile(cmd.Context(), remotePath, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error uploading file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("File uploaded to %s\n", remotePath)
	},
}

// sandboxFilesDownloadCmd downloads a file to local.
var sandboxFilesDownloadCmd = &cobra.Command{
	Use:   "download <remote-path> <local-path>",
	Short: "Download a file",
	Long:  `Download a file from the sandbox to local filesystem.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		remotePath := args[0]
		localPath := args[1]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		data, err := client.Sandbox(filesSandboxID).ReadFile(cmd.Context(), remotePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error downloading file: %v\n", err)
			os.Exit(1)
		}

		err = os.WriteFile(localPath, data, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing local file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("File downloaded to %s\n", localPath)
	},
}

// sandboxFilesWriteCmd writes content to a file.
var sandboxFilesWriteCmd = &cobra.Command{
	Use:   "write <path>",
	Short: "Write content to a file",
	Long:  `Write content to a file. Use --stdin to read from stdin or --data to provide content directly.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		var data []byte
		var err error

		if filesStdin {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				data = append(data, scanner.Bytes()...)
				data = append(data, '\n')
			}
			if err = scanner.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
				os.Exit(1)
			}
		} else if filesData != "" {
			data = []byte(filesData)
		} else {
			fmt.Fprintln(os.Stderr, "Error: must specify either --stdin or --data")
			os.Exit(1)
		}

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.Sandbox(filesSandboxID).WriteFile(cmd.Context(), path, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("File %s written successfully\n", path)
	},
}

// sandboxFilesWatchCmd watches for file changes.
var sandboxFilesWatchCmd = &cobra.Command{
	Use:   "watch <path>",
	Short: "Watch for file changes",
	Long:  `Watch for file changes in the specified path. Use -r for recursive watching.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		events, errs, unsubscribe, err := client.Sandbox(filesSandboxID).WatchFiles(ctx, path, filesRecursive)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error watching files: %v\n", err)
			os.Exit(1)
		}
		defer func() { _ = unsubscribe() }()

		fmt.Printf("Watching %s (recursive: %v). Press Ctrl+C to stop.\n", path, filesRecursive)

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
	sandboxFilesCmd.AddCommand(sandboxFilesLsCmd)
	sandboxFilesCmd.AddCommand(sandboxFilesCatCmd)
	sandboxFilesCmd.AddCommand(sandboxFilesStatCmd)
	sandboxFilesCmd.AddCommand(sandboxFilesMkdirCmd)
	sandboxFilesCmd.AddCommand(sandboxFilesRmCmd)
	sandboxFilesCmd.AddCommand(sandboxFilesMvCmd)
	sandboxFilesCmd.AddCommand(sandboxFilesUploadCmd)
	sandboxFilesCmd.AddCommand(sandboxFilesDownloadCmd)
	sandboxFilesCmd.AddCommand(sandboxFilesWriteCmd)
	sandboxFilesCmd.AddCommand(sandboxFilesWatchCmd)

	// Sandbox ID flag (required for all subcommands)
	sandboxFilesCmd.PersistentFlags().StringVarP(&filesSandboxID, "sandbox-id", "s", "", "sandbox ID (required)")
	_ = sandboxFilesCmd.MarkPersistentFlagRequired("sandbox-id")

	// Mkdir flags
	sandboxFilesMkdirCmd.Flags().BoolVar(&filesParents, "parents", false, "create parent directories as needed")

	// Watch flags
	sandboxFilesWatchCmd.Flags().BoolVarP(&filesRecursive, "recursive", "r", false, "watch recursively")

	// Write flags
	sandboxFilesWriteCmd.Flags().BoolVar(&filesStdin, "stdin", false, "read content from stdin")
	sandboxFilesWriteCmd.Flags().StringVar(&filesData, "data", "", "content to write directly")

	sandboxCmd.AddCommand(sandboxFilesCmd)
}
