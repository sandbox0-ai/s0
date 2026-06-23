package output

import (
	"strings"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// TeamListItem adds local CLI selection state to an API team.
type TeamListItem struct {
	ID           string    `json:"id" yaml:"id"`
	Name         string    `json:"name" yaml:"name"`
	Slug         string    `json:"slug" yaml:"slug"`
	OwnerID      *string   `json:"owner_id,omitempty" yaml:"owner_id,omitempty"`
	HomeRegionID *string   `json:"home_region_id,omitempty" yaml:"home_region_id,omitempty"`
	CreatedAt    time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" yaml:"updated_at"`
	Current      bool      `json:"current" yaml:"current"`
}

type TeamList []TeamListItem

func NewTeamList(teams []apispec.Team, currentTeamID string) TeamList {
	currentTeamID = strings.TrimSpace(currentTeamID)
	items := make(TeamList, 0, len(teams))
	for _, team := range teams {
		items = append(items, TeamListItem{
			ID:           team.ID,
			Name:         team.Name,
			Slug:         team.Slug,
			OwnerID:      optNilStringPtr(team.OwnerID),
			HomeRegionID: optNilStringPtr(team.HomeRegionID),
			CreatedAt:    team.CreatedAt,
			UpdatedAt:    team.UpdatedAt,
			Current:      currentTeamID != "" && strings.TrimSpace(team.ID) == currentTeamID,
		})
	}
	return items
}

func optNilStringPtr(value apispec.OptNilString) *string {
	if s, ok := value.Get(); ok {
		return &s
	}
	return nil
}
