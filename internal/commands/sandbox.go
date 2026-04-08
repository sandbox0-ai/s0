package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	sandboxTemplate           string
	sandboxTTL                int32
	sandboxHardTTL            int32
	sandboxConfigFile         string
	sandboxMounts             []string
	sandboxWaitForMounts      bool
	sandboxMountWaitTimeoutMS int32
	// list flags
	sandboxListStatus     string
	sandboxListTemplateID string
	sandboxListPaused     string
	sandboxListLimit      int
	sandboxListOffset     int
	// update flags
	sandboxUpdateTTL        int32
	sandboxUpdateHardTTL    int32
	sandboxUpdateAutoResume string
	sandboxUpdateConfigFile string
)

// sandboxCmd represents the sandbox command.
var sandboxCmd = &cobra.Command{
	Use:   "sandbox",
	Short: "Manage sandboxes",
	Long:  `Create, get, delete, pause, resume, refresh, and check status of sandboxes.`,
}

// sandboxCreateCmd creates a new sandbox.
var sandboxCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create (claim) a new sandbox",
	Long:  `Create a new sandbox from a template.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		request, err := buildSandboxCreateRequest(cmd.Flags().Changed("wait-for-mounts"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building sandbox create request: %v\n", err)
			os.Exit(1)
		}

		sandbox, err := client.ClaimSandboxRequest(cmd.Context(), request)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating sandbox: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, sandbox); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxGetCmd gets a sandbox by ID.
var sandboxGetCmd = &cobra.Command{
	Use:   "get <sandbox-id>",
	Short: "Get sandbox details",
	Long:  `Get details of a sandbox by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		sandbox, err := client.GetSandbox(cmd.Context(), sandboxID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting sandbox: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, sandbox); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxDeleteCmd deletes a sandbox.
var sandboxDeleteCmd = &cobra.Command{
	Use:   "delete <sandbox-id>",
	Short: "Delete a sandbox",
	Long:  `Delete (terminate) a sandbox by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.DeleteSandbox(cmd.Context(), sandboxID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting sandbox: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Sandbox %s deleted successfully\n", sandboxID)
	},
}

// sandboxPauseCmd pauses a sandbox.
var sandboxPauseCmd = &cobra.Command{
	Use:   "pause <sandbox-id>",
	Short: "Pause a sandbox",
	Long:  `Pause (suspend) a sandbox by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.PauseSandbox(cmd.Context(), sandboxID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error pausing sandbox: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Pause requested for sandbox %s\n", sandboxID)
	},
}

// sandboxResumeCmd resumes a sandbox.
var sandboxResumeCmd = &cobra.Command{
	Use:   "resume <sandbox-id>",
	Short: "Resume a sandbox",
	Long:  `Resume a paused sandbox by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.ResumeSandbox(cmd.Context(), sandboxID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resuming sandbox: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Resume requested for sandbox %s\n", sandboxID)
	},
}

// sandboxRefreshCmd refreshes a sandbox's TTL.
var sandboxRefreshCmd = &cobra.Command{
	Use:   "refresh <sandbox-id>",
	Short: "Refresh sandbox TTL",
	Long:  `Refresh the TTL of a sandbox by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		resp, err := client.RefreshSandbox(cmd.Context(), sandboxID, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error refreshing sandbox: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, resp); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxStatusCmd gets the status of a sandbox.
var sandboxStatusCmd = &cobra.Command{
	Use:   "status <sandbox-id>",
	Short: "Get sandbox status",
	Long:  `Get the status of a sandbox by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		status, err := client.StatusSandbox(cmd.Context(), sandboxID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting sandbox status: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, status); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxUpdateCmd updates a sandbox's configuration.
var sandboxUpdateCmd = &cobra.Command{
	Use:   "update <sandbox-id>",
	Short: "Update sandbox configuration",
	Long:  `Update the configuration of a sandbox (TTL, env vars, auto-resume, etc.).`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		config, hasConfig, err := buildSandboxUpdateConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building sandbox update config: %v\n", err)
			os.Exit(1)
		}

		if !hasConfig {
			fmt.Fprintln(os.Stderr, "Error: at least one update input is required (--config-file, --ttl, --hard-ttl, --auto-resume)")
			os.Exit(1)
		}

		req := apispec.SandboxUpdateRequest{
			Config: apispec.NewOptSandboxUpdateConfig(config),
		}

		sandbox, err := client.UpdateSandbox(cmd.Context(), sandboxID, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating sandbox: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, sandbox); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxListCmd lists all sandboxes.
var sandboxListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sandboxes",
	Long:  `List all sandboxes for the authenticated team.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		opts := &sandbox0.ListSandboxesOptions{}
		if sandboxListStatus != "" {
			opts.Status = sandboxListStatus
		}
		if sandboxListTemplateID != "" {
			opts.TemplateID = sandboxListTemplateID
		}
		if sandboxListPaused != "" {
			paused := sandboxListPaused == "true"
			opts.Paused = &paused
		}
		if sandboxListLimit > 0 {
			opts.Limit = &sandboxListLimit
		}
		if sandboxListOffset > 0 {
			opts.Offset = &sandboxListOffset
		}

		resp, err := client.ListSandboxes(cmd.Context(), opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing sandboxes: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, resp); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(sandboxCmd)

	// Create command flags
	sandboxCreateCmd.Flags().StringVarP(&sandboxTemplate, "template", "t", "", "template ID (required unless config file includes template)")
	sandboxCreateCmd.Flags().StringVarP(&sandboxConfigFile, "config-file", "f", "", "path to sandbox config or claim request YAML/JSON file, or - for stdin")
	sandboxCreateCmd.Flags().Int32Var(&sandboxTTL, "ttl", 0, "soft TTL in seconds")
	sandboxCreateCmd.Flags().Int32Var(&sandboxHardTTL, "hard-ttl", 0, "hard TTL in seconds")
	sandboxCreateCmd.Flags().StringArrayVar(&sandboxMounts, "mount", nil, "bootstrap mount in the form <sandboxvolume-id>:/absolute/path (repeatable)")
	sandboxCreateCmd.Flags().BoolVar(&sandboxWaitForMounts, "wait-for-mounts", false, "wait best-effort for bootstrap mounts before claim returns")
	sandboxCreateCmd.Flags().Int32Var(&sandboxMountWaitTimeoutMS, "mount-wait-timeout-ms", 0, "best-effort bootstrap mount wait budget in milliseconds")

	sandboxCmd.AddCommand(sandboxCreateCmd)
	sandboxCmd.AddCommand(sandboxGetCmd)
	sandboxCmd.AddCommand(sandboxDeleteCmd)
	sandboxCmd.AddCommand(sandboxPauseCmd)
	sandboxCmd.AddCommand(sandboxResumeCmd)
	sandboxCmd.AddCommand(sandboxRefreshCmd)
	sandboxCmd.AddCommand(sandboxStatusCmd)
	sandboxCmd.AddCommand(sandboxUpdateCmd)

	// Update command flags
	sandboxUpdateCmd.Flags().StringVarP(&sandboxUpdateConfigFile, "config-file", "f", "", "path to sandbox update config YAML/JSON file, or - for stdin")
	sandboxUpdateCmd.Flags().Int32Var(&sandboxUpdateTTL, "ttl", 0, "soft TTL in seconds")
	sandboxUpdateCmd.Flags().Int32Var(&sandboxUpdateHardTTL, "hard-ttl", 0, "hard TTL in seconds")
	sandboxUpdateCmd.Flags().StringVar(&sandboxUpdateAutoResume, "auto-resume", "", "auto resume on access (true/false)")

	// List command flags
	sandboxListCmd.Flags().StringVar(&sandboxListStatus, "status", "", "filter by status (starting, running, failed, completed)")
	sandboxListCmd.Flags().StringVar(&sandboxListTemplateID, "template-id", "", "filter by template ID")
	sandboxListCmd.Flags().StringVar(&sandboxListPaused, "paused", "", "filter by paused state (true/false)")
	sandboxListCmd.Flags().IntVar(&sandboxListLimit, "limit", 50, "maximum number of results")
	sandboxListCmd.Flags().IntVar(&sandboxListOffset, "offset", 0, "pagination offset")
	sandboxCmd.AddCommand(sandboxListCmd)
}

func buildSandboxCreateRequest(waitForMountsSet bool) (apispec.ClaimRequest, error) {
	request := apispec.ClaimRequest{}
	if sandboxConfigFile != "" {
		var err error
		request, err = readSandboxCreateInputFile(sandboxConfigFile)
		if err != nil {
			return apispec.ClaimRequest{}, err
		}
	}
	if sandboxTemplate != "" {
		request.Template = apispec.NewOptString(sandboxTemplate)
	}

	configOverrides, hasConfigOverrides, err := buildSandboxCreateConfigOverrides()
	if err != nil {
		return apispec.ClaimRequest{}, err
	}
	if hasConfigOverrides {
		config := apispec.SandboxConfig{}
		if existing, ok := request.Config.Get(); ok {
			config = existing
		}
		mergeSandboxCreateConfig(&config, configOverrides)
		if err := config.Validate(); err != nil {
			return apispec.ClaimRequest{}, fmt.Errorf("invalid sandbox config: %w", err)
		}
		request.Config = apispec.NewOptSandboxConfig(config)
	}

	mounts, err := parseSandboxCreateMounts(sandboxMounts)
	if err != nil {
		return apispec.ClaimRequest{}, err
	}
	if len(mounts) > 0 {
		request.Mounts = append(request.Mounts, mounts...)
	}
	if waitForMountsSet {
		request.WaitForMounts = apispec.NewOptBool(sandboxWaitForMounts)
	}
	if sandboxMountWaitTimeoutMS > 0 {
		request.MountWaitTimeoutMs = apispec.NewOptInt32(sandboxMountWaitTimeoutMS)
		if !waitForMountsSet {
			request.WaitForMounts = apispec.NewOptBool(true)
		}
	}
	if _, ok := request.Template.Get(); !ok {
		return apispec.ClaimRequest{}, fmt.Errorf("--template is required unless provided in config file")
	}
	if err := request.Validate(); err != nil {
		return apispec.ClaimRequest{}, fmt.Errorf("invalid sandbox create request: %w", err)
	}
	return request, nil
}

func buildSandboxCreateConfigOverrides() (apispec.SandboxConfig, bool, error) {
	var (
		config apispec.SandboxConfig
	)
	hasConfig := false
	if sandboxTTL > 0 {
		config.TTL = apispec.NewOptInt32(sandboxTTL)
		hasConfig = true
	}
	if sandboxHardTTL > 0 {
		config.HardTTL = apispec.NewOptInt32(sandboxHardTTL)
		hasConfig = true
	}
	if hasConfig {
		if err := config.Validate(); err != nil {
			return apispec.SandboxConfig{}, false, fmt.Errorf("invalid sandbox config: %w", err)
		}
	}
	return config, hasConfig, nil
}

func mergeSandboxCreateConfig(dst *apispec.SandboxConfig, src apispec.SandboxConfig) {
	if value, ok := src.TTL.Get(); ok {
		dst.TTL = apispec.NewOptInt32(value)
	}
	if value, ok := src.HardTTL.Get(); ok {
		dst.HardTTL = apispec.NewOptInt32(value)
	}
}

func parseSandboxCreateMounts(values []string) ([]apispec.ClaimMountRequest, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]apispec.ClaimMountRequest, 0, len(values))
	for _, raw := range values {
		volumeID, mountPoint, ok := strings.Cut(raw, ":")
		if !ok || volumeID == "" || mountPoint == "" {
			return nil, fmt.Errorf("invalid --mount %q: expected <sandboxvolume-id>:/absolute/path", raw)
		}
		if !strings.HasPrefix(mountPoint, "/") {
			return nil, fmt.Errorf("invalid --mount %q: mount path must be absolute", raw)
		}
		out = append(out, apispec.ClaimMountRequest{
			SandboxvolumeID: volumeID,
			MountPoint:      mountPoint,
		})
	}
	return out, nil
}

func readSandboxCreateInputFile(path string) (apispec.ClaimRequest, error) {
	data, err := readConfigFile(path)
	if err != nil {
		return apispec.ClaimRequest{}, err
	}
	claimLike, err := isSandboxCreateClaimRequest(data)
	if err != nil {
		return apispec.ClaimRequest{}, err
	}
	if claimLike {
		var request apispec.ClaimRequest
		if err := yaml.Unmarshal(data, &request); err != nil {
			return apispec.ClaimRequest{}, fmt.Errorf("parse sandbox create file: %w", err)
		}
		return request, nil
	}
	config, err := readSandboxConfigFile(path)
	if err != nil {
		return apispec.ClaimRequest{}, err
	}
	return apispec.ClaimRequest{Config: apispec.NewOptSandboxConfig(config)}, nil
}

func isSandboxCreateClaimRequest(data []byte) (bool, error) {
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return false, fmt.Errorf("parse sandbox create file: %w", err)
	}
	for _, key := range []string{"template", "config", "mounts", "wait_for_mounts", "mount_wait_timeout_ms"} {
		if _, ok := raw[key]; ok {
			return true, nil
		}
	}
	return false, nil
}

func buildSandboxUpdateConfig() (apispec.SandboxUpdateConfig, bool, error) {
	var (
		config apispec.SandboxUpdateConfig
		err    error
	)
	hasConfig := false
	if sandboxUpdateConfigFile != "" {
		config, err = readSandboxUpdateConfigFile(sandboxUpdateConfigFile)
		if err != nil {
			return apispec.SandboxUpdateConfig{}, false, err
		}
		hasConfig = true
	}
	if sandboxUpdateTTL > 0 {
		config.TTL = apispec.NewOptInt32(sandboxUpdateTTL)
		hasConfig = true
	}
	if sandboxUpdateHardTTL > 0 {
		config.HardTTL = apispec.NewOptInt32(sandboxUpdateHardTTL)
		hasConfig = true
	}
	if sandboxUpdateAutoResume != "" {
		autoResume := sandboxUpdateAutoResume == "true"
		config.AutoResume = apispec.NewOptBool(autoResume)
		hasConfig = true
	}
	if hasConfig {
		if err := config.Validate(); err != nil {
			return apispec.SandboxUpdateConfig{}, false, fmt.Errorf("invalid sandbox update config: %w", err)
		}
	}
	return config, hasConfig, nil
}

func readSandboxConfigFile(path string) (apispec.SandboxConfig, error) {
	data, err := readConfigFile(path)
	if err != nil {
		return apispec.SandboxConfig{}, err
	}
	var config apispec.SandboxConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return apispec.SandboxConfig{}, fmt.Errorf("parse sandbox config file: %w", err)
	}
	if err := config.Validate(); err != nil {
		return apispec.SandboxConfig{}, fmt.Errorf("invalid sandbox config: %w", err)
	}
	return config, nil
}

func readSandboxUpdateConfigFile(path string) (apispec.SandboxUpdateConfig, error) {
	data, err := readConfigFile(path)
	if err != nil {
		return apispec.SandboxUpdateConfig{}, err
	}
	var config apispec.SandboxUpdateConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return apispec.SandboxUpdateConfig{}, fmt.Errorf("parse sandbox update config file: %w", err)
	}
	if err := config.Validate(); err != nil {
		return apispec.SandboxUpdateConfig{}, fmt.Errorf("invalid sandbox update config: %w", err)
	}
	return config, nil
}

func readConfigFile(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}
