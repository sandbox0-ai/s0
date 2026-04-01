package commands

import "testing"

func TestBuildCreateTeamRequest(t *testing.T) {
	req := buildCreateTeamRequest("Team One", " team-one ", " aws/us-east-1 ")

	if req.Name != "Team One" {
		t.Fatalf("Name = %q, want Team One", req.Name)
	}

	slug, ok := req.Slug.Get()
	if !ok || slug != "team-one" {
		t.Fatalf("Slug = %q, want team-one", slug)
	}

	homeRegion, ok := req.HomeRegionID.Get()
	if !ok || homeRegion != "aws/us-east-1" {
		t.Fatalf("HomeRegionID = %q, want aws/us-east-1", homeRegion)
	}
}

func TestBuildCreateTeamRequestOmitsOptionalFieldsWhenBlank(t *testing.T) {
	req := buildCreateTeamRequest("Team One", "   ", "   ")

	if _, ok := req.Slug.Get(); ok {
		t.Fatal("Slug should be unset")
	}
	if _, ok := req.HomeRegionID.Get(); ok {
		t.Fatal("HomeRegionID should be unset")
	}
}

func TestBuildActivateTeamRequest(t *testing.T) {
	req := buildActivateTeamRequest(" team-1 ")

	defaultTeamID, ok := req.DefaultTeamID.Get()
	if !ok || defaultTeamID != "team-1" {
		t.Fatalf("DefaultTeamID = %q, want team-1", defaultTeamID)
	}
}
