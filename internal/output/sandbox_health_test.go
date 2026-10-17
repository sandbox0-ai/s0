package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestHealthFormatsKeepIndependentEvidenceAndExactGeneration(t *testing.T) {
	at := time.Date(2026, 10, 15, 0, 0, 1, 0, time.UTC)
	sourced := func(state, reason, source string) apispec.SandboxHealthObservation {
		return apispec.SandboxHealthObservation{State: apispec.SandboxHealthObservationState(state), Reason: apispec.SandboxHealthObservationReason(reason), Source: apispec.SandboxHealthObservationSource(source), ObservedAt: apispec.NewNilDateTime(at)}
	}
	unknown := apispec.SandboxHealthObservation{State: "unknown", Reason: "storage_io_unobserved", Source: "none", ObservedAt: apispec.NilDateTime{Null: true}}
	h := &apispec.SandboxHealth{SandboxID: "sandbox", RuntimeGeneration: 9007199254740993, LifecycleEpoch: 8, DesiredState: "active", ObservedAt: at.Add(time.Second),
		Runtime:   apispec.SandboxRuntimeHealth{LaunchAuthority: sourced("available", "external_launch_authority_present", "regional_authority"), Liveness: sourced("available", "runsc_running", "node_runtime")},
		Execution: sourced("unavailable", "command_start_failed", "guest_diagnostic"), Terminal: unknown,
		Storage: apispec.SandboxStorageHealth{Identity: sourced("available", "rootfs_identity_present", "regional_authority"), WriterAuthority: sourced("available", "current_writer_lease_present", "regional_authority"), Io: unknown}}
	for _, format := range []Format{FormatTable, FormatJSON, FormatYAML} {
		var out bytes.Buffer
		if err := NewFormatter(format).Format(&out, h); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"9007199254740993", "command_start_failed", "storage_io_unobserved", "guest_diagnostic", "node_runtime", "2026-10-15T00:00:01Z", "unknown", "unavailable"} {
			if !strings.Contains(out.String(), want) {
				t.Fatalf("%s lost %s: %s", format, want, out.String())
			}
		}
	}
}
