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

type healthCommandBackend struct {
	calls int
	block bool
	err   error
}

func (b *healthCommandBackend) GetSandboxHealth(ctx context.Context, id string) (*apispec.SandboxHealth, error) {
	b.calls++
	if id != "sandbox" {
		return nil, errors.New("wrong sandbox")
	}
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("missing request deadline")
	}
	if b.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if b.err != nil {
		return nil, b.err
	}
	return &apispec.SandboxHealth{SandboxID: id, RuntimeGeneration: 9007199254740993,
		Execution: apispec.SandboxHealthObservation{State: "unavailable", Reason: "command_start_failed", Source: "guest_diagnostic", ObservedAt: apispec.NewNilDateTime(time.Now())},
		Terminal:  apispec.SandboxHealthObservation{State: "unknown", Reason: "guest_diagnostics_unavailable", Source: "none", ObservedAt: apispec.NilDateTime{Null: true}},
	}, nil
}
func TestHealthCommandReturnsIndependentFailureAsObservation(t *testing.T) {
	old := cfgFormat
	cfgFormat = "json"
	t.Cleanup(func() { cfgFormat = old })
	b := &healthCommandBackend{}
	cmd := newSandboxHealthCommand(func(*cobra.Command) (sandboxHealthClient, error) { return b, nil })
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"sandbox"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"9007199254740993", `"state": "unavailable"`, `"state": "unknown"`, `"observed_at": null`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %s in %s", want, out.String())
		}
	}
	if b.calls != 1 {
		t.Fatal("unexpected retry")
	}
}
func TestHealthCommandBoundsStallAndDoesNotRetry(t *testing.T) {
	for _, b := range []*healthCommandBackend{{block: true}, {err: errors.New("unavailable")}} {
		cmd := newSandboxHealthCommand(func(*cobra.Command) (sandboxHealthClient, error) { return b, nil })
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"sandbox", "--timeout", "10ms"})
		start := time.Now()
		if err := cmd.Execute(); err == nil {
			t.Fatal("missing error")
		}
		if time.Since(start) > time.Second || b.calls != 1 || out.Len() != 0 {
			t.Fatal("query retried, hung or invented evidence")
		}
	}
}
func TestHealthCommandRejectsInvalidArgumentsBeforeClient(t *testing.T) {
	for _, args := range [][]string{{}, {"sandbox", "extra"}, {"sandbox", "--timeout", "0s"}, {" sandbox"}} {
		cmd := newSandboxHealthCommand(func(*cobra.Command) (sandboxHealthClient, error) {
			t.Fatal("invalid query created client")
			return nil, nil
		})
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil {
			t.Fatal("invalid arguments accepted")
		}
	}
}
