package output

import (
	"fmt"
	"io"
	"os"
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
	case *apispec.ContextStatsResponse:
		return f.formatContextStats(w, v)
	case *apispec.SandboxNetworkPolicy:
		return f.formatSandboxNetworkPolicy(w, v)
	case *sandbox0.SandboxServicesResponse:
		return f.formatSandboxServices(w, v)
	case *sandbox0.PublicGatewayResponse:
		return f.formatPublicGateway(w, v)
	case []apispec.FunctionRecord:
		return f.formatFunctionList(w, v)
	case apispec.FunctionRecord:
		return f.formatFunction(w, &v)
	case *apispec.FunctionRecord:
		return f.formatFunction(w, v)
	case sandbox0.FunctionCreateResult:
		return f.formatFunctionCreateResult(w, &v)
	case *sandbox0.FunctionCreateResult:
		return f.formatFunctionCreateResult(w, v)
	case []apispec.FunctionRevision:
		return f.formatFunctionRevisionList(w, v)
	case apispec.FunctionRevision:
		return f.formatFunctionRevision(w, &v)
	case *apispec.FunctionRevision:
		return f.formatFunctionRevision(w, v)
	case sandbox0.FunctionRevisionCreateResult:
		return f.formatFunctionRevisionCreateResult(w, &v)
	case *sandbox0.FunctionRevisionCreateResult:
		return f.formatFunctionRevisionCreateResult(w, v)
	case apispec.FunctionAlias:
		return f.formatFunctionAlias(w, &v)
	case *apispec.FunctionAlias:
		return f.formatFunctionAlias(w, v)
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
	t.Header([]string{"ID", "TEAM ID", "CREATED"})

	for _, v := range volumes {
		_ = t.Append([]string{
			v.ID,
			v.TeamID,
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
	_ = t.Append([]string{"Created:", v.CreatedAt.Format("2006-01-02 15:04:05")})
	_ = t.Append([]string{"Updated:", v.UpdatedAt.Format("2006-01-02 15:04:05")})
	return t.Render()
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

func (f *TableFormatter) formatContextStats(w io.Writer, stats *apispec.ContextStatsResponse) error {
	t := newTable(w)
	if contextID, ok := stats.ContextID.Get(); ok {
		_ = t.Append([]string{"Context ID:", contextID})
	}
	if ctxType, ok := stats.Type.Get(); ok {
		_ = t.Append([]string{"Type:", ctxType})
	}
	if alias, ok := stats.Alias.Get(); ok {
		_ = t.Append([]string{"Alias:", alias})
	}
	if running, ok := stats.Running.Get(); ok {
		_ = t.Append([]string{"Running:", fmt.Sprintf("%v", running)})
	}
	if paused, ok := stats.Paused.Get(); ok {
		_ = t.Append([]string{"Paused:", fmt.Sprintf("%v", paused)})
	}
	if usage, ok := stats.Usage.Get(); ok {
		_ = t.Append([]string{"---", "---"})
		_ = t.Append([]string{"Resource Usage:", ""})
		if cpu, ok := usage.CPUPercent.Get(); ok {
			_ = t.Append([]string{"CPU %:", fmt.Sprintf("%.2f", cpu)})
		}
		if memRss, ok := usage.MemoryRss.Get(); ok {
			_ = t.Append([]string{"Memory RSS:", formatBytes(memRss)})
		}
		if memVms, ok := usage.MemoryVms.Get(); ok {
			_ = t.Append([]string{"Memory VMS:", formatBytes(memVms)})
		}
		if memBytes, ok := usage.MemoryBytes.Get(); ok {
			_ = t.Append([]string{"Memory Bytes:", formatBytes(memBytes)})
		}
		if threads, ok := usage.ThreadCount.Get(); ok {
			_ = t.Append([]string{"Threads:", fmt.Sprintf("%d", threads)})
		}
		if openFiles, ok := usage.OpenFiles.Get(); ok {
			_ = t.Append([]string{"Open Files:", fmt.Sprintf("%d", openFiles)})
		}
		if ioRead, ok := usage.IoReadBytes.Get(); ok {
			_ = t.Append([]string{"IO Read:", formatBytes(ioRead)})
		}
		if ioWrite, ok := usage.IoWriteBytes.Get(); ok {
			_ = t.Append([]string{"IO Write:", formatBytes(ioWrite)})
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
		if len(egress.CredentialRules) > 0 {
			_ = t.Append([]string{"Credential Rules:", fmt.Sprintf("%d", len(egress.CredentialRules))})
		}
	}
	if len(policy.CredentialBindings) > 0 {
		_ = t.Append([]string{"", ""})
		_ = t.Append([]string{"Credential Bindings:", fmt.Sprintf("%d", len(policy.CredentialBindings))})
	}
	return t.Render()
}

func (f *TableFormatter) formatPublicGateway(w io.Writer, resp *sandbox0.PublicGatewayResponse) error {
	policy := resp.PublicGateway
	if !policy.Enabled || len(policy.Routes) == 0 {
		state := "disabled"
		if policy.Enabled {
			state = "enabled with no routes"
		}
		_, _ = fmt.Fprintf(w, "Public Gateway: %s\n", state)
		if resp.ExposureDomain != "" {
			_, _ = fmt.Fprintf(w, "Exposure Domain: %s\n", resp.ExposureDomain)
		}
		return nil
	}

	t := newTable(w)
	t.Header([]string{"ID", "PORT", "PATH", "METHODS", "AUTH", "RATE LIMIT", "TIMEOUT", "RESUME"})
	for _, route := range policy.Routes {
		_ = t.Append([]string{
			route.ID,
			fmt.Sprintf("%d", route.Port),
			route.PathPrefix.Or("/"),
			formatGatewayMethods(route.Methods),
			formatGatewayAuth(route.Auth),
			formatGatewayRateLimit(route.RateLimit),
			formatGatewayTimeout(route.TimeoutSeconds),
			fmt.Sprintf("%v", route.Resume),
		})
	}
	if err := t.Render(); err != nil {
		return err
	}

	if resp.ExposureDomain != "" {
		_, _ = fmt.Fprintf(w, "Exposure Domain: %s\n", resp.ExposureDomain)
	}
	return nil
}

func (f *TableFormatter) formatSandboxServices(w io.Writer, resp *sandbox0.SandboxServicesResponse) error {
	if len(resp.Services) == 0 {
		_, _ = fmt.Fprintln(w, "No sandbox services configured.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"SERVICE", "PORT", "PUBLIC", "ROUTE", "PATH", "METHODS", "AUTH", "RATE LIMIT", "TIMEOUT", "RESUME", "PUBLISHABLE"})
	for _, service := range resp.Services {
		routes := service.Ingress.Routes
		if len(routes) == 0 {
			_ = t.Append([]string{
				service.ID,
				fmt.Sprintf("%d", service.Port),
				fmt.Sprintf("%v", service.Ingress.Public),
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
				fmt.Sprintf("%d", service.Port),
				fmt.Sprintf("%v", service.Ingress.Public),
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

func formatGatewayAuth(auth apispec.OptPublicGatewayAuth) string {
	value, ok := auth.Get()
	if !ok {
		return "none"
	}
	return string(value.Mode)
}

func formatGatewayRateLimit(rateLimit apispec.OptPublicGatewayRateLimit) string {
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

func (f *TableFormatter) formatFunctionList(w io.Writer, functions []apispec.FunctionRecord) error {
	if len(functions) == 0 {
		_, _ = fmt.Fprintln(w, "No functions found.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"ID", "NAME", "SLUG", "HOST", "ACTIVE REVISION", "UPDATED AT"})
	for _, fn := range functions {
		_ = t.Append([]string{
			fn.ID,
			fn.Name,
			fn.Slug,
			fn.Host,
			formatOptString(fn.ActiveRevisionID),
			formatTimestamp(fn.UpdatedAt),
		})
	}
	return t.Render()
}

func (f *TableFormatter) formatFunction(w io.Writer, fn *apispec.FunctionRecord) error {
	t := newTable(w)
	_ = t.Append([]string{"ID:", fn.ID})
	_ = t.Append([]string{"Team ID:", fn.TeamID})
	_ = t.Append([]string{"Name:", fn.Name})
	_ = t.Append([]string{"Slug:", fn.Slug})
	_ = t.Append([]string{"Domain Label:", fn.DomainLabel})
	_ = t.Append([]string{"Host:", fn.Host})
	_ = t.Append([]string{"URL:", fn.URL})
	_ = t.Append([]string{"Active Revision:", formatOptString(fn.ActiveRevisionID)})
	_ = t.Append([]string{"Created By:", formatOptString(fn.CreatedBy)})
	_ = t.Append([]string{"Created At:", formatTimestamp(fn.CreatedAt)})
	_ = t.Append([]string{"Updated At:", formatTimestamp(fn.UpdatedAt)})
	return t.Render()
}

func (f *TableFormatter) formatFunctionCreateResult(w io.Writer, result *sandbox0.FunctionCreateResult) error {
	t := newTable(w)
	_ = t.Append([]string{"Function ID:", result.Function.ID})
	_ = t.Append([]string{"Name:", result.Function.Name})
	_ = t.Append([]string{"Slug:", result.Function.Slug})
	_ = t.Append([]string{"URL:", result.Function.URL})
	_ = t.Append([]string{"Revision ID:", result.Revision.ID})
	_ = t.Append([]string{"Revision Number:", fmt.Sprintf("%d", result.Revision.RevisionNumber)})
	_ = t.Append([]string{"Alias:", result.Alias.Alias})
	return t.Render()
}

func (f *TableFormatter) formatFunctionRevisionList(w io.Writer, revisions []apispec.FunctionRevision) error {
	if len(revisions) == 0 {
		_, _ = fmt.Fprintln(w, "No function revisions found.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"REVISION", "ID", "SOURCE", "PORT", "RUNTIME", "RUNTIME SANDBOX", "CREATED AT"})
	for _, revision := range revisions {
		_ = t.Append([]string{
			fmt.Sprintf("%d", revision.RevisionNumber),
			revision.ID,
			formatFunctionRevisionSource(revision),
			fmt.Sprintf("%d", revision.ServiceSnapshot.Port),
			formatFunctionServiceRuntime(revision.ServiceSnapshot),
			formatOptString(revision.RuntimeSandboxID),
			formatTimestamp(revision.CreatedAt),
		})
	}
	return t.Render()
}

func (f *TableFormatter) formatFunctionRevision(w io.Writer, revision *apispec.FunctionRevision) error {
	t := newTable(w)
	_ = t.Append([]string{"ID:", revision.ID})
	_ = t.Append([]string{"Function ID:", revision.FunctionID})
	_ = t.Append([]string{"Team ID:", revision.TeamID})
	_ = t.Append([]string{"Revision Number:", fmt.Sprintf("%d", revision.RevisionNumber)})
	_ = t.Append([]string{"Source Sandbox ID:", revision.SourceSandboxID})
	_ = t.Append([]string{"Source Service ID:", revision.SourceServiceID})
	_ = t.Append([]string{"Source Template ID:", revision.SourceTemplateID})
	_ = t.Append([]string{"Service Port:", fmt.Sprintf("%d", revision.ServiceSnapshot.Port)})
	_ = t.Append([]string{"Service Runtime:", formatFunctionServiceRuntime(revision.ServiceSnapshot)})
	_ = t.Append([]string{"Runtime Sandbox ID:", formatOptString(revision.RuntimeSandboxID)})
	_ = t.Append([]string{"Runtime Context ID:", formatOptString(revision.RuntimeContextID)})
	_ = t.Append([]string{"Runtime Updated At:", formatOptDateTime(revision.RuntimeUpdatedAt)})
	_ = t.Append([]string{"Restore Mounts:", formatFunctionRestoreMounts(revision.RestoreMounts)})
	_ = t.Append([]string{"Created By:", formatOptString(revision.CreatedBy)})
	_ = t.Append([]string{"Created At:", formatTimestamp(revision.CreatedAt)})
	return t.Render()
}

func (f *TableFormatter) formatFunctionRevisionCreateResult(w io.Writer, result *sandbox0.FunctionRevisionCreateResult) error {
	t := newTable(w)
	_ = t.Append([]string{"Revision ID:", result.Revision.ID})
	_ = t.Append([]string{"Function ID:", result.Revision.FunctionID})
	_ = t.Append([]string{"Revision Number:", fmt.Sprintf("%d", result.Revision.RevisionNumber)})
	_ = t.Append([]string{"Promoted:", fmt.Sprintf("%v", result.Promoted)})
	_ = t.Append([]string{"Runtime Sandbox ID:", formatOptString(result.Revision.RuntimeSandboxID)})
	_ = t.Append([]string{"Created At:", formatTimestamp(result.Revision.CreatedAt)})
	return t.Render()
}

func (f *TableFormatter) formatFunctionAlias(w io.Writer, alias *apispec.FunctionAlias) error {
	t := newTable(w)
	_ = t.Append([]string{"Function ID:", alias.FunctionID})
	_ = t.Append([]string{"Alias:", alias.Alias})
	_ = t.Append([]string{"Revision ID:", alias.RevisionID})
	_ = t.Append([]string{"Revision Number:", fmt.Sprintf("%d", alias.RevisionNumber)})
	_ = t.Append([]string{"Updated By:", formatOptString(alias.UpdatedBy)})
	_ = t.Append([]string{"Updated At:", formatTimestamp(alias.UpdatedAt)})
	return t.Render()
}

func formatFunctionRevisionSource(revision apispec.FunctionRevision) string {
	return fmt.Sprintf("%s/%s", valueOrDash(revision.SourceSandboxID), valueOrDash(revision.SourceServiceID))
}

func formatFunctionServiceRuntime(service apispec.SandboxAppService) string {
	runtime, ok := service.Runtime.Get()
	if !ok || runtime.Type == "" {
		return "-"
	}
	return string(runtime.Type)
}

func formatFunctionRestoreMounts(mounts []apispec.FunctionRestoreMount) string {
	if len(mounts) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(mounts))
	for _, mount := range mounts {
		parts = append(parts, fmt.Sprintf("%s:%s", mount.SandboxvolumeID, mount.MountPoint))
	}
	return strings.Join(parts, ",")
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
	if len(teams) == 0 {
		_, _ = fmt.Fprintln(w, "No teams found.")
		return nil
	}

	t := newTable(w)
	t.Header([]string{"ID", "NAME", "SLUG", "OWNER ID", "CREATED AT"})
	for _, team := range teams {
		_ = t.Append([]string{
			team.ID,
			team.Name,
			team.Slug,
			formatOptNilString(team.OwnerID),
			formatTimestamp(team.CreatedAt),
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
	t.Header([]string{"ID", "USER ID", "ROLE", "JOINED AT"})
	for _, m := range members {
		_ = t.Append([]string{
			m.ID,
			m.UserID,
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
