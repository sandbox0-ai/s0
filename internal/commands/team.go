package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	teamName       string
	teamSlug       string
	teamHomeRegion string

	teamMemberTeamID string
	teamMemberEmail  string
	teamMemberRole   string
)

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Manage teams and team members",
	Long:  `List, get, create, update, and delete teams. Manage team members.`,
}

var teamListCmd = &cobra.Command{
	Use:   "list",
	Short: "List teams",
	Long:  `List teams available to the current user.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		res, err := client.API().TeamsGet(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing teams: %v\n", err)
			os.Exit(1)
		}

		successRes, ok := res.(*apispec.SuccessTeamListResponse)
		if !ok {
			fmt.Fprintln(os.Stderr, "Error listing teams: unexpected response type")
			os.Exit(1)
		}

		data, ok := successRes.Data.Get()
		if !ok {
			fmt.Fprintln(os.Stderr, "Error listing teams: missing response data")
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, data.Teams); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var teamGetCmd = &cobra.Command{
	Use:   "get <team-id>",
	Short: "Get team details",
	Long:  `Get details of a team by team ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		res, err := client.API().TeamsIDGet(cmd.Context(), apispec.TeamsIDGetParams{
			ID: args[0],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting team: %v\n", err)
			os.Exit(1)
		}

		successRes, ok := res.(*apispec.SuccessTeamResponse)
		if !ok {
			fmt.Fprintln(os.Stderr, "Error getting team: unexpected response type")
			os.Exit(1)
		}

		data, ok := successRes.Data.Get()
		if !ok {
			fmt.Fprintln(os.Stderr, "Error getting team: missing response data")
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, data); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var teamCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a team",
	Long:  `Create a new team.`,
	Run: func(cmd *cobra.Command, args []string) {
		if strings.TrimSpace(teamName) == "" {
			fmt.Fprintln(os.Stderr, "Error: --name is required")
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		req := buildCreateTeamRequest(teamName, teamSlug, teamHomeRegion)

		res, err := client.API().TeamsPost(cmd.Context(), req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating team: %v\n", err)
			os.Exit(1)
		}

		successRes, ok := res.(*apispec.SuccessTeamResponse)
		if !ok {
			fmt.Fprintln(os.Stderr, "Error creating team: unexpected response type")
			os.Exit(1)
		}

		data, ok := successRes.Data.Get()
		if !ok {
			fmt.Fprintln(os.Stderr, "Error creating team: missing response data")
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, data); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var teamUseCmd = &cobra.Command{
	Use:   "use <team-id>",
	Short: "Set the current team locally",
	Long:  `Set the current team in local CLI config.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		teamID := strings.TrimSpace(args[0])
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		res, err := client.API().TeamsIDGet(cmd.Context(), apispec.TeamsIDGetParams{
			ID: teamID,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error validating team: %v\n", err)
			os.Exit(1)
		}

		successRes, ok := res.(*apispec.SuccessTeamResponse)
		if !ok {
			fmt.Fprintln(os.Stderr, "Error validating team: unexpected response type")
			os.Exit(1)
		}

		data, ok := successRes.Data.Get()
		if !ok {
			fmt.Fprintln(os.Stderr, "Error validating team: missing response data")
			os.Exit(1)
		}

		cfg, err := getConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		profileName := cfg.GetActiveProfile()
		cfg.SetCurrentTeam(profileName, teamID)
		if err := cfg.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Current team for profile %q set to %s (%s)\n", profileName, data.ID, data.Name)
	},
}

var teamUpdateCmd = &cobra.Command{
	Use:   "update <team-id>",
	Short: "Update a team",
	Long:  `Update team name and/or slug.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		req := &apispec.UpdateTeamRequest{}
		hasChange := false

		if strings.TrimSpace(teamName) != "" {
			req.Name = apispec.NewOptString(teamName)
			hasChange = true
		}
		if strings.TrimSpace(teamSlug) != "" {
			req.Slug = apispec.NewOptString(teamSlug)
			hasChange = true
		}

		if !hasChange {
			fmt.Fprintln(os.Stderr, "Error: at least one field must be set for update")
			fmt.Fprintln(os.Stderr, "Use --name and/or --slug")
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		res, err := client.API().TeamsIDPut(cmd.Context(), req, apispec.TeamsIDPutParams{
			ID: args[0],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating team: %v\n", err)
			os.Exit(1)
		}

		successRes, ok := res.(*apispec.SuccessTeamResponse)
		if !ok {
			fmt.Fprintln(os.Stderr, "Error updating team: unexpected response type")
			os.Exit(1)
		}

		data, ok := successRes.Data.Get()
		if !ok {
			fmt.Fprintln(os.Stderr, "Error updating team: missing response data")
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, data); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var teamDeleteCmd = &cobra.Command{
	Use:   "delete <team-id>",
	Short: "Delete a team",
	Long:  `Delete a team by ID.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		res, err := client.API().TeamsIDDelete(cmd.Context(), apispec.TeamsIDDeleteParams{
			ID: args[0],
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting team: %v\n", err)
			os.Exit(1)
		}

		successRes, ok := res.(*apispec.SuccessMessageResponse)
		if !ok {
			fmt.Fprintln(os.Stderr, "Error deleting team: unexpected response type")
			os.Exit(1)
		}

		if data, ok := successRes.Data.Get(); ok {
			if message, ok := data.Message.Get(); ok && strings.TrimSpace(message) != "" {
				fmt.Println(message)
				return
			}
		}
		fmt.Printf("Team %s deleted successfully\n", args[0])
	},
}

var teamMemberCmd = &cobra.Command{
	Use:   "member",
	Short: "Manage team members",
	Long:  `List, add, update, and remove members in a team.`,
}

var teamMemberListCmd = &cobra.Command{
	Use:   "list",
	Short: "List team members",
	Long:  `List all members of a team.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		res, err := client.API().TeamsIDMembersGet(cmd.Context(), apispec.TeamsIDMembersGetParams{
			ID: teamMemberTeamID,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing team members: %v\n", err)
			os.Exit(1)
		}

		successRes, ok := res.(*apispec.SuccessTeamMemberListResponse)
		if !ok {
			fmt.Fprintln(os.Stderr, "Error listing team members: unexpected response type")
			os.Exit(1)
		}

		data, ok := successRes.Data.Get()
		if !ok {
			fmt.Fprintln(os.Stderr, "Error listing team members: missing response data")
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, data.Members); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var teamMemberAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a team member",
	Long:  `Invite/add a team member by email and role.`,
	Run: func(cmd *cobra.Command, args []string) {
		if strings.TrimSpace(teamMemberEmail) == "" {
			fmt.Fprintln(os.Stderr, "Error: --email is required")
			os.Exit(1)
		}

		role, err := parseAddTeamMemberRole(teamMemberRole)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		req := &apispec.AddTeamMemberRequest{
			Email: teamMemberEmail,
			Role:  role,
		}

		res, err := client.API().TeamsIDMembersPost(
			cmd.Context(),
			req,
			apispec.TeamsIDMembersPostParams{ID: teamMemberTeamID},
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error adding team member: %v\n", err)
			os.Exit(1)
		}

		successRes, ok := res.(*apispec.SuccessTeamMemberResponse)
		if !ok {
			fmt.Fprintln(os.Stderr, "Error adding team member: unexpected response type")
			os.Exit(1)
		}

		data, ok := successRes.Data.Get()
		if !ok {
			fmt.Fprintln(os.Stderr, "Error adding team member: missing response data")
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, data); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var teamMemberUpdateCmd = &cobra.Command{
	Use:   "update <user-id>",
	Short: "Update team member role",
	Long:  `Update role of a team member.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		role, err := parseUpdateTeamMemberRole(teamMemberRole)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		req := &apispec.UpdateTeamMemberRequest{
			Role: role,
		}

		res, err := client.API().TeamsIDMembersUserIdPut(
			cmd.Context(),
			req,
			apispec.TeamsIDMembersUserIdPutParams{
				ID:     teamMemberTeamID,
				UserId: args[0],
			},
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating team member: %v\n", err)
			os.Exit(1)
		}

		successRes, ok := res.(*apispec.SuccessMessageResponse)
		if !ok {
			fmt.Fprintln(os.Stderr, "Error updating team member: unexpected response type")
			os.Exit(1)
		}

		if data, ok := successRes.Data.Get(); ok {
			if message, ok := data.Message.Get(); ok && strings.TrimSpace(message) != "" {
				fmt.Println(message)
				return
			}
		}
		fmt.Printf("Member %s updated in team %s\n", args[0], teamMemberTeamID)
	},
}

var teamMemberRemoveCmd = &cobra.Command{
	Use:   "remove <user-id>",
	Short: "Remove a team member",
	Long:  `Remove a member from a team.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		res, err := client.API().TeamsIDMembersUserIdDelete(
			cmd.Context(),
			apispec.TeamsIDMembersUserIdDeleteParams{
				ID:     teamMemberTeamID,
				UserId: args[0],
			},
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error removing team member: %v\n", err)
			os.Exit(1)
		}

		successRes, ok := res.(*apispec.SuccessMessageResponse)
		if !ok {
			fmt.Fprintln(os.Stderr, "Error removing team member: unexpected response type")
			os.Exit(1)
		}

		if data, ok := successRes.Data.Get(); ok {
			if message, ok := data.Message.Get(); ok && strings.TrimSpace(message) != "" {
				fmt.Println(message)
				return
			}
		}
		fmt.Printf("Member %s removed from team %s\n", args[0], teamMemberTeamID)
	},
}

func parseAddTeamMemberRole(s string) (apispec.AddTeamMemberRequestRole, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", string(apispec.AddTeamMemberRequestRoleDeveloper):
		return apispec.AddTeamMemberRequestRoleDeveloper, nil
	case string(apispec.AddTeamMemberRequestRoleAdmin):
		return apispec.AddTeamMemberRequestRoleAdmin, nil
	case string(apispec.AddTeamMemberRequestRoleViewer):
		return apispec.AddTeamMemberRequestRoleViewer, nil
	default:
		return "", fmt.Errorf("invalid --role %q, must be one of: admin, developer, viewer", s)
	}
}

func parseUpdateTeamMemberRole(s string) (apispec.UpdateTeamMemberRequestRole, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", string(apispec.UpdateTeamMemberRequestRoleDeveloper):
		return apispec.UpdateTeamMemberRequestRoleDeveloper, nil
	case string(apispec.UpdateTeamMemberRequestRoleAdmin):
		return apispec.UpdateTeamMemberRequestRoleAdmin, nil
	case string(apispec.UpdateTeamMemberRequestRoleViewer):
		return apispec.UpdateTeamMemberRequestRoleViewer, nil
	default:
		return "", fmt.Errorf("invalid --role %q, must be one of: admin, developer, viewer", s)
	}
}

func buildCreateTeamRequest(name, slug, homeRegion string) *apispec.CreateTeamRequest {
	req := &apispec.CreateTeamRequest{
		Name: name,
	}
	if trimmedSlug := strings.TrimSpace(slug); trimmedSlug != "" {
		req.Slug = apispec.NewOptString(trimmedSlug)
	}
	if trimmedHomeRegion := strings.TrimSpace(homeRegion); trimmedHomeRegion != "" {
		req.HomeRegionID = apispec.NewOptNilString(trimmedHomeRegion)
	}
	return req
}

func init() {
	rootCmd.AddCommand(teamCmd)

	teamCmd.AddCommand(teamListCmd)
	teamCmd.AddCommand(teamGetCmd)
	teamCmd.AddCommand(teamCreateCmd)
	teamCmd.AddCommand(teamUseCmd)
	teamCmd.AddCommand(teamUpdateCmd)
	teamCmd.AddCommand(teamDeleteCmd)
	teamCmd.AddCommand(teamMemberCmd)

	teamMemberCmd.AddCommand(teamMemberListCmd)
	teamMemberCmd.AddCommand(teamMemberAddCmd)
	teamMemberCmd.AddCommand(teamMemberUpdateCmd)
	teamMemberCmd.AddCommand(teamMemberRemoveCmd)
	teamMemberCmd.PersistentFlags().StringVarP(&teamMemberTeamID, "team-id", "t", "", "team ID (required)")
	_ = teamMemberCmd.MarkPersistentFlagRequired("team-id")

	teamCreateCmd.Flags().StringVar(&teamName, "name", "", "team name (required)")
	teamCreateCmd.Flags().StringVar(&teamSlug, "slug", "", "team slug")
	teamCreateCmd.Flags().StringVar(&teamHomeRegion, "home-region", "", "team home region ID")

	teamUpdateCmd.Flags().StringVar(&teamName, "name", "", "new team name")
	teamUpdateCmd.Flags().StringVar(&teamSlug, "slug", "", "new team slug")

	teamMemberAddCmd.Flags().StringVar(&teamMemberEmail, "email", "", "member email (required)")
	teamMemberAddCmd.Flags().StringVar(&teamMemberRole, "role", "developer", "member role (admin, developer, viewer)")
	teamMemberUpdateCmd.Flags().StringVar(&teamMemberRole, "role", "developer", "member role (admin, developer, viewer)")
}
