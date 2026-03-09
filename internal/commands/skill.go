package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sandbox0-ai/s0/internal/skills"
	"github.com/spf13/cobra"
)

var (
	skillInstallForce  bool
	skillInstallActive bool
	skillSyncVersion   string
	skillSyncAgent     string
	skillSyncTargetDir string
	skillSyncForce     bool
)

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage Sandbox0 coding-agent skills",
	Long:  `Install, inspect, activate, and sync versioned Sandbox0 skill artifacts.`,
}

var skillInstallCmd = &cobra.Command{
	Use:   "install [skill-name]",
	Short: "Install the deployment-matched skill artifact",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := "sandbox0"
		if len(args) == 1 {
			name = args[0]
		}

		api, store, err := getSkillDependencies()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error preparing skill install: %v\n", err)
			os.Exit(1)
		}

		installed, err := store.Install(cmd.Context(), api, name, skillInstallForce, skillInstallActive)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error installing skill: %v\n", err)
			os.Exit(1)
		}

		if err := getFormatter().Format(os.Stdout, []skills.InstalledVersion{*installed}); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var skillListCmd = &cobra.Command{
	Use:   "list [skill-name]",
	Short: "List installed skill versions",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := ""
		if len(args) == 1 {
			name = args[0]
		}
		store, err := getSkillStore()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error preparing skill list: %v\n", err)
			os.Exit(1)
		}

		installed, err := store.List(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing skills: %v\n", err)
			os.Exit(1)
		}
		if err := getFormatter().Format(os.Stdout, installed); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var skillGetCmd = &cobra.Command{
	Use:   "get [skill-name]",
	Short: "Fetch deployment-matched skill metadata from the API",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := "sandbox0"
		if len(args) == 1 {
			name = args[0]
		}
		api, _, err := getSkillDependencies()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error preparing skill request: %v\n", err)
			os.Exit(1)
		}
		release, err := api.GetRelease(cmd.Context(), name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting skill metadata: %v\n", err)
			os.Exit(1)
		}
		if err := getFormatter().Format(os.Stdout, release); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var skillActivateCmd = &cobra.Command{
	Use:   "activate <skill-name> <version>",
	Short: "Set the active local skill version",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		store, err := getSkillStore()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error preparing skill activation: %v\n", err)
			os.Exit(1)
		}
		if err := store.Activate(args[0], args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "Error activating skill: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "Activated %s %s\n", args[0], args[1])
	},
}

var skillSyncCmd = &cobra.Command{
	Use:   "sync <skill-name>",
	Short: "Sync an installed skill to an agent directory",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		store, err := getSkillStore()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error preparing skill sync: %v\n", err)
			os.Exit(1)
		}
		targetDir, err := resolveSkillSyncTarget(args[0], skillSyncAgent, skillSyncTargetDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving sync target: %v\n", err)
			os.Exit(1)
		}
		if err := store.Sync(args[0], skillSyncVersion, targetDir, skillSyncForce); err != nil {
			fmt.Fprintf(os.Stderr, "Error syncing skill: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "Synced %s to %s\n", args[0], targetDir)
	},
}

func getSkillDependencies() (*skills.APIClient, *skills.Store, error) {
	profile, err := getProfileWithFreshToken()
	if err != nil {
		return nil, nil, err
	}
	store, err := getSkillStore()
	if err != nil {
		return nil, nil, err
	}
	api := skills.NewAPIClient(profile.GetAPIURL(), profile.GetToken(), fmt.Sprintf("s0/%s", cfgVersion))
	return api, store, nil
}

func getSkillStore() (*skills.Store, error) {
	rootDir, err := skills.DefaultRootDir()
	if err != nil {
		return nil, err
	}
	return skills.NewStore(rootDir), nil
}

func resolveSkillSyncTarget(name, agent, targetDir string) (string, error) {
	if strings.TrimSpace(targetDir) != "" {
		return filepath.Clean(targetDir), nil
	}
	switch strings.TrimSpace(agent) {
	case "codex":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, ".codex", "skills", name), nil
	case "":
		return "", fmt.Errorf("either --agent or --target-dir is required")
	default:
		return "", fmt.Errorf("unsupported agent %q", agent)
	}
}

func init() {
	rootCmd.AddCommand(skillCmd)

	skillInstallCmd.Flags().BoolVar(&skillInstallForce, "force", false, "reinstall even when the same version is already stored locally")
	skillInstallCmd.Flags().BoolVar(&skillInstallActive, "activate", true, "mark the installed version as active")

	skillSyncCmd.Flags().StringVar(&skillSyncVersion, "version", "", "installed version to sync (defaults to the active version)")
	skillSyncCmd.Flags().StringVar(&skillSyncAgent, "agent", "", "agent preset to sync to (supported: codex)")
	skillSyncCmd.Flags().StringVar(&skillSyncTargetDir, "target-dir", "", "explicit target directory for the synced skill")
	skillSyncCmd.Flags().BoolVar(&skillSyncForce, "force", false, "replace an existing target directory")

	skillCmd.AddCommand(skillInstallCmd)
	skillCmd.AddCommand(skillListCmd)
	skillCmd.AddCommand(skillGetCmd)
	skillCmd.AddCommand(skillActivateCmd)
	skillCmd.AddCommand(skillSyncCmd)
}
