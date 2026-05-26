package commands

import (
	"testing"

	"github.com/sandbox0-ai/s0/internal/client"
	"github.com/spf13/cobra"
)

func TestCommandRouteScopeRoutesUserSSHKeysToHomeRegion(t *testing.T) {
	user := &cobra.Command{Use: "user"}
	sshKey := &cobra.Command{Use: "ssh-key"}
	add := &cobra.Command{Use: "add"}
	user.AddCommand(sshKey)
	sshKey.AddCommand(add)

	if got := commandRouteScope(add); got != client.RouteScopeHomeRegion {
		t.Fatalf("commandRouteScope(user ssh-key add) = %q, want %q", got, client.RouteScopeHomeRegion)
	}
}

func TestCommandRouteScopeRoutesFunctionsToHomeRegion(t *testing.T) {
	function := &cobra.Command{Use: "function"}
	list := &cobra.Command{Use: "list"}
	function.AddCommand(list)

	if got := commandRouteScope(list); got != client.RouteScopeHomeRegion {
		t.Fatalf("commandRouteScope(function list) = %q, want %q", got, client.RouteScopeHomeRegion)
	}
}

func TestCommandRouteScopeKeepsUserProfileOnEntrypoint(t *testing.T) {
	user := &cobra.Command{Use: "user"}
	get := &cobra.Command{Use: "get"}
	user.AddCommand(get)

	if got := commandRouteScope(get); got != client.RouteScopeEntrypoint {
		t.Fatalf("commandRouteScope(user get) = %q, want %q", got, client.RouteScopeEntrypoint)
	}
}
