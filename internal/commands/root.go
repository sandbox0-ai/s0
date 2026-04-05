package commands

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/sandbox0-ai/s0/internal/client"
	"github.com/sandbox0-ai/s0/internal/config"
	"github.com/sandbox0-ai/s0/internal/output"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/spf13/cobra"
)

// ErrNoToken is returned when no API token is configured.
var ErrNoToken = errors.New("no API token configured. Set SANDBOX0_TOKEN environment variable, use --token flag, or run `s0 auth login`")

var (
	cfgVersion string
	cfgFormat  string
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "s0",
	Short: "Sandbox0 CLI - Manage sandboxes, templates, volumes, and images",
	Long: `s0 is the command-line interface for Sandbox0.

It provides comprehensive management of sandboxes, templates, volumes,
snapshots, and container images.`,
}

// Execute runs the root command.
func Execute(version string) {
	cfgVersion = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(config.GetConfigFile(), "config", "c", "", "config file (default is ~/.s0/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&cfgFormat, "output", "o", "table", "output format (table, json, yaml)")
	rootCmd.PersistentFlags().StringVarP(config.GetProfileVar(), "profile", "p", "", "profile name (default is \"default\")")
	rootCmd.PersistentFlags().StringVar(config.GetAPIURLVar(), "api-url", "", "override API URL")
	rootCmd.PersistentFlags().StringVar(config.GetTokenVar(), "token", "", "override API token")
}

// getFormatter returns the output formatter based on the format flag.
func getFormatter() output.Formatter {
	return output.NewFormatter(output.ParseFormat(cfgFormat))
}

// getFormatterWithOptions returns formatter with custom options.
func getFormatterWithOptions(opts output.Options) output.Formatter {
	return output.NewFormatterWithOptions(output.ParseFormat(cfgFormat), opts)
}

// getConfig loads and returns the configuration.
func getConfig() (*config.Config, error) {
	return config.Load()
}

// getClient creates a new wrapped SDK client from the configuration.
func getClient(cmd *cobra.Command) (*client.Client, error) {
	resolved, userAgent, err := resolveClientTarget(cmd)
	if err != nil {
		return nil, err
	}

	opts := []sandbox0.Option{
		sandbox0.WithBaseURL(resolved.BaseURL),
		sandbox0.WithToken(resolved.Token),
	}
	if userAgent != "" {
		opts = append(opts, sandbox0.WithUserAgent(userAgent))
	}

	return client.NewClient(opts...)
}

// getClientRaw creates a raw SDK client for operations that don't need the wrapper.
func getClientRaw(cmd *cobra.Command) (*sandbox0.Client, error) {
	resolved, userAgent, err := resolveClientTarget(cmd)
	if err != nil {
		return nil, err
	}

	opts := []sandbox0.Option{
		sandbox0.WithBaseURL(resolved.BaseURL),
		sandbox0.WithToken(resolved.Token),
	}
	if userAgent != "" {
		opts = append(opts, sandbox0.WithUserAgent(userAgent))
	}

	return sandbox0.NewClient(opts...)
}

func resolveClientTarget(cmd *cobra.Command) (*client.ResolvedTarget, string, error) {
	p, err := getProfileWithFreshToken()
	if err != nil {
		return nil, "", err
	}

	token := p.GetToken()
	if token == "" {
		return nil, "", ErrNoToken
	}

	var configuredMode config.GatewayMode
	if mode, ok := p.GetConfiguredGatewayMode(); ok {
		configuredMode = mode
	}

	resolved, err := client.ResolveTarget(
		cmd.Context(),
		client.ResolveTargetOptions{
			BaseURL:               p.GetAPIURL(),
			Token:                 token,
			ConfiguredGatewayMode: configuredMode,
			CurrentTeamID:         p.GetCurrentTeamID(),
			RegionalSession: func() *config.RegionalSession {
				if session, ok := p.GetRegionalSession(p.GetCurrentTeamID()); ok {
					return session
				}
				return nil
			}(),
			Scope:     commandRouteScope(cmd),
			UserAgent: buildUserAgent(),
		},
	)
	if err != nil {
		return nil, "", err
	}

	return resolved, buildUserAgent(), nil
}

func buildUserAgent() string {
	if cfgVersion == "" {
		return ""
	}
	return fmt.Sprintf("s0/%s", cfgVersion)
}

func commandRouteScope(cmd *cobra.Command) client.RouteScope {
	for current := cmd; current != nil; current = current.Parent() {
		switch current.Name() {
		case "sandbox", "template", "volume", "sync", "credential", "apikey", "image":
			return client.RouteScopeHomeRegion
		case "auth", "team", "user":
			return client.RouteScopeEntrypoint
		}
	}
	return client.RouteScopeEntrypoint
}

// parseInt32 parses a string to int32 with error handling.
func parseInt32(s string, name string) int32 {
	val, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid %s %q: %v\n", name, s, err)
		os.Exit(1)
	}
	return int32(val)
}
