package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ghodss/yaml"
	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	templateSpecFile       string
	templateOverridesFile  string
	templateID             string
	templateFromSandbox    string
	templateIdempotencyKey string
	templateWait           bool
	templateWaitTimeout    time.Duration
	templatePollInterval   time.Duration
)

// templateCmd represents the template command.
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage templates",
	Long:  `List, get, create, update, and delete sandbox templates.`,
}

func loadTemplateSpecFile(path string) (templateSpec, error) {
	var spec templateSpec

	specData, err := os.ReadFile(path)
	if err != nil {
		return spec, err
	}

	specJSON, err := yaml.YAMLToJSON(specData)
	if err != nil {
		return spec, err
	}
	if err := rejectTemplateCPU(specJSON); err != nil {
		return spec, err
	}
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return spec, err
	}

	return spec, nil
}

// rejectTemplateCPU prevents removed CPU settings from being silently ignored by generated decoders.
func rejectTemplateCPU(specJSON []byte) error {
	var document map[string]any
	if err := json.Unmarshal(specJSON, &document); err != nil {
		return err
	}

	spec, _ := document["spec"].(map[string]any)
	mainContainer, _ := spec["mainContainer"].(map[string]any)
	resources, _ := mainContainer["resources"].(map[string]any)
	if _, ok := resources["cpu"]; ok {
		return fmt.Errorf("spec.mainContainer.resources.cpu is not supported; set memory only")
	}
	return nil
}

func buildTemplateCreateRequest(templateID, specFile string) (apispec.TemplateCreateRequest, error) {
	spec, err := loadTemplateSpecFile(specFile)
	if err != nil {
		return apispec.TemplateCreateRequest{}, err
	}

	return apispec.TemplateCreateRequest{
		TemplateID: templateID,
		Spec:       spec.Spec,
	}, nil
}

func buildTemplateFromSandboxCreateRequest(templateID, sandboxID, overridesFile string) (apispec.TemplateFromSandboxCreateRequest, error) {
	request := sandbox0.NewTemplateFromSandboxCreateRequest(templateID, sandboxID, nil)
	if overridesFile == "" {
		return request, nil
	}

	data, err := os.ReadFile(overridesFile)
	if err != nil {
		return apispec.TemplateFromSandboxCreateRequest{}, err
	}
	specJSON, err := yaml.YAMLToJSON(data)
	if err != nil {
		return apispec.TemplateFromSandboxCreateRequest{}, err
	}
	if err := rejectUnsupportedTemplateFromSandboxOverrides(specJSON); err != nil {
		return apispec.TemplateFromSandboxCreateRequest{}, err
	}
	var overrides apispec.TemplateFromSandboxSpecOverrides
	if err := json.Unmarshal(specJSON, &overrides); err != nil {
		return apispec.TemplateFromSandboxCreateRequest{}, err
	}
	return sandbox0.NewTemplateFromSandboxCreateRequest(templateID, sandboxID, &overrides), nil
}

// rejectUnsupportedTemplateFromSandboxOverrides prevents fields outside the public override contract from being silently ignored.
func rejectUnsupportedTemplateFromSandboxOverrides(specJSON []byte) error {
	var overrides map[string]json.RawMessage
	if err := json.Unmarshal(specJSON, &overrides); err != nil || overrides == nil {
		return fmt.Errorf("overrides file must contain an object")
	}
	for field := range overrides {
		switch field {
		case "description", "displayName", "tags", "pool":
		default:
			return fmt.Errorf("%s is not supported in an overrides file", field)
		}
	}
	if rawPool, ok := overrides["pool"]; ok {
		var pool map[string]json.RawMessage
		if err := json.Unmarshal(rawPool, &pool); err != nil || pool == nil {
			return fmt.Errorf("pool must be an object")
		}
		for field := range pool {
			if field != "minIdle" && field != "maxIdle" {
				return fmt.Errorf("pool.%s is not supported", field)
			}
		}
	}
	return nil
}

type templateCreateModeOptions struct {
	templateID          string
	specFile            string
	overridesFile       string
	fromSandbox         string
	idempotencyKey      string
	wait                bool
	waitTimeoutChanged  bool
	pollIntervalChanged bool
	waitTimeout         time.Duration
	pollInterval        time.Duration
}

func validateTemplateCreateMode(opts templateCreateModeOptions) error {
	if opts.templateID == "" {
		return fmt.Errorf("--id is required")
	}
	if opts.fromSandbox == "" {
		if opts.specFile == "" {
			return fmt.Errorf("--spec-file is required when --from-sandbox is not set")
		}
		if opts.overridesFile != "" {
			return fmt.Errorf("--overrides-file requires --from-sandbox")
		}
		if opts.idempotencyKey != "" || opts.wait || opts.waitTimeoutChanged || opts.pollIntervalChanged {
			return fmt.Errorf("--idempotency-key and wait flags require --from-sandbox")
		}
		return nil
	}
	if opts.specFile != "" {
		return fmt.Errorf("--spec-file cannot be used with --from-sandbox; use --overrides-file")
	}
	if opts.waitTimeoutChanged && !opts.wait {
		return fmt.Errorf("--wait-timeout requires --wait")
	}
	if opts.pollIntervalChanged && !opts.wait {
		return fmt.Errorf("--poll-interval requires --wait")
	}
	if opts.wait && opts.waitTimeout <= 0 {
		return fmt.Errorf("--wait-timeout must be greater than zero")
	}
	if opts.wait && opts.pollInterval <= 0 {
		return fmt.Errorf("--poll-interval must be greater than zero")
	}
	return nil
}

func buildTemplateUpdateRequest(specFile string) (apispec.TemplateUpdateRequest, error) {
	spec, err := loadTemplateSpecFile(specFile)
	if err != nil {
		return apispec.TemplateUpdateRequest{}, err
	}

	return apispec.TemplateUpdateRequest{
		Spec: spec.Spec,
	}, nil
}

// templateListCmd lists all templates.
var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List templates",
	Long:  `List all available sandbox templates.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		templates, err := client.ListTemplate(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing templates: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, templates); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// templateGetCmd gets a template by ID.
var templateGetCmd = &cobra.Command{
	Use:   "get <template-id>",
	Short: "Get template details",
	Long:  `Get details of a template by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		templateID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		template, err := client.GetTemplate(cmd.Context(), templateID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting template: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, template); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// templateCreateCmd creates a new template.
var templateCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a template",
	Long:  `Create a template from an image spec file or from an existing sandbox root filesystem.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := validateTemplateCreateMode(templateCreateModeOptions{
			templateID:          templateID,
			specFile:            templateSpecFile,
			overridesFile:       templateOverridesFile,
			fromSandbox:         templateFromSandbox,
			idempotencyKey:      templateIdempotencyKey,
			wait:                templateWait,
			waitTimeoutChanged:  cmd.Flags().Changed("wait-timeout"),
			pollIntervalChanged: cmd.Flags().Changed("poll-interval"),
			waitTimeout:         templateWaitTimeout,
			pollInterval:        templatePollInterval,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		var created *apispec.Template
		if templateFromSandbox == "" {
			req, buildErr := buildTemplateCreateRequest(templateID, templateSpecFile)
			if buildErr != nil {
				fmt.Fprintf(os.Stderr, "Error parsing spec file: %v\n", buildErr)
				os.Exit(1)
			}
			created, err = client.CreateTemplate(cmd.Context(), req)
		} else {
			req, buildErr := buildTemplateFromSandboxCreateRequest(templateID, templateFromSandbox, templateOverridesFile)
			if buildErr != nil {
				fmt.Fprintf(os.Stderr, "Error parsing overrides file: %v\n", buildErr)
				os.Exit(1)
			}
			created, err = client.CreateTemplateFromSandbox(
				cmd.Context(),
				req,
				&sandbox0.CreateTemplateFromSandboxOptions{IdempotencyKey: templateIdempotencyKey},
			)
			if err == nil && templateWait {
				waitContext, cancel := context.WithTimeout(cmd.Context(), templateWaitTimeout)
				defer cancel()
				created, err = client.WaitTemplateReady(
					waitContext,
					templateID,
					&sandbox0.WaitTemplateReadyOptions{PollInterval: templatePollInterval},
				)
			}
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating template: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, created); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// templateUpdateCmd updates a template.
var templateUpdateCmd = &cobra.Command{
	Use:   "update <template-id>",
	Short: "Update a template",
	Long:  `Update an existing sandbox template from a spec file.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		templateID := args[0]

		if templateSpecFile == "" {
			fmt.Fprintln(os.Stderr, "Error: --spec-file is required")
			os.Exit(1)
		}

		req, err := buildTemplateUpdateRequest(templateSpecFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing spec file: %v\n", err)
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		template, err := client.UpdateTemplate(cmd.Context(), templateID, req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating template: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, template); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// templateDeleteCmd deletes a template.
var templateDeleteCmd = &cobra.Command{
	Use:   "delete <template-id>",
	Short: "Delete a template",
	Long:  `Delete a sandbox template by its ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		templateID := args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		_, err = client.DeleteTemplate(cmd.Context(), templateID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting template: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Template %s deleted successfully\n", templateID)
	},
}

// templateSpec represents the YAML structure for template spec files.
type templateSpec struct {
	Spec apispec.SandboxTemplateSpec `yaml:"spec"`
}

func init() {
	rootCmd.AddCommand(templateCmd)

	// Create command flags
	templateCreateCmd.Flags().StringVar(&templateID, "id", "", "template ID (required)")
	templateCreateCmd.Flags().StringVarP(&templateSpecFile, "spec-file", "f", "", "template spec file for image-based creation")
	templateCreateCmd.Flags().StringVar(&templateFromSandbox, "from-sandbox", "", "source sandbox ID whose current root filesystem should become the template image")
	templateCreateCmd.Flags().StringVar(&templateOverridesFile, "overrides-file", "", "optional override object YAML file for from-sandbox creation")
	templateCreateCmd.Flags().StringVar(&templateIdempotencyKey, "idempotency-key", "", "safe retry key for a from-sandbox create request")
	templateCreateCmd.Flags().BoolVar(&templateWait, "wait", false, "wait until from-sandbox template creation is ready or failed")
	templateCreateCmd.Flags().DurationVar(&templateWaitTimeout, "wait-timeout", 10*time.Minute, "maximum time to wait for template readiness")
	templateCreateCmd.Flags().DurationVar(&templatePollInterval, "poll-interval", time.Second, "template readiness polling interval")

	// Update command flags
	templateUpdateCmd.Flags().StringVarP(&templateSpecFile, "spec-file", "f", "", "path to template spec YAML file (required)")

	templateCmd.AddCommand(templateListCmd)
	templateCmd.AddCommand(templateGetCmd)
	templateCmd.AddCommand(templateCreateCmd)
	templateCmd.AddCommand(templateUpdateCmd)
	templateCmd.AddCommand(templateDeleteCmd)

	// Image commands as subcommands of template
	templateCmd.AddCommand(imageCmd)
}
