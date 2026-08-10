package commands

import (
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/sandbox0-ai/s0/internal/config"
	"github.com/sandbox0-ai/s0/internal/output"
	"github.com/spf13/cobra"
)

const cloudInvitationPathPrefix = "/cloud/v1"

type teamInvitationCreateOptions struct {
	teamID string
	email  string
	role   string
}

type teamInvitationCreateRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// cloudTeamInvitation is the Cloud-only invitation response contract. It is
// intentionally local to the CLI because the open-source API does not expose
// invitation endpoints.
type cloudTeamInvitation struct {
	ID                    string     `json:"id"`
	TeamID                string     `json:"team_id"`
	TeamName              string     `json:"team_name,omitempty"`
	InviteeEmail          string     `json:"invitee_email"`
	Role                  string     `json:"role"`
	Status                string     `json:"status"`
	ExpiresAt             time.Time  `json:"expires_at"`
	LastSendRequestedAt   time.Time  `json:"last_send_requested_at"`
	LastSentAt            *time.Time `json:"last_sent_at,omitempty"`
	SendCount             int        `json:"send_count"`
	DeliveryState         string     `json:"delivery_state"`
	DeliveryLastError     string     `json:"delivery_last_error,omitempty"`
	DeliveryNextAttemptAt *time.Time `json:"delivery_next_attempt_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func newTeamInvitationCommand() *cobra.Command {
	command := &cobra.Command{
		Use:     "invitation",
		Aliases: []string{"invite"},
		Short:   "Manage Sandbox0 Cloud team invitations",
		Long:    "Create pending team invitations through the Sandbox0 Cloud Global Gateway.",
	}
	command.AddCommand(newTeamInvitationCreateCommand())
	return command
}

func newTeamInvitationCreateCommand() *cobra.Command {
	options := &teamInvitationCreateOptions{}
	command := &cobra.Command{
		Use:   "create",
		Short: "Invite a user to a team",
		Long:  "Create and send a pending invitation to an email address. The recipient can register before accepting it.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			teamID, request, err := buildTeamInvitationCreateRequest(*options)
			if err != nil {
				return err
			}

			baseURL, token, err := resolveCloudInvitationTarget(cmd)
			if err != nil {
				return err
			}

			var invitation cloudTeamInvitation
			if err := authRequest(
				cmd.Context(),
				http.MethodPost,
				cloudTeamInvitationCollectionURL(baseURL, teamID),
				token,
				request,
				&invitation,
			); err != nil {
				return fmt.Errorf("create team invitation: %w", err)
			}

			return getFormatter().Format(cmd.OutOrStdout(), invitation.output())
		},
	}
	command.Flags().StringVarP(&options.teamID, "team-id", "t", "", "team ID (required)")
	command.Flags().StringVar(&options.email, "email", "", "invitee email address (required)")
	command.Flags().StringVar(&options.role, "role", "developer", "team role (admin, developer, builder, viewer)")
	return command
}

func buildTeamInvitationCreateRequest(options teamInvitationCreateOptions) (string, teamInvitationCreateRequest, error) {
	teamID := strings.TrimSpace(options.teamID)
	if teamID == "" {
		return "", teamInvitationCreateRequest{}, fmt.Errorf("--team-id is required")
	}

	email := strings.ToLower(strings.TrimSpace(options.email))
	if email == "" {
		return "", teamInvitationCreateRequest{}, fmt.Errorf("--email is required")
	}
	address, err := mail.ParseAddress(email)
	if err != nil || !strings.EqualFold(address.Address, email) {
		return "", teamInvitationCreateRequest{}, fmt.Errorf("--email must be a valid email address")
	}

	role := strings.ToLower(strings.TrimSpace(options.role))
	switch role {
	case "admin", "developer", "builder", "viewer":
	default:
		return "", teamInvitationCreateRequest{}, fmt.Errorf(
			"invalid --role %q, must be one of: admin, developer, builder, viewer",
			options.role,
		)
	}

	return teamID, teamInvitationCreateRequest{Email: email, Role: role}, nil
}

func resolveCloudInvitationTarget(cmd *cobra.Command) (string, string, error) {
	target, _, _, _, err := resolveClientTarget(cmd)
	if err != nil {
		return "", "", fmt.Errorf("resolve Cloud Gateway target: %w", err)
	}
	if target.GatewayMode != config.GatewayModeGlobal {
		return "", "", fmt.Errorf("team invitations require a Sandbox0 Cloud Global Gateway")
	}
	return target.BaseURL, target.Token, nil
}

func cloudTeamInvitationCollectionURL(baseURL, teamID string) string {
	return strings.TrimRight(baseURL, "/") + cloudInvitationPathPrefix + "/teams/" +
		url.PathEscape(teamID) + "/invitations"
}

func (invitation cloudTeamInvitation) output() *output.TeamInvitation {
	return &output.TeamInvitation{
		ID:                    invitation.ID,
		TeamID:                invitation.TeamID,
		TeamName:              invitation.TeamName,
		InviteeEmail:          invitation.InviteeEmail,
		Role:                  invitation.Role,
		Status:                invitation.Status,
		ExpiresAt:             invitation.ExpiresAt,
		LastSendRequestedAt:   invitation.LastSendRequestedAt,
		LastSentAt:            invitation.LastSentAt,
		SendCount:             invitation.SendCount,
		DeliveryState:         invitation.DeliveryState,
		DeliveryLastError:     invitation.DeliveryLastError,
		DeliveryNextAttemptAt: invitation.DeliveryNextAttemptAt,
		CreatedAt:             invitation.CreatedAt,
		UpdatedAt:             invitation.UpdatedAt,
	}
}
