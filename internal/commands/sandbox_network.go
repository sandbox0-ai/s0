package commands

import (
	"fmt"
	"os"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	networkSandboxID      string
	networkMode           string
	networkAllowedCidrs   []string
	networkAllowedDomains []string
	networkDeniedCidrs    []string
	networkDeniedDomains  []string
)

// sandboxNetworkCmd represents the sandbox network command group.
var sandboxNetworkCmd = &cobra.Command{
	Use:   "network",
	Short: "Manage network policy",
	Long:  `Get and update network policy for a sandbox.`,
}

// sandboxNetworkGetCmd gets the network policy.
var sandboxNetworkGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get network policy",
	Long:  `Get the network policy configuration for the sandbox.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(networkSandboxID).GetNetworkPolicy(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting network policy: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxNetworkUpdateCmd updates the network policy.
var sandboxNetworkUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update network policy",
	Long:  `Update the network policy configuration for the sandbox.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		// Build the policy from flags
		policy := apispec.TplSandboxNetworkPolicy{}

		if networkMode != "" {
			policy.Mode = apispec.TplSandboxNetworkPolicyMode(networkMode)
		} else {
			fmt.Fprintln(os.Stderr, "Error: --mode is required")
			os.Exit(1)
		}

		// Add egress policy if any egress flags are set
		if len(networkAllowedCidrs) > 0 || len(networkAllowedDomains) > 0 ||
			len(networkDeniedCidrs) > 0 || len(networkDeniedDomains) > 0 {
			policy.Egress = apispec.NewOptNetworkEgressPolicy(apispec.NetworkEgressPolicy{
				AllowedCidrs:   networkAllowedCidrs,
				AllowedDomains: networkAllowedDomains,
				DeniedCidrs:    networkDeniedCidrs,
				DeniedDomains:  networkDeniedDomains,
			})
		}

		result, err := client.Sandbox(networkSandboxID).UpdateNetworkPolicy(cmd.Context(), policy)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating network policy: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	sandboxNetworkCmd.AddCommand(sandboxNetworkGetCmd)
	sandboxNetworkCmd.AddCommand(sandboxNetworkUpdateCmd)

	// Sandbox ID flag (required for all subcommands)
	sandboxNetworkCmd.PersistentFlags().StringVarP(&networkSandboxID, "sandbox-id", "s", "", "sandbox ID (required)")
	_ = sandboxNetworkCmd.MarkPersistentFlagRequired("sandbox-id")

	// Update command flags
	sandboxNetworkUpdateCmd.Flags().StringVar(&networkMode, "mode", "", "network mode (allow-all, block-all) (required)")
	sandboxNetworkUpdateCmd.Flags().StringArrayVar(&networkAllowedCidrs, "allow-cidr", nil, "allowed CIDR (can be repeated)")
	sandboxNetworkUpdateCmd.Flags().StringArrayVar(&networkAllowedDomains, "allow-domain", nil, "allowed domain (can be repeated)")
	sandboxNetworkUpdateCmd.Flags().StringArrayVar(&networkDeniedCidrs, "deny-cidr", nil, "denied CIDR (can be repeated)")
	sandboxNetworkUpdateCmd.Flags().StringArrayVar(&networkDeniedDomains, "deny-domain", nil, "denied domain (can be repeated)")

	sandboxCmd.AddCommand(sandboxNetworkCmd)
}
