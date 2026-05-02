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

// sandboxGatewayCmd represents the sandbox gateway command group.
var sandboxGatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Manage public gateway policy",
	Long:  `Get, update, and clear request-level public gateway policy for a sandbox.`,
}

// sandboxGatewayGetCmd gets the public gateway policy.
var sandboxGatewayGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get public gateway policy",
	Long:  `Get the request-level public gateway policy for the sandbox.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(gatewaySandboxID).GetPublicGateway(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting public gateway policy: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxGatewayUpdateCmd updates the public gateway policy.
var sandboxGatewayUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update public gateway policy",
	Long:  `Replace the request-level public gateway policy for the sandbox.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		policy, err := readPublicGatewayPolicyFile(gatewayPolicyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading public gateway policy: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(gatewaySandboxID).UpdatePublicGateway(cmd.Context(), *policy)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating public gateway policy: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxGatewayClearCmd disables request-level public gateway enforcement.
var sandboxGatewayClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear public gateway policy",
	Long:  `Disable request-level public gateway enforcement for the sandbox.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(gatewaySandboxID).ClearPublicGateway(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error clearing public gateway policy: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

func readPublicGatewayPolicyFile(path string) (*apispec.PublicGatewayConfig, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("--policy-file is required")
	}
	data, err := readConfigFile(path)
	if err != nil {
		return nil, err
	}
	return parsePublicGatewayPolicy(data)
}

func parsePublicGatewayPolicy(data []byte) (*apispec.PublicGatewayConfig, error) {
	var policy apispec.PublicGatewayConfig
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("parse public gateway policy file: %w", err)
	}
	if err := policy.Validate(); err != nil {
		return nil, fmt.Errorf("invalid public gateway policy: %w", err)
	}
	return &policy, nil
}

func init() {
	sandboxGatewayCmd.AddCommand(sandboxGatewayGetCmd)
	sandboxGatewayCmd.AddCommand(sandboxGatewayUpdateCmd)
	sandboxGatewayCmd.AddCommand(sandboxGatewayClearCmd)

	sandboxGatewayCmd.PersistentFlags().StringVarP(&gatewaySandboxID, "sandbox-id", "s", "", "sandbox ID (required)")
	_ = sandboxGatewayCmd.MarkPersistentFlagRequired("sandbox-id")

	sandboxGatewayUpdateCmd.Flags().StringVarP(&gatewayPolicyFile, "policy-file", "f", "", "path to public gateway policy YAML/JSON file, or - for stdin")
	_ = sandboxGatewayUpdateCmd.MarkFlagRequired("policy-file")

	sandboxCmd.AddCommand(sandboxGatewayCmd)
}
