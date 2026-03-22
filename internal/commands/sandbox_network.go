package commands

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	networkSandboxID       string
	networkMode            string
	networkPolicyFile      string
	networkAllowedCidrs    []string
	networkAllowedDomains  []string
	networkAllowedPorts    []string
	networkDeniedCidrs     []string
	networkDeniedDomains   []string
	networkDeniedPorts     []string
	networkTrafficRules    []string
	networkCredentialRules []string
	networkCredentialBinds []string
)

type networkUpdateOptions struct {
	Mode            string
	PolicyFile      string
	AllowedCidrs    []string
	AllowedDomains  []string
	AllowedPorts    []string
	DeniedCidrs     []string
	DeniedDomains   []string
	DeniedPorts     []string
	TrafficRules    []string
	CredentialRules []string
	CredentialBinds []string
}

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
		client, err := getClientRaw(cmd)
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
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		policy, err := buildNetworkPolicyFromUpdateOptions(networkUpdateOptions{
			Mode:            networkMode,
			PolicyFile:      networkPolicyFile,
			AllowedCidrs:    networkAllowedCidrs,
			AllowedDomains:  networkAllowedDomains,
			AllowedPorts:    networkAllowedPorts,
			DeniedCidrs:     networkDeniedCidrs,
			DeniedDomains:   networkDeniedDomains,
			DeniedPorts:     networkDeniedPorts,
			TrafficRules:    networkTrafficRules,
			CredentialRules: networkCredentialRules,
			CredentialBinds: networkCredentialBinds,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building network policy: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(networkSandboxID).UpdateNetworkPolicy(cmd.Context(), *policy)
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

func buildNetworkPolicyFromUpdateOptions(opts networkUpdateOptions) (*apispec.SandboxNetworkPolicy, error) {
	if strings.TrimSpace(opts.PolicyFile) != "" {
		if hasNetworkNonFileInputs(opts) {
			return nil, fmt.Errorf("--policy-file cannot be combined with other network update flags")
		}
		return readNetworkPolicyUpdateFile(opts.PolicyFile)
	}

	mode := strings.TrimSpace(opts.Mode)
	if mode == "" {
		return nil, fmt.Errorf("--mode is required unless --policy-file is provided")
	}

	if hasLegacyNetworkInputs(opts) && len(opts.TrafficRules) > 0 {
		return nil, fmt.Errorf("--traffic-rule cannot be combined with legacy allow/deny flags")
	}

	allowedPorts, err := parsePortSpecs(opts.AllowedPorts)
	if err != nil {
		return nil, fmt.Errorf("parse --allow-port: %w", err)
	}
	deniedPorts, err := parsePortSpecs(opts.DeniedPorts)
	if err != nil {
		return nil, fmt.Errorf("parse --deny-port: %w", err)
	}
	trafficRules, err := parseNetworkObjects[apispec.TrafficRule](opts.TrafficRules, "--traffic-rule")
	if err != nil {
		return nil, err
	}
	credentialRules, err := parseNetworkObjects[apispec.EgressCredentialRule](opts.CredentialRules, "--credential-rule")
	if err != nil {
		return nil, err
	}
	credentialBinds, err := parseNetworkObjects[apispec.CredentialBinding](opts.CredentialBinds, "--credential-binding")
	if err != nil {
		return nil, err
	}

	policy := &apispec.SandboxNetworkPolicy{
		Mode: apispec.SandboxNetworkPolicyMode(mode),
	}

	if hasAnyNetworkEgressInputs(opts, allowedPorts, deniedPorts, trafficRules, credentialRules) {
		policy.Egress = apispec.NewOptNetworkEgressPolicy(apispec.NetworkEgressPolicy{
			AllowedCidrs:    opts.AllowedCidrs,
			AllowedDomains:  opts.AllowedDomains,
			AllowedPorts:    allowedPorts,
			DeniedCidrs:     opts.DeniedCidrs,
			DeniedDomains:   opts.DeniedDomains,
			DeniedPorts:     deniedPorts,
			TrafficRules:    trafficRules,
			CredentialRules: credentialRules,
		})
	}

	if len(credentialBinds) > 0 {
		policy.CredentialBindings = credentialBinds
	}

	if err := validateNetworkPolicy(policy); err != nil {
		return nil, err
	}
	return policy, nil
}

func hasNetworkNonFileInputs(opts networkUpdateOptions) bool {
	return strings.TrimSpace(opts.Mode) != "" ||
		len(opts.AllowedCidrs) > 0 ||
		len(opts.AllowedDomains) > 0 ||
		len(opts.AllowedPorts) > 0 ||
		len(opts.DeniedCidrs) > 0 ||
		len(opts.DeniedDomains) > 0 ||
		len(opts.DeniedPorts) > 0 ||
		len(opts.TrafficRules) > 0 ||
		len(opts.CredentialRules) > 0 ||
		len(opts.CredentialBinds) > 0
}

func hasLegacyNetworkInputs(opts networkUpdateOptions) bool {
	return len(opts.AllowedCidrs) > 0 ||
		len(opts.AllowedDomains) > 0 ||
		len(opts.AllowedPorts) > 0 ||
		len(opts.DeniedCidrs) > 0 ||
		len(opts.DeniedDomains) > 0 ||
		len(opts.DeniedPorts) > 0
}

func hasAnyNetworkEgressInputs(
	opts networkUpdateOptions,
	allowedPorts []apispec.PortSpec,
	deniedPorts []apispec.PortSpec,
	trafficRules []apispec.TrafficRule,
	credentialRules []apispec.EgressCredentialRule,
) bool {
	return len(opts.AllowedCidrs) > 0 ||
		len(opts.AllowedDomains) > 0 ||
		len(allowedPorts) > 0 ||
		len(opts.DeniedCidrs) > 0 ||
		len(opts.DeniedDomains) > 0 ||
		len(deniedPorts) > 0 ||
		len(trafficRules) > 0 ||
		len(credentialRules) > 0
}

func readNetworkPolicyUpdateFile(path string) (*apispec.SandboxNetworkPolicy, error) {
	var (
		data []byte
		err  error
	)
	if strings.TrimSpace(path) == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}
	return parseNetworkPolicyUpdateFile(data)
}

func parseNetworkPolicyUpdateFile(data []byte) (*apispec.SandboxNetworkPolicy, error) {
	var policy apispec.SandboxNetworkPolicy
	if err := yaml.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("parse network policy file: %w", err)
	}
	if err := validateNetworkPolicy(&policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

func validateNetworkPolicy(policy *apispec.SandboxNetworkPolicy) error {
	if policy == nil {
		return fmt.Errorf("network policy is required")
	}
	if err := policy.Validate(); err != nil {
		return fmt.Errorf("invalid network policy: %w", err)
	}
	if egress, ok := policy.Egress.Get(); ok {
		if len(egress.TrafficRules) > 0 && hasLegacyEgressFields(egress) {
			return fmt.Errorf("egress trafficRules cannot be combined with legacy allowed*/denied* fields")
		}
	}
	return nil
}

//nolint:staticcheck // Legacy allow/deny fields remain intentionally supported for CLI compatibility.
func hasLegacyEgressFields(egress apispec.NetworkEgressPolicy) bool {
	return len(egress.AllowedCidrs) > 0 ||
		len(egress.AllowedDomains) > 0 ||
		len(egress.AllowedPorts) > 0 ||
		len(egress.DeniedCidrs) > 0 ||
		len(egress.DeniedDomains) > 0 ||
		len(egress.DeniedPorts) > 0
}

func parseNetworkObjects[T any](values []string, flagName string) ([]T, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]T, 0, len(values))
	for _, value := range values {
		var parsed T
		if err := yaml.Unmarshal([]byte(value), &parsed); err != nil {
			return nil, fmt.Errorf("parse %s value: %w", flagName, err)
		}
		out = append(out, parsed)
	}
	return out, nil
}

func parsePortSpecs(values []string) ([]apispec.PortSpec, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]apispec.PortSpec, 0, len(values))
	for _, value := range values {
		spec, err := parsePortSpec(value)
		if err != nil {
			return nil, err
		}
		out = append(out, spec)
	}
	return out, nil
}

func parsePortSpec(raw string) (apispec.PortSpec, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return apispec.PortSpec{}, fmt.Errorf("value is required")
	}

	rangePart := value
	protocolPart := ""
	if idx := strings.Index(value, "/"); idx >= 0 {
		rangePart = strings.TrimSpace(value[:idx])
		protocolPart = strings.ToLower(strings.TrimSpace(value[idx+1:]))
	}
	if rangePart == "" {
		return apispec.PortSpec{}, fmt.Errorf("port is required")
	}
	if protocolPart != "" && protocolPart != "tcp" && protocolPart != "udp" {
		return apispec.PortSpec{}, fmt.Errorf("protocol must be tcp or udp")
	}

	startText := rangePart
	endText := ""
	if idx := strings.Index(rangePart, "-"); idx >= 0 {
		startText = strings.TrimSpace(rangePart[:idx])
		endText = strings.TrimSpace(rangePart[idx+1:])
	}

	start, err := parsePortNumber(startText)
	if err != nil {
		return apispec.PortSpec{}, fmt.Errorf("invalid start port %q: %w", startText, err)
	}
	spec := apispec.PortSpec{Port: start}
	if protocolPart != "" {
		spec.Protocol = apispec.NewOptString(protocolPart)
	}

	if endText != "" {
		end, err := parsePortNumber(endText)
		if err != nil {
			return apispec.PortSpec{}, fmt.Errorf("invalid end port %q: %w", endText, err)
		}
		if end < start {
			return apispec.PortSpec{}, fmt.Errorf("end port %d must be greater than or equal to start port %d", end, start)
		}
		spec.EndPort = apispec.NewOptInt32(end)
	}

	return spec, nil
}

func parsePortNumber(raw string) (int32, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
	if err != nil {
		return 0, err
	}
	if value < 1 || value > 65535 {
		return 0, fmt.Errorf("must be between 1 and 65535")
	}
	return int32(value), nil
}

func init() {
	sandboxNetworkCmd.AddCommand(sandboxNetworkGetCmd)
	sandboxNetworkCmd.AddCommand(sandboxNetworkUpdateCmd)

	// Sandbox ID flag (required for all subcommands)
	sandboxNetworkCmd.PersistentFlags().StringVarP(&networkSandboxID, "sandbox-id", "s", "", "sandbox ID (required)")
	_ = sandboxNetworkCmd.MarkPersistentFlagRequired("sandbox-id")

	// Update command flags
	sandboxNetworkUpdateCmd.Flags().StringVar(&networkMode, "mode", "", "network mode (allow-all, block-all)")
	sandboxNetworkUpdateCmd.Flags().StringVarP(&networkPolicyFile, "policy-file", "f", "", "path to network policy YAML/JSON file, or - for stdin")
	sandboxNetworkUpdateCmd.Flags().StringArrayVar(&networkAllowedCidrs, "allow-cidr", nil, "allowed CIDR (can be repeated)")
	sandboxNetworkUpdateCmd.Flags().StringArrayVar(&networkAllowedDomains, "allow-domain", nil, "allowed domain (can be repeated)")
	sandboxNetworkUpdateCmd.Flags().StringArrayVar(&networkAllowedPorts, "allow-port", nil, "allowed port spec: <port>[/tcp|udp] or <start>-<end>[/tcp|udp] (can be repeated)")
	sandboxNetworkUpdateCmd.Flags().StringArrayVar(&networkDeniedCidrs, "deny-cidr", nil, "denied CIDR (can be repeated)")
	sandboxNetworkUpdateCmd.Flags().StringArrayVar(&networkDeniedDomains, "deny-domain", nil, "denied domain (can be repeated)")
	sandboxNetworkUpdateCmd.Flags().StringArrayVar(&networkDeniedPorts, "deny-port", nil, "denied port spec: <port>[/tcp|udp] or <start>-<end>[/tcp|udp] (can be repeated)")
	sandboxNetworkUpdateCmd.Flags().StringArrayVar(&networkTrafficRules, "traffic-rule", nil, "traffic rule object as JSON/YAML (can be repeated)")
	sandboxNetworkUpdateCmd.Flags().StringArrayVar(&networkCredentialRules, "credential-rule", nil, "credential rule object as JSON/YAML (can be repeated)")
	sandboxNetworkUpdateCmd.Flags().StringArrayVar(&networkCredentialBinds, "credential-binding", nil, "credential binding object as JSON/YAML (can be repeated)")

	sandboxCmd.AddCommand(sandboxNetworkCmd)
}
