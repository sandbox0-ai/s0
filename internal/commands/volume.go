package commands

import (
	"fmt"
	"os"
	"strings"

	sdk "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

// Volume create flags.
var (
	volumeAccessMode       string
	volumeBackend          string
	volumeCreateSnapshotID string
	volumeS3Provider       string
	volumeS3Bucket         string
	volumeS3Prefix         string
	volumeS3Region         string
	volumeS3EndpointURL    string
	volumeS3AccessKey      string
	volumeS3SecretKey      string
	volumeS3SessionToken   string
	volumeDeleteForce      bool
)

type createVolumeOptions struct {
	AccessMode  string
	Backend     string
	SnapshotID  string
	S3Provider  string
	S3Bucket    string
	S3Prefix    string
	S3Region    string
	S3Endpoint  string
	S3AccessKey string
	S3SecretKey string
	S3Token     string
}

// volumeCmd represents the volume command.
var volumeCmd = &cobra.Command{
	Use:   "volume",
	Short: "Manage volumes",
	Long:  `List, get, create, fork, and delete sandbox volumes.`,
}

// volumeListCmd lists all volumes.
var volumeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List volumes",
	Long:  `List all sandbox volumes.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
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

		client, err := getClientRaw(cmd)
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
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		req, err := buildCreateVolumeRequest(createVolumeOptions{
			AccessMode:  volumeAccessMode,
			Backend:     volumeBackend,
			SnapshotID:  volumeCreateSnapshotID,
			S3Provider:  volumeS3Provider,
			S3Bucket:    volumeS3Bucket,
			S3Prefix:    volumeS3Prefix,
			S3Region:    volumeS3Region,
			S3Endpoint:  volumeS3EndpointURL,
			S3AccessKey: volumeS3AccessKey,
			S3SecretKey: volumeS3SecretKey,
			S3Token:     volumeS3SessionToken,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building volume request: %v\n", err)
			os.Exit(1)
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

func buildCreateVolumeRequest(opts createVolumeOptions) (apispec.CreateSandboxVolumeRequest, error) {
	req := apispec.CreateSandboxVolumeRequest{}

	if opts.AccessMode != "" {
		req.AccessMode = apispec.NewOptVolumeAccessMode(apispec.VolumeAccessMode(opts.AccessMode))
	}
	if opts.SnapshotID != "" {
		req.SnapshotID = apispec.NewOptString(opts.SnapshotID)
	}

	backend := strings.ToLower(strings.TrimSpace(opts.Backend))
	if backend == "" && opts.hasS3Config() {
		backend = string(apispec.VolumeBackendS3)
	}
	switch backend {
	case "":
	case string(apispec.VolumeBackendS0fs):
		if opts.hasS3Config() {
			return req, fmt.Errorf("S3 flags require --backend s3")
		}
		req.Backend = apispec.NewOptVolumeBackend(apispec.VolumeBackendS0fs)
	case string(apispec.VolumeBackendS3):
		s3, err := buildCreateVolumeS3Config(opts)
		if err != nil {
			return req, err
		}
		req.Backend = apispec.NewOptVolumeBackend(apispec.VolumeBackendS3)
		req.S3 = apispec.NewOptCreateSandboxVolumeS3Config(s3)
	default:
		return req, fmt.Errorf("--backend must be s0fs or s3")
	}

	return req, nil
}

func (opts createVolumeOptions) hasS3Config() bool {
	return strings.TrimSpace(opts.S3Provider) != "" ||
		strings.TrimSpace(opts.S3Bucket) != "" ||
		strings.TrimSpace(opts.S3Prefix) != "" ||
		strings.TrimSpace(opts.S3Region) != "" ||
		strings.TrimSpace(opts.S3Endpoint) != "" ||
		strings.TrimSpace(opts.S3AccessKey) != "" ||
		strings.TrimSpace(opts.S3SecretKey) != "" ||
		strings.TrimSpace(opts.S3Token) != ""
}

func buildCreateVolumeS3Config(opts createVolumeOptions) (apispec.CreateSandboxVolumeS3Config, error) {
	bucket := strings.TrimSpace(opts.S3Bucket)
	if bucket == "" {
		return apispec.CreateSandboxVolumeS3Config{}, fmt.Errorf("--s3-bucket is required for --backend s3")
	}
	if (strings.TrimSpace(opts.S3AccessKey) == "") != (strings.TrimSpace(opts.S3SecretKey) == "") {
		return apispec.CreateSandboxVolumeS3Config{}, fmt.Errorf("--s3-access-key and --s3-secret-key must be provided together")
	}

	s3 := apispec.CreateSandboxVolumeS3Config{Bucket: bucket}
	if provider := strings.ToLower(strings.TrimSpace(opts.S3Provider)); provider != "" {
		switch provider {
		case string(apispec.CreateSandboxVolumeS3ConfigProviderAWS),
			string(apispec.CreateSandboxVolumeS3ConfigProviderAli),
			string(apispec.CreateSandboxVolumeS3ConfigProviderR2):
			s3.Provider = apispec.NewOptCreateSandboxVolumeS3ConfigProvider(apispec.CreateSandboxVolumeS3ConfigProvider(provider))
		default:
			return apispec.CreateSandboxVolumeS3Config{}, fmt.Errorf("--s3-provider must be aws, ali, or r2")
		}
	}
	if prefix := strings.TrimSpace(opts.S3Prefix); prefix != "" {
		s3.Prefix = apispec.NewOptString(prefix)
	}
	if region := strings.TrimSpace(opts.S3Region); region != "" {
		s3.Region = apispec.NewOptString(region)
	}
	if endpoint := strings.TrimSpace(opts.S3Endpoint); endpoint != "" {
		s3.EndpointURL = apispec.NewOptString(endpoint)
	}
	if accessKey := strings.TrimSpace(opts.S3AccessKey); accessKey != "" {
		s3.AccessKey = apispec.NewOptString(accessKey)
		s3.SecretKey = apispec.NewOptString(strings.TrimSpace(opts.S3SecretKey))
	}
	if token := strings.TrimSpace(opts.S3Token); token != "" {
		s3.SessionToken = apispec.NewOptString(token)
	}
	return s3, nil
}

// volumeDeleteCmd deletes a volume.
var volumeDeleteCmd = &cobra.Command{
	Use:   "delete <volume-id>",
	Short: "Delete a volume",
	Long:  `Delete a sandbox volume by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]

		client, err := getClientRaw(cmd)
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

// volumeForkCmd forks a volume.
var volumeForkCmd = &cobra.Command{
	Use:   "fork <volume-id>",
	Short: "Fork a volume",
	Long:  `Create an independent copy of a volume using Copy-On-Write.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		req := &apispec.ForkVolumeRequest{}

		if volumeAccessMode != "" {
			req.AccessMode = apispec.NewOptVolumeAccessMode(apispec.VolumeAccessMode(volumeAccessMode))
		}

		volume, err := client.ForkVolume(cmd.Context(), volumeID, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error forking volume: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, volume); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(volumeCmd)

	volumeCmd.AddCommand(volumeListCmd)
	volumeCmd.AddCommand(volumeGetCmd)
	volumeCmd.AddCommand(volumeCreateCmd)
	volumeCmd.AddCommand(volumeDeleteCmd)
	volumeCmd.AddCommand(volumeForkCmd)

	// Volume create flags
	volumeCreateCmd.Flags().StringVar(&volumeAccessMode, "access-mode", "", "access mode (RWO, ROX, or RWX)")
	volumeCreateCmd.Flags().StringVar(&volumeBackend, "backend", "", "volume backend (s0fs or s3)")
	volumeCreateCmd.Flags().StringVar(&volumeCreateSnapshotID, "snapshot-id", "", "snapshot ID used to initialize the new volume")
	volumeCreateCmd.Flags().StringVar(&volumeS3Provider, "s3-provider", "", "S3-compatible provider (aws, ali, or r2)")
	volumeCreateCmd.Flags().StringVar(&volumeS3Bucket, "s3-bucket", "", "S3 bucket name for --backend s3")
	volumeCreateCmd.Flags().StringVar(&volumeS3Prefix, "s3-prefix", "", "S3 object key prefix to expose as the volume root")
	volumeCreateCmd.Flags().StringVar(&volumeS3Region, "s3-region", "", "S3 region override")
	volumeCreateCmd.Flags().StringVar(&volumeS3EndpointURL, "s3-endpoint-url", "", "S3-compatible endpoint URL override")
	volumeCreateCmd.Flags().StringVar(&volumeS3AccessKey, "s3-access-key", "", "S3 access key override")
	volumeCreateCmd.Flags().StringVar(&volumeS3SecretKey, "s3-secret-key", "", "S3 secret key override")
	volumeCreateCmd.Flags().StringVar(&volumeS3SessionToken, "s3-session-token", "", "S3 temporary credential session token")
	volumeDeleteCmd.Flags().BoolVar(&volumeDeleteForce, "force", false, "force delete volume even if it has active mounts")

	// Volume fork flags
	volumeForkCmd.Flags().StringVar(&volumeAccessMode, "access-mode", "", "access mode override (RWO, ROX, or RWX)")
}
