package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var credentialSourceSpecFile string

var credentialSourceCmd = &cobra.Command{
	Use:   "credential",
	Short: "Manage egress credential sources",
	Long:  `List, get, create, update, and delete team-scoped egress credential sources.`,
}

var credentialSourceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List credential sources",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		sources, err := client.ListCredentialSources(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing credential sources: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, sources); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var credentialSourceGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get credential source details",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		source, err := client.GetCredentialSource(cmd.Context(), args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting credential source: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, source); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var credentialSourceCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a credential source",
	Run: func(cmd *cobra.Command, args []string) {
		if strings.TrimSpace(credentialSourceSpecFile) == "" {
			fmt.Fprintln(os.Stderr, "Error: --file is required")
			os.Exit(1)
		}

		req, err := readCredentialSourceWriteRequest(credentialSourceSpecFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading credential source spec: %v\n", err)
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		source, err := client.CreateCredentialSource(cmd.Context(), *req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating credential source: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, source); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var credentialSourceUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Update a credential source",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if strings.TrimSpace(credentialSourceSpecFile) == "" {
			fmt.Fprintln(os.Stderr, "Error: --file is required")
			os.Exit(1)
		}

		req, err := readCredentialSourceWriteRequest(credentialSourceSpecFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading credential source spec: %v\n", err)
			os.Exit(1)
		}
		req.Name = args[0]

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		source, err := client.UpdateCredentialSource(cmd.Context(), args[0], *req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating credential source: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, source); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var credentialSourceDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a credential source",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		resp, err := client.DeleteCredentialSource(cmd.Context(), args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting credential source: %v\n", err)
			os.Exit(1)
		}

		if data, ok := resp.Data.Get(); ok {
			if message, ok := data.Message.Get(); ok && strings.TrimSpace(message) != "" {
				fmt.Println(message)
				return
			}
		}
		fmt.Printf("Credential source %s deleted successfully\n", args[0])
	},
}

type credentialSourceWriteRequestFile struct {
	Name         string                               `json:"name" yaml:"name"`
	ResolverKind string                               `json:"resolverKind" yaml:"resolverKind"`
	Spec         credentialSourceWriteRequestSpecFile `json:"spec" yaml:"spec"`
}

type credentialSourceWriteRequestSpecFile struct {
	StaticHeaders              *credentialSourceStaticHeadersFile              `json:"staticHeaders" yaml:"staticHeaders"`
	StaticTLSClientCertificate *credentialSourceStaticTLSClientCertificateFile `json:"staticTLSClientCertificate" yaml:"staticTLSClientCertificate"`
	StaticUsernamePassword     *credentialSourceStaticUsernamePasswordFile     `json:"staticUsernamePassword" yaml:"staticUsernamePassword"`
}

type credentialSourceStaticHeadersFile struct {
	Values map[string]string `json:"values" yaml:"values"`
}

type credentialSourceStaticTLSClientCertificateFile struct {
	CertificatePem string `json:"certificatePem" yaml:"certificatePem"`
	PrivateKeyPem  string `json:"privateKeyPem" yaml:"privateKeyPem"`
	CaPem          string `json:"caPem" yaml:"caPem"`
}

type credentialSourceStaticUsernamePasswordFile struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

func readCredentialSourceWriteRequest(path string) (*apispec.CredentialSourceWriteRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseCredentialSourceWriteRequest(data)
}

func parseCredentialSourceWriteRequest(data []byte) (*apispec.CredentialSourceWriteRequest, error) {
	var file credentialSourceWriteRequestFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse spec file: %w", err)
	}
	return credentialSourceWriteRequestFromFile(file)
}

func credentialSourceWriteRequestFromFile(file credentialSourceWriteRequestFile) (*apispec.CredentialSourceWriteRequest, error) {
	name := strings.TrimSpace(file.Name)
	if name == "" {
		return nil, fmt.Errorf("credential source name is required")
	}

	resolverKind, err := parseCredentialSourceResolverKind(file.ResolverKind)
	if err != nil {
		return nil, err
	}

	req := &apispec.CredentialSourceWriteRequest{
		Name:         name,
		ResolverKind: resolverKind,
	}

	switch resolverKind {
	case apispec.CredentialSourceResolverKindStaticHeaders:
		if file.Spec.StaticHeaders == nil {
			return nil, fmt.Errorf("spec.staticHeaders is required for resolverKind %q", resolverKind)
		}
		req.Spec.StaticHeaders = apispec.NewOptStaticHeadersSourceSpec(apispec.StaticHeadersSourceSpec{
			Values: apispec.NewOptStaticHeadersSourceSpecValues(apispec.StaticHeadersSourceSpecValues(file.Spec.StaticHeaders.Values)),
		})
	case apispec.CredentialSourceResolverKindStaticTLSClientCertificate:
		if file.Spec.StaticTLSClientCertificate == nil {
			return nil, fmt.Errorf("spec.staticTLSClientCertificate is required for resolverKind %q", resolverKind)
		}
		spec := apispec.StaticTLSClientCertificateSourceSpec{
			CertificatePem: strings.TrimSpace(file.Spec.StaticTLSClientCertificate.CertificatePem),
			PrivateKeyPem:  strings.TrimSpace(file.Spec.StaticTLSClientCertificate.PrivateKeyPem),
		}
		if spec.CertificatePem == "" {
			return nil, fmt.Errorf("spec.staticTLSClientCertificate.certificatePem is required")
		}
		if spec.PrivateKeyPem == "" {
			return nil, fmt.Errorf("spec.staticTLSClientCertificate.privateKeyPem is required")
		}
		if caPem := strings.TrimSpace(file.Spec.StaticTLSClientCertificate.CaPem); caPem != "" {
			spec.CaPem = apispec.NewOptString(caPem)
		}
		req.Spec.StaticTLSClientCertificate = apispec.NewOptStaticTLSClientCertificateSourceSpec(spec)
	case apispec.CredentialSourceResolverKindStaticUsernamePassword:
		if file.Spec.StaticUsernamePassword == nil {
			return nil, fmt.Errorf("spec.staticUsernamePassword is required for resolverKind %q", resolverKind)
		}
		spec := apispec.StaticUsernamePasswordSourceSpec{
			Username: strings.TrimSpace(file.Spec.StaticUsernamePassword.Username),
			Password: strings.TrimSpace(file.Spec.StaticUsernamePassword.Password),
		}
		if spec.Username == "" {
			return nil, fmt.Errorf("spec.staticUsernamePassword.username is required")
		}
		if spec.Password == "" {
			return nil, fmt.Errorf("spec.staticUsernamePassword.password is required")
		}
		req.Spec.StaticUsernamePassword = apispec.NewOptStaticUsernamePasswordSourceSpec(spec)
	default:
		return nil, fmt.Errorf("unsupported resolverKind %q", resolverKind)
	}

	return req, nil
}

func parseCredentialSourceResolverKind(raw string) (apispec.CredentialSourceResolverKind, error) {
	switch apispec.CredentialSourceResolverKind(strings.TrimSpace(raw)) {
	case apispec.CredentialSourceResolverKindStaticHeaders:
		return apispec.CredentialSourceResolverKindStaticHeaders, nil
	case apispec.CredentialSourceResolverKindStaticTLSClientCertificate:
		return apispec.CredentialSourceResolverKindStaticTLSClientCertificate, nil
	case apispec.CredentialSourceResolverKindStaticUsernamePassword:
		return apispec.CredentialSourceResolverKindStaticUsernamePassword, nil
	default:
		return "", fmt.Errorf(
			"invalid resolverKind %q, must be one of: %s, %s, %s",
			raw,
			apispec.CredentialSourceResolverKindStaticHeaders,
			apispec.CredentialSourceResolverKindStaticTLSClientCertificate,
			apispec.CredentialSourceResolverKindStaticUsernamePassword,
		)
	}
}

func init() {
	rootCmd.AddCommand(credentialSourceCmd)

	credentialSourceCmd.AddCommand(credentialSourceListCmd)
	credentialSourceCmd.AddCommand(credentialSourceGetCmd)
	credentialSourceCmd.AddCommand(credentialSourceCreateCmd)
	credentialSourceCmd.AddCommand(credentialSourceUpdateCmd)
	credentialSourceCmd.AddCommand(credentialSourceDeleteCmd)

	credentialSourceCreateCmd.Flags().StringVarP(&credentialSourceSpecFile, "file", "f", "", "path to credential source YAML/JSON file (required)")
	credentialSourceUpdateCmd.Flags().StringVarP(&credentialSourceSpecFile, "file", "f", "", "path to credential source YAML/JSON file (required)")
}
