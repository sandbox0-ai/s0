package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	sandboxTemplate   string
	sandboxTTL        int32
	sandboxHardTTL    int32
	sandboxMemory     string
	sandboxConfigFile string
	sandboxMounts     []string
	sandboxSnapshotID string
	// list flags
	sandboxListStatus     string
	sandboxListTemplateID string
	sandboxListPaused     string
	sandboxListLimit      int
	sandboxListOffset     int
	// observability flags
	sandboxObsLimit      int
	sandboxObsCursor     string
	sandboxObsStartTime  string
	sandboxObsEndTime    string
	sandboxObsSince      string
	sandboxObsWatch      bool
	sandboxObsContextID  string
	sandboxObsStream     string
	sandboxObsNames      []string
	sandboxObsSource     string
	sandboxObsEventType  string
	sandboxObsOutcome    string
	sandboxMetricStep    int
	sandboxMetricStat    string
	sandboxMetricPoints  int
	sandboxLogsFollow    bool
	sandboxLogsTailLines int
	sandboxLogsSinceSecs int64
	// update flags
	sandboxUpdateTTL        int32
	sandboxUpdateHardTTL    int32
	sandboxUpdateMemory     string
	sandboxUpdateAutoResume string
	sandboxUpdateConfigFile string
)

type sandboxCreateOutput struct {
	ID              string                `json:"id"`
	Template        string                `json:"template"`
	ClusterID       *string               `json:"cluster_id,omitempty"`
	PodName         string                `json:"pod_name"`
	Status          string                `json:"status"`
	BootstrapMounts []apispec.MountStatus `json:"bootstrap_mounts"`
}

func sandboxCreateOutputValue(sandbox *sandbox0.Sandbox) any {
	if cfgFormat != "json" && cfgFormat != "yaml" {
		return sandbox
	}
	return sandboxCreateOutput{
		ID:              sandbox.ID,
		Template:        sandbox.Template,
		ClusterID:       sandbox.ClusterID,
		PodName:         sandbox.PodName,
		Status:          sandbox.Status,
		BootstrapMounts: sandbox.BootstrapMounts,
	}
}

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

		request, err := buildSandboxCreateRequest()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building sandbox create request: %v\n", err)
			os.Exit(1)
		}

		sandbox, err := client.ClaimSandboxRequest(cmd.Context(), request)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating sandbox: %s\n", formatSandboxCreateError(err))
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, sandboxCreateOutputValue(sandbox)); err != nil {
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
			fmt.Fprintln(os.Stderr, "Error: at least one update input is required (--config-file, --ttl, --hard-ttl, --memory, --auto-resume)")
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

// sandboxLogsCmd queries sandbox log observability.
var sandboxLogsCmd = &cobra.Command{
	Use:   "logs <sandbox-id>",
	Short: "Query sandbox logs",
	Long:  `Query sandbox logs from the per-sandbox observability backend. Use --watch for realtime NDJSON streaming.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		options, watch, err := buildSandboxLogObservabilityOptions(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building sandbox logs request: %v\n", err)
			os.Exit(1)
		}

		sandbox := client.Sandbox(sandboxID)
		if watch {
			stream, err := sandbox.WatchLogs(cmd.Context(), options)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error streaming sandbox logs: %v\n", err)
				os.Exit(1)
			}
			defer stream.Close()
			if err := writeObservabilityWatch(stream); err != nil {
				fmt.Fprintf(os.Stderr, "Error reading sandbox log stream: %v\n", err)
				os.Exit(1)
			}
			return
		}

		logs, err := sandbox.ListLogs(cmd.Context(), options)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting sandbox logs: %v\n", err)
			os.Exit(1)
		}
		if cfgFormat == "json" || cfgFormat == "yaml" {
			if err := getFormatter().Format(os.Stdout, logs); err != nil {
				fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
				os.Exit(1)
			}
			return
		}
		writeObservabilityLogs(os.Stdout, logs.Logs)
	},
}

// sandboxEventsCmd queries sandbox observability events.
var sandboxEventsCmd = &cobra.Command{
	Use:   "events <sandbox-id>",
	Short: "Query sandbox observability events",
	Long:  `Query sandbox lifecycle, network audit, and runtime stats events from the per-sandbox observability backend.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}
		options, watch, err := buildSandboxEventObservabilityOptions(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building sandbox events request: %v\n", err)
			os.Exit(1)
		}
		sandbox := client.Sandbox(sandboxID)
		if watch {
			stream, err := sandbox.WatchObservabilityEvents(cmd.Context(), options)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error streaming sandbox events: %v\n", err)
				os.Exit(1)
			}
			defer stream.Close()
			if err := writeObservabilityWatch(stream); err != nil {
				fmt.Fprintf(os.Stderr, "Error reading sandbox event stream: %v\n", err)
				os.Exit(1)
			}
			return
		}
		events, err := sandbox.ListObservabilityEvents(cmd.Context(), options)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting sandbox events: %v\n", err)
			os.Exit(1)
		}
		if err := getFormatter().Format(os.Stdout, events); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxMetricsCmd queries chart-ready sandbox runtime metrics.
var sandboxMetricsCmd = &cobra.Command{
	Use:   "metrics <sandbox-id>",
	Short: "Query sandbox metrics",
	Long:  `Query bounded, downsampled sandbox runtime metric series.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}
		options, err := buildSandboxMetricObservabilityOptions(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building sandbox metrics request: %v\n", err)
			os.Exit(1)
		}
		sandbox := client.Sandbox(sandboxID)
		metrics, err := sandbox.ListMetrics(cmd.Context(), options)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting sandbox metrics: %v\n", err)
			os.Exit(1)
		}
		if err := getFormatter().Format(os.Stdout, metrics); err != nil {
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
	sandboxCreateCmd.Flags().StringVar(&sandboxMemory, "memory", "", "sandbox memory limit, for example 512Mi or 2Gi")
	sandboxCreateCmd.Flags().StringArrayVar(&sandboxMounts, "mount", nil, "bootstrap mount in the form <sandboxvolume-id>:/absolute/path (repeatable)")
	sandboxCreateCmd.Flags().StringVar(&sandboxSnapshotID, "snapshot-id", "", "rootfs snapshot ID used to initialize the new sandbox")

	sandboxCmd.AddCommand(sandboxCreateCmd)
	sandboxCmd.AddCommand(sandboxGetCmd)
	sandboxCmd.AddCommand(sandboxDeleteCmd)
	sandboxCmd.AddCommand(sandboxPauseCmd)
	sandboxCmd.AddCommand(sandboxResumeCmd)
	sandboxCmd.AddCommand(sandboxRefreshCmd)
	sandboxCmd.AddCommand(sandboxStatusCmd)
	sandboxCmd.AddCommand(sandboxUpdateCmd)
	sandboxCmd.AddCommand(sandboxLogsCmd)
	sandboxCmd.AddCommand(sandboxEventsCmd)
	sandboxCmd.AddCommand(sandboxMetricsCmd)

	// Update command flags
	sandboxUpdateCmd.Flags().StringVarP(&sandboxUpdateConfigFile, "config-file", "f", "", "path to sandbox update config YAML/JSON file, or - for stdin")
	sandboxUpdateCmd.Flags().Int32Var(&sandboxUpdateTTL, "ttl", 0, "soft TTL in seconds")
	sandboxUpdateCmd.Flags().Int32Var(&sandboxUpdateHardTTL, "hard-ttl", 0, "hard TTL in seconds")
	sandboxUpdateCmd.Flags().StringVar(&sandboxUpdateMemory, "memory", "", "sandbox memory limit, for example 512Mi or 2Gi")
	sandboxUpdateCmd.Flags().StringVar(&sandboxUpdateAutoResume, "auto-resume", "", "auto resume on access (true/false)")

	// List command flags
	sandboxListCmd.Flags().StringVar(&sandboxListStatus, "status", "", "filter by status (starting, running, failed, completed)")
	sandboxListCmd.Flags().StringVar(&sandboxListTemplateID, "template-id", "", "filter by template ID")
	sandboxListCmd.Flags().StringVar(&sandboxListPaused, "paused", "", "filter by paused state (true/false)")
	sandboxListCmd.Flags().IntVar(&sandboxListLimit, "limit", 50, "maximum number of results")
	sandboxListCmd.Flags().IntVar(&sandboxListOffset, "offset", 0, "pagination offset")
	sandboxCmd.AddCommand(sandboxListCmd)

	addSandboxObservabilityFlags(sandboxLogsCmd)
	sandboxLogsCmd.Flags().StringVar(&sandboxObsContextID, "context-id", "", "filter by context ID")
	sandboxLogsCmd.Flags().StringVar(&sandboxObsStream, "stream", "", "filter by log stream (stdout, stderr, pty)")
	sandboxLogsCmd.Flags().BoolVarP(&sandboxLogsFollow, "follow", "f", false, "deprecated alias for --watch")
	sandboxLogsCmd.Flags().IntVar(&sandboxLogsTailLines, "tail", 0, "deprecated alias for --limit")
	sandboxLogsCmd.Flags().Int64Var(&sandboxLogsSinceSecs, "since-seconds", 0, "deprecated alias for --since")
	_ = sandboxLogsCmd.Flags().MarkDeprecated("follow", "use --watch")
	_ = sandboxLogsCmd.Flags().MarkDeprecated("tail", "use --limit")
	_ = sandboxLogsCmd.Flags().MarkDeprecated("since-seconds", "use --since")

	addSandboxObservabilityFlags(sandboxEventsCmd)
	addSandboxEventFilterFlags(sandboxEventsCmd)

	sandboxMetricsCmd.Flags().StringVar(&sandboxObsStartTime, "start-time", "", "inclusive start time (RFC3339)")
	sandboxMetricsCmd.Flags().StringVar(&sandboxObsEndTime, "end-time", "", "inclusive end time (RFC3339)")
	sandboxMetricsCmd.Flags().StringVar(&sandboxObsSince, "since", "", "relative start time, for example 10m or 1h")
	sandboxMetricsCmd.Flags().StringArrayVar(&sandboxObsNames, "name", nil, "canonical metric name (repeatable or comma-separated)")
	sandboxMetricsCmd.Flags().IntVar(&sandboxMetricStep, "step-seconds", 0, "requested bucket width in seconds")
	sandboxMetricsCmd.Flags().StringVar(&sandboxMetricStat, "statistic", "", "aggregation (auto, average, minimum, maximum, last, rate)")
	sandboxMetricsCmd.Flags().IntVar(&sandboxMetricPoints, "max-points", 240, "maximum points per returned series")
}

func buildSandboxCreateRequest() (apispec.ClaimRequest, error) {
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
	if sandboxSnapshotID != "" {
		request.SnapshotID = apispec.NewOptString(sandboxSnapshotID)
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
	if _, ok := request.Template.Get(); !ok {
		return apispec.ClaimRequest{}, fmt.Errorf("--template is required unless provided in config file")
	}
	if err := request.Validate(); err != nil {
		return apispec.ClaimRequest{}, fmt.Errorf("invalid sandbox create request: %w", err)
	}
	return request, nil
}

func addSandboxObservabilityFlags(cmd *cobra.Command) {
	cmd.Flags().IntVar(&sandboxObsLimit, "limit", 100, "maximum number of records to return per query or watch poll")
	cmd.Flags().StringVar(&sandboxObsCursor, "cursor", "", "resume cursor")
	cmd.Flags().StringVar(&sandboxObsStartTime, "start-time", "", "inclusive start time (RFC3339)")
	cmd.Flags().StringVar(&sandboxObsEndTime, "end-time", "", "exclusive end time (RFC3339, not valid with --watch)")
	cmd.Flags().StringVar(&sandboxObsSince, "since", "", "relative start time, for example 10m or 1h")
	cmd.Flags().BoolVar(&sandboxObsWatch, "watch", false, "watch realtime records as they are ingested")
}

func addSandboxEventFilterFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&sandboxObsSource, "source", "", "filter by event source (manager, netd, procd)")
	cmd.Flags().StringVar(&sandboxObsEventType, "event-type", "", "filter by event type (lifecycle, network_audit, runtime_stats)")
	cmd.Flags().StringVar(&sandboxObsOutcome, "outcome", "", "filter by outcome (completed, denied, error, succeeded, failed)")
}

func buildSandboxLogObservabilityOptions(cmd *cobra.Command) (*sandbox0.SandboxObservabilityLogOptions, bool, error) {
	query, watch, err := buildSandboxObservabilityQueryOptions(cmd)
	if err != nil {
		return nil, false, err
	}
	if sandboxLogsFollow {
		if query.EndTime != nil {
			return nil, false, fmt.Errorf("--end-time cannot be used with --follow")
		}
		watch = true
	}
	if cmd.Flags().Changed("tail") {
		if sandboxLogsTailLines < 1 {
			return nil, false, fmt.Errorf("--tail must be greater than 0")
		}
		if cmd.Flags().Changed("limit") {
			return nil, false, fmt.Errorf("--tail and --limit cannot both be set")
		}
		query.Limit = sandboxLogsTailLines
	}
	if cmd.Flags().Changed("since-seconds") {
		if sandboxLogsSinceSecs < 1 {
			return nil, false, fmt.Errorf("--since-seconds must be greater than 0")
		}
		if cmd.Flags().Changed("since") || cmd.Flags().Changed("start-time") {
			return nil, false, fmt.Errorf("--since-seconds cannot be combined with --since or --start-time")
		}
		start := time.Now().Add(-time.Duration(sandboxLogsSinceSecs) * time.Second)
		query.StartTime = &start
	}
	options := &sandbox0.SandboxObservabilityLogOptions{SandboxObservabilityQueryOptions: query}
	if sandboxObsContextID != "" {
		options.ContextID = sandboxObsContextID
	}
	if sandboxObsStream != "" {
		stream, err := parseSandboxObservabilityLogStream(sandboxObsStream)
		if err != nil {
			return nil, false, err
		}
		options.Stream = stream
	}
	return options, watch, nil
}

func buildSandboxEventObservabilityOptions(cmd *cobra.Command) (*sandbox0.SandboxObservabilityEventOptions, bool, error) {
	query, watch, err := buildSandboxObservabilityQueryOptions(cmd)
	if err != nil {
		return nil, false, err
	}
	options := &sandbox0.SandboxObservabilityEventOptions{SandboxObservabilityQueryOptions: query}
	if sandboxObsSource != "" {
		source, err := parseObservabilityEventSource(sandboxObsSource)
		if err != nil {
			return nil, false, err
		}
		options.Source = source
	}
	if sandboxObsEventType != "" {
		eventType, err := parseSandboxObservabilityEventType(sandboxObsEventType)
		if err != nil {
			return nil, false, err
		}
		options.EventType = eventType
	}
	if sandboxObsOutcome != "" {
		outcome, err := parseSandboxObservabilityOutcome(sandboxObsOutcome)
		if err != nil {
			return nil, false, err
		}
		options.Outcome = outcome
	}
	return options, watch, nil
}

func buildSandboxMetricObservabilityOptions(cmd *cobra.Command) (*sandbox0.SandboxObservabilityMetricOptions, error) {
	options := &sandbox0.SandboxObservabilityMetricOptions{}
	if sandboxObsStartTime != "" && sandboxObsSince != "" {
		return nil, fmt.Errorf("--start-time and --since cannot both be set")
	}
	if sandboxObsStartTime != "" {
		start, err := time.Parse(time.RFC3339, sandboxObsStartTime)
		if err != nil {
			return nil, fmt.Errorf("parse --start-time: %w", err)
		}
		options.StartTime = &start
	}
	if sandboxObsSince != "" {
		duration, err := time.ParseDuration(sandboxObsSince)
		if err != nil {
			return nil, fmt.Errorf("parse --since: %w", err)
		}
		start := time.Now().Add(-duration)
		options.StartTime = &start
	}
	if sandboxObsEndTime != "" {
		end, err := time.Parse(time.RFC3339, sandboxObsEndTime)
		if err != nil {
			return nil, fmt.Errorf("parse --end-time: %w", err)
		}
		options.EndTime = &end
	}
	for _, name := range splitObservabilityNames(sandboxObsNames) {
		var metric apispec.SandboxRuntimeMetricName
		if err := metric.UnmarshalText([]byte(name)); err != nil {
			return nil, fmt.Errorf("invalid --name %q", name)
		}
		options.Metrics = append(options.Metrics, metric)
	}
	if sandboxMetricStep < 0 {
		return nil, fmt.Errorf("--step-seconds cannot be negative")
	}
	options.StepSeconds = sandboxMetricStep
	if sandboxMetricStat != "" {
		var statistic apispec.SandboxRuntimeMetricStatistic
		if err := statistic.UnmarshalText([]byte(strings.ToLower(sandboxMetricStat))); err != nil {
			return nil, fmt.Errorf("invalid --statistic %q", sandboxMetricStat)
		}
		options.Statistic = statistic
	}
	if sandboxMetricPoints < 1 || sandboxMetricPoints > 1000 {
		return nil, fmt.Errorf("--max-points must be between 1 and 1000")
	}
	options.MaxPoints = sandboxMetricPoints
	return options, nil
}

func buildSandboxObservabilityQueryOptions(cmd *cobra.Command) (sandbox0.SandboxObservabilityQueryOptions, bool, error) {
	options := sandbox0.SandboxObservabilityQueryOptions{}
	if sandboxObsLimit < 1 {
		return options, false, fmt.Errorf("--limit must be greater than 0")
	}
	options.Limit = sandboxObsLimit
	if sandboxObsCursor != "" {
		options.Cursor = sandboxObsCursor
	}
	if sandboxObsStartTime != "" && sandboxObsSince != "" {
		return options, false, fmt.Errorf("--start-time and --since cannot both be set")
	}
	if sandboxObsStartTime != "" {
		start, err := time.Parse(time.RFC3339, sandboxObsStartTime)
		if err != nil {
			return options, false, fmt.Errorf("parse --start-time: %w", err)
		}
		options.StartTime = &start
	}
	if sandboxObsSince != "" {
		duration, err := time.ParseDuration(sandboxObsSince)
		if err != nil {
			return options, false, fmt.Errorf("parse --since: %w", err)
		}
		start := time.Now().Add(-duration)
		options.StartTime = &start
	}
	if sandboxObsEndTime != "" {
		if sandboxObsWatch {
			return options, false, fmt.Errorf("--end-time cannot be used with --watch")
		}
		end, err := time.Parse(time.RFC3339, sandboxObsEndTime)
		if err != nil {
			return options, false, fmt.Errorf("parse --end-time: %w", err)
		}
		options.EndTime = &end
	}
	return options, sandboxObsWatch, nil
}

func writeObservabilityLogs(w io.Writer, logs []apispec.SandboxObservabilityLogEntry) {
	for _, entry := range logs {
		if strings.HasSuffix(entry.Message, "\n") {
			_, _ = io.WriteString(w, entry.Message)
			continue
		}
		_, _ = fmt.Fprintln(w, entry.Message)
	}
}

func writeObservabilityWatch(stream *sandbox0.SandboxObservabilityStream) error {
	for {
		line, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := writeObservabilityWatchLine(line); err != nil {
			return err
		}
	}
}

func writeObservabilityWatchLine(line *sandbox0.SandboxObservabilityWatchLine) error {
	if cfgFormat == "json" || cfgFormat == "yaml" {
		return json.NewEncoder(os.Stdout).Encode(line)
	}
	switch line.Type {
	case "heartbeat", "watermark":
		return nil
	case "error":
		if line.Error != "" {
			return fmt.Errorf("%s", line.Error)
		}
		return fmt.Errorf("observability watch stream returned an error")
	case "log":
		var entry apispec.SandboxObservabilityLogEntry
		if err := json.Unmarshal(line.Data, &entry); err != nil {
			return err
		}
		writeObservabilityLogs(os.Stdout, []apispec.SandboxObservabilityLogEntry{entry})
	case "event":
		var event apispec.SandboxObservabilityEvent
		if err := json.Unmarshal(line.Data, &event); err != nil {
			return err
		}
		writeObservabilityEvent(os.Stdout, event)
	default:
		return nil
	}
	return nil
}

func writeObservabilityEvent(w io.Writer, event apispec.SandboxObservabilityEvent) {
	outcome := ""
	if value, ok := event.Outcome.Get(); ok {
		outcome = string(value)
	}
	_, _ = fmt.Fprintf(
		w,
		"%s\t%s\t%s\t%s\t%s\n",
		event.OccurredAt.Format(time.RFC3339),
		event.Source,
		event.EventType,
		outcome,
		event.Cursor,
	)
}

func parseSandboxObservabilityLogStream(value string) (apispec.SandboxObservabilityLogStream, error) {
	switch strings.ToLower(value) {
	case "stdout":
		return apispec.SandboxObservabilityLogStreamStdout, nil
	case "stderr":
		return apispec.SandboxObservabilityLogStreamStderr, nil
	case "pty":
		return apispec.SandboxObservabilityLogStreamPty, nil
	default:
		return "", fmt.Errorf("invalid --stream %q: expected stdout, stderr, or pty", value)
	}
}

func parseObservabilityEventSource(value string) (apispec.ObservabilityEventSource, error) {
	switch strings.ToLower(value) {
	case "manager":
		return apispec.ObservabilityEventSourceManager, nil
	case "netd":
		return apispec.ObservabilityEventSourceNetd, nil
	case "procd":
		return apispec.ObservabilityEventSourceProcd, nil
	default:
		return "", fmt.Errorf("invalid --source %q: expected manager, netd, or procd", value)
	}
}

func parseSandboxObservabilityEventType(value string) (apispec.SandboxObservabilityEventType, error) {
	switch strings.ToLower(value) {
	case "lifecycle":
		return apispec.SandboxObservabilityEventTypeLifecycle, nil
	case "network_audit":
		return apispec.SandboxObservabilityEventTypeNetworkAudit, nil
	case "runtime_stats":
		return apispec.SandboxObservabilityEventTypeRuntimeStats, nil
	default:
		return "", fmt.Errorf("invalid --event-type %q: expected lifecycle, network_audit, or runtime_stats", value)
	}
}

func parseSandboxObservabilityOutcome(value string) (apispec.SandboxObservabilityOutcome, error) {
	switch strings.ToLower(value) {
	case "completed":
		return apispec.SandboxObservabilityOutcomeCompleted, nil
	case "denied":
		return apispec.SandboxObservabilityOutcomeDenied, nil
	case "error":
		return apispec.SandboxObservabilityOutcomeError, nil
	case "succeeded":
		return apispec.SandboxObservabilityOutcomeSucceeded, nil
	case "failed":
		return apispec.SandboxObservabilityOutcomeFailed, nil
	default:
		return "", fmt.Errorf("invalid --outcome %q: expected completed, denied, error, succeeded, or failed", value)
	}
}

func splitObservabilityNames(values []string) []string {
	var out []string
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			name := strings.TrimSpace(part)
			if name != "" {
				out = append(out, name)
			}
		}
	}
	return out
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
	if sandboxMemory != "" {
		config.Resources = apispec.NewOptSandboxResourceConfig(apispec.SandboxResourceConfig{
			Memory: apispec.NewOptString(sandboxMemory),
		})
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
	if value, ok := src.Resources.Get(); ok {
		dst.Resources = apispec.NewOptSandboxResourceConfig(value)
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
	for _, key := range []string{"template", "config", "mounts", "snapshot_id"} {
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
	if sandboxUpdateMemory != "" {
		config.Resources = apispec.NewOptSandboxResourceConfig(apispec.SandboxResourceConfig{
			Memory: apispec.NewOptString(sandboxUpdateMemory),
		})
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
