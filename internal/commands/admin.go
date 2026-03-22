package commands

import "github.com/spf13/cobra"

var adminCmd = newAdminCommand()

func newAdminCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Global Gateway platform administration",
		Long:  `Global Gateway and platform administration commands for system operators.`,
	}

	cmd.AddCommand(newAdminRegionCommand())
	return cmd
}

func init() {
	rootCmd.AddCommand(adminCmd)
}
