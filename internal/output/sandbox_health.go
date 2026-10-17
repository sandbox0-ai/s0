package output

import (
	"fmt"
	"io"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func (f *TableFormatter) formatSandboxHealth(w io.Writer, h *apispec.SandboxHealth) error {
	if _, err := fmt.Fprintf(w, "Sandbox: %s  Generation: %d  Lifecycle epoch: %d\nDesired state: %s  Active lifecycle: %t  Observed: %s\n", h.SandboxID, h.RuntimeGeneration, h.LifecycleEpoch, h.DesiredState, h.ActiveLifecycle, h.ObservedAt.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	t := tablewriter.NewWriter(w)
	t.Header([]string{"Check", "State", "Reason", "Source", "Observed at"})
	for _, row := range []struct {
		name  string
		value apispec.SandboxHealthObservation
	}{
		{"Runtime launch authority", h.Runtime.LaunchAuthority}, {"Runtime liveness", h.Runtime.Liveness},
		{"Execution", h.Execution}, {"Terminal", h.Terminal}, {"Storage identity", h.Storage.Identity},
		{"Storage writer authority", h.Storage.WriterAuthority}, {"Storage I/O", h.Storage.Io},
	} {
		at := "unobserved"
		if value, ok := row.value.ObservedAt.Get(); ok {
			at = value.Format(time.RFC3339Nano)
		}
		if err := t.Append([]string{row.name, string(row.value.State), string(row.value.Reason), string(row.value.Source), at}); err != nil {
			return err
		}
	}
	if err := t.Render(); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, "Guest diagnostics are advisory. Storage identity and writer authority do not establish current I/O health.")
	return err
}
