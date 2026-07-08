package commands

import (
	"fmt"
	"os"

	sdk "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

// Volume create flags.
var (
	volumeAccessMode       string
	volumeCreateSnapshotID string
	volumeCreateBackend    string
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
	accessMode     string
	snapshotID     string
	backend        string
	s3Provider     string
	s3Bucket       string
	s3Prefix       string
	s3Region       string
	s3EndpointURL  string
	s3AccessKey    string
	s3SecretKey    string
	s3SessionToken string
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
			accessMode:     volumeAccessMode,
			snapshotID:     volumeCreateSnapshotID,
			backend:        volumeCreateBackend,
			s3Provider:     volumeS3Provider,
			s3Bucket:       volumeS3Bucket,
			s3Prefix:       volumeS3Prefix,
			s3Region:       volumeS3Region,
			s3EndpointURL:  volumeS3EndpointURL,
			s3AccessKey:    volumeS3AccessKey,
			s3SecretKey:    volumeS3SecretKey,
			s3SessionToken: volumeS3SessionToken,
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

	if opts.accessMode != "" {
		req.AccessMode = apispec.NewOptVolumeAccessMode(apispec.VolumeAccessMode(opts.accessMode))
	}
	if opts.snapshotID != "" {
		req.SnapshotID = apispec.NewOptString(opts.snapshotID)
	}

	hasS3Config := opts.s3Provider != "" ||
		opts.s3Bucket != "" ||
		opts.s3Prefix != "" ||
		opts.s3Region != "" ||
		opts.s3EndpointURL != "" ||
		opts.s3AccessKey != "" ||
		opts.s3SecretKey != "" ||
		opts.s3SessionToken != ""

	switch opts.backend {
	case "":
	case string(apispec.VolumeBackendS0fs):
		req.Backend = apispec.NewOptVolumeBackend(apispec.VolumeBackendS0fs)
		if hasS3Config {
			return req, fmt.Errorf("S3 flags require --backend s3")
		}
	case string(apispec.VolumeBackendS3):
		req.Backend = apispec.NewOptVolumeBackend(apispec.VolumeBackendS3)
		hasS3Config = true
	default:
		return req, fmt.Errorf("unsupported volume backend %q (expected s0fs or s3)", opts.backend)
	}

	if !hasS3Config {
		return req, nil
	}

	if opts.snapshotID != "" {
		return req, fmt.Errorf("S3 volumes do not support --snapshot-id")
	}
	if opts.accessMode == string(apispec.VolumeAccessModeRWX) {
		return req, fmt.Errorf("S3 volumes do not support --access-mode RWX")
	}
	if opts.s3Bucket == "" {
		return req, fmt.Errorf("--s3-bucket is required for S3 volumes")
	}
	if (opts.s3AccessKey == "") != (opts.s3SecretKey == "") {
		return req, fmt.Errorf("--s3-access-key and --s3-secret-key must be provided together")
	}

	s3 := apispec.CreateSandboxVolumeS3Config{
		Bucket: opts.s3Bucket,
	}
	if opts.s3Provider != "" {
		provider, err := parseCreateVolumeS3Provider(opts.s3Provider)
		if err != nil {
			return req, err
		}
		s3.Provider = apispec.NewOptCreateSandboxVolumeS3ConfigProvider(provider)
		if provider != apispec.CreateSandboxVolumeS3ConfigProviderAWS && opts.s3EndpointURL == "" {
			return req, fmt.Errorf("--s3-endpoint-url is required for provider %s", provider)
		}
	}
	if opts.s3Prefix != "" {
		s3.Prefix = apispec.NewOptString(opts.s3Prefix)
	}
	if opts.s3Region != "" {
		s3.Region = apispec.NewOptString(opts.s3Region)
	}
	if opts.s3EndpointURL != "" {
		s3.EndpointURL = apispec.NewOptString(opts.s3EndpointURL)
	}
	if opts.s3AccessKey != "" {
		s3.AccessKey = opts.s3AccessKey
		s3.SecretKey = opts.s3SecretKey
	}
	if opts.s3SessionToken != "" {
		s3.SessionToken = apispec.NewOptString(opts.s3SessionToken)
	}

	req.Backend = apispec.NewOptVolumeBackend(apispec.VolumeBackendS3)
	req.S3 = apispec.NewOptCreateSandboxVolumeS3Config(s3)
	return req, nil
}

func parseCreateVolumeS3Provider(value string) (apispec.CreateSandboxVolumeS3ConfigProvider, error) {
	switch value {
	case string(apispec.CreateSandboxVolumeS3ConfigProviderAWS):
		return apispec.CreateSandboxVolumeS3ConfigProviderAWS, nil
	case string(apispec.CreateSandboxVolumeS3ConfigProviderAli):
		return apispec.CreateSandboxVolumeS3ConfigProviderAli, nil
	case string(apispec.CreateSandboxVolumeS3ConfigProviderR2):
		return apispec.CreateSandboxVolumeS3ConfigProviderR2, nil
	default:
		return "", fmt.Errorf("unsupported S3 provider %q (expected aws, ali, or r2)", value)
	}
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
	volumeCreateCmd.Flags().StringVar(&volumeCreateSnapshotID, "snapshot-id", "", "snapshot ID used to initialize the new volume")
	volumeCreateCmd.Flags().StringVar(&volumeCreateBackend, "backend", "", "volume backend (s0fs or s3)")
	volumeCreateCmd.Flags().StringVar(&volumeS3Provider, "s3-provider", "", "S3-compatible provider (aws, ali, or r2; defaults to aws)")
	volumeCreateCmd.Flags().StringVar(&volumeS3Bucket, "s3-bucket", "", "S3 bucket to expose as the volume root")
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
