package commands

import (
	"fmt"
	"strings"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

func newQuotaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quota",
		Short: "Inspect team quotas",
		Long:  `List all quota policies and statuses for the current team, or inspect one quota dimension.`,
	}
	cmd.AddCommand(newQuotaListCommand())
	cmd.AddCommand(newQuotaGetCommand())
	return cmd
}

func newQuotaListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List team quotas",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClientRaw(cmd)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			quotas, err := client.ListTeamQuotas(cmd.Context())
			if err != nil {
				return fmt.Errorf("list team quotas: %w", err)
			}
			return getFormatter().Format(cmd.OutOrStdout(), quotas)
		},
	}
}

func newQuotaGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:       "get <dimension>",
		Short:     "Get one team quota",
		Args:      cobra.ExactArgs(1),
		ValidArgs: teamQuotaDimensionValues(),
		RunE: func(cmd *cobra.Command, args []string) error {
			dimension, err := parseTeamQuotaDimension(args[0])
			if err != nil {
				return err
			}

			client, err := getClientRaw(cmd)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			quota, err := client.GetTeamQuota(cmd.Context(), dimension)
			if err != nil {
				return fmt.Errorf("get team quota: %w", err)
			}
			return getFormatter().Format(cmd.OutOrStdout(), quota)
		},
	}
}

func parseTeamQuotaDimension(value string) (apispec.QuotaDimension, error) {
	normalized := strings.TrimSpace(value)
	for _, dimension := range apispec.QuotaDimension("").AllValues() {
		if string(dimension) == normalized {
			return dimension, nil
		}
	}
	return "", fmt.Errorf(
		"unsupported quota dimension %q (expected one of: %s)",
		value,
		strings.Join(teamQuotaDimensionValues(), ", "),
	)
}

func teamQuotaDimensionValues() []string {
	dimensions := apispec.QuotaDimension("").AllValues()
	values := make([]string, 0, len(dimensions))
	for _, dimension := range dimensions {
		values = append(values, string(dimension))
	}
	return values
}

func init() {
	rootCmd.AddCommand(newQuotaCommand())
}
