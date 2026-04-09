package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	userName      string
	userAvatarURL string
	sshKeyName    string
	sshPublicKey  string
	sshKeyFile    string
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage current user profile",
	Long:  `Get and update the current user profile.`,
}

var userGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get current user profile",
	Long:  `Get profile information for the authenticated user.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		res, err := client.API().UsersMeGet(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting user profile: %v\n", err)
			os.Exit(1)
		}

		successRes, ok := res.(*apispec.SuccessUserResponse)
		if !ok {
			fmt.Fprintln(os.Stderr, "Error getting user profile: unexpected response type")
			os.Exit(1)
		}

		data, ok := successRes.Data.Get()
		if !ok {
			fmt.Fprintln(os.Stderr, "Error getting user profile: missing response data")
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, data); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var userUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update current user profile",
	Long:  `Update profile information for the authenticated user.`,
	Run: func(cmd *cobra.Command, args []string) {
		req := &apispec.UpdateUserRequest{}
		hasChange := false

		if strings.TrimSpace(userName) != "" {
			req.Name = apispec.NewOptString(userName)
			hasChange = true
		}

		if strings.TrimSpace(userAvatarURL) != "" {
			req.AvatarURL = apispec.NewOptString(userAvatarURL)
			hasChange = true
		}

		if !hasChange {
			fmt.Fprintln(os.Stderr, "Error: at least one field must be set for update")
			fmt.Fprintln(os.Stderr, "Use --name and/or --avatar-url")
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		res, err := client.API().UsersMePut(cmd.Context(), req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating user profile: %v\n", err)
			os.Exit(1)
		}

		successRes, ok := res.(*apispec.SuccessUserResponse)
		if !ok {
			fmt.Fprintln(os.Stderr, "Error updating user profile: unexpected response type")
			os.Exit(1)
		}

		data, ok := successRes.Data.Get()
		if !ok {
			fmt.Fprintln(os.Stderr, "Error updating user profile: missing response data")
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, data); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var userSSHKeyCmd = &cobra.Command{
	Use:   "ssh-key",
	Short: "Manage SSH public keys",
	Long:  `List, add, and delete SSH public keys for the current user.`,
}

var userSSHKeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List SSH public keys",
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		keys, err := client.ListUserSSHPublicKeys(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing SSH public keys: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, keys); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var userSSHKeyAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add an SSH public key",
	Run: func(cmd *cobra.Command, args []string) {
		req, err := buildCreateSSHPublicKeyRequest()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		key, err := client.CreateUserSSHPublicKey(cmd.Context(), *req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error adding SSH public key: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, key); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var userSSHKeyDeleteCmd = &cobra.Command{
	Use:   "delete <ssh-key-id>",
	Short: "Delete an SSH public key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		resp, err := client.DeleteUserSSHPublicKey(cmd.Context(), args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting SSH public key: %v\n", err)
			os.Exit(1)
		}

		if data, ok := resp.Data.Get(); ok {
			if message, ok := data.Message.Get(); ok && strings.TrimSpace(message) != "" {
				fmt.Println(message)
				return
			}
		}
		fmt.Printf("SSH public key %s deleted successfully\n", args[0])
	},
}

func init() {
	rootCmd.AddCommand(userCmd)

	userCmd.AddCommand(userGetCmd)
	userCmd.AddCommand(userUpdateCmd)
	userCmd.AddCommand(userSSHKeyCmd)
	userSSHKeyCmd.AddCommand(userSSHKeyListCmd)
	userSSHKeyCmd.AddCommand(userSSHKeyAddCmd)
	userSSHKeyCmd.AddCommand(userSSHKeyDeleteCmd)

	userUpdateCmd.Flags().StringVar(&userName, "name", "", "new display name")
	userUpdateCmd.Flags().StringVar(&userAvatarURL, "avatar-url", "", "new avatar URL")
	userSSHKeyAddCmd.Flags().StringVar(&sshKeyName, "name", "", "SSH key name (defaults to the public key file name)")
	userSSHKeyAddCmd.Flags().StringVar(&sshPublicKey, "public-key", "", "SSH public key content")
	userSSHKeyAddCmd.Flags().StringVar(&sshKeyFile, "public-key-file", "", "path to an SSH public key file")
}

func buildCreateSSHPublicKeyRequest() (*apispec.CreateSSHPublicKeyRequest, error) {
	inlineKey := strings.TrimSpace(sshPublicKey)
	keyFile := strings.TrimSpace(sshKeyFile)
	switch {
	case inlineKey == "" && keyFile == "":
		return nil, fmt.Errorf("either --public-key or --public-key-file is required")
	case inlineKey != "" && keyFile != "":
		return nil, fmt.Errorf("use only one of --public-key or --public-key-file")
	}

	keyName := strings.TrimSpace(sshKeyName)
	if keyFile != "" {
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, err
		}
		inlineKey = strings.TrimSpace(string(data))
		if keyName == "" {
			keyName = strings.TrimSuffix(filepath.Base(keyFile), filepath.Ext(keyFile))
		}
	}

	if inlineKey == "" {
		return nil, fmt.Errorf("SSH public key content is empty")
	}
	if keyName == "" {
		return nil, fmt.Errorf("--name is required when using --public-key")
	}

	return &apispec.CreateSSHPublicKeyRequest{
		Name:      keyName,
		PublicKey: inlineKey,
	}, nil
}
