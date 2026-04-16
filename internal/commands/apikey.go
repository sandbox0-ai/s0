package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sandbox0-ai/s0/internal/output"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	apiKeyName      string
	apiKeyScope     string
	apiKeyRoles     []string
	apiKeyExpiresIn string
	apiKeyRaw       bool
)

const (
	apiKeyScopeTeam     = "team"
	apiKeyScopePlatform = "platform"
)

var apiKeyCmd = &cobra.Command{
	Use:   "apikey",
	Short: "Manage API keys",
	Long:  `List, create, deactivate, and delete API keys.`,
}

var apiKeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List API keys",
	Long:  `List API keys in current scope.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		res, err := client.API().APIKeysGet(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing API keys: %v\n", withSelectedTeamAuthHint(err))
			os.Exit(1)
		}

		successRes, ok := res.(*apispec.SuccessAPIKeyListResponse)
		if !ok {
			fmt.Fprintln(os.Stderr, "Error listing API keys: unexpected response type")
			os.Exit(1)
		}

		data, ok := successRes.Data.Get()
		if !ok {
			fmt.Fprintln(os.Stderr, "Error listing API keys: missing response data")
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, data.APIKeys); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var apiKeyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an API key",
	Long:  `Create a new API key for machine access.`,
	Run: func(cmd *cobra.Command, args []string) {
		if strings.TrimSpace(apiKeyName) == "" {
			fmt.Fprintln(os.Stderr, "Error: --name is required")
			os.Exit(1)
		}
		scope, err := normalizeAPIKeyScope(apiKeyScope)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := validateAPIKeyCreateOptions(scope, apiKeyRoles); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		req := &apispec.CreateAPIKeyRequest{
			Name:  apiKeyName,
			Scope: apispec.NewOptString(scope),
		}
		if len(apiKeyRoles) > 0 {
			req.Roles = apiKeyRoles
		}
		if strings.TrimSpace(apiKeyExpiresIn) != "" {
			req.ExpiresIn = apispec.NewOptString(apiKeyExpiresIn)
		}

		res, err := client.API().APIKeysPost(cmd.Context(), req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating API key: %v\n", withSelectedTeamAuthHint(err))
			os.Exit(1)
		}

		successRes, ok := res.(*apispec.SuccessCreateAPIKeyResponse)
		if !ok {
			fmt.Fprintln(os.Stderr, "Error creating API key: unexpected response type")
			os.Exit(1)
		}

		data, ok := successRes.Data.Get()
		if !ok {
			fmt.Fprintln(os.Stderr, "Error creating API key: missing response data")
			os.Exit(1)
		}

		if apiKeyRaw {
			if err := printCreatedAPIKeyRaw(os.Stdout, &data); err != nil {
				fmt.Fprintf(os.Stderr, "Error printing API key: %v\n", err)
				os.Exit(1)
			}
			return
		}

		if err := getFormatterWithOptions(output.Options{ShowSecrets: true}).Format(os.Stdout, data); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
		if output.ParseFormat(cfgFormat) == output.FormatTable {
			fmt.Fprintln(os.Stderr, "Warning: API key is shown only once. Save it now; it cannot be retrieved again.")
		}
	},
}

var apiKeyDeactivateCmd = &cobra.Command{
	Use:   "deactivate <api-key-id>",
	Short: "Deactivate an API key",
	Long:  `Deactivate an API key by ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		res, err := client.API().APIKeysIDDeactivatePost(cmd.Context(), apispec.APIKeysIDDeactivatePostParams{
			ID: args[0],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deactivating API key: %v\n", withSelectedTeamAuthHint(err))
			os.Exit(1)
		}

		successRes, ok := res.(*apispec.SuccessMessageResponse)
		if !ok {
			fmt.Fprintln(os.Stderr, "Error deactivating API key: unexpected response type")
			os.Exit(1)
		}

		if data, ok := successRes.Data.Get(); ok {
			if message, ok := data.Message.Get(); ok && strings.TrimSpace(message) != "" {
				fmt.Println(message)
				return
			}
		}
		fmt.Printf("API key %s deactivated successfully\n", args[0])
	},
}

var apiKeyDeleteCmd = &cobra.Command{
	Use:   "delete <api-key-id>",
	Short: "Delete an API key",
	Long:  `Delete an API key by ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		res, err := client.API().APIKeysIDDelete(cmd.Context(), apispec.APIKeysIDDeleteParams{
			ID: args[0],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting API key: %v\n", withSelectedTeamAuthHint(err))
			os.Exit(1)
		}

		successRes, ok := res.(*apispec.SuccessMessageResponse)
		if !ok {
			fmt.Fprintln(os.Stderr, "Error deleting API key: unexpected response type")
			os.Exit(1)
		}

		if data, ok := successRes.Data.Get(); ok {
			if message, ok := data.Message.Get(); ok && strings.TrimSpace(message) != "" {
				fmt.Println(message)
				return
			}
		}
		fmt.Printf("API key %s deleted successfully\n", args[0])
	},
}

func printCreatedAPIKeyRaw(w io.Writer, data *apispec.CreateAPIKeyResponse) error {
	if data == nil {
		return fmt.Errorf("missing API key data")
	}
	key, ok := data.Key.Get()
	if !ok || strings.TrimSpace(key) == "" {
		return fmt.Errorf("API key not returned by server")
	}
	_, err := fmt.Fprintln(w, key)
	return err
}

func normalizeAPIKeyScope(scope string) (string, error) {
	normalized := strings.TrimSpace(scope)
	if normalized == "" {
		return apiKeyScopeTeam, nil
	}
	switch normalized {
	case apiKeyScopeTeam, apiKeyScopePlatform:
		return normalized, nil
	default:
		return "", fmt.Errorf("--scope must be team or platform")
	}
}

func validateAPIKeyCreateOptions(scope string, roles []string) error {
	switch scope {
	case apiKeyScopePlatform:
		if len(roles) > 0 {
			return fmt.Errorf("platform API keys do not support --role")
		}
	case apiKeyScopeTeam:
		if len(roles) == 0 {
			return fmt.Errorf("at least one --role is required for team API keys")
		}
	default:
		return fmt.Errorf("--scope must be team or platform")
	}
	return nil
}

func init() {
	rootCmd.AddCommand(apiKeyCmd)

	apiKeyCmd.AddCommand(apiKeyListCmd)
	apiKeyCmd.AddCommand(apiKeyCreateCmd)
	apiKeyCmd.AddCommand(apiKeyDeactivateCmd)
	apiKeyCmd.AddCommand(apiKeyDeleteCmd)

	apiKeyCreateCmd.Flags().StringVar(&apiKeyName, "name", "", "API key name (required)")
	apiKeyCreateCmd.Flags().StringVar(&apiKeyScope, "scope", apiKeyScopeTeam, "API key scope (team or platform)")
	apiKeyCreateCmd.Flags().StringArrayVar(&apiKeyRoles, "role", nil, "role to grant (admin, developer, builder, viewer; can be repeated; required for team scope)")
	apiKeyCreateCmd.Flags().StringVar(&apiKeyExpiresIn, "expires-in", "", "key expiry (30d, 90d, 180d, 365d, or never)")
	apiKeyCreateCmd.Flags().BoolVar(&apiKeyRaw, "raw", false, "print only the API key value (for scripts/pipes)")
}
