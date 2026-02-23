package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	portsSandboxID string
	portsResume    bool
)

// sandboxPortsCmd represents the sandbox ports command group.
var sandboxPortsCmd = &cobra.Command{
	Use:   "ports",
	Short: "Manage exposed ports",
	Long:  `List, expose, unexpose, and clear exposed ports for a sandbox.`,
}

// sandboxPortsListCmd lists all exposed ports.
var sandboxPortsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List exposed ports",
	Long:  `List all exposed ports for the sandbox.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(portsSandboxID).GetExposedPorts(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting exposed ports: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxPortsExposeCmd exposes a port.
var sandboxPortsExposeCmd = &cobra.Command{
	Use:   "expose <port>",
	Short: "Expose a port",
	Long:  `Expose a port to make it publicly accessible.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		port := parseInt32(args[0], "port")

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(portsSandboxID).ExposePort(cmd.Context(), port, portsResume)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error exposing port: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxPortsUnexposeCmd unexposes a port.
var sandboxPortsUnexposeCmd = &cobra.Command{
	Use:   "unexpose <port>",
	Short: "Unexpose a port",
	Long:  `Remove a port from public access.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		port := parseInt32(args[0], "port")

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		result, err := client.Sandbox(portsSandboxID).UnexposePort(cmd.Context(), port)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error unexposing port: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

// sandboxPortsClearCmd clears all exposed ports.
var sandboxPortsClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all exposed ports",
	Long:  `Remove all exposed ports from the sandbox.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		err = client.Sandbox(portsSandboxID).ClearExposedPorts(cmd.Context())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error clearing exposed ports: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("All exposed ports cleared successfully")
	},
}

func init() {
	sandboxPortsCmd.AddCommand(sandboxPortsListCmd)
	sandboxPortsCmd.AddCommand(sandboxPortsExposeCmd)
	sandboxPortsCmd.AddCommand(sandboxPortsUnexposeCmd)
	sandboxPortsCmd.AddCommand(sandboxPortsClearCmd)

	// Sandbox ID flag (required for all subcommands)
	sandboxPortsCmd.PersistentFlags().StringVarP(&portsSandboxID, "sandbox-id", "s", "", "sandbox ID (required)")
	_ = sandboxPortsCmd.MarkPersistentFlagRequired("sandbox-id")

	// Expose command flags
	sandboxPortsExposeCmd.Flags().BoolVar(&portsResume, "resume", false, "resume sandbox on port access")

	sandboxCmd.AddCommand(sandboxPortsCmd)
}
