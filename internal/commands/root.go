package commands

import (
	"fmt"
	"os"

	"s0/internal/client"
	"s0/internal/config"
	"s0/internal/output"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/spf13/cobra"
)

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

// getConfig loads and returns the configuration.
func getConfig() (*config.Config, error) {
	return config.Load()
}

// getClient creates a new wrapped SDK client from the configuration.
func getClient() (*client.Client, error) {
	cfg, err := getConfig()
	if err != nil {
		return nil, err
	}

	// Get profile info
	activeProfile := cfg.GetActiveProfile()
	p, err := cfg.GetProfile(activeProfile)
	if err != nil {
		return nil, err
	}

	opts := []sandbox0.Option{
		sandbox0.WithBaseURL(p.GetAPIURL()),
	}

	if t := p.GetToken(); t != "" {
		opts = append(opts, sandbox0.WithToken(t))
	}

	userAgent := fmt.Sprintf("s0/%s", cfgVersion)
	if userAgent != "" {
		opts = append(opts, sandbox0.WithUserAgent(userAgent))
	}

	return client.NewClient(opts...)
}

// getClientRaw creates a raw SDK client for operations that don't need the wrapper.
func getClientRaw() (*sandbox0.Client, error) {
	cfg, err := getConfig()
	if err != nil {
		return nil, err
	}

	activeProfile := cfg.GetActiveProfile()
	p, err := cfg.GetProfile(activeProfile)
	if err != nil {
		return nil, err
	}

	opts := []sandbox0.Option{
		sandbox0.WithBaseURL(p.GetAPIURL()),
	}

	if t := p.GetToken(); t != "" {
		opts = append(opts, sandbox0.WithToken(t))
	}

	userAgent := fmt.Sprintf("s0/%s", cfgVersion)
	if userAgent != "" {
		opts = append(opts, sandbox0.WithUserAgent(userAgent))
	}

	return sandbox0.NewClient(opts...)
}
