package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestTableFormatterFormatsCapacityAndRateQuotas(t *testing.T) {
	nullValue := apispec.NewNilInt64(0)
	nullValue.SetToNull()
	quotas := []apispec.TeamQuota{
		{
			TeamID:     "team-1",
			Dimension:  apispec.QuotaDimensionActiveSandboxes,
			Kind:       apispec.TeamQuotaKindCapacity,
			LimitValue: apispec.NewNilInt64(10),
			IntervalMs: nullValue,
			BurstValue: nullValue,
			Current:    apispec.NewNilInt64(3),
			Remaining:  apispec.NewNilInt64(7),
			Unit:       apispec.TeamQuotaUnitCount,
			Source:     apispec.TeamQuotaSourceRegionDefault,
		},
		{
			TeamID:     "team-1",
			Dimension:  apispec.QuotaDimensionAPIRequests,
			Kind:       apispec.TeamQuotaKindRate,
			LimitValue: apispec.NewNilInt64(100),
			IntervalMs: apispec.NewNilInt64(1000),
			BurstValue: apispec.NewNilInt64(200),
			Current:    nullValue,
			Remaining:  nullValue,
			Unit:       apispec.TeamQuotaUnitRequests,
			Source:     apispec.TeamQuotaSourceTeamOverride,
		},
	}

	var output bytes.Buffer
	if err := NewFormatter(FormatTable).Format(&output, quotas); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	got := output.String()
	for _, want := range []string{
		"DIMENSION",
		"active_sandboxes",
		"api_requests",
		"capacity",
		"rate",
		"1s",
		"region_default",
		"team_override",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestTableFormatterFormatsUnlimitedQuota(t *testing.T) {
	nullValue := apispec.NewNilInt64(0)
	nullValue.SetToNull()
	quota := &apispec.TeamQuota{
		TeamID:     "team-1",
		Dimension:  apispec.QuotaDimensionMemoryMib,
		Kind:       apispec.TeamQuotaKindCapacity,
		LimitValue: nullValue,
		IntervalMs: nullValue,
		BurstValue: nullValue,
		Current:    apispec.NewNilInt64(256),
		Remaining:  nullValue,
		Unlimited:  true,
		Unit:       apispec.TeamQuotaUnitMiB,
		Source:     apispec.TeamQuotaSourceUnlimited,
	}

	var output bytes.Buffer
	if err := NewFormatter(FormatTable).Format(&output, quota); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	for _, want := range []string{"memory_mib", "unlimited", "Current:", "256"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, output.String())
		}
	}
}
