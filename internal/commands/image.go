package commands

import (
	"fmt"
	"os"

	"s0/internal/docker"

	"github.com/spf13/cobra"
)

var (
	imageTag        string
	imageDockerfile string
	imagePlatform   string
	imageNoCache    bool
	imagePull       bool
)

// imageCmd represents the image command.
var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Manage template images",
	Long:  `Build, push, and manage container images for sandbox templates.

Sandboxes are created from templates, and templates reference container images.
Use these commands to build and push images to the Sandbox0 registry.`,
}

// imageBuildCmd builds a Docker image.
var imageBuildCmd = &cobra.Command{
	Use:   "build [CONTEXT]",
	Short: "Build a template image",
	Long:  `Build a container image from a Dockerfile for use in sandbox templates.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contextPath := "."
		if len(args) > 0 {
			contextPath = args[0]
		}

		if imageTag == "" {
			fmt.Fprintln(os.Stderr, "Error: --tag (-t) is required")
			os.Exit(1)
		}

		builder, err := docker.NewBuilder()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating builder: %v\n", err)
			os.Exit(1)
		}

		opts := docker.BuildOptions{
			Context:    contextPath,
			Dockerfile: imageDockerfile,
			Tags:       []string{imageTag},
			Platform:   imagePlatform,
			NoCache:    imageNoCache,
			Pull:       imagePull,
			Progress:   os.Stdout,
		}

		if err := builder.Build(cmd.Context(), opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error building image: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nImage built successfully: %s\n", imageTag)
	},
}

// imagePushCmd pushes a Docker image to the registry.
var imagePushCmd = &cobra.Command{
	Use:   "push <local-image>",
	Short: "Push a template image",
	Long:  `Push a container image to the Sandbox0 registry for use in sandbox templates.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		localImage := args[0]

		if imageTag == "" {
			fmt.Fprintln(os.Stderr, "Error: --tag (-t) is required")
			os.Exit(1)
		}

		client, err := getClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		// Get registry credentials
		creds, err := client.GetRegistryCredentials(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting registry credentials: %v\n", err)
			os.Exit(1)
		}

		pusher, err := docker.NewPusher()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating pusher: %v\n", err)
			os.Exit(1)
		}

		// Prepend registry to tag if not already present
		targetImage := imageTag
		if creds.Registry != "" {
			targetImage = fmt.Sprintf("%s/%s", creds.Registry, imageTag)
		}

		opts := docker.PushOptions{
			SourceImage: localImage,
			TargetImage: targetImage,
			Registry:    creds.Registry,
			Username:    creds.Username,
			Password:    creds.Password,
			Progress:    os.Stdout,
		}

		if err := pusher.Push(cmd.Context(), opts); err != nil {
			fmt.Fprintf(os.Stderr, "Error pushing image: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nImage pushed successfully: %s\n", targetImage)
	},
}

func init() {
	// Build command flags
	imageBuildCmd.Flags().StringVarP(&imageTag, "tag", "t", "", "image name:tag (required)")
	imageBuildCmd.Flags().StringVarP(&imageDockerfile, "file", "f", "Dockerfile", "path to Dockerfile")
	imageBuildCmd.Flags().StringVar(&imagePlatform, "platform", "", "target platform (e.g., linux/amd64)")
	imageBuildCmd.Flags().BoolVar(&imageNoCache, "no-cache", false, "do not use cache when building")
	imageBuildCmd.Flags().BoolVar(&imagePull, "pull", false, "always attempt to pull a newer version of the image")

	// Push command flags
	imagePushCmd.Flags().StringVarP(&imageTag, "tag", "t", "", "target image name:tag (required)")

	imageCmd.AddCommand(imageBuildCmd)
	imageCmd.AddCommand(imagePushCmd)
}
