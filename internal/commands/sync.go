package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	pathpkg "path"
	"path/filepath"
	"strings"
	"time"

	"github.com/sandbox0-ai/s0/internal/config"
	"github.com/sandbox0-ai/s0/internal/output"
	"github.com/sandbox0-ai/s0/internal/syncapi"
	"github.com/sandbox0-ai/s0/internal/syncstate"
	"github.com/sandbox0-ai/s0/internal/syncview"
	"github.com/sandbox0-ai/s0/internal/syncworker"
	syncsdk "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
	"github.com/spf13/cobra"
)

var (
	syncAttachForeground bool
	syncAttachInitFrom   string
	syncAttachName       string

	syncDetachPurgeState bool
	syncDetachForce      bool

	syncLogsFollow bool

	syncConflictsIgnore bool

	syncWorkerAttachmentID string
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Manage local-first workspace sync attachments",
	Long:  `Attach a local workspace to a volume and manage the local sync worker lifecycle.`,
}

var syncAttachCmd = &cobra.Command{
	Use:   "attach <volume-id> [path]",
	Short: "Attach the current or specified workspace to a volume",
	Long:  `Attach a local workspace to a volume and start the local sync worker in the foreground or background.`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		volumeID := strings.TrimSpace(args[0])
		workspacePath := "."
		if len(args) == 2 {
			workspacePath = args[1]
		}

		attachment, err := upsertAttachment(workspacePath, volumeID, syncAttachName, syncAttachInitFrom)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error preparing attachment: %v\n", err)
			os.Exit(1)
		}

		status := syncstate.EffectiveStatus(attachment)
		if status == "running" {
			if syncAttachForeground {
				fmt.Fprintf(os.Stderr, "Sync worker is already running for %s\n", attachment.WorkspaceRoot)
				os.Exit(1)
			}
			if err := renderSyncOutput(os.Stdout, attachment); err != nil {
				fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
				os.Exit(1)
			}
			return
		}
		if needsWorkerRecovery(status) {
			attachment, err = syncstate.ResetWorkerState(attachment.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error preparing local worker recovery: %v\n", err)
				os.Exit(1)
			}
		}

		if syncAttachForeground {
			fmt.Fprintf(os.Stdout, "Attached %s to %s. Running sync worker in foreground. Press Ctrl+C to stop.\n", attachment.WorkspaceRoot, attachment.VolumeID)
			client, err := getClientRaw(cmd)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error preparing API client: %v\n", err)
				os.Exit(1)
			}
			if err := runForegroundSyncWorker(cmd.Context(), client, attachment.ID); err != nil {
				fmt.Fprintf(os.Stderr, "Error running sync worker: %v\n", err)
				os.Exit(1)
			}
			return
		}

		if err := startBackgroundSyncWorker(attachment.ID); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting background sync worker: %v\n", err)
			os.Exit(1)
		}

		attachment, err = syncstate.LoadAttachmentByID(attachment.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reloading attachment: %v\n", err)
			os.Exit(1)
		}
		if err := renderSyncOutput(os.Stdout, attachment); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var syncDetachCmd = &cobra.Command{
	Use:   "detach [path|volume-id]",
	Short: "Detach a workspace from sync",
	Long:  `Stop the local sync worker and remove the local attachment for the current or specified workspace.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		attachment, err := resolveSyncAttachment(cmd, args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving attachment: %v\n", err)
			os.Exit(1)
		}

		if err := stopAttachmentWorker(attachment, syncDetachForce); err != nil {
			fmt.Fprintf(os.Stderr, "Error stopping sync worker: %v\n", err)
			os.Exit(1)
		}
		if err := syncstate.DeleteAttachment(attachment.ID); err != nil {
			fmt.Fprintf(os.Stderr, "Error removing attachment: %v\n", err)
			os.Exit(1)
		}
		if syncDetachPurgeState {
			_ = os.Remove(attachment.Worker.LogPath)
		}

		fmt.Fprintf(os.Stdout, "Detached %s from volume %s\n", attachment.WorkspaceRoot, attachment.VolumeID)
	},
}

var syncListCmd = &cobra.Command{
	Use:   "list",
	Short: "List local sync attachments",
	Long:  `List all local workspaces currently attached for sync on this machine.`,
	Run: func(cmd *cobra.Command, args []string) {
		attachments, err := syncstate.ListAttachments()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing attachments: %v\n", err)
			os.Exit(1)
		}
		if err := renderSyncOutput(os.Stdout, attachments); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var syncStatusCmd = &cobra.Command{
	Use:   "status [path|volume-id]",
	Short: "Show sync status for the current or specified workspace",
	Long:  `Show the local attachment state, worker status, and persisted sync checkpoint metadata.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		attachment, err := resolveSyncAttachment(cmd, args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving attachment: %v\n", err)
			os.Exit(1)
		}

		view := buildStatusView(cmd, attachment)
		if err := renderSyncOutput(os.Stdout, view); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var syncLogsCmd = &cobra.Command{
	Use:   "logs [path|volume-id]",
	Short: "Read sync worker logs",
	Long:  `Read the log output for the current or specified sync attachment.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		attachment, err := resolveSyncAttachment(cmd, args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving attachment: %v\n", err)
			os.Exit(1)
		}
		if err := printAttachmentLogs(cmd.Context(), attachment, syncLogsFollow); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading logs: %v\n", err)
			os.Exit(1)
		}
	},
}

var syncConflictsCmd = &cobra.Command{
	Use:   "conflicts",
	Short: "Inspect and resolve sync conflicts",
	Long:  `List and resolve persisted sync conflicts for the current workspace or attached volume.`,
}

var syncConflictsListCmd = &cobra.Command{
	Use:   "list [path|volume-id]",
	Short: "List unresolved sync conflicts",
	Long:  `List unresolved sync conflicts for the current workspace or attached volume.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		attachment, err := resolveSyncAttachment(cmd, args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving attachment: %v\n", err)
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error preparing API client: %v\n", err)
			os.Exit(1)
		}
		conflicts, err := syncapi.New(client).ListConflicts(cmd.Context(), attachment.VolumeID, "open", 256)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing conflicts: %v\n", err)
			os.Exit(1)
		}
		view := syncview.BuildConflictListView(attachment, conflicts, 0)
		if err := renderSyncOutput(os.Stdout, view); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var syncConflictsShowCmd = &cobra.Command{
	Use:   "show <path>",
	Short: "Show one conflict path",
	Long:  `Show one unresolved sync conflict addressed by its workspace-relative path.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		attachment, err := syncstate.ResolveAttachmentFromPath(mustGetwd())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving current workspace: %v\n", err)
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error preparing API client: %v\n", err)
			os.Exit(1)
		}
		conflictPath, err := resolveWorkspaceRelativePath(attachment, args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving conflict path: %v\n", err)
			os.Exit(1)
		}
		conflict, err := resolveConflictByPath(cmd.Context(), syncapi.New(client), attachment, conflictPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error showing conflict: %v\n", err)
			os.Exit(1)
		}
		view := syncview.BuildConflictDetailView(conflict)
		if err := renderSyncOutput(os.Stdout, view); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var syncConflictsMarkCmd = &cobra.Command{
	Use:   "mark <path>",
	Short: "Mark one conflict as resolved or ignored",
	Long:  `Mark one unresolved sync conflict as resolved or ignored after local repair.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		attachment, err := syncstate.ResolveAttachmentFromPath(mustGetwd())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving current workspace: %v\n", err)
			os.Exit(1)
		}

		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error preparing API client: %v\n", err)
			os.Exit(1)
		}
		api := syncapi.New(client)
		conflictPath, err := resolveWorkspaceRelativePath(attachment, args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving conflict path: %v\n", err)
			os.Exit(1)
		}
		conflict, err := resolveConflictByPath(cmd.Context(), api, attachment, conflictPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving conflict path: %v\n", err)
			os.Exit(1)
		}
		conflictID, ok := conflict.ID.Get()
		if !ok || strings.TrimSpace(conflictID) == "" {
			fmt.Fprintln(os.Stderr, "Error resolving conflict path: conflict id is missing")
			os.Exit(1)
		}
		resolved, err := api.ResolveConflict(cmd.Context(), attachment.VolumeID, conflictID, syncConflictsIgnore)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating conflict: %v\n", err)
			os.Exit(1)
		}
		refreshAttachmentConflictCount(cmd.Context(), api, attachment.ID, attachment.VolumeID)
		view := syncview.BuildConflictDetailView(resolved)
		if err := renderSyncOutput(os.Stdout, view); err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}
	},
}

var syncWorkerCmd = &cobra.Command{
	Use:    "__sync-worker",
	Short:  "Run the internal sync worker",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		if strings.TrimSpace(syncWorkerAttachmentID) == "" {
			fmt.Fprintln(os.Stderr, "missing attachment id")
			os.Exit(1)
		}
		client, err := getClientRaw(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error preparing API client: %v\n", err)
			os.Exit(1)
		}
		if err := runBackgroundSyncWorker(cmd.Context(), client, syncWorkerAttachmentID); err != nil {
			fmt.Fprintf(os.Stderr, "Error running sync worker: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(syncWorkerCmd)

	syncCmd.AddCommand(syncAttachCmd)
	syncCmd.AddCommand(syncDetachCmd)
	syncCmd.AddCommand(syncListCmd)
	syncCmd.AddCommand(syncStatusCmd)
	syncCmd.AddCommand(syncLogsCmd)
	syncCmd.AddCommand(syncConflictsCmd)

	syncConflictsCmd.AddCommand(syncConflictsListCmd)
	syncConflictsCmd.AddCommand(syncConflictsShowCmd)
	syncConflictsCmd.AddCommand(syncConflictsMarkCmd)

	syncAttachCmd.Flags().BoolVar(&syncAttachForeground, "foreground", false, "run the sync worker in the foreground")
	syncAttachCmd.Flags().StringVar(&syncAttachInitFrom, "init-from", "auto", "bootstrap policy (auto, volume, local)")
	syncAttachCmd.Flags().StringVar(&syncAttachName, "name", "", "human-readable local workspace name")

	syncDetachCmd.Flags().BoolVar(&syncDetachPurgeState, "purge-state", false, "remove local log files while detaching")
	syncDetachCmd.Flags().BoolVar(&syncDetachForce, "force", false, "forcefully stop a worker even when the local heartbeat is stale")

	syncLogsCmd.Flags().BoolVarP(&syncLogsFollow, "follow", "f", false, "follow log output")

	syncConflictsMarkCmd.Flags().BoolVar(&syncConflictsIgnore, "ignore", false, "mark the conflict placeholder as ignored instead of resolved")

	syncWorkerCmd.Flags().StringVar(&syncWorkerAttachmentID, "attachment-id", "", "internal attachment id")
	_ = syncWorkerCmd.MarkFlagRequired("attachment-id")
}

func upsertAttachment(workspacePath, volumeID, displayName, initFrom string) (*syncstate.Attachment, error) {
	workspaceRoot, err := filepath.Abs(workspacePath)
	if err != nil {
		return nil, err
	}
	existing, err := syncstate.ResolveAttachmentFromPath(workspaceRoot)
	if err == nil && filepath.Clean(existing.WorkspaceRoot) == filepath.Clean(workspaceRoot) {
		if existing.VolumeID != strings.TrimSpace(volumeID) {
			return nil, fmt.Errorf("workspace %s is already attached to volume %s", existing.WorkspaceRoot, existing.VolumeID)
		}
		existing.InitFrom = strings.TrimSpace(initFrom)
		if strings.TrimSpace(displayName) != "" {
			existing.DisplayName = strings.TrimSpace(displayName)
		}
		if err := syncstate.SaveAttachment(existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	attachment, err := syncstate.NewAttachment(workspaceRoot, volumeID, displayName, initFrom)
	if err != nil {
		return nil, err
	}
	if err := syncstate.SaveAttachment(attachment); err != nil {
		return nil, err
	}
	return attachment, nil
}

func resolveSyncAttachment(cmd *cobra.Command, args []string) (*syncstate.Attachment, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	target := ""
	if len(args) > 0 {
		target = args[0]
	}
	return syncstate.ResolveAttachmentTarget(target, cwd)
}

func renderSyncOutput(w io.Writer, data any) error {
	if output.ParseFormat(cfgFormat) == output.FormatTable {
		return getFormatter().Format(w, data)
	}
	return getFormatter().Format(w, data)
}

func runForegroundSyncWorker(parent context.Context, client *syncsdk.Client, attachmentID string) error {
	ctx, cancel := signal.NotifyContext(parent, forwardingSignals()...)
	defer cancel()
	return syncworker.Run(ctx, client, attachmentID, "foreground", os.Stdout)
}

func runBackgroundSyncWorker(parent context.Context, client *syncsdk.Client, attachmentID string) error {
	attachment, err := syncstate.LoadAttachmentByID(attachmentID)
	if err != nil {
		return err
	}
	if err := syncstate.EnsureLogDir(); err != nil {
		return err
	}
	logFile, err := os.OpenFile(attachment.Worker.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer logFile.Close()

	ctx, cancel := signal.NotifyContext(parent, forwardingSignals()...)
	defer cancel()
	return syncworker.Run(ctx, client, attachmentID, "background", logFile)
}

func startBackgroundSyncWorker(attachmentID string) error {
	attachment, err := syncstate.LoadAttachmentByID(attachmentID)
	if err != nil {
		return err
	}
	if err := syncstate.EnsureLogDir(); err != nil {
		return err
	}

	logFile, err := os.OpenFile(attachment.Worker.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer logFile.Close()

	executable, err := os.Executable()
	if err != nil {
		return err
	}
	args := backgroundWorkerArgs(attachmentID)
	command := exec.Command(executable, args...)
	command.Stdin = nil
	command.Stdout = logFile
	command.Stderr = logFile
	command.Env = backgroundWorkerEnv()

	if err := command.Start(); err != nil {
		return err
	}
	pid := command.Process.Pid
	if err := command.Process.Release(); err != nil {
		return err
	}

	_, err = syncstate.TouchWorkerHeartbeat(attachmentID, "background", pid)
	return err
}

func stopAttachmentWorker(attachment *syncstate.Attachment, force bool) error {
	if attachment == nil {
		return errors.New("attachment is nil")
	}
	if attachment.Worker.PID == 0 {
		return nil
	}
	if syncstate.EffectiveStatus(attachment) == "stale" && !force {
		return fmt.Errorf("worker heartbeat is stale; rerun with --force to detach anyway")
	}

	process, err := os.FindProcess(attachment.Worker.PID)
	if err == nil {
		_ = process.Kill()
	}
	attachment.Worker.Status = "stopped"
	attachment.Worker.PID = 0
	attachment.Worker.LastStoppedAt = ptrTime(time.Now().UTC())
	return syncstate.SaveAttachment(attachment)
}

func printAttachmentLogs(ctx context.Context, attachment *syncstate.Attachment, follow bool) error {
	file, err := os.Open(attachment.Worker.LogPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer file.Close()

	if _, err := io.Copy(os.Stdout, file); err != nil {
		return err
	}
	if !follow {
		return nil
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	buffer := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			for {
				n, err := file.Read(buffer)
				if n > 0 {
					if _, writeErr := os.Stdout.Write(buffer[:n]); writeErr != nil {
						return writeErr
					}
				}
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return err
				}
				if n == 0 {
					break
				}
			}
		}
	}
}

func mustGetwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func backgroundWorkerArgs(attachmentID string) []string {
	args := make([]string, 0, 10)
	if configFile := strings.TrimSpace(*config.GetConfigFile()); configFile != "" {
		args = append(args, "--config", configFile)
	}
	if profile := strings.TrimSpace(*config.GetProfileVar()); profile != "" {
		args = append(args, "--profile", profile)
	}
	if apiURL := strings.TrimSpace(*config.GetAPIURLVar()); apiURL != "" {
		args = append(args, "--api-url", apiURL)
	}
	args = append(args, "__sync-worker", "--attachment-id", attachmentID)
	return args
}

func backgroundWorkerEnv() []string {
	env := os.Environ()
	if token := strings.TrimSpace(*config.GetTokenVar()); token != "" {
		env = append(env, config.EnvToken+"="+token)
	}
	if apiURL := strings.TrimSpace(*config.GetAPIURLVar()); apiURL != "" {
		env = append(env, config.EnvBaseURL+"="+apiURL)
	}
	return env
}

func needsWorkerRecovery(status string) bool {
	switch strings.TrimSpace(status) {
	case "stale":
		return true
	default:
		return false
	}
}

func buildStatusView(cmd *cobra.Command, attachment *syncstate.Attachment) *syncview.StatusView {
	if attachment == nil {
		return syncview.BuildStatusView(nil, nil, nil)
	}
	client, err := getClientRaw(cmd)
	if err != nil {
		return syncview.BuildStatusView(attachment, nil, err)
	}
	conflicts, err := syncapi.New(client).ListConflicts(cmd.Context(), attachment.VolumeID, "open", 256)
	return syncview.BuildStatusView(attachment, conflicts, err)
}

func refreshAttachmentConflictCount(ctx context.Context, api *syncapi.Client, attachmentID, volumeID string) {
	conflicts, err := api.ListConflicts(ctx, volumeID, "open", 256)
	if err != nil {
		return
	}
	_, _ = syncstate.UpdateAttachment(attachmentID, func(attachment *syncstate.Attachment) error {
		if attachment.LastSync == nil {
			attachment.LastSync = &syncstate.SyncCheckpoint{}
		}
		attachment.LastSync.OpenConflictCount = len(conflicts)
		return nil
	})
}

func resolveConflictByPath(ctx context.Context, api *syncapi.Client, attachment *syncstate.Attachment, relativePath string) (*apispec.SyncConflict, error) {
	conflicts, err := api.ListConflicts(ctx, attachment.VolumeID, "open", 256)
	if err != nil {
		return nil, err
	}
	candidate := normalizeConflictPath(relativePath)
	for _, conflict := range conflicts {
		if conflictMatchesPath(conflict, candidate) {
			return &conflict, nil
		}
	}
	return nil, fmt.Errorf("no open conflict found for %q", candidate)
}

func conflictMatchesPath(conflict apispec.SyncConflict, candidate string) bool {
	for _, path := range []string{
		optString(conflict.Path),
		optNilString(conflict.IncomingPath),
		optNilString(conflict.IncomingOldPath),
	} {
		if normalizeConflictPath(path) == candidate {
			return true
		}
	}
	return false
}

func resolveWorkspaceRelativePath(attachment *syncstate.Attachment, value string) (string, error) {
	if attachment == nil {
		return "", fmt.Errorf("attachment is nil")
	}
	candidate := strings.TrimSpace(value)
	if candidate == "" {
		return "", fmt.Errorf("path is empty")
	}
	if strings.HasPrefix(candidate, "/") && !isAncestorPath(attachment.WorkspaceRoot, filepath.Clean(candidate)) {
		logical := normalizeConflictPath(candidate)
		if logical == "" {
			return "", fmt.Errorf("%q is outside workspace %s", value, attachment.WorkspaceRoot)
		}
		return logical, nil
	}
	absolute := candidate
	if !filepath.IsAbs(absolute) {
		absolute = filepath.Join(mustGetwd(), candidate)
	}
	relative, err := filepath.Rel(attachment.WorkspaceRoot, absolute)
	if err != nil {
		return "", err
	}
	relative = filepath.ToSlash(filepath.Clean(relative))
	if relative == "." || strings.HasPrefix(relative, "../") {
		return "", fmt.Errorf("%q is outside workspace %s", value, attachment.WorkspaceRoot)
	}
	return relative, nil
}

func normalizeConflictPath(path string) string {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" {
		return ""
	}
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "./")
	path = pathpkg.Clean(path)
	if path == "." || path == "/" {
		return ""
	}
	return filepath.ToSlash(path)
}

func isAncestorPath(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	if root == path {
		return true
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func optString(value apispec.OptString) string {
	v, ok := value.Get()
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

func optNilString(value apispec.OptNilString) string {
	v, ok := value.Get()
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}
