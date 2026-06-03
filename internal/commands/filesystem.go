package commands

import (
	"fmt"
	"os"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	filesystemCreateTemplate        string
	filesystemCreateSnapshotID      string
	filesystemCreateBaseImageDigest string
	filesystemCreateS0FSHead        string
	filesystemForkTemplate          string
)

// filesystemCmd represents the filesystem command.
var filesystemCmd = &cobra.Command{
	Use:   "filesystem",
	Short: "Manage sandbox filesystems",
	Long:  `List, get, create, fork, and delete persistent sandbox filesystems.`,
}

// filesystemListCmd lists all filesystems.
var filesystemListCmd = &cobra.Command{
	Use:   "list",
	Short: "List filesystems",
	Long:  `List all persistent sandbox filesystems.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		filesystems, err := client.ListFilesystems(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing filesystems: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, filesystems); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// filesystemGetCmd gets a filesystem by ID.
var filesystemGetCmd = &cobra.Command{
	Use:   "get <filesystem-id>",
	Short: "Get filesystem details",
	Long:  `Get details of a persistent sandbox filesystem by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filesystemID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		filesystem, err := client.GetFilesystem(cmd.Context(), filesystemID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting filesystem: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, filesystem); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// filesystemCreateCmd creates a filesystem.
var filesystemCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a filesystem",
	Long:  `Create a persistent sandbox filesystem.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		req := buildCreateFilesystemRequest(
			filesystemCreateTemplate,
			filesystemCreateSnapshotID,
			filesystemCreateBaseImageDigest,
			filesystemCreateS0FSHead,
		)

		filesystem, err := client.CreateFilesystem(cmd.Context(), req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating filesystem: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, filesystem); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

func buildCreateFilesystemRequest(templateID, snapshotID, baseImageDigest, s0fsHead string) apispec.CreateSandboxFilesystemRequest {
	req := apispec.CreateSandboxFilesystemRequest{}
	if templateID != "" {
		req.Template = apispec.NewOptString(templateID)
	}
	if snapshotID != "" {
		req.SnapshotID = apispec.NewOptString(snapshotID)
	}
	if baseImageDigest != "" {
		req.BaseImageDigest = apispec.NewOptString(baseImageDigest)
	}
	if s0fsHead != "" {
		req.S0fsHead = apispec.NewOptString(s0fsHead)
	}
	return req
}

// filesystemDeleteCmd deletes a filesystem.
var filesystemDeleteCmd = &cobra.Command{
	Use:   "delete <filesystem-id>",
	Short: "Delete a filesystem",
	Long:  `Delete a persistent sandbox filesystem by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filesystemID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.DeleteFilesystem(cmd.Context(), filesystemID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting filesystem: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Filesystem %s deleted successfully\n", filesystemID)
	},
}

// filesystemForkCmd forks a filesystem.
var filesystemForkCmd = &cobra.Command{
	Use:   "fork <filesystem-id>",
	Short: "Fork a filesystem",
	Long:  `Create an independent persistent sandbox filesystem using Copy-On-Write.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filesystemID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		filesystem, err := client.ForkFilesystem(cmd.Context(), filesystemID, buildForkFilesystemRequest(filesystemForkTemplate))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error forking filesystem: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, filesystem); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

func buildForkFilesystemRequest(templateID string) *apispec.ForkSandboxFilesystemRequest {
	if templateID == "" {
		return nil
	}
	return &apispec.ForkSandboxFilesystemRequest{
		Template: apispec.NewOptString(templateID),
	}
}

func init() {
	rootCmd.AddCommand(filesystemCmd)

	filesystemCmd.AddCommand(filesystemListCmd)
	filesystemCmd.AddCommand(filesystemGetCmd)
	filesystemCmd.AddCommand(filesystemCreateCmd)
	filesystemCmd.AddCommand(filesystemDeleteCmd)
	filesystemCmd.AddCommand(filesystemForkCmd)

	filesystemCreateCmd.Flags().StringVar(&filesystemCreateTemplate, "template", "", "template ID associated with the filesystem")
	filesystemCreateCmd.Flags().StringVar(&filesystemCreateSnapshotID, "snapshot-id", "", "filesystem snapshot ID used to initialize the new filesystem")
	filesystemCreateCmd.Flags().StringVar(&filesystemCreateBaseImageDigest, "base-image-digest", "", "resolved immutable base image digest")
	filesystemCreateCmd.Flags().StringVar(&filesystemCreateS0FSHead, "s0fs-head", "", "initial s0fs head for low-level restore/import paths")

	filesystemForkCmd.Flags().StringVar(&filesystemForkTemplate, "template", "", "template ID override for the forked filesystem")
}
