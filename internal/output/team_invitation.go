package output

import "time"

// TeamInvitation represents a pending or completed Cloud team invitation.
type TeamInvitation struct {
	ID                    string     `json:"id" yaml:"id"`
	TeamID                string     `json:"team_id" yaml:"team_id"`
	TeamName              string     `json:"team_name,omitempty" yaml:"team_name,omitempty"`
	InviteeEmail          string     `json:"invitee_email" yaml:"invitee_email"`
	Role                  string     `json:"role" yaml:"role"`
	Status                string     `json:"status" yaml:"status"`
	ExpiresAt             time.Time  `json:"expires_at" yaml:"expires_at"`
	LastSendRequestedAt   time.Time  `json:"last_send_requested_at" yaml:"last_send_requested_at"`
	LastSentAt            *time.Time `json:"last_sent_at,omitempty" yaml:"last_sent_at,omitempty"`
	SendCount             int        `json:"send_count" yaml:"send_count"`
	DeliveryState         string     `json:"delivery_state" yaml:"delivery_state"`
	DeliveryLastError     string     `json:"delivery_last_error,omitempty" yaml:"delivery_last_error,omitempty"`
	DeliveryNextAttemptAt *time.Time `json:"delivery_next_attempt_at,omitempty" yaml:"delivery_next_attempt_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at" yaml:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" yaml:"updated_at"`
}
