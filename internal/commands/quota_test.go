package commands

import (
	"slices"
	"strings"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

func TestParseTeamQuotaDimensionUsesGeneratedValues(t *testing.T) {
	for _, want := range apispec.QuotaDimension("").AllValues() {
		got, err := parseTeamQuotaDimension(" " + string(want) + " ")
		if err != nil {
			t.Fatalf("parseTeamQuotaDimension(%q) error = %v", want, err)
		}
		if got != want {
			t.Fatalf("parseTeamQuotaDimension(%q) = %q, want %q", want, got, want)
		}
	}
}

func TestParseTeamQuotaDimensionRejectsLegacyTrafficTotals(t *testing.T) {
	for _, legacy := range []string{"egress", "ingress"} {
		_, err := parseTeamQuotaDimension(legacy)
		if err == nil {
			t.Fatalf("parseTeamQuotaDimension(%q) unexpectedly succeeded", legacy)
		}
		if !strings.Contains(err.Error(), "network_egress_bytes") ||
			!strings.Contains(err.Error(), "network_ingress_bytes") {
			t.Fatalf("error %q does not list network bandwidth dimensions", err)
		}
	}
}

func TestQuotaCommandUsesHomeRegionRouting(t *testing.T) {
	root := &cobra.Command{Use: "s0"}
	quota := newQuotaCommand()
	root.AddCommand(quota)

	list, _, err := quota.Find([]string{"list"})
	if err != nil {
		t.Fatalf("find quota list: %v", err)
	}
	if got := commandRouteScope(list); got != "home-region" {
		t.Fatalf("commandRouteScope(quota list) = %q, want home-region", got)
	}
}

func TestQuotaGetCompletionListsAllDimensions(t *testing.T) {
	quota := newQuotaCommand()
	get, _, err := quota.Find([]string{"get"})
	if err != nil {
		t.Fatalf("find quota get: %v", err)
	}

	for _, dimension := range apispec.QuotaDimension("").AllValues() {
		if !slices.Contains(get.ValidArgs, string(dimension)) {
			t.Fatalf("quota get completion missing %q", dimension)
		}
	}
}
