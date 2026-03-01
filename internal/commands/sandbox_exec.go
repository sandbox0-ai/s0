package commands

import (
	"fmt"
	"os"
	"strings"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/spf13/cobra"
)

var (
	execCwd    string
	execEnv    []string
	execNoWait bool
	execTTL    int32
)

// sandboxExecCmd executes a command in a sandbox.
var sandboxExecCmd = &cobra.Command{
	Use:   "exec <sandbox-id> -- <command> [args...]",
	Short: "Execute a command in a sandbox",
	Long: `Execute a command in a sandbox and wait for completion.

The command must be preceded by '--' to separate it from flags.

Examples:
  s0 sandbox exec sb_abc123 -- echo "Hello"
  s0 sandbox exec sb_abc123 --cwd /app -- python script.py
  s0 sandbox exec sb_abc123 --env FOO=bar -- ls -la`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		sandboxID := args[0]

		// Find the command after '--' separator
		// Cobra may remove '--' or keep it depending on configuration
		var command []string
		separatorIndex := -1
		for i, arg := range args {
			if arg == "--" {
				separatorIndex = i
				break
			}
		}

		if separatorIndex >= 0 {
			// '--' found in args, take everything after it
			command = args[separatorIndex+1:]
		} else {
			// '--' was consumed by Cobra, args[1:] is the command
			command = args[1:]
		}

		if len(command) == 0 {
			fmt.Fprintln(os.Stderr, "Error: command is required (use '--' followed by command)")
			os.Exit(1)
		}

		client, err := getClientRaw()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
			os.Exit(1)
		}

		// Build opts
		opts := []sandbox0.CmdOption{
			sandbox0.WithCommand(command),
		}
		if execCwd != "" {
			opts = append(opts, sandbox0.WithCmdCWD(execCwd))
		}
		if len(execEnv) > 0 {
			envMap := make(map[string]string)
			for _, e := range execEnv {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) != 2 {
					fmt.Fprintf(os.Stderr, "Error: invalid env format %q, expected KEY=VALUE\n", e)
					os.Exit(1)
				}
				envMap[parts[0]] = parts[1]
			}
			opts = append(opts, sandbox0.WithCmdEnvVars(envMap))
		}
		if execNoWait {
			opts = append(opts, sandbox0.WithCmdWait(false))
		}
		if execTTL > 0 {
			opts = append(opts, sandbox0.WithCmdTTL(execTTL))
		}

		// Pass command as string for SDK validation, WithCommand overrides the actual command array
		cmdStr := strings.Join(command, " ")
		result, err := client.Sandbox(sandboxID).Cmd(cmd.Context(), cmdStr, opts...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
			os.Exit(1)
		}

		// Output the command result
		fmt.Print(result.OutputRaw)
	},
}

func init() {
	sandboxExecCmd.Flags().StringVar(&execCwd, "cwd", "", "working directory")
	sandboxExecCmd.Flags().StringArrayVar(&execEnv, "env", nil, "environment variables (KEY=VALUE, can be repeated)")
	sandboxExecCmd.Flags().BoolVar(&execNoWait, "no-wait", false, "don't wait for command completion")
	sandboxExecCmd.Flags().Int32Var(&execTTL, "ttl", 0, "context TTL in seconds")

	sandboxCmd.AddCommand(sandboxExecCmd)
}
