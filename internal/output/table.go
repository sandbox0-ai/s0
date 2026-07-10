package output

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sandbox0-ai/s0/internal/client"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

const timeLayout = "2006-01-02 15:04:05"

// TableFormatter formats output as a table.
type TableFormatter struct {
	showSecrets bool
}

// Format writes the data as a table to the writer.
func (f *TableFormatter) Format(w io.Writer, data interface{}) error {
	switch v := data.(type) {
	case []apispec.Template:
		return f.formatTemplates(w, v)
	case *apispec.Template:
		return f.formatTemplate(w, v)
	case []apispec.SandboxVolume:
		return f.formatVolumes(w, v)
	case *apispec.SandboxVolume:
		return f.formatVolume(w, v)
	case []apispec.Snapshot:
		return f.formatSnapshots(w, v)
	case *apispec.Snapshot:
		return f.formatSnapshot(w, v)
	case []apispec.SandboxRootFSSnapshot:
		return f.formatSandboxRootFSSnapshots(w, v)
	case *apispec.SandboxRootFSSnapshotList:
		return f.formatSandboxRootFSSnapshotList(w, v)
	case *apispec.SandboxRootFSSnapshot:
		return f.formatSandboxRootFSSnapshot(w, v)
	case *apispec.RestoreSandboxRootFSResponse:
		return f.formatRestoreSandboxRootFSResponse(w, v)
	case *apispec.ForkSandboxResponse:
		return f.formatForkSandboxResponse(w, v)
	case *apispec.Sandbox:
		return f.formatSandbox(w, v)
	case *apispec.SandboxStatus:
		return f.formatSandboxStatus(w, v)
	case *apispec.RefreshResponse:
		return f.formatRefreshResponse(w, v)
	case *apispec.SuccessMessageResponse:
		return f.formatSuccessMessage(w, v)
	case *apispec.SuccessDeletedResponse:
		return f.formatSuccessDeleted(w, v)
	case *sandbox0.ListSandboxesResponse:
		return f.formatSandboxList(w, v)
	case *sandbox0.Sandbox:
		return f.formatSDKSandbox(w, v)
	case *client.RegistryCredentials:
		return f.formatRegistryCredentials(w, v)
	case []apispec.CredentialSourceMetadata:
		return f.formatCredentialSourceList(w, v)
	case apispec.CredentialSourceMetadata:
		return f.formatCredentialSource(w, &v)
	case *apispec.CredentialSourceMetadata:
		return f.formatCredentialSource(w, v)
	case []apispec.FileInfo:
		return f.formatFileList(w, v)
	case *apispec.FileInfo:
		return f.formatFileInfo(w, v)
	case []apispec.ContextResponse:
		return f.formatContextList(w, v)
	case *apispec.ContextResponse:
		return f.formatContext(w, v)
	case []apispec.ExecutionSession:
		return f.formatExecutionSessionList(w, v)
	case *apispec.ExecutionSession:
		return f.formatExecutionSession(w, v)
	case *apispec.ExecutionSessionEventPage:
		return f.formatExecutionSessionEvents(w, v)
	case *apispec.ExecutionSessionInputResponse:
		return f.formatExecutionSessionInput(w, v)
	case *apispec.SandboxObservabilityEventsResponse:
		return f.formatSandboxObservabilityEvents(w, v)
	case *apispec.SandboxObservabilityLogsResponse:
		return f.formatSandboxObservabilityLogs(w, v)
	case *apispec.SandboxRuntimeMetricsResponse:
		return f.formatSandboxRuntimeMetrics(w, v)
	case *apispec.SandboxNetworkPolicy:
		return f.formatSandboxNetworkPolicy(w, v)
	case *sandbox0.SandboxServicesResponse:
		return f.formatSandboxServices(w, v)
	case []apispec.MountStatus:
		return f.formatMountStatusList(w, v)
	case []apispec.APIKey:
		return f.formatAPIKeyList(w, v)
	case apispec.CreateAPIKeyResponse:
		return f.formatCreatedAPIKey(w, &v)
	case *apispec.CreateAPIKeyResponse:
		return f.formatCreatedAPIKey(w, v)
	case []apispec.Team:
		return f.formatTeamList(w, v)
	case TeamList:
		return f.formatTeamListWithCurrent(w, v)
	case apispec.Team:
		return f.formatTeam(w, &v)
	case *apispec.Team:
		return f.formatTeam(w, v)
	case []apispec.Region:
		return f.formatRegionList(w, v)
	case apispec.Region:
		return f.formatRegion(w, &v)
	case *apispec.Region:
		return f.formatRegion(w, v)
	case []apispec.TeamMember:
		return f.formatTeamMemberList(w, v)
	case apispec.TeamMember:
		return f.formatTeamMember(w, &v)
	case *apispec.TeamMember:
		return f.formatTeamMember(w, v)
	case apispec.User:
		return f.formatUser(w, &v)
	case *apispec.User:
		return f.formatUser(w, v)
	case []apispec.SSHPublicKey:
		return f.formatSSHPublicKeyList(w, v)
	case apispec.SSHPublicKey:
		return f.formatSSHPublicKey(w, &v)
	case *apispec.SSHPublicKey:
		return f.formatSSHPublicKey(w, v)
	case string:
		_, _ = fmt.Fprintln(w, v)
		return nil
	default:
		_, _ = fmt.Fprintf(w, "%v\n", data)
		return nil
	}
}

func newTable(w io.Writer) *tablewriter.Table {
	return tablewriter.NewTable(w, tablewriter.WithRendition(tw.Rendition{
		Borders: tw.Border{Left: tw.Off, Right: tw.Off, Top: tw.Off, Bottom: tw.Off},
		Settings: tw.Settings{
			Lines: tw.Lines{
				ShowTop:        tw.Off,
				ShowBottom:     tw.Off,
				ShowHeaderLine: tw.Off,
				ShowFooterLine: tw.Off,
			},
			Separators: tw.Separators{
				ShowHeader:     tw.Off,
				ShowFooter:     tw.Off,
				BetweenRows:    tw.Off,
				BetweenColumns: tw.Off,
			},
		},
	}))
}

func (f *TableFormatter) formatTemplates(w io.Writer, templates []apispec.Template) error {
	if len(templates) == 0 {
		_, _ = fmt.Fprintln(w, "No templates found.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"TEMPLATE ID", "SCOPE", "CREATED AT"})

	for _, tmpl := range templates {
		_ = t.Append([]string{
			tmpl.TemplateID,
			tmpl.Scope,
			tmpl.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return t.Render()
}

func (f *TableFormatter) formatTemplate(w io.Writer, tmpl *apispec.Template) error {
	t := newTable(w)
	_ = t.Append([]string{"Template ID:", tmpl.TemplateID})
	_ = t.Append([]string{"Scope:", tmpl.Scope})
	if v, ok := tmpl.TeamID.Get(); ok {
		_ = t.Append([]string{"Team ID:", v})
	}
	if v, ok := tmpl.UserID.Get(); ok {
		_ = t.Append([]string{"User ID:", v})
	}
	_ = t.Append([]string{"Created:", tmpl.CreatedAt.Format("2006-01-02 15:04:05")})
	_ = t.Append([]string{"Updated:", tmpl.UpdatedAt.Format("2006-01-02 15:04:05")})
	return t.Render()
}

func (f *TableFormatter) formatVolumes(w io.Writer, volumes []apispec.SandboxVolume) error {
	if len(volumes) == 0 {
		_, _ = fmt.Fprintln(w, "No volumes found.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"ID", "TEAM ID", "BACKEND", "CREATED"})

	for _, v := range volumes {
		_ = t.Append([]string{
			v.ID,
			v.TeamID,
			formatVolumeBackend(v.Backend),
			v.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return t.Render()
}

func (f *TableFormatter) formatVolume(w io.Writer, v *apispec.SandboxVolume) error {
	t := newTable(w)
	_ = t.Append([]string{"ID:", v.ID})
	_ = t.Append([]string{"Team ID:", v.TeamID})
	_ = t.Append([]string{"User ID:", v.UserID})
	_ = t.Append([]string{"Backend:", formatVolumeBackend(v.Backend)})
	if s3, ok := v.S3.Get(); ok {
		_ = t.Append([]string{"S3 Provider:", string(s3.Provider)})
		_ = t.Append([]string{"S3 Bucket:", s3.Bucket})
		if prefix, ok := s3.Prefix.Get(); ok {
			_ = t.Append([]string{"S3 Prefix:", prefix})
		}
		if region, ok := s3.Region.Get(); ok {
			_ = t.Append([]string{"S3 Region:", region})
		}
		if endpointURL, ok := s3.EndpointURL.Get(); ok {
			_ = t.Append([]string{"S3 Endpoint URL:", endpointURL})
		}
	}
	_ = t.Append([]string{"Created:", v.CreatedAt.Format("2006-01-02 15:04:05")})
	_ = t.Append([]string{"Updated:", v.UpdatedAt.Format("2006-01-02 15:04:05")})
	return t.Render()
}

func formatVolumeBackend(backend apispec.VolumeBackend) string {
	if backend == "" {
		return "-"
	}
	return string(backend)
}

func (f *TableFormatter) formatSnapshots(w io.Writer, snapshots []apispec.Snapshot) error {
	if len(snapshots) == 0 {
		_, _ = fmt.Fprintln(w, "No snapshots found.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"ID", "NAME", "SIZE", "CREATED"})

	for _, s := range snapshots {
		name := s.Name
		if name == "" {
			name = "-"
		}
		_ = t.Append([]string{
			s.ID,
			name,
			fmt.Sprintf("%d bytes", s.SizeBytes),
			s.CreatedAt,
		})
	}
	return t.Render()
}

func valueOrDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func intOrDash(value int) string {
	if value == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", value)
}

func optInt32OrDash(value apispec.OptInt32) string {
	if !value.Set || value.Value == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", value.Value)
}

func (f *TableFormatter) formatSnapshot(w io.Writer, s *apispec.Snapshot) error {
	t := newTable(w)
	_ = t.Append([]string{"ID:", s.ID})
	_ = t.Append([]string{"Volume ID:", s.VolumeID})
	_ = t.Append([]string{"Name:", s.Name})
	if v, ok := s.Description.Get(); ok {
		_ = t.Append([]string{"Description:", v})
	}
	_ = t.Append([]string{"Size:", fmt.Sprintf("%d bytes", s.SizeBytes)})
	_ = t.Append([]string{"Created:", s.CreatedAt})
	return t.Render()
}

func (f *TableFormatter) formatSandboxRootFSSnapshotList(w io.Writer, snapshots *apispec.SandboxRootFSSnapshotList) error {
	if snapshots == nil {
		_, _ = fmt.Fprintln(w, "No sandbox rootfs snapshots found.")
		return nil
	}
	if err := f.formatSandboxRootFSSnapshots(w, snapshots.Snapshots); err != nil {
		return err
	}
	if len(snapshots.Snapshots) > 0 {
		_, _ = fmt.Fprintf(w, "Total: %d\n", snapshots.Count)
	}
	return nil
}

func (f *TableFormatter) formatSandboxRootFSSnapshots(w io.Writer, snapshots []apispec.SandboxRootFSSnapshot) error {
	if len(snapshots) == 0 {
		_, _ = fmt.Fprintln(w, "No sandbox rootfs snapshots found.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"ID", "SANDBOX ID", "NAME", "CREATED", "EXPIRES"})
	for _, s := range snapshots {
		_ = t.Append([]string{
			s.ID,
			s.SandboxID,
			formatOptString(s.Name),
			formatTimestamp(s.CreatedAt),
			formatOptDateTime(s.ExpiresAt),
		})
	}
	return t.Render()
}

func (f *TableFormatter) formatSandboxRootFSSnapshot(w io.Writer, s *apispec.SandboxRootFSSnapshot) error {
	t := newTable(w)
	_ = t.Append([]string{"ID:", s.ID})
	_ = t.Append([]string{"Sandbox ID:", s.SandboxID})
	_ = t.Append([]string{"Name:", formatOptString(s.Name)})
	_ = t.Append([]string{"Description:", formatOptString(s.Description)})
	_ = t.Append([]string{"Created At:", formatTimestamp(s.CreatedAt)})
	_ = t.Append([]string{"Expires At:", formatOptDateTime(s.ExpiresAt)})
	return t.Render()
}

func (f *TableFormatter) formatRestoreSandboxRootFSResponse(w io.Writer, r *apispec.RestoreSandboxRootFSResponse) error {
	t := newTable(w)
	_ = t.Append([]string{"Sandbox ID:", r.SandboxID})
	_ = t.Append([]string{"Snapshot ID:", r.SnapshotID})
	_ = t.Append([]string{"Status:", string(r.Status)})
	return t.Render()
}

func (f *TableFormatter) formatForkSandboxResponse(w io.Writer, r *apispec.ForkSandboxResponse) error {
	t := newTable(w)
	_ = t.Append([]string{"Source Sandbox ID:", r.SourceSandboxID})
	_ = t.Append([]string{"Fork Sandbox ID:", r.Sandbox.ID})
	_ = t.Append([]string{"Template ID:", r.Sandbox.TemplateID})
	_ = t.Append([]string{"Status:", string(r.Sandbox.Status)})
	_ = t.Append([]string{"Paused:", fmt.Sprintf("%v", r.Sandbox.Paused)})
	return t.Render()
}

func (f *TableFormatter) formatSandbox(w io.Writer, s *apispec.Sandbox) error {
	t := newTable(w)
	_ = t.Append([]string{"ID:", s.ID})
	_ = t.Append([]string{"Template ID:", s.TemplateID})
	_ = t.Append([]string{"Team ID:", s.TeamID})
	if v, ok := s.UserID.Get(); ok {
		_ = t.Append([]string{"User ID:", v})
	}
	_ = t.Append([]string{"Status:", string(s.Status)})
	_ = t.Append([]string{"Paused:", fmt.Sprintf("%v", s.Paused)})
	if resources, ok := s.Resources.Get(); ok {
		if memory, ok := resources.Memory.Get(); ok {
			_ = t.Append([]string{"Memory:", memory})
		}
	}
	_ = t.Append([]string{"Pod Name:", s.PodName})
	_ = t.Append([]string{"Claimed At:", s.ClaimedAt.Format(timeLayout)})
	_ = t.Append([]string{"Soft Expires At:", formatTimestamp(s.ExpiresAt)})
	_ = t.Append([]string{"Hard Expires At:", formatTimestamp(s.HardExpiresAt)})
	if ssh, ok := s.SSH.Get(); ok {
		_ = t.Append([]string{"SSH Host:", valueOrDash(ssh.Host)})
		_ = t.Append([]string{"SSH Port:", intOrDash(ssh.Port)})
		_ = t.Append([]string{"SSH Username:", valueOrDash(ssh.Username)})
	}
	return t.Render()
}

func (f *TableFormatter) formatSSHPublicKeyList(w io.Writer, keys []apispec.SSHPublicKey) error {
	if len(keys) == 0 {
		_, _ = fmt.Fprintln(w, "No SSH public keys found.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"ID", "NAME", "KEY TYPE", "FINGERPRINT", "CREATED AT"})
	for _, key := range keys {
		_ = t.Append([]string{
			key.ID,
			key.Name,
			key.KeyType,
			key.FingerprintSHA256,
			key.CreatedAt.Format(timeLayout),
		})
	}
	return t.Render()
}

func (f *TableFormatter) formatSSHPublicKey(w io.Writer, key *apispec.SSHPublicKey) error {
	t := newTable(w)
	_ = t.Append([]string{"ID:", key.ID})
	_ = t.Append([]string{"Name:", key.Name})
	_ = t.Append([]string{"Key Type:", key.KeyType})
	_ = t.Append([]string{"Fingerprint:", key.FingerprintSHA256})
	_ = t.Append([]string{"Comment:", formatOptString(key.Comment)})
	_ = t.Append([]string{"Created At:", key.CreatedAt.Format(timeLayout)})
	_ = t.Append([]string{"Updated At:", key.UpdatedAt.Format(timeLayout)})
	return t.Render()
}

func (f *TableFormatter) formatSandboxStatus(w io.Writer, s *apispec.SandboxStatus) error {
	t := newTable(w)
	if v, ok := s.Status.Get(); ok {
		_ = t.Append([]string{"Status:", string(v)})
	}
	if v, ok := s.ClaimedAt.Get(); ok {
		_ = t.Append([]string{"Claimed At:", v})
	}
	if v, ok := s.ExpiresAt.Get(); ok {
		_ = t.Append([]string{"Soft Expires At:", formatTimestampText(v)})
	}
	if v, ok := s.HardExpiresAt.Get(); ok {
		_ = t.Append([]string{"Hard Expires At:", formatTimestampText(v)})
	}
	if v, ok := s.CreatedAt.Get(); ok {
		_ = t.Append([]string{"Created At:", v})
	}
	return t.Render()
}

func (f *TableFormatter) formatRefreshResponse(w io.Writer, r *apispec.RefreshResponse) error {
	t := newTable(w)
	_ = t.Append([]string{"Sandbox ID:", r.SandboxID})
	_ = t.Append([]string{"Soft Expires At:", formatTimestamp(r.ExpiresAt)})
	_ = t.Append([]string{"Hard Expires At:", formatTimestamp(r.HardExpiresAt)})
	return t.Render()
}

func (f *TableFormatter) formatSuccessMessage(w io.Writer, r *apispec.SuccessMessageResponse) error {
	t := newTable(w)
	_ = t.Append([]string{"Success:", fmt.Sprintf("%v", r.Success)})
	if v, ok := r.Data.Get(); ok {
		if msg, ok := v.Message.Get(); ok {
			_ = t.Append([]string{"Message:", msg})
		}
	}
	return t.Render()
}

func (f *TableFormatter) formatSuccessDeleted(w io.Writer, _ *apispec.SuccessDeletedResponse) error {
	_, _ = fmt.Fprintln(w, "Resource deleted successfully.")
	return nil
}

func (f *TableFormatter) formatSandboxList(w io.Writer, r *sandbox0.ListSandboxesResponse) error {
	if len(r.Sandboxes) == 0 {
		_, _ = fmt.Fprintln(w, "No sandboxes found.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"ID", "TEMPLATE ID", "STATUS", "PAUSED", "CREATED AT", "HARD EXPIRES AT"})

	for _, s := range r.Sandboxes {
		_ = t.Append([]string{
			s.ID,
			s.TemplateID,
			string(s.Status),
			fmt.Sprintf("%v", s.Paused),
			s.CreatedAt.Format(timeLayout),
			formatTimestamp(s.HardExpiresAt),
		})
	}
	if err := t.Render(); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(w, "Total: %d", r.Count)
	if r.HasMore {
		_, _ = fmt.Fprintf(w, " (more available)")
	}
	_, _ = fmt.Fprintln(w)
	return nil
}

func (f *TableFormatter) formatSDKSandbox(w io.Writer, s *sandbox0.Sandbox) error {
	t := newTable(w)
	_ = t.Append([]string{"ID:", s.ID})
	_ = t.Append([]string{"Template:", s.Template})
	_ = t.Append([]string{"Status:", s.Status})
	if s.ClusterID != nil {
		_ = t.Append([]string{"Cluster ID:", *s.ClusterID})
	}
	if s.PodName != "" {
		_ = t.Append([]string{"Pod Name:", s.PodName})
	}
	return t.Render()
}

func (f *TableFormatter) formatRegistryCredentials(w io.Writer, c *client.RegistryCredentials) error {
	t := newTable(w)
	password := c.Password
	if !f.showSecrets {
		password = maskSecret(password)
	}
	_ = t.Append([]string{"Provider:", c.Provider})
	_ = t.Append([]string{"Push Registry:", c.PushRegistry})
	_ = t.Append([]string{"Pull Registry:", c.PullRegistry})
	_ = t.Append([]string{"Username:", c.Username})
	_ = t.Append([]string{"Password:", password})
	if c.ExpiresAt != "" {
		_ = t.Append([]string{"Expires At:", c.ExpiresAt})
	}
	return t.Render()
}

func (f *TableFormatter) formatCredentialSourceList(w io.Writer, sources []apispec.CredentialSourceMetadata) error {
	if len(sources) == 0 {
		_, _ = fmt.Fprintln(w, "No credential sources found.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"NAME", "RESOLVER KIND", "VERSION", "STATUS", "CREATED AT", "UPDATED AT"})
	for _, source := range sources {
		_ = t.Append([]string{
			source.Name,
			string(source.ResolverKind),
			formatOptInt64(source.CurrentVersion),
			formatOptString(source.Status),
			formatOptNilDateTime(source.CreatedAt),
			formatOptNilDateTime(source.UpdatedAt),
		})
	}
	return t.Render()
}

func (f *TableFormatter) formatCredentialSource(w io.Writer, source *apispec.CredentialSourceMetadata) error {
	t := newTable(w)
	_ = t.Append([]string{"Name:", source.Name})
	_ = t.Append([]string{"Resolver Kind:", string(source.ResolverKind)})
	_ = t.Append([]string{"Current Version:", formatOptInt64(source.CurrentVersion)})
	_ = t.Append([]string{"Status:", formatOptString(source.Status)})
	_ = t.Append([]string{"Created At:", formatOptNilDateTime(source.CreatedAt)})
	_ = t.Append([]string{"Updated At:", formatOptNilDateTime(source.UpdatedAt)})
	return t.Render()
}

// PrintTable is a helper for printing tabular data.
func PrintTable(headers []string, rows [][]string) {
	t := newTable(os.Stdout)
	t.Header(headers)
	for _, row := range rows {
		_ = t.Append(row)
	}
	_ = t.Render()
}

func (f *TableFormatter) formatFileList(w io.Writer, files []apispec.FileInfo) error {
	if len(files) == 0 {
		_, _ = fmt.Fprintln(w, "No files found.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"NAME", "TYPE", "SIZE", "MODIFIED"})

	for _, file := range files {
		name, _ := file.Name.Get()
		fileType, _ := file.Type.Get()
		size, _ := file.Size.Get()
		modTime, _ := file.ModTime.Get()

		typeStr := string(fileType)
		if isLink, ok := file.IsLink.Get(); ok && isLink {
			if target, ok := file.LinkTarget.Get(); ok {
				typeStr = fmt.Sprintf("%s -> %s", typeStr, target)
			}
		}

		_ = t.Append([]string{
			name,
			typeStr,
			fmt.Sprintf("%d", size),
			modTime.Format("2006-01-02 15:04:05"),
		})
	}
	return t.Render()
}

func (f *TableFormatter) formatFileInfo(w io.Writer, file *apispec.FileInfo) error {
	t := newTable(w)
	if name, ok := file.Name.Get(); ok {
		_ = t.Append([]string{"Name:", name})
	}
	if path, ok := file.Path.Get(); ok {
		_ = t.Append([]string{"Path:", path})
	}
	if fileType, ok := file.Type.Get(); ok {
		_ = t.Append([]string{"Type:", string(fileType)})
	}
	if size, ok := file.Size.Get(); ok {
		_ = t.Append([]string{"Size:", fmt.Sprintf("%d bytes", size)})
	}
	if mode, ok := file.Mode.Get(); ok {
		_ = t.Append([]string{"Mode:", mode})
	}
	if modTime, ok := file.ModTime.Get(); ok {
		_ = t.Append([]string{"Modified:", modTime.Format("2006-01-02 15:04:05")})
	}
	if isLink, ok := file.IsLink.Get(); ok && isLink {
		_ = t.Append([]string{"Is Link:", fmt.Sprintf("%v", isLink)})
		if target, ok := file.LinkTarget.Get(); ok {
			_ = t.Append([]string{"Link Target:", target})
		}
	}
	return t.Render()
}

func (f *TableFormatter) formatContextList(w io.Writer, contexts []apispec.ContextResponse) error {
	if len(contexts) == 0 {
		_, _ = fmt.Fprintln(w, "No contexts found.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"ID", "TYPE", "ALIAS", "RUNNING", "PAUSED", "CREATED"})

	for _, ctx := range contexts {
		alias := "-"
		if lang, ok := ctx.Alias.Get(); ok {
			alias = lang
		}

		_ = t.Append([]string{
			ctx.ID,
			string(ctx.Type),
			alias,
			fmt.Sprintf("%v", ctx.Running),
			fmt.Sprintf("%v", ctx.Paused),
			ctx.CreatedAt,
		})
	}
	return t.Render()
}

func (f *TableFormatter) formatContext(w io.Writer, ctx *apispec.ContextResponse) error {
	t := newTable(w)
	_ = t.Append([]string{"ID:", ctx.ID})
	_ = t.Append([]string{"Type:", string(ctx.Type)})
	if alias, ok := ctx.Alias.Get(); ok {
		_ = t.Append([]string{"Alias:", alias})
	}
	if cwd, ok := ctx.Cwd.Get(); ok {
		_ = t.Append([]string{"Working Dir:", cwd})
	}
	_ = t.Append([]string{"Running:", fmt.Sprintf("%v", ctx.Running)})
	_ = t.Append([]string{"Paused:", fmt.Sprintf("%v", ctx.Paused)})
	_ = t.Append([]string{"Created:", ctx.CreatedAt})
	if outputRaw, ok := ctx.OutputRaw.Get(); ok {
		_ = t.Append([]string{"Output:", outputRaw})
	}
	return t.Render()
}

func (f *TableFormatter) formatExecutionSessionList(w io.Writer, sessions []apispec.ExecutionSession) error {
	if len(sessions) == 0 {
		_, _ = fmt.Fprintln(w, "No execution sessions found.")
		return nil
	}
	t := newTable(w)
	t.Header([]string{"ID", "Name", "Phase", "Attempt", "PID", "Restarts", "Latest event", "Updated"})
	for _, session := range sessions {
		name := "-"
		if value, ok := session.Spec.Name.Get(); ok && value != "" {
			name = value
		}
		attemptID := "-"
		pid := "-"
		if attempt, ok := session.Attempt.Get(); ok {
			attemptID = attempt.ID
			if value, ok := attempt.Pid.Get(); ok {
				pid = strconv.Itoa(int(value))
			}
		}
		_ = t.Append([]string{
			session.ID,
			name,
			string(session.Phase),
			attemptID,
			pid,
			strconv.Itoa(int(session.RestartCount)),
			strconv.FormatInt(session.Cursor.Latest, 10),
			session.UpdatedAt.Format(timeLayout),
		})
	}
	return t.Render()
}

func (f *TableFormatter) formatExecutionSession(w io.Writer, session *apispec.ExecutionSession) error {
	t := newTable(w)
	_ = t.Append([]string{"ID:", session.ID})
	if name, ok := session.Spec.Name.Get(); ok {
		_ = t.Append([]string{"Name:", name})
	}
	_ = t.Append([]string{"Phase:", string(session.Phase)})
	_ = t.Append([]string{"Command:", strings.Join(session.Spec.Command, " ")})
	if cwd, ok := session.Spec.Cwd.Get(); ok {
		_ = t.Append([]string{"Working dir:", cwd})
	}
	_ = t.Append([]string{"Spec version:", strconv.FormatInt(session.SpecVersion, 10)})
	_ = t.Append([]string{"Runtime generation:", strconv.FormatInt(session.RuntimeGeneration, 10)})
	_ = t.Append([]string{"Restart count:", strconv.Itoa(int(session.RestartCount))})
	_ = t.Append([]string{"Event cursor:", fmt.Sprintf("%d..%d", session.Cursor.Earliest, session.Cursor.Latest)})
	if attempt, ok := session.Attempt.Get(); ok {
		_ = t.Append([]string{"Attempt:", attempt.ID})
		_ = t.Append([]string{"Attempt number:", strconv.FormatInt(attempt.Number, 10)})
		if pid, ok := attempt.Pid.Get(); ok {
			_ = t.Append([]string{"PID:", strconv.Itoa(int(pid))})
		}
		if exitCode, ok := attempt.ExitCode.Get(); ok {
			_ = t.Append([]string{"Exit code:", strconv.Itoa(int(exitCode))})
		}
		if reason, ok := attempt.Reason.Get(); ok {
			_ = t.Append([]string{"Attempt reason:", reason})
		}
	}
	_ = t.Append([]string{"Created:", session.CreatedAt.Format(timeLayout)})
	_ = t.Append([]string{"Updated:", session.UpdatedAt.Format(timeLayout)})
	return t.Render()
}

func (f *TableFormatter) formatExecutionSessionEvents(w io.Writer, page *apispec.ExecutionSessionEventPage) error {
	if len(page.Events) == 0 {
		_, _ = fmt.Fprintf(w, "No execution session events found (cursor %d..%d).\n", page.Cursor.Earliest, page.Cursor.Latest)
		return nil
	}
	t := newTable(w)
	t.Header([]string{"Seq", "Occurred", "Attempt", "Type", "Stream", "Data", "Reason"})
	for _, event := range page.Events {
		attempt := "-"
		if value, ok := event.AttemptID.Get(); ok {
			attempt = value
		}
		stream := "-"
		if value, ok := event.Stream.Get(); ok {
			stream = string(value)
		}
		data := ""
		if value, ok := event.DataBase64.Get(); ok {
			if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
				data = strings.TrimRight(string(decoded), "\r\n")
			} else {
				data = value
			}
		}
		reason := ""
		if value, ok := event.Reason.Get(); ok {
			reason = value
		}
		_ = t.Append([]string{
			strconv.FormatInt(event.Seq, 10),
			event.OccurredAt.Format(timeLayout),
			attempt,
			event.Type,
			stream,
			data,
			reason,
		})
	}
	return t.Render()
}

func (f *TableFormatter) formatExecutionSessionInput(w io.Writer, response *apispec.ExecutionSessionInputResponse) error {
	t := newTable(w)
	_ = t.Append([]string{"Input ID:", response.InputID})
	_ = t.Append([]string{"Attempt ID:", response.AttemptID})
	_ = t.Append([]string{"Accepted:", strconv.FormatBool(response.Accepted)})
	_ = t.Append([]string{"Duplicate:", strconv.FormatBool(response.Duplicate)})
	return t.Render()
}

func (f *TableFormatter) formatSandboxObservabilityEvents(w io.Writer, resp *apispec.SandboxObservabilityEventsResponse) error {
	t := newTable(w)
	t.Header([]string{"Occurred", "Source", "Type", "Outcome", "Cursor"})
	for _, event := range resp.Events {
		outcome := ""
		if value, ok := event.Outcome.Get(); ok {
			outcome = string(value)
		}
		_ = t.Append([]string{
			event.OccurredAt.Format(timeLayout),
			string(event.Source),
			string(event.EventType),
			outcome,
			event.Cursor,
		})
	}
	return t.Render()
}

func (f *TableFormatter) formatSandboxObservabilityLogs(w io.Writer, resp *apispec.SandboxObservabilityLogsResponse) error {
	t := newTable(w)
	t.Header([]string{"Occurred", "Stream", "Context", "Message", "Cursor"})
	for _, entry := range resp.Logs {
		stream := ""
		if value, ok := entry.Stream.Get(); ok {
			stream = string(value)
		}
		contextID := ""
		if value, ok := entry.ContextID.Get(); ok {
			contextID = value
		}
		_ = t.Append([]string{
			entry.OccurredAt.Format(timeLayout),
			stream,
			contextID,
			strings.TrimRight(entry.Message, "\r\n"),
			entry.Cursor,
		})
	}
	return t.Render()
}

func (f *TableFormatter) formatSandboxRuntimeMetrics(w io.Writer, resp *apispec.SandboxRuntimeMetricsResponse) error {
	t := newTable(w)
	t.Header([]string{"Time", "Metric", "Value", "Unit", "Statistic", "Dimensions"})
	for _, series := range resp.Series {
		dimensions := ""
		if value, ok := series.Dimensions.Get(); ok {
			pairs := make([]string, 0, len(value))
			for key, item := range value {
				pairs = append(pairs, key+"="+item)
			}
			slices.Sort(pairs)
			dimensions = strings.Join(pairs, ",")
		}
		for _, segment := range series.Segments {
			for _, point := range segment.Points {
				_ = t.Append([]string{
					point.Time.Format(timeLayout),
					string(series.Metric),
					fmt.Sprintf("%.6g", point.Value),
					string(series.Unit),
					string(series.Statistic),
					dimensions,
				})
			}
		}
	}
	return t.Render()
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

//nolint:staticcheck // The CLI still displays legacy allow/deny fields for compatibility with older policies.
func (f *TableFormatter) formatSandboxNetworkPolicy(w io.Writer, policy *apispec.SandboxNetworkPolicy) error {
	t := newTable(w)
	_ = t.Append([]string{"Mode:", string(policy.Mode)})
	if egress, ok := policy.Egress.Get(); ok {
		_ = t.Append([]string{"", ""})
		_ = t.Append([]string{"Egress Policy:", ""})
		if len(egress.AllowedCidrs) > 0 {
			_ = t.Append([]string{"Allowed CIDRs:", fmt.Sprintf("%v", egress.AllowedCidrs)})
		}
		if len(egress.AllowedDomains) > 0 {
			_ = t.Append([]string{"Allowed Domains:", fmt.Sprintf("%v", egress.AllowedDomains)})
		}
		if len(egress.DeniedCidrs) > 0 {
			_ = t.Append([]string{"Denied CIDRs:", fmt.Sprintf("%v", egress.DeniedCidrs)})
		}
		if len(egress.DeniedDomains) > 0 {
			_ = t.Append([]string{"Denied Domains:", fmt.Sprintf("%v", egress.DeniedDomains)})
		}
		if len(egress.TrafficRules) > 0 {
			_ = t.Append([]string{"Traffic Rules:", fmt.Sprintf("%d", len(egress.TrafficRules))})
		}
		if len(egress.ProtocolRules) > 0 {
			_ = t.Append([]string{"Protocol Rules:", fmt.Sprintf("%d", len(egress.ProtocolRules))})
		}
		if len(egress.CredentialRules) > 0 {
			_ = t.Append([]string{"Credential Rules:", fmt.Sprintf("%d", len(egress.CredentialRules))})
		}
		if proxy, ok := egress.Proxy.Get(); ok {
			_ = t.Append([]string{"Proxy:", fmt.Sprintf("%s %s", proxy.Type, proxy.Address)})
			if credentialRef, ok := proxy.CredentialRef.Get(); ok {
				_ = t.Append([]string{"Proxy Credential:", credentialRef})
			}
		}
	}
	if len(policy.CredentialBindings) > 0 {
		_ = t.Append([]string{"", ""})
		_ = t.Append([]string{"Credential Bindings:", fmt.Sprintf("%d", len(policy.CredentialBindings))})
	}
	return t.Render()
}

func (f *TableFormatter) formatSandboxServices(w io.Writer, resp *sandbox0.SandboxServicesResponse) error {
	if len(resp.Services) == 0 {
		_, _ = fmt.Fprintln(w, "No sandbox services configured.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"SERVICE", "PORT", "PUBLIC", "URL", "ROUTE", "PATH", "METHODS", "AUTH", "RATE LIMIT", "TIMEOUT", "RESUME", "PUBLISHABLE"})
	for _, service := range resp.Services {
		routes := service.Ingress.Routes
		if len(routes) == 0 {
			_ = t.Append([]string{
				service.ID,
				optInt32OrDash(service.Port),
				fmt.Sprintf("%v", service.Ingress.Public),
				service.PublicURL.Or("-"),
				"-",
				"-",
				"-",
				"-",
				"-",
				"-",
				"-",
				formatPublishable(service),
			})
			continue
		}
		for _, route := range routes {
			_ = t.Append([]string{
				service.ID,
				optInt32OrDash(service.Port),
				fmt.Sprintf("%v", service.Ingress.Public),
				service.PublicURL.Or("-"),
				route.ID,
				route.PathPrefix.Or("/"),
				formatGatewayMethods(route.Methods),
				formatGatewayAuth(route.Auth),
				formatGatewayRateLimit(route.RateLimit),
				formatGatewayTimeout(route.TimeoutSeconds),
				fmt.Sprintf("%v", route.Resume),
				formatPublishable(service),
			})
		}
	}
	return t.Render()
}

func formatPublishable(service apispec.SandboxAppServiceView) string {
	if service.Publishable {
		return "true"
	}
	if len(service.PublishBlockers) == 0 {
		return "false"
	}
	return "false: " + strings.Join(service.PublishBlockers, ",")
}

func formatGatewayMethods(methods []string) string {
	if len(methods) == 0 {
		return "*"
	}
	return strings.Join(methods, ",")
}

func formatGatewayAuth(auth apispec.OptSandboxAppServiceRouteAuth) string {
	value, ok := auth.Get()
	if !ok {
		return "none"
	}
	return string(value.Mode)
}

func formatGatewayRateLimit(rateLimit apispec.OptSandboxAppServiceRouteRateLimit) string {
	value, ok := rateLimit.Get()
	if !ok {
		return "-"
	}
	return fmt.Sprintf("%d/%d", value.Rps, value.Burst)
}

func formatGatewayTimeout(timeout apispec.OptInt32) string {
	value, ok := timeout.Get()
	if !ok || value == 0 {
		return "-"
	}
	return fmt.Sprintf("%ds", value)
}

func formatTimestamp(ts time.Time) string {
	if ts.IsZero() {
		return "-"
	}
	return ts.Format(timeLayout)
}

func formatTimestampText(v string) string {
	if v == "" {
		return "-"
	}
	parsed, err := time.Parse(time.RFC3339, v)
	if err == nil {
		return formatTimestamp(parsed)
	}
	if v == "0001-01-01T00:00:00Z" || v == "0001-01-01 00:00:00" {
		return "-"
	}
	return v
}

func (f *TableFormatter) formatMountStatusList(w io.Writer, mounts []apispec.MountStatus) error {
	if len(mounts) == 0 {
		_, _ = fmt.Fprintln(w, "No mounted volumes.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"VOLUME ID", "MOUNT POINT", "STATE", "MOUNTED AT", "DURATION", "ERROR"})

	for _, m := range mounts {
		volumeID := m.SandboxvolumeID
		mountPoint := m.MountPoint
		mountedAt, _ := m.MountedAt.Get()
		duration := "-"
		if d, ok := m.MountedDurationSec.Get(); ok {
			duration = formatDuration(d)
		}
		errorText := "-"
		errorCode, hasErrorCode := m.ErrorCode.Get()
		errorMessage, hasErrorMessage := m.ErrorMessage.Get()
		switch {
		case hasErrorCode && hasErrorMessage:
			errorText = fmt.Sprintf("%s: %s", errorCode, errorMessage)
		case hasErrorMessage:
			errorText = errorMessage
		case hasErrorCode:
			errorText = errorCode
		}

		_ = t.Append([]string{
			volumeID,
			mountPoint,
			string(m.State),
			formatTimestampText(mountedAt),
			duration,
			errorText,
		})
	}
	return t.Render()
}

func formatDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm%ds", seconds/60, seconds%60)
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	return fmt.Sprintf("%dh%dm", hours, minutes)
}

func (f *TableFormatter) formatAPIKeyList(w io.Writer, keys []apispec.APIKey) error {
	if len(keys) == 0 {
		_, _ = fmt.Fprintln(w, "No API keys found.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"ID", "NAME", "SCOPE", "TEAM ID", "USER ID", "ROLES", "ACTIVE", "EXPIRES AT", "LAST USED"})
	for _, k := range keys {
		_ = t.Append([]string{
			k.ID,
			k.Name,
			formatAPIKeyScope(k.Scope),
			k.TeamID,
			formatOptNilString(k.UserID),
			formatStringSlice(k.Roles),
			fmt.Sprintf("%v", k.IsActive),
			formatTimestamp(k.ExpiresAt),
			formatOptDateTime(k.LastUsedAt),
		})
	}
	return t.Render()
}

func (f *TableFormatter) formatCreatedAPIKey(w io.Writer, k *apispec.CreateAPIKeyResponse) error {
	t := newTable(w)
	_ = t.Append([]string{"ID:", k.ID})
	_ = t.Append([]string{"Name:", k.Name})
	_ = t.Append([]string{"Scope:", formatAPIKeyScope(k.Scope)})
	_ = t.Append([]string{"Team ID:", k.TeamID})
	_ = t.Append([]string{"Roles:", formatStringSlice(k.Roles)})
	_ = t.Append([]string{"Expires At:", formatTimestamp(k.ExpiresAt)})
	_ = t.Append([]string{"Created At:", formatTimestamp(k.CreatedAt)})
	if key, ok := k.Key.Get(); ok && key != "" {
		if !f.showSecrets {
			key = maskSecret(key)
		}
		_ = t.Append([]string{"Key:", key})
	}
	return t.Render()
}

func formatAPIKeyScope(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return "team"
	}
	return scope
}

func (f *TableFormatter) formatTeamList(w io.Writer, teams []apispec.Team) error {
	return f.formatTeamListWithCurrent(w, NewTeamList(teams, ""))
}

func (f *TableFormatter) formatTeamListWithCurrent(w io.Writer, teams TeamList) error {
	if len(teams) == 0 {
		_, _ = fmt.Fprintln(w, "No teams found.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"CURRENT", "ID", "NAME", "SLUG", "OWNER ID", "CREATED AT"})
	for _, item := range teams {
		current := ""
		if item.Current {
			current = "*"
		}
		_ = t.Append([]string{
			current,
			item.ID,
			item.Name,
			item.Slug,
			formatStringPtr(item.OwnerID),
			formatTimestamp(item.CreatedAt),
		})
	}
	return t.Render()
}

func (f *TableFormatter) formatTeam(w io.Writer, team *apispec.Team) error {
	t := newTable(w)
	_ = t.Append([]string{"ID:", team.ID})
	_ = t.Append([]string{"Name:", team.Name})
	_ = t.Append([]string{"Slug:", team.Slug})
	_ = t.Append([]string{"Owner ID:", formatOptNilString(team.OwnerID)})
	_ = t.Append([]string{"Created At:", formatTimestamp(team.CreatedAt)})
	_ = t.Append([]string{"Updated At:", formatTimestamp(team.UpdatedAt)})
	return t.Render()
}

func (f *TableFormatter) formatRegionList(w io.Writer, regions []apispec.Region) error {
	if len(regions) == 0 {
		_, _ = fmt.Fprintln(w, "No regions found.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"ID", "DISPLAY NAME", "REGIONAL GATEWAY URL", "METERING EXPORT URL", "ENABLED"})
	for _, region := range regions {
		_ = t.Append([]string{
			region.ID,
			formatOptString(region.DisplayName),
			region.RegionalGatewayURL,
			formatOptNilString(region.MeteringExportURL),
			fmt.Sprintf("%v", region.Enabled),
		})
	}
	return t.Render()
}

func (f *TableFormatter) formatRegion(w io.Writer, region *apispec.Region) error {
	t := newTable(w)
	_ = t.Append([]string{"ID:", region.ID})
	_ = t.Append([]string{"Display Name:", formatOptString(region.DisplayName)})
	_ = t.Append([]string{"Regional Gateway URL:", region.RegionalGatewayURL})
	_ = t.Append([]string{"Metering Export URL:", formatOptNilString(region.MeteringExportURL)})
	_ = t.Append([]string{"Enabled:", fmt.Sprintf("%v", region.Enabled)})
	return t.Render()
}

func (f *TableFormatter) formatTeamMemberList(w io.Writer, members []apispec.TeamMember) error {
	if len(members) == 0 {
		_, _ = fmt.Fprintln(w, "No team members found.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"ID", "USER ID", "EMAIL", "NAME", "ROLE", "JOINED AT"})
	for _, m := range members {
		_ = t.Append([]string{
			m.ID,
			m.UserID,
			formatOptString(m.Email),
			formatOptString(m.Name),
			m.Role,
			formatTimestamp(m.JoinedAt),
		})
	}
	return t.Render()
}

func (f *TableFormatter) formatTeamMember(w io.Writer, m *apispec.TeamMember) error {
	t := newTable(w)
	_ = t.Append([]string{"ID:", m.ID})
	_ = t.Append([]string{"User ID:", m.UserID})
	_ = t.Append([]string{"Email:", formatOptString(m.Email)})
	_ = t.Append([]string{"Name:", formatOptString(m.Name)})
	_ = t.Append([]string{"Avatar URL:", formatOptString(m.AvatarURL)})
	_ = t.Append([]string{"Role:", m.Role})
	_ = t.Append([]string{"Joined At:", formatTimestamp(m.JoinedAt)})
	return t.Render()
}

func (f *TableFormatter) formatUser(w io.Writer, u *apispec.User) error {
	t := newTable(w)
	_ = t.Append([]string{"ID:", u.ID})
	_ = t.Append([]string{"Email:", u.Email})
	_ = t.Append([]string{"Name:", u.Name})
	_ = t.Append([]string{"Avatar URL:", formatOptNilString(u.AvatarURL)})
	_ = t.Append([]string{"Email Verified:", fmt.Sprintf("%v", u.EmailVerified)})
	_ = t.Append([]string{"Is Admin:", fmt.Sprintf("%v", u.IsAdmin)})
	_ = t.Append([]string{"Created At:", formatTimestamp(u.CreatedAt)})
	_ = t.Append([]string{"Updated At:", formatTimestamp(u.UpdatedAt)})
	return t.Render()
}

func formatStringSlice(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ",")
}

func formatOptString(v apispec.OptString) string {
	if s, ok := v.Get(); ok && s != "" {
		return s
	}
	return "-"
}

func formatOptInt64(v apispec.OptInt64) string {
	if n, ok := v.Get(); ok {
		return fmt.Sprintf("%d", n)
	}
	return "-"
}

func formatOptNilString(v apispec.OptNilString) string {
	if v.IsNull() {
		return "-"
	}
	if s, ok := v.Get(); ok && s != "" {
		return s
	}
	return "-"
}

func formatStringPtr(v *string) string {
	if v != nil && *v != "" {
		return *v
	}
	return "-"
}

func formatOptDateTime(v apispec.OptDateTime) string {
	if ts, ok := v.Get(); ok {
		return formatTimestamp(ts)
	}
	return "-"
}

func formatOptNilDateTime(v apispec.OptNilDateTime) string {
	if v.IsNull() {
		return "-"
	}
	if ts, ok := v.Get(); ok {
		return formatTimestamp(ts)
	}
	return "-"
}
