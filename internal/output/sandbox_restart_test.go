package output

import (
	"bytes"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"math"
	"strings"
	"testing"
)

func TestRestartReceiptFormatsKeepExactGeneration(t *testing.T) {
	r := &apispec.SandboxRestartOperation{ID: "40ea02de-5ef4-4710-a742-c9854cdfb6ce", SandboxID: "sandbox", RequestKey: "retry-key", FromRuntimeGeneration: math.MaxInt64 - 1, State: "succeeded", ResultRuntimeGeneration: apispec.NewOptInt64(math.MaxInt64)}
	for _, format := range []Format{FormatTable, FormatJSON, FormatYAML} {
		var out bytes.Buffer
		if err := NewFormatter(format).Format(&out, r); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{r.ID, "retry-key", "succeeded", "9223372036854775806", "9223372036854775807"} {
			if !strings.Contains(out.String(), want) {
				t.Fatalf("%s lost receipt field %s: %s", format, want, out.String())
			}
		}
	}
}
