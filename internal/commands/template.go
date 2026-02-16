package commands

import (
	"fmt"
	"os"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	templateSpecFile string
	templateID       string
)

// templateCmd represents the template command.
var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Manage templates",
	Long:  `List, get, create, update, and delete sandbox templates.`,
}

// templateListCmd lists all templates.
var templateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List templates",
	Long:  `List all available sandbox templates.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw()
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

		client, err := getClientRaw()
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
	Long:  `Create a new sandbox template from a spec file.`,
	Run: func(cmd *cobra.Command, args []string) {
		if templateID == "" {
			fmt.Fprintln(os.Stderr, "Error: --id is required")
			os.Exit(1)
		}
		if templateSpecFile == "" {
			fmt.Fprintln(os.Stderr, "Error: --spec-file is required")
			os.Exit(1)
		}

		// Read spec file
		specData, err := os.ReadFile(templateSpecFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading spec file: %v\n", err)
			os.Exit(1)
		}

		var spec templateSpec
		if err := yaml.Unmarshal(specData, &spec); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing spec file: %v\n", err)
			os.Exit(1)
		}

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		req := apispec.TemplateCreateRequest{
			TemplateID: templateID,
			Spec:       spec.Spec,
		}

		template, err := client.CreateTemplate(cmd.Context(), req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating template: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, template); err != nil {
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

		// Read spec file
		specData, err := os.ReadFile(templateSpecFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading spec file: %v\n", err)
			os.Exit(1)
		}

		var spec templateSpec
		if err := yaml.Unmarshal(specData, &spec); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing spec file: %v\n", err)
			os.Exit(1)
		}

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		req := apispec.TemplateUpdateRequest{
			Spec: spec.Spec,
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

		client, err := getClientRaw()
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
	templateCreateCmd.Flags().StringVarP(&templateSpecFile, "spec-file", "f", "", "path to template spec YAML file (required)")

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
