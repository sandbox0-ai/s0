package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	userName      string
	userAvatarURL string
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

func init() {
	rootCmd.AddCommand(userCmd)

	userCmd.AddCommand(userGetCmd)
	userCmd.AddCommand(userUpdateCmd)

	userUpdateCmd.Flags().StringVar(&userName, "name", "", "new display name")
	userUpdateCmd.Flags().StringVar(&userAvatarURL, "avatar-url", "", "new avatar URL")
}
