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

const restartCommandID = "40ea02de-5ef4-4710-a742-c9854cdfb6ce"

type restartCommandBackend struct {
	submitted, read, found int
	submitErr              error
	states                 []apispec.SandboxRestartOperationState
	cancel                 context.CancelFunc
}

func (b *restartCommandBackend) receipt() *apispec.SandboxRestartOperation {
	state := apispec.SandboxRestartOperationState("pausing")
	if len(b.states) > 0 {
		state = b.states[0]
		if len(b.states) > 1 {
			b.states = b.states[1:]
		}
	}
	r := &apispec.SandboxRestartOperation{ID: restartCommandID, SandboxID: "sandbox", RequestKey: "retry-key", FromRuntimeGeneration: 7, State: state}
	if state == "succeeded" {
		r.ResultRuntimeGeneration = apispec.NewOptInt64(8)
	}
	return r
}
func (b *restartCommandBackend) RestartSandbox(ctx context.Context, id string, generation int64, key string) (*apispec.SandboxRestartOperation, error) {
	b.submitted++
	if id != "sandbox" || generation != 7 || key != "retry-key" {
		return nil, errors.New("identity changed")
	}
	if b.cancel != nil {
		b.cancel()
	}
	return b.receipt(), b.submitErr
}
func (b *restartCommandBackend) GetSandboxRestartOperation(ctx context.Context, id, key string) (*apispec.SandboxRestartOperation, error) {
	b.read++
	if id != "sandbox" || key != restartCommandID {
		return nil, errors.New("lookup identity changed")
	}
	return b.receipt(), nil
}
func (b *restartCommandBackend) FindSandboxRestartOperation(ctx context.Context, id, key string) (*apispec.SandboxRestartOperation, error) {
	b.found++
	if id != "sandbox" || key != "retry-key" {
		return nil, errors.New("lookup identity changed")
	}
	return b.receipt(), nil
}
func runRestartCommand(t *testing.T, b *restartCommandBackend, readOnly bool, ctx context.Context, args ...string) (string, string, error) {
	t.Helper()
	old := cfgFormat
	cfgFormat = "json"
	t.Cleanup(func() { cfgFormat = old })
	cmd := newSandboxRestartCommand(func(*cobra.Command) (sandboxRestartClient, error) { return b, nil }, readOnly)
	var out, stderr bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	cmd.SetContext(ctx)
	err := cmd.Execute()
	return out.String(), stderr.String(), err
}
func TestRestartCommandSubmitsOnceAndPollsExactReceipt(t *testing.T) {
	b := &restartCommandBackend{states: []apispec.SandboxRestartOperationState{"pausing", "paused", "resuming", "succeeded"}}
	out, _, err := runRestartCommand(t, b, false, t.Context(), "sandbox", "--expected-runtime-generation", "7", "--idempotency-key", "retry-key", "--wait", "--poll-interval", "1ms")
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
func TestRestartStatusNeverSubmits(t *testing.T) {
	for _, args := range [][]string{{"sandbox", "--idempotency-key", "retry-key"}, {"sandbox", "--operation-id", restartCommandID}} {
		b := &restartCommandBackend{states: []apispec.SandboxRestartOperationState{"succeeded"}}
		_, _, err := runRestartCommand(t, b, true, t.Context(), args...)
		if err != nil {
			t.Fatal(err)
		}
		if b.submitted != 0 || b.read+b.found != 1 {
			t.Fatalf("calls: %+v", b)
		}
	}
}
func TestRestartCommandRejectsAmbiguousIntentBeforeClient(t *testing.T) {
	for _, tc := range []struct {
		read bool
		args []string
	}{
		{false, []string{"sandbox", "--idempotency-key", "retry-key"}},
		{false, []string{"sandbox", "--expected-runtime-generation", "7"}},
		{false, []string{"sandbox", "--expected-runtime-generation", "-1", "--idempotency-key", "retry-key"}},
		{false, []string{"sandbox", "--expected-runtime-generation", "7", "--idempotency-key", "bad key"}},
		{true, []string{"sandbox"}},
		{true, []string{"sandbox", "--operation-id", restartCommandID, "--idempotency-key", "retry-key"}},
		{true, []string{"sandbox", "--operation-id", "bad"}},
	} {
		cmd := newSandboxRestartCommand(func(*cobra.Command) (sandboxRestartClient, error) {
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
func TestRestartCommandLostResponseRetainsOriginalRetryCoordinates(t *testing.T) {
	b := &restartCommandBackend{submitErr: errors.New("lost response")}
	_, stderr, err := runRestartCommand(t, b, false, t.Context(), "sandbox", "--expected-runtime-generation", "7", "--idempotency-key", "retry-key")
	if err == nil || b.submitted != 1 || b.read != 0 {
		t.Fatal("lost response replayed")
	}
	for _, text := range []string{"restart-status sandbox --idempotency-key retry-key", "--expected-runtime-generation 7"} {
		if !strings.Contains(stderr, text) {
			t.Fatalf("missing retry coordinates: %s", stderr)
		}
	}
}
func TestRestartCommandWaitCancellationPreservesPendingReceipt(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	b := &restartCommandBackend{cancel: cancel}
	out, stderr, err := runRestartCommand(t, b, false, ctx, "sandbox", "--expected-runtime-generation", "7", "--idempotency-key", "retry-key", "--wait")
	if !errors.Is(err, context.Canceled) || b.submitted != 1 || b.read != 0 {
		t.Fatalf("cancel result: %v %+v", err, b)
	}
	if !strings.Contains(out, restartCommandID) || !strings.Contains(stderr, "not canceled") {
		t.Fatal("lost pending receipt or cancellation semantics")
	}
}
func TestRestartCommandTerminalFailureIsNotSuccess(t *testing.T) {
	for _, state := range []apispec.SandboxRestartOperationState{"failed", "superseded"} {
		b := &restartCommandBackend{states: []apispec.SandboxRestartOperationState{state}}
		out, _, err := runRestartCommand(t, b, true, t.Context(), "sandbox", "--operation-id", restartCommandID, "--wait")
		if err == nil || !strings.Contains(out, string(state)) || b.read != 1 {
			t.Fatal("terminal failure hidden")
		}
	}
}
func TestRestartCommandWaitDeadlineDoesNotResubmit(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()
	b := &restartCommandBackend{}
	_, _, err := runRestartCommand(t, b, false, ctx, "sandbox", "--expected-runtime-generation", "7", "--idempotency-key", "retry-key", "--wait", "--poll-interval", "1h")
	if !errors.Is(err, context.DeadlineExceeded) || b.submitted != 1 || b.read != 0 {
		t.Fatalf("deadline result: %v %+v", err, b)
	}
}
