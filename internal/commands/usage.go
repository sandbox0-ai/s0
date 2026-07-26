package commands

import (
	"fmt"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/spf13/cobra"
)

func newUsageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Inspect team usage windows",
		Long:  `List immutable, closed usage windows for the current team.`,
	}
	cmd.AddCommand(newUsageListCommand())
	return cmd
}

func newUsageListCommand() *cobra.Command {
	var cursor string
	var limit int
	var windowType string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List team usage windows",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit < 1 || limit > 1000 {
				return fmt.Errorf("limit must be between 1 and 1000")
			}

			client, err := getClientRaw(cmd)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}
			page, err := client.ListUsageWindows(
				cmd.Context(),
				&sandbox0.ListUsageWindowsOptions{
					Cursor:     cursor,
					Limit:      limit,
					WindowType: windowType,
				},
			)
			if err != nil {
				return fmt.Errorf("list team usage windows: %w", err)
			}
			return getFormatter().Format(cmd.OutOrStdout(), page)
		},
	}
	cmd.Flags().StringVar(&cursor, "cursor", "", "opaque cursor returned by a previous response")
	cmd.Flags().IntVar(&limit, "limit", 100, "maximum number of windows to return (1-1000)")
	cmd.Flags().StringVar(&windowType, "window-type", "", "filter by one exact usage window type")
	return cmd
}

func init() {
	rootCmd.AddCommand(newUsageCommand())
}
