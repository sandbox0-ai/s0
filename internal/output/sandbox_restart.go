package output

import (
	"io"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func (f *TableFormatter) formatSandboxRestart(w io.Writer, r *apispec.SandboxRestartOperation) error {
	t := tablewriter.NewWriter(w)
	t.Header([]string{"Field", "Value"})
	for _, row := range [][]string{{"Operation", r.ID}, {"Sandbox", r.SandboxID}, {"Request key", r.RequestKey}, {"Source generation", strconv.FormatInt(r.FromRuntimeGeneration, 10)}, {"State", string(r.State)}, {"Created", r.CreatedAt.Format(timeLayout)}, {"Updated", r.UpdatedAt.Format(timeLayout)}} {
		_ = t.Append(row)
	}
	if generation, ok := r.ResultRuntimeGeneration.Get(); ok {
		_ = t.Append([]string{"Replacement generation", strconv.FormatInt(generation, 10)})
	}
	if reason, ok := r.Reason.Get(); ok {
		_ = t.Append([]string{"Terminal reason", string(reason)})
	}
	if retry, ok := r.Retry.Get(); ok {
		_ = t.Append([]string{"Attempts", strconv.FormatInt(retry.Attempts, 10)})
		if reason, ok := retry.BlockedReason.Get(); ok {
			_ = t.Append([]string{"Blocked reason", string(reason)})
		}
		if observed, ok := retry.ObservedAt.Get(); ok {
			_ = t.Append([]string{"Observed", observed.Format(timeLayout)})
		}
		if stage, ok := retry.ObservedStage.Get(); ok {
			_ = t.Append([]string{"Observed stage", string(stage)})
		}
		if next, ok := retry.NextAttemptAt.Get(); ok {
			_ = t.Append([]string{"Next attempt", next.Format(timeLayout)})
		}
	}
	return t.Render()
}
