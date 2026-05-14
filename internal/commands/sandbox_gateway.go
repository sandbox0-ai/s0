package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	gatewaySandboxID  string
	gatewayPolicyFile string
)

// sandboxGatewayCmd represents the sandbox service command group.
var sandboxGatewayCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage sandbox services",
	Long:  `Get, update, and clear canonical services for a sandbox.`,
}

// sandboxGatewayGetCmd gets sandbox services.
var sandboxGatewayGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get sandbox services",
	Long:  `Get canonical services configured for the sandbox.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(gatewaySandboxID).GetServices(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting sandbox services: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxGatewayUpdateCmd updates sandbox services.
var sandboxGatewayUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update sandbox services",
	Long:  `Replace canonical services configured for the sandbox.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		services, err := readSandboxServicesFile(gatewayPolicyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading sandbox services: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(gatewaySandboxID).UpdateServices(cmd.Context(), services.Services)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating sandbox services: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxGatewayClearCmd clears sandbox services.
var sandboxGatewayClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear sandbox services",
	Long:  `Remove all canonical services from the sandbox.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(gatewaySandboxID).ClearServices(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error clearing sandbox services: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

func readSandboxServicesFile(path string) (*apispec.SandboxServicesUpdateRequest, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("--services-file is required")
	}
	data, err := readConfigFile(path)
	if err != nil {
		return nil, err
	}
	return parseSandboxServices(data)
}

func parseSandboxServices(data []byte) (*apispec.SandboxServicesUpdateRequest, error) {
	var services apispec.SandboxServicesUpdateRequest
	if err := yaml.Unmarshal(data, &services); err != nil {
		return nil, fmt.Errorf("parse sandbox services file: %w", err)
	}
	if services.Services == nil {
		services.Services = []apispec.SandboxAppService{}
	}
	if err := services.Validate(); err != nil {
		return nil, fmt.Errorf("invalid sandbox services: %w", err)
	}
	return &services, nil
}

func init() {
	sandboxGatewayCmd.AddCommand(sandboxGatewayGetCmd)
	sandboxGatewayCmd.AddCommand(sandboxGatewayUpdateCmd)
	sandboxGatewayCmd.AddCommand(sandboxGatewayClearCmd)

	sandboxGatewayCmd.PersistentFlags().StringVarP(&gatewaySandboxID, "sandbox-id", "s", "", "sandbox ID (required)")
	_ = sandboxGatewayCmd.MarkPersistentFlagRequired("sandbox-id")

	sandboxGatewayUpdateCmd.Flags().StringVarP(&gatewayPolicyFile, "services-file", "f", "", "path to sandbox services YAML/JSON file, or - for stdin")

	sandboxCmd.AddCommand(sandboxGatewayCmd)
}
