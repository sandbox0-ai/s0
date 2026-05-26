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
	functionName                  string
	functionSlug                  string
	functionTemplate              string
	functionServiceID             string
	functionServiceDisplayName    string
	functionPort                  int32
	functionCommandString         string
	functionCommand               []string
	functionCWD                   string
	functionWarmProcessName       string
	functionHealthPath            string
	functionMounts                []string
	functionEnv                   []string
	functionSpecFile              string
	functionTargetID              string
	functionNoActivate            bool
	functionMaxInstances          int32
	functionTargetConcurrency     int32
	functionIdleTimeoutSeconds    int32
	functionStartupTimeoutSeconds int32
	functionUpdateName            string
	functionUpdateEnable          bool
	functionUpdateDisable         bool
)

var functionCmd = &cobra.Command{
	Use:   "function",
	Short: "Manage production functions",
	Long:  `Create, deploy, update, and delete production functions.`,
}

var functionDeployCmd = &cobra.Command{
	Use:   "deploy [function-id-or-slug]",
	Short: "Deploy a function from snapshots",
	Long:  `Deploy a production function revision from immutable snapshots and a sandbox template.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		spec, err := buildFunctionDeploySpec(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building function deploy request: %v\n", err)
			os.Exit(1)
		}

		var result *apispec.FunctionDeployResult
		if len(args) > 0 {
			result, err = client.DeployFunctionRevision(cmd.Context(), args[0], spec)
		} else {
			result, err = client.DeployFunction(cmd.Context(), spec)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deploying function: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionDeployServiceCmd = &cobra.Command{
	Use:   "deploy-service <sandbox-id> <service-id>",
	Short: "Deploy a function from a sandbox service",
	Long:  `Deploy a production function revision by snapshotting the mounts used by an existing sandbox service.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		scale, err := buildFunctionScalePolicy(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building scale policy: %v\n", err)
			os.Exit(1)
		}
		activate, hasActivate := buildFunctionActivateOption()

		var result *apispec.FunctionDeployResult
		if functionTargetID != "" {
			request := apispec.FunctionDeployRequest{
				Source: apispec.NewOptFunctionSource(apispec.FunctionSource{
					Type: apispec.FunctionSourceTypeSandboxService,
					SandboxService: apispec.NewOptSandboxServiceFunctionSource(apispec.SandboxServiceFunctionSource{
						SandboxID: args[0],
						ServiceID: args[1],
					}),
				}),
			}
			applyFunctionRequestFlags(&request, scale, activate, hasActivate)
			result, err = client.DeployFunctionRevisionRequest(cmd.Context(), functionTargetID, request)
		} else {
			opts := buildFunctionDeployOptions(scale, activate, hasActivate)
			result, err = client.DeployFunctionFromSandboxService(cmd.Context(), args[0], args[1], opts...)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deploying function: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List functions",
	Long:  `List production functions for the authenticated team.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		functions, err := client.ListFunctions(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing functions: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, functions); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionGetCmd = &cobra.Command{
	Use:   "get <function-id-or-slug>",
	Short: "Get function details",
	Long:  `Get details of a production function by ID or slug.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		fn, err := client.GetFunction(cmd.Context(), args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting function: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, fn); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionUpdateCmd = &cobra.Command{
	Use:   "update <function-id-or-slug>",
	Short: "Update a function",
	Long:  `Update mutable production function metadata and scale-to-zero policy.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		request, err := buildFunctionUpdateRequest(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building function update request: %v\n", err)
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		fn, err := client.UpdateFunction(cmd.Context(), args[0], request)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating function: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, fn); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionDeleteCmd = &cobra.Command{
	Use:   "delete <function-id-or-slug>",
	Short: "Delete a function",
	Long:  `Delete a production function and disable its public domain.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		if _, err := client.DeleteFunction(cmd.Context(), args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting function: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Function %s deleted successfully\n", args[0])
	},
}

var functionRevisionCmd = &cobra.Command{
	Use:     "revision",
	Aliases: []string{"revisions"},
	Short:   "Manage function revisions",
	Long:    `List and activate immutable function revisions.`,
}

var functionRevisionListCmd = &cobra.Command{
	Use:   "list <function-id-or-slug>",
	Short: "List function revisions",
	Long:  `List immutable revisions for a production function.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		revisions, err := client.ListFunctionRevisions(cmd.Context(), args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing function revisions: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, revisions); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var functionRevisionActivateCmd = &cobra.Command{
	Use:   "activate <function-id-or-slug> <revision-id>",
	Short: "Activate a function revision",
	Long:  `Switch production traffic to a specific immutable function revision.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.ActivateFunctionRevision(cmd.Context(), args[0], args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error activating function revision: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(functionCmd)

	addFunctionIdentityFlags(functionDeployCmd)
	addFunctionSpecFlags(functionDeployCmd)
	addFunctionScaleFlags(functionDeployCmd)
	functionDeployCmd.Flags().StringVarP(&functionSpecFile, "file", "f", "", "path to function deploy YAML/JSON file, or - for stdin")
	functionDeployCmd.Flags().BoolVar(&functionNoActivate, "no-activate", false, "create the revision without activating it")

	addFunctionIdentityFlags(functionDeployServiceCmd)
	addFunctionScaleFlags(functionDeployServiceCmd)
	functionDeployServiceCmd.Flags().StringVar(&functionTargetID, "function-id", "", "existing function ID or slug to deploy a new revision")
	functionDeployServiceCmd.Flags().BoolVar(&functionNoActivate, "no-activate", false, "create the revision without activating it")

	functionUpdateCmd.Flags().StringVar(&functionUpdateName, "name", "", "new function display name")
	functionUpdateCmd.Flags().BoolVar(&functionUpdateEnable, "enable", false, "enable the production function domain")
	functionUpdateCmd.Flags().BoolVar(&functionUpdateDisable, "disable", false, "disable the production function domain")
	addFunctionScaleFlags(functionUpdateCmd)

	functionCmd.AddCommand(functionDeployCmd)
	functionCmd.AddCommand(functionDeployServiceCmd)
	functionCmd.AddCommand(functionListCmd)
	functionCmd.AddCommand(functionGetCmd)
	functionCmd.AddCommand(functionUpdateCmd)
	functionCmd.AddCommand(functionDeleteCmd)
	functionCmd.AddCommand(functionRevisionCmd)

	functionRevisionCmd.AddCommand(functionRevisionListCmd)
	functionRevisionCmd.AddCommand(functionRevisionActivateCmd)
}

func addFunctionIdentityFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&functionName, "name", "", "function display name")
	cmd.Flags().StringVar(&functionSlug, "slug", "", "stable function slug")
}

func addFunctionSpecFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&functionTemplate, "template", "t", "", "sandbox template ID")
	cmd.Flags().StringVar(&functionServiceID, "service-id", "app", "service ID inside the function revision")
	cmd.Flags().StringVar(&functionServiceDisplayName, "service-name", "", "service display name")
	cmd.Flags().Int32Var(&functionPort, "port", 0, "service port")
	cmd.Flags().StringVar(&functionCommandString, "cmd", "", "service command as a shell-like string")
	cmd.Flags().StringArrayVar(&functionCommand, "command", nil, "service command argv item (repeatable)")
	cmd.Flags().StringVar(&functionCWD, "cwd", "", "service working directory")
	cmd.Flags().StringVar(&functionWarmProcessName, "warm-process", "", "warm process alias or context ID")
	cmd.Flags().StringVar(&functionHealthPath, "health-path", "", "HTTP health check path")
	cmd.Flags().StringArrayVar(&functionMounts, "mount", nil, "snapshot mount in the form <snapshot-id>:/absolute/path (repeatable)")
	cmd.Flags().StringArrayVar(&functionEnv, "env", nil, "service environment variable (KEY=VALUE, repeatable)")
}

func addFunctionScaleFlags(cmd *cobra.Command) {
	cmd.Flags().Int32Var(&functionMaxInstances, "max-instances", 0, "maximum runtime instances")
	cmd.Flags().Int32Var(&functionTargetConcurrency, "target-concurrency", 0, "target requests per runtime instance")
	cmd.Flags().Int32Var(&functionIdleTimeoutSeconds, "idle-timeout", 0, "seconds before scaling idle runtime back to zero")
	cmd.Flags().Int32Var(&functionStartupTimeoutSeconds, "startup-timeout", 0, "maximum seconds to wait for cold start health")
}

func buildFunctionDeploySpec(cmd *cobra.Command) (sandbox0.FunctionDeploySpec, error) {
	spec := sandbox0.FunctionDeploySpec{}
	if functionSpecFile != "" {
		fileSpec, err := readFunctionDeploySpecFile(functionSpecFile)
		if err != nil {
			return sandbox0.FunctionDeploySpec{}, err
		}
		spec = fileSpec
	}

	if cmd.Flags().Changed("name") {
		spec.Name = functionName
	}
	if cmd.Flags().Changed("slug") {
		spec.Slug = functionSlug
	}
	if cmd.Flags().Changed("template") {
		spec.Template = functionTemplate
	}
	if cmd.Flags().Changed("service-id") {
		spec.Service.ID = functionServiceID
	}
	if cmd.Flags().Changed("service-name") {
		spec.Service.DisplayName = functionServiceDisplayName
	}
	if cmd.Flags().Changed("port") {
		spec.Service.Port = functionPort
	}
	if cmd.Flags().Changed("cmd") || cmd.Flags().Changed("command") {
		command, err := parseFunctionCommand(functionCommandString, functionCommand)
		if err != nil {
			return sandbox0.FunctionDeploySpec{}, err
		}
		spec.Service.Command = command
	}
	if cmd.Flags().Changed("cwd") {
		spec.Service.CWD = functionCWD
	}
	if cmd.Flags().Changed("warm-process") {
		spec.Service.WarmProcessName = functionWarmProcessName
	}
	if cmd.Flags().Changed("health-path") {
		spec.Service.HealthPath = functionHealthPath
	}
	if cmd.Flags().Changed("mount") {
		mounts, err := parseFunctionMounts(functionMounts)
		if err != nil {
			return sandbox0.FunctionDeploySpec{}, err
		}
		spec.Mounts = mounts
	}
	if cmd.Flags().Changed("env") {
		env, err := parseExecEnvVars(functionEnv)
		if err != nil {
			return sandbox0.FunctionDeploySpec{}, err
		}
		spec.Service.EnvVars = env
	}
	scale, err := buildFunctionScalePolicy(cmd)
	if err != nil {
		return sandbox0.FunctionDeploySpec{}, err
	}
	if scale != nil {
		spec.Scale = scale
	}
	if functionNoActivate {
		activate := false
		spec.Activate = &activate
	}
	return spec, nil
}

func buildFunctionScalePolicy(cmd *cobra.Command) (*apispec.FunctionScalePolicy, error) {
	scale := apispec.FunctionScalePolicy{}
	hasScale := false
	if cmd.Flags().Changed("max-instances") {
		if functionMaxInstances < 1 {
			return nil, fmt.Errorf("--max-instances must be greater than 0")
		}
		scale.MaxInstances = apispec.NewOptInt32(functionMaxInstances)
		hasScale = true
	}
	if cmd.Flags().Changed("target-concurrency") {
		if functionTargetConcurrency < 1 {
			return nil, fmt.Errorf("--target-concurrency must be greater than 0")
		}
		scale.TargetConcurrency = apispec.NewOptInt32(functionTargetConcurrency)
		hasScale = true
	}
	if cmd.Flags().Changed("idle-timeout") {
		if functionIdleTimeoutSeconds < 1 {
			return nil, fmt.Errorf("--idle-timeout must be greater than 0")
		}
		scale.IdleTimeoutSeconds = apispec.NewOptInt32(functionIdleTimeoutSeconds)
		hasScale = true
	}
	if cmd.Flags().Changed("startup-timeout") {
		if functionStartupTimeoutSeconds < 1 {
			return nil, fmt.Errorf("--startup-timeout must be greater than 0")
		}
		scale.StartupTimeoutSeconds = apispec.NewOptInt32(functionStartupTimeoutSeconds)
		hasScale = true
	}
	if !hasScale {
		return nil, nil
	}
	return &scale, nil
}

func buildFunctionUpdateRequest(cmd *cobra.Command) (apispec.FunctionUpdateRequest, error) {
	request := apispec.FunctionUpdateRequest{}
	hasUpdate := false
	if cmd.Flags().Changed("name") {
		request.Name = apispec.NewOptString(functionUpdateName)
		hasUpdate = true
	}
	if functionUpdateEnable && functionUpdateDisable {
		return apispec.FunctionUpdateRequest{}, fmt.Errorf("--enable and --disable cannot be used together")
	}
	if functionUpdateEnable || functionUpdateDisable {
		request.Enabled = apispec.NewOptBool(functionUpdateEnable)
		hasUpdate = true
	}
	scale, err := buildFunctionScalePolicy(cmd)
	if err != nil {
		return apispec.FunctionUpdateRequest{}, err
	}
	if scale != nil {
		request.Scale = apispec.NewOptFunctionScalePolicy(*scale)
		hasUpdate = true
	}
	if !hasUpdate {
		return apispec.FunctionUpdateRequest{}, fmt.Errorf("at least one update input is required")
	}
	return request, nil
}

func buildFunctionActivateOption() (bool, bool) {
	if functionNoActivate {
		return false, true
	}
	return false, false
}

func buildFunctionDeployOptions(scale *apispec.FunctionScalePolicy, activate bool, hasActivate bool) []sandbox0.FunctionDeployOption {
	opts := make([]sandbox0.FunctionDeployOption, 0, 4)
	if functionName != "" {
		opts = append(opts, sandbox0.WithFunctionName(functionName))
	}
	if functionSlug != "" {
		opts = append(opts, sandbox0.WithFunctionSlug(functionSlug))
	}
	if scale != nil {
		opts = append(opts, sandbox0.WithFunctionScale(*scale))
	}
	if hasActivate {
		opts = append(opts, sandbox0.WithFunctionActivate(activate))
	}
	return opts
}

func applyFunctionRequestFlags(request *apispec.FunctionDeployRequest, scale *apispec.FunctionScalePolicy, activate bool, hasActivate bool) {
	if functionName != "" {
		request.Name = apispec.NewOptString(functionName)
	}
	if functionSlug != "" {
		request.Slug = apispec.NewOptString(functionSlug)
	}
	if scale != nil {
		request.Scale = apispec.NewOptFunctionScalePolicy(*scale)
	}
	if hasActivate {
		request.Activate = apispec.NewOptBool(activate)
	}
}

func parseFunctionMounts(values []string) ([]sandbox0.FunctionSnapshotMount, error) {
	if len(values) == 0 {
		return nil, nil
	}
	mounts := make([]sandbox0.FunctionSnapshotMount, 0, len(values))
	for _, raw := range values {
		snapshotID, mountPath, ok := strings.Cut(raw, ":")
		if !ok || snapshotID == "" || mountPath == "" {
			return nil, fmt.Errorf("invalid --mount %q: expected <snapshot-id>:/absolute/path", raw)
		}
		if !strings.HasPrefix(mountPath, "/") {
			return nil, fmt.Errorf("invalid --mount %q: mount path must be absolute", raw)
		}
		mounts = append(mounts, sandbox0.FunctionSnapshotMount{
			SnapshotID: snapshotID,
			MountPath:  mountPath,
		})
	}
	return mounts, nil
}

func parseFunctionCommand(commandString string, command []string) ([]string, error) {
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

func readFunctionDeploySpecFile(path string) (sandbox0.FunctionDeploySpec, error) {
	data, err := readConfigFile(path)
	if err != nil {
		return sandbox0.FunctionDeploySpec{}, err
	}
	var file functionDeploySpecFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return sandbox0.FunctionDeploySpec{}, fmt.Errorf("parse function deploy file: %w", err)
	}
	spec, err := file.toSDKSpec()
	if err != nil {
		return sandbox0.FunctionDeploySpec{}, err
	}
	return spec, nil
}

type functionDeploySpecFile struct {
	Name     string              `yaml:"name"`
	Slug     string              `yaml:"slug"`
	Template string              `yaml:"template"`
	Service  functionServiceFile `yaml:"service"`
	Mounts   []functionMountFile `yaml:"mounts"`
	EnvVars  map[string]string   `yaml:"envVars"`
	EnvVars2 map[string]string   `yaml:"env_vars"`
	Scale    functionScaleFile   `yaml:"scale"`
	Activate *bool               `yaml:"activate"`
}

type functionServiceFile struct {
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

type functionMountFile struct {
	SnapshotID  string `yaml:"snapshotID"`
	SnapshotID2 string `yaml:"snapshot_id"`
	MountPath   string `yaml:"mountPath"`
	MountPath2  string `yaml:"mount_path"`
}

type functionScaleFile struct {
	MaxInstances           *int32 `yaml:"maxInstances"`
	MaxInstances2          *int32 `yaml:"max_instances"`
	TargetConcurrency      *int32 `yaml:"targetConcurrency"`
	TargetConcurrency2     *int32 `yaml:"target_concurrency"`
	IdleTimeoutSeconds     *int32 `yaml:"idleTimeoutSeconds"`
	IdleTimeoutSeconds2    *int32 `yaml:"idle_timeout_seconds"`
	StartupTimeoutSeconds  *int32 `yaml:"startupTimeoutSeconds"`
	StartupTimeoutSeconds2 *int32 `yaml:"startup_timeout_seconds"`
}

func (f functionDeploySpecFile) toSDKSpec() (sandbox0.FunctionDeploySpec, error) {
	mounts := make([]sandbox0.FunctionSnapshotMount, 0, len(f.Mounts))
	for _, mount := range f.Mounts {
		snapshotID := firstNonEmpty(mount.SnapshotID, mount.SnapshotID2)
		mountPath := firstNonEmpty(mount.MountPath, mount.MountPath2)
		if snapshotID == "" || mountPath == "" {
			return sandbox0.FunctionDeploySpec{}, fmt.Errorf("function file mount requires snapshotID and mountPath")
		}
		mounts = append(mounts, sandbox0.FunctionSnapshotMount{
			SnapshotID: snapshotID,
			MountPath:  mountPath,
		})
	}

	spec := sandbox0.FunctionDeploySpec{
		Name:     f.Name,
		Slug:     f.Slug,
		Template: f.Template,
		Service: sandbox0.FunctionServiceSpec{
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
		return sandbox0.FunctionDeploySpec{}, err
	}
	spec.Scale = scale
	return spec, nil
}

func (f functionScaleFile) toSDKScale() (*apispec.FunctionScalePolicy, error) {
	scale := apispec.FunctionScalePolicy{}
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
	if value, ok := firstInt32(f.StartupTimeoutSeconds, f.StartupTimeoutSeconds2); ok {
		if value < 1 {
			return nil, fmt.Errorf("scale.startupTimeoutSeconds must be greater than 0")
		}
		scale.StartupTimeoutSeconds = apispec.NewOptInt32(value)
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
