package commands

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

const snapshotRestoreCommandID = "40ea02de-5ef4-4710-a742-c9854cdfb6ce"

type snapshotRestoreCommandBackend struct {
	submitted, read, found int
	submitErr              error
	states                 []apispec.SandboxSnapshotRestoreOperationState
	cancel                 context.CancelFunc
}

func (b *snapshotRestoreCommandBackend) receipt() *apispec.SandboxSnapshotRestoreOperation {
	state := apispec.SandboxSnapshotRestoreOperationState("pausing")
	if len(b.states) > 0 {
		state = b.states[0]
		if len(b.states) > 1 {
			b.states = b.states[1:]
		}
	}
	r := &apispec.SandboxSnapshotRestoreOperation{ID: snapshotRestoreCommandID, SandboxID: "sandbox", RequestKey: "retry-key", FromRuntimeGeneration: 7, State: state, SnapshotID: "snapshot", SnapshotGenerationID: "generation", DiscardProcessState: true}
	if state == "succeeded" {
		r.ResultRuntimeGeneration = apispec.NewOptInt64(8)
	}
	return r
}
func (b *snapshotRestoreCommandBackend) RestoreSandboxSnapshot(ctx context.Context, id string, request apispec.SandboxSnapshotRestoreRequest, key string) (*apispec.SandboxSnapshotRestoreOperation, error) {
	b.submitted++
	if id != "sandbox" || request.ExpectedRuntimeGeneration != 7 || request.SnapshotID != "snapshot" || request.SnapshotGenerationID != "generation" || !bool(request.DiscardProcessState) || key != "retry-key" {
		return nil, errors.New("identity changed")
	}
	if b.cancel != nil {
		b.cancel()
	}
	return b.receipt(), b.submitErr
}
func (b *snapshotRestoreCommandBackend) GetSandboxSnapshotRestoreOperation(ctx context.Context, id, key string) (*apispec.SandboxSnapshotRestoreOperation, error) {
	b.read++
	if id != "sandbox" || key != snapshotRestoreCommandID {
		return nil, errors.New("lookup identity changed")
	}
	return b.receipt(), nil
}
func (b *snapshotRestoreCommandBackend) FindSandboxSnapshotRestoreOperation(ctx context.Context, id, key string) (*apispec.SandboxSnapshotRestoreOperation, error) {
	b.found++
	if id != "sandbox" || key != "retry-key" {
		return nil, errors.New("lookup identity changed")
	}
	return b.receipt(), nil
}
func runSnapshotRestoreCommand(t *testing.T, b *snapshotRestoreCommandBackend, readOnly bool, ctx context.Context, args ...string) (string, string, error) {
	t.Helper()
	old := cfgFormat
	cfgFormat = "json"
	t.Cleanup(func() { cfgFormat = old })
	cmd := newSandboxSnapshotRestoreCommand(func(*cobra.Command) (sandboxSnapshotRestoreClient, error) { return b, nil }, readOnly)
	var out, stderr bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	if !readOnly {
		args = append(args, "--snapshot-id", "snapshot", "--snapshot-generation-id", "generation", "--discard-process-state")
	}
	cmd.SetArgs(args)
	cmd.SetContext(ctx)
	err := cmd.Execute()
	return out.String(), stderr.String(), err
}
func TestSnapshotRestoreCommandSubmitsOnceAndPollsExactReceipt(t *testing.T) {
	b := &snapshotRestoreCommandBackend{states: []apispec.SandboxSnapshotRestoreOperationState{"pausing", "paused", "resuming", "succeeded"}}
	out, _, err := runSnapshotRestoreCommand(t, b, false, t.Context(), "sandbox", "--expected-runtime-generation", "7", "--idempotency-key", "retry-key", "--wait", "--poll-interval", "1ms")
	if err != nil {
		t.Fatal(err)
	}
	if b.submitted != 1 || b.read != 3 || b.found != 0 {
		t.Fatalf("calls: %+v", b)
	}
	if !strings.Contains(out, `"state": "succeeded"`) || !strings.Contains(out, `"result_runtime_generation": 8`) {
		t.Fatalf("missing terminal receipt: %s", out)
	}
}
func TestSnapshotRestoreStatusNeverSubmits(t *testing.T) {
	for _, args := range [][]string{{"sandbox", "--idempotency-key", "retry-key"}, {"sandbox", "--operation-id", snapshotRestoreCommandID}} {
		b := &snapshotRestoreCommandBackend{states: []apispec.SandboxSnapshotRestoreOperationState{"succeeded"}}
		_, _, err := runSnapshotRestoreCommand(t, b, true, t.Context(), args...)
		if err != nil {
			t.Fatal(err)
		}
		if b.submitted != 0 || b.read+b.found != 1 {
			t.Fatalf("calls: %+v", b)
		}
	}
}
func TestSnapshotRestoreCommandRejectsAmbiguousIntentBeforeClient(t *testing.T) {
	for _, tc := range []struct {
		read bool
		args []string
	}{
		{false, []string{"sandbox", "--idempotency-key", "retry-key"}},
		{false, []string{"sandbox", "--expected-runtime-generation", "7"}},
		{false, []string{"sandbox", "--expected-runtime-generation", "-1", "--idempotency-key", "retry-key"}},
		{false, []string{"sandbox", "--expected-runtime-generation", "7", "--idempotency-key", "bad key"}},
		{true, []string{"sandbox"}},
		{true, []string{"sandbox", "--operation-id", snapshotRestoreCommandID, "--idempotency-key", "retry-key"}},
		{true, []string{"sandbox", "--operation-id", "bad"}},
	} {
		cmd := newSandboxSnapshotRestoreCommand(func(*cobra.Command) (sandboxSnapshotRestoreClient, error) {
			t.Fatal("invalid intent created client")
			return nil, nil
		}, tc.read)
		cmd.SetArgs(tc.args)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		if err := cmd.Execute(); err == nil {
			t.Fatal("invalid intent accepted")
		}
	}
}
func TestSnapshotRestoreCommandLostResponseRetainsOriginalRetryCoordinates(t *testing.T) {
	b := &snapshotRestoreCommandBackend{submitErr: errors.New("lost response")}
	_, stderr, err := runSnapshotRestoreCommand(t, b, false, t.Context(), "sandbox", "--expected-runtime-generation", "7", "--idempotency-key", "retry-key")
	if err == nil || b.submitted != 1 || b.read != 0 {
		t.Fatal("lost response replayed")
	}
	for _, text := range []string{"restore-snapshot-status sandbox --idempotency-key retry-key", "--expected-runtime-generation 7"} {
		if !strings.Contains(stderr, text) {
			t.Fatalf("missing retry coordinates: %s", stderr)
		}
	}
}
func TestSnapshotRestoreCommandWaitCancellationPreservesPendingReceipt(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	b := &snapshotRestoreCommandBackend{cancel: cancel}
	out, stderr, err := runSnapshotRestoreCommand(t, b, false, ctx, "sandbox", "--expected-runtime-generation", "7", "--idempotency-key", "retry-key", "--wait")
	if !errors.Is(err, context.Canceled) || b.submitted != 1 || b.read != 0 {
		t.Fatalf("cancel result: %v %+v", err, b)
	}
	if !strings.Contains(out, snapshotRestoreCommandID) || !strings.Contains(stderr, "not canceled") {
		t.Fatal("lost pending receipt or cancellation semantics")
	}
}
func TestSnapshotRestoreCommandTerminalFailureIsNotSuccess(t *testing.T) {
	for _, state := range []apispec.SandboxSnapshotRestoreOperationState{"failed", "superseded"} {
		b := &snapshotRestoreCommandBackend{states: []apispec.SandboxSnapshotRestoreOperationState{state}}
		out, _, err := runSnapshotRestoreCommand(t, b, true, t.Context(), "sandbox", "--operation-id", snapshotRestoreCommandID, "--wait")
		if err == nil || !strings.Contains(out, string(state)) || b.read != 1 {
			t.Fatal("terminal failure hidden")
		}
	}
}
func TestSnapshotRestoreCommandWaitDeadlineDoesNotResubmit(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()
	b := &snapshotRestoreCommandBackend{}
	_, _, err := runSnapshotRestoreCommand(t, b, false, ctx, "sandbox", "--expected-runtime-generation", "7", "--idempotency-key", "retry-key", "--wait", "--poll-interval", "1h")
	if !errors.Is(err, context.DeadlineExceeded) || b.submitted != 1 || b.read != 0 {
		t.Fatalf("deadline result: %v %+v", err, b)
	}
}

func TestSnapshotRestoreCommandRequiresExplicitTargetAndDiscard(t *testing.T) {
	for _, change := range []struct{ flag, value string }{{"--snapshot-id", ""}, {"--snapshot-generation-id", ""}, {"--discard-process-state", "false"}, {"--expected-runtime-generation", "0"}, {"--idempotency-key", "bad key"}} {
		cmd := newSandboxSnapshotRestoreCommand(func(*cobra.Command) (sandboxSnapshotRestoreClient, error) {
			t.Fatal("invalid recovery intent reached client")
			return nil, nil
		}, false)
		args := []string{"sandbox", "--snapshot-id", "snapshot", "--snapshot-generation-id", "generation", "--discard-process-state=true", "--expected-runtime-generation", "7", "--idempotency-key", "retry-key", change.flag + "=" + change.value}
		cmd.SetArgs(args)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		if err := cmd.Execute(); err == nil {
			t.Fatalf("accepted invalid %s", change.flag)
		}
	}
}
