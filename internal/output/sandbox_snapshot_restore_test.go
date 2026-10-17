package output

import (
	"bytes"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"strings"
	"testing"
)

func TestSnapshotRestoreReceiptFormatsKeepExactGeneration(t *testing.T) {
	r := &apispec.SandboxSnapshotRestoreOperation{ID: "40ea02de-5ef4-4710-a742-c9854cdfb6ce", SandboxID: "sandbox", RequestKey: "retry-key", FromRuntimeGeneration: 9007199254740990, State: "succeeded", SnapshotID: "snapshot-target", SnapshotGenerationID: "snapshot-generation", DiscardProcessState: true, ResultRuntimeGeneration: apispec.NewOptInt64(9007199254740991)}
	for _, format := range []Format{FormatTable, FormatJSON, FormatYAML} {
		var out bytes.Buffer
		if err := NewFormatter(format).Format(&out, r); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{r.ID, "retry-key", "succeeded", "snapshot-target", "snapshot-generation", "9007199254740990", "9007199254740991"} {
			if !strings.Contains(out.String(), want) {
				t.Fatalf("%s lost receipt field %s: %s", format, want, out.String())
			}
		}
	}
}
