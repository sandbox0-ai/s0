package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestTableFormatterFormatsUsageWindowPage(t *testing.T) {
	page := &apispec.UsageWindowPage{
		Windows: []apispec.UsageWindow{
			{
				WindowID:    "window-1",
				WindowType:  "sandbox.runtime_mib_milliseconds",
				SubjectType: "sandbox",
				SubjectID:   "sandbox-1",
				SandboxID:   apispec.NewOptString("sandbox-1"),
				WindowStart: time.Date(2026, time.July, 26, 0, 0, 0, 0, time.UTC),
				WindowEnd:   time.Date(2026, time.July, 26, 1, 0, 0, 0, time.UTC),
				Value:       3_686_400_000,
				Unit:        "mib_milliseconds",
				RecordedAt:  time.Date(2026, time.July, 26, 1, 0, 1, 0, time.UTC),
			},
		},
		NextCursor: "page-2",
	}

	var output bytes.Buffer
	if err := NewFormatter(FormatTable).Format(&output, page); err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	for _, want := range []string{
		"WINDOW ID",
		"window-1",
		"sandbox.runtime_mib_milliseconds",
		"sandbox:sandbox-1",
		"3686400000",
		"mib_milliseconds",
		"Next cursor: page-2",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, output.String())
		}
	}
}

func TestTableFormatterFormatsEmptyUsageWindowPage(t *testing.T) {
	var output bytes.Buffer
	if err := NewFormatter(FormatTable).Format(
		&output,
		&apispec.UsageWindowPage{},
	); err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	if got := output.String(); got != "No usage windows found.\n" {
		t.Fatalf("output = %q, want empty usage message", got)
	}
}
