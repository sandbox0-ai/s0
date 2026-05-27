package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/shlex"
	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"
)

var (
	runName               string
	runSlug               string
	runTemplate           string
	runServiceID          string
	runServiceDisplayName string
	runPort               int32
	runCommandString      string
	runCommand            []string
	runCWD                string
	runWarmProcessName    string
	runHealthPath         string
	runMounts             []string
	runEnv                []string
	runSpecFile           string
	runTargetID           string
	runNoActivate         bool
	runMaxInstances       int32
	runTargetConcurrency  int32
	runIdleTimeoutSeconds int32
	runUpdateName         string
	runUpdateEnable       bool
	runUpdateDisable      bool
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Manage production runs",
	Long:  `Create, deploy, update, and delete production runs.`,
}

var runDeployCmd = &cobra.Command{
	Use:   "deploy [run-id-or-slug]",
	Short: "Deploy a run from snapshots",
	Long:  `Deploy a production run revision from immutable snapshots and a sandbox template.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		spec, err := buildRunDeploySpec(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building run deploy request: %v\n", err)
			os.Exit(1)
		}

		var result *apispec.RunDeployResult
		if len(args) > 0 {
			result, err = client.DeployRunRevision(cmd.Context(), args[0], spec)
		} else {
			result, err = client.DeployRun(cmd.Context(), spec)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deploying run: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var runDeployServiceCmd = &cobra.Command{
	Use:   "deploy-service <sandbox-id> <service-id>",
	Short: "Deploy a run from a sandbox service",
	Long:  `Deploy a production run revision by snapshotting the mounts used by an existing sandbox service.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		scale, err := buildRunScalePolicy(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building scale policy: %v\n", err)
			os.Exit(1)
		}
		activate, hasActivate := buildRunActivateOption()

		var result *apispec.RunDeployResult
		if runTargetID != "" {
			request := apispec.RunDeployRequest{
				Source: apispec.NewOptRunSource(apispec.RunSource{
					Type: apispec.RunSourceTypeSandboxService,
					SandboxService: apispec.NewOptSandboxServiceRunSource(apispec.SandboxServiceRunSource{
						SandboxID: args[0],
						ServiceID: args[1],
					}),
				}),
			}
			applyRunRequestFlags(&request, scale, activate, hasActivate)
			result, err = client.DeployRunRevisionRequest(cmd.Context(), runTargetID, request)
		} else {
			opts := buildRunDeployOptions(scale, activate, hasActivate)
			result, err = client.DeployRunFromSandboxService(cmd.Context(), args[0], args[1], opts...)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deploying run: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var runListCmd = &cobra.Command{
	Use:   "list",
	Short: "List runs",
	Long:  `List production runs for the authenticated team.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		runs, err := client.ListRuns(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing runs: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, runs); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var runGetCmd = &cobra.Command{
	Use:   "get <run-id-or-slug>",
	Short: "Get run details",
	Long:  `Get details of a production run by ID or slug.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		run, err := client.GetRun(cmd.Context(), args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting run: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, run); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var runUpdateCmd = &cobra.Command{
	Use:   "update <run-id-or-slug>",
	Short: "Update a run",
	Long:  `Update mutable production run metadata and scale-to-zero policy.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		request, err := buildRunUpdateRequest(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building run update request: %v\n", err)
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		run, err := client.UpdateRun(cmd.Context(), args[0], request)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating run: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, run); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var runDeleteCmd = &cobra.Command{
	Use:   "delete <run-id-or-slug>",
	Short: "Delete a run",
	Long:  `Delete a production run and disable its public domain.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		if _, err := client.DeleteRun(cmd.Context(), args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting run: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Run %s deleted successfully\n", args[0])
	},
}

var runRevisionCmd = &cobra.Command{
	Use:     "revision",
	Aliases: []string{"revisions"},
	Short:   "Manage run revisions",
	Long:    `List and activate immutable run revisions.`,
}

var runRevisionListCmd = &cobra.Command{
	Use:   "list <run-id-or-slug>",
	Short: "List run revisions",
	Long:  `List immutable revisions for a production run.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		revisions, err := client.ListRunRevisions(cmd.Context(), args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing run revisions: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, revisions); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var runRevisionActivateCmd = &cobra.Command{
	Use:   "activate <run-id-or-slug> <revision-id>",
	Short: "Activate a run revision",
	Long:  `Switch production traffic to a specific immutable run revision.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.ActivateRunRevision(cmd.Context(), args[0], args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error activating run revision: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	addRunIdentityFlags(runDeployCmd)
	addRunSpecFlags(runDeployCmd)
	addRunScaleFlags(runDeployCmd)
	runDeployCmd.Flags().StringVarP(&runSpecFile, "file", "f", "", "path to run deploy YAML/JSON file, or - for stdin")
	runDeployCmd.Flags().BoolVar(&runNoActivate, "no-activate", false, "create the revision without activating it")

	addRunIdentityFlags(runDeployServiceCmd)
	addRunScaleFlags(runDeployServiceCmd)
	runDeployServiceCmd.Flags().StringVar(&runTargetID, "run-id", "", "existing run ID or slug to deploy a new revision")
	runDeployServiceCmd.Flags().BoolVar(&runNoActivate, "no-activate", false, "create the revision without activating it")

	runUpdateCmd.Flags().StringVar(&runUpdateName, "name", "", "new run display name")
	runUpdateCmd.Flags().BoolVar(&runUpdateEnable, "enable", false, "enable the production run domain")
	runUpdateCmd.Flags().BoolVar(&runUpdateDisable, "disable", false, "disable the production run domain")
	addRunScaleFlags(runUpdateCmd)

	runCmd.AddCommand(runDeployCmd)
	runCmd.AddCommand(runDeployServiceCmd)
	runCmd.AddCommand(runListCmd)
	runCmd.AddCommand(runGetCmd)
	runCmd.AddCommand(runUpdateCmd)
	runCmd.AddCommand(runDeleteCmd)
	runCmd.AddCommand(runRevisionCmd)

	runRevisionCmd.AddCommand(runRevisionListCmd)
	runRevisionCmd.AddCommand(runRevisionActivateCmd)
}

func addRunIdentityFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&runName, "name", "", "run display name")
	cmd.Flags().StringVar(&runSlug, "slug", "", "stable run slug")
}

func addRunSpecFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&runTemplate, "template", "t", "", "sandbox template ID")
	cmd.Flags().StringVar(&runServiceID, "service-id", "app", "service ID inside the run revision")
	cmd.Flags().StringVar(&runServiceDisplayName, "service-name", "", "service display name")
	cmd.Flags().Int32Var(&runPort, "port", 0, "service port")
	cmd.Flags().StringVar(&runCommandString, "cmd", "", "service command as a shell-like string")
	cmd.Flags().StringArrayVar(&runCommand, "command", nil, "service command argv item (repeatable)")
	cmd.Flags().StringVar(&runCWD, "cwd", "", "service working directory")
	cmd.Flags().StringVar(&runWarmProcessName, "warm-process", "", "warm process alias or context ID")
	cmd.Flags().StringVar(&runHealthPath, "health-path", "", "HTTP health check path")
	cmd.Flags().StringArrayVar(&runMounts, "mount", nil, "snapshot mount in the form <snapshot-id>:/absolute/path (repeatable)")
	cmd.Flags().StringArrayVar(&runEnv, "env", nil, "service environment variable (KEY=VALUE, repeatable)")
}

func addRunScaleFlags(cmd *cobra.Command) {
	cmd.Flags().Int32Var(&runMaxInstances, "max-instances", 0, "maximum runtime instances")
	cmd.Flags().Int32Var(&runTargetConcurrency, "target-concurrency", 0, "target requests per runtime instance")
	cmd.Flags().Int32Var(&runIdleTimeoutSeconds, "idle-timeout", 0, "seconds before scaling idle runtime back to zero")
}

func buildRunDeploySpec(cmd *cobra.Command) (sandbox0.RunDeploySpec, error) {
	spec := sandbox0.RunDeploySpec{}
	if runSpecFile != "" {
		fileSpec, err := readRunDeploySpecFile(runSpecFile)
		if err != nil {
			return sandbox0.RunDeploySpec{}, err
		}
		spec = fileSpec
	}

	if cmd.Flags().Changed("name") {
		spec.Name = runName
	}
	if cmd.Flags().Changed("slug") {
		spec.Slug = runSlug
	}
	if cmd.Flags().Changed("template") {
		spec.Template = runTemplate
	}
	if cmd.Flags().Changed("service-id") {
		spec.Service.ID = runServiceID
	}
	if cmd.Flags().Changed("service-name") {
		spec.Service.DisplayName = runServiceDisplayName
	}
	if cmd.Flags().Changed("port") {
		spec.Service.Port = runPort
	}
	if cmd.Flags().Changed("cmd") || cmd.Flags().Changed("command") {
		command, err := parseRunCommand(runCommandString, runCommand)
		if err != nil {
			return sandbox0.RunDeploySpec{}, err
		}
		spec.Service.Command = command
	}
	if cmd.Flags().Changed("cwd") {
		spec.Service.CWD = runCWD
	}
	if cmd.Flags().Changed("warm-process") {
		spec.Service.WarmProcessName = runWarmProcessName
	}
	if cmd.Flags().Changed("health-path") {
		spec.Service.HealthPath = runHealthPath
	}
	if cmd.Flags().Changed("mount") {
		mounts, err := parseRunMounts(runMounts)
		if err != nil {
			return sandbox0.RunDeploySpec{}, err
		}
		spec.Mounts = mounts
	}
	if cmd.Flags().Changed("env") {
		env, err := parseExecEnvVars(runEnv)
		if err != nil {
			return sandbox0.RunDeploySpec{}, err
		}
		spec.Service.EnvVars = env
	}
	scale, err := buildRunScalePolicy(cmd)
	if err != nil {
		return sandbox0.RunDeploySpec{}, err
	}
	if scale != nil {
		spec.Scale = scale
	}
	if runNoActivate {
		activate := false
		spec.Activate = &activate
	}
	return spec, nil
}

func buildRunScalePolicy(cmd *cobra.Command) (*apispec.RunScalePolicy, error) {
	scale := apispec.RunScalePolicy{}
	hasScale := false
	if cmd.Flags().Changed("max-instances") {
		if runMaxInstances < 1 {
			return nil, fmt.Errorf("--max-instances must be greater than 0")
		}
		scale.MaxInstances = apispec.NewOptInt32(runMaxInstances)
		hasScale = true
	}
	if cmd.Flags().Changed("target-concurrency") {
		if runTargetConcurrency < 1 {
			return nil, fmt.Errorf("--target-concurrency must be greater than 0")
		}
		scale.TargetConcurrency = apispec.NewOptInt32(runTargetConcurrency)
		hasScale = true
	}
	if cmd.Flags().Changed("idle-timeout") {
		if runIdleTimeoutSeconds < 1 {
			return nil, fmt.Errorf("--idle-timeout must be greater than 0")
		}
		scale.IdleTimeoutSeconds = apispec.NewOptInt32(runIdleTimeoutSeconds)
		hasScale = true
	}
	if !hasScale {
		return nil, nil
	}
	return &scale, nil
}

func buildRunUpdateRequest(cmd *cobra.Command) (apispec.RunUpdateRequest, error) {
	request := apispec.RunUpdateRequest{}
	hasUpdate := false
	if cmd.Flags().Changed("name") {
		request.Name = apispec.NewOptString(runUpdateName)
		hasUpdate = true
	}
	if runUpdateEnable && runUpdateDisable {
		return apispec.RunUpdateRequest{}, fmt.Errorf("--enable and --disable cannot be used together")
	}
	if runUpdateEnable || runUpdateDisable {
		request.Enabled = apispec.NewOptBool(runUpdateEnable)
		hasUpdate = true
	}
	scale, err := buildRunScalePolicy(cmd)
	if err != nil {
		return apispec.RunUpdateRequest{}, err
	}
	if scale != nil {
		request.Scale = apispec.NewOptRunScalePolicy(*scale)
		hasUpdate = true
	}
	if !hasUpdate {
		return apispec.RunUpdateRequest{}, fmt.Errorf("at least one update input is required")
	}
	return request, nil
}

func buildRunActivateOption() (bool, bool) {
	if runNoActivate {
		return false, true
	}
	return false, false
}

func buildRunDeployOptions(scale *apispec.RunScalePolicy, activate bool, hasActivate bool) []sandbox0.RunDeployOption {
	opts := make([]sandbox0.RunDeployOption, 0, 4)
	if runName != "" {
		opts = append(opts, sandbox0.WithRunName(runName))
	}
	if runSlug != "" {
		opts = append(opts, sandbox0.WithRunSlug(runSlug))
	}
	if scale != nil {
		opts = append(opts, sandbox0.WithRunScale(*scale))
	}
	if hasActivate {
		opts = append(opts, sandbox0.WithRunActivate(activate))
	}
	return opts
}

func applyRunRequestFlags(request *apispec.RunDeployRequest, scale *apispec.RunScalePolicy, activate bool, hasActivate bool) {
	if runName != "" {
		request.Name = apispec.NewOptString(runName)
	}
	if runSlug != "" {
		request.Slug = apispec.NewOptString(runSlug)
	}
	if scale != nil {
		request.Scale = apispec.NewOptRunScalePolicy(*scale)
	}
	if hasActivate {
		request.Activate = apispec.NewOptBool(activate)
	}
}

func parseRunMounts(values []string) ([]sandbox0.RunSnapshotMount, error) {
	if len(values) == 0 {
		return nil, nil
	}
	mounts := make([]sandbox0.RunSnapshotMount, 0, len(values))
	for _, raw := range values {
		snapshotID, mountPath, ok := strings.Cut(raw, ":")
		if !ok || snapshotID == "" || mountPath == "" {
			return nil, fmt.Errorf("invalid --mount %q: expected <snapshot-id>:/absolute/path", raw)
		}
		if !strings.HasPrefix(mountPath, "/") {
			return nil, fmt.Errorf("invalid --mount %q: mount path must be absolute", raw)
		}
		mounts = append(mounts, sandbox0.RunSnapshotMount{
			SnapshotID: snapshotID,
			MountPath:  mountPath,
		})
	}
	return mounts, nil
}

func parseRunCommand(commandString string, command []string) ([]string, error) {
	if commandString != "" && len(command) > 0 {
		return nil, fmt.Errorf("--cmd and --command cannot be used together")
	}
	if len(command) > 0 {
		return append([]string(nil), command...), nil
	}
	if commandString == "" {
		return nil, nil
	}
	parts, err := shlex.Split(commandString)
	if err != nil {
		return nil, fmt.Errorf("parse --cmd: %w", err)
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("--cmd cannot be empty")
	}
	return parts, nil
}

func readRunDeploySpecFile(path string) (sandbox0.RunDeploySpec, error) {
	data, err := readConfigFile(path)
	if err != nil {
		return sandbox0.RunDeploySpec{}, err
	}
	var file runDeploySpecFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return sandbox0.RunDeploySpec{}, fmt.Errorf("parse run deploy file: %w", err)
	}
	spec, err := file.toSDKSpec()
	if err != nil {
		return sandbox0.RunDeploySpec{}, err
	}
	return spec, nil
}

type runDeploySpecFile struct {
	Name     string            `yaml:"name"`
	Slug     string            `yaml:"slug"`
	Template string            `yaml:"template"`
	Service  runServiceFile    `yaml:"service"`
	Mounts   []runMountFile    `yaml:"mounts"`
	EnvVars  map[string]string `yaml:"envVars"`
	EnvVars2 map[string]string `yaml:"env_vars"`
	Scale    runScaleFile      `yaml:"scale"`
	Activate *bool             `yaml:"activate"`
}

type runServiceFile struct {
	ID               string            `yaml:"id"`
	DisplayName      string            `yaml:"displayName"`
	DisplayName2     string            `yaml:"display_name"`
	Port             int32             `yaml:"port"`
	Command          []string          `yaml:"command"`
	CWD              string            `yaml:"cwd"`
	EnvVars          map[string]string `yaml:"envVars"`
	EnvVars2         map[string]string `yaml:"env_vars"`
	WarmProcessName  string            `yaml:"warmProcessName"`
	WarmProcessName2 string            `yaml:"warm_process_name"`
	HealthPath       string            `yaml:"healthPath"`
	HealthPath2      string            `yaml:"health_path"`
}

type runMountFile struct {
	SnapshotID  string `yaml:"snapshotID"`
	SnapshotID2 string `yaml:"snapshot_id"`
	MountPath   string `yaml:"mountPath"`
	MountPath2  string `yaml:"mount_path"`
}

type runScaleFile struct {
	MaxInstances        *int32 `yaml:"maxInstances"`
	MaxInstances2       *int32 `yaml:"max_instances"`
	TargetConcurrency   *int32 `yaml:"targetConcurrency"`
	TargetConcurrency2  *int32 `yaml:"target_concurrency"`
	IdleTimeoutSeconds  *int32 `yaml:"idleTimeoutSeconds"`
	IdleTimeoutSeconds2 *int32 `yaml:"idle_timeout_seconds"`
}

func (f runDeploySpecFile) toSDKSpec() (sandbox0.RunDeploySpec, error) {
	mounts := make([]sandbox0.RunSnapshotMount, 0, len(f.Mounts))
	for _, mount := range f.Mounts {
		snapshotID := firstNonEmpty(mount.SnapshotID, mount.SnapshotID2)
		mountPath := firstNonEmpty(mount.MountPath, mount.MountPath2)
		if snapshotID == "" || mountPath == "" {
			return sandbox0.RunDeploySpec{}, fmt.Errorf("run file mount requires snapshotID and mountPath")
		}
		mounts = append(mounts, sandbox0.RunSnapshotMount{
			SnapshotID: snapshotID,
			MountPath:  mountPath,
		})
	}

	spec := sandbox0.RunDeploySpec{
		Name:     f.Name,
		Slug:     f.Slug,
		Template: f.Template,
		Service: sandbox0.RunServiceSpec{
			ID:              f.Service.ID,
			DisplayName:     firstNonEmpty(f.Service.DisplayName, f.Service.DisplayName2),
			Port:            f.Service.Port,
			Command:         append([]string(nil), f.Service.Command...),
			CWD:             f.Service.CWD,
			EnvVars:         firstMap(f.Service.EnvVars, f.Service.EnvVars2),
			WarmProcessName: firstNonEmpty(f.Service.WarmProcessName, f.Service.WarmProcessName2),
			HealthPath:      firstNonEmpty(f.Service.HealthPath, f.Service.HealthPath2),
		},
		Mounts:   mounts,
		EnvVars:  firstMap(f.EnvVars, f.EnvVars2),
		Activate: f.Activate,
	}
	scale, err := f.Scale.toSDKScale()
	if err != nil {
		return sandbox0.RunDeploySpec{}, err
	}
	spec.Scale = scale
	return spec, nil
}

func (f runScaleFile) toSDKScale() (*apispec.RunScalePolicy, error) {
	scale := apispec.RunScalePolicy{}
	hasScale := false
	if value, ok := firstInt32(f.MaxInstances, f.MaxInstances2); ok {
		if value < 1 {
			return nil, fmt.Errorf("scale.maxInstances must be greater than 0")
		}
		scale.MaxInstances = apispec.NewOptInt32(value)
		hasScale = true
	}
	if value, ok := firstInt32(f.TargetConcurrency, f.TargetConcurrency2); ok {
		if value < 1 {
			return nil, fmt.Errorf("scale.targetConcurrency must be greater than 0")
		}
		scale.TargetConcurrency = apispec.NewOptInt32(value)
		hasScale = true
	}
	if value, ok := firstInt32(f.IdleTimeoutSeconds, f.IdleTimeoutSeconds2); ok {
		if value < 1 {
			return nil, fmt.Errorf("scale.idleTimeoutSeconds must be greater than 0")
		}
		scale.IdleTimeoutSeconds = apispec.NewOptInt32(value)
		hasScale = true
	}
	if !hasScale {
		return nil, nil
	}
	return &scale, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstMap(values ...map[string]string) map[string]string {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}

func firstInt32(values ...*int32) (int32, bool) {
	for _, value := range values {
		if value != nil {
			return *value, true
		}
	}
	return 0, false
}
