package output

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sandbox0-ai/s0/internal/client"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

const timeLayout = "2006-01-02 15:04:05"

// TableFormatter formats output as a table.
type TableFormatter struct{}

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
	case *apispec.TplSandboxNetworkPolicy:
		return f.formatNetworkPolicy(w, v)
	case *sandbox0.ExposedPortsResponse:
		return f.formatExposedPorts(w, v)
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
	t.Header([]string{"ID", "TEAM ID", "CACHE SIZE", "CREATED"})

	for _, v := range volumes {
		_ = t.Append([]string{
			v.ID,
			v.TeamID,
			v.CacheSize,
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
	_ = t.Append([]string{"Cache Size:", v.CacheSize})
	_ = t.Append([]string{"Buffer Size:", v.BufferSize})
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
	_ = t.Append([]string{"Provider:", c.Provider})
	_ = t.Append([]string{"Registry:", c.Registry})
	_ = t.Append([]string{"Username:", c.Username})
	_ = t.Append([]string{"Password:", c.Password})
	if c.ExpiresAt != "" {
		_ = t.Append([]string{"Expires At:", c.ExpiresAt})
	}
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

func (f *TableFormatter) formatNetworkPolicy(w io.Writer, policy *apispec.TplSandboxNetworkPolicy) error {
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
	}
	return t.Render()
}

func (f *TableFormatter) formatExposedPorts(w io.Writer, resp *sandbox0.ExposedPortsResponse) error {
	if len(resp.Ports) == 0 {
		_, _ = fmt.Fprintln(w, "No exposed ports.")
		if resp.ExposureDomain != "" {
			_, _ = fmt.Fprintf(w, "Exposure Domain: %s\n", resp.ExposureDomain)
		}
		return nil
	}

	t := newTable(w)
	t.Header([]string{"PORT", "RESUME", "PUBLIC URL"})

	for _, port := range resp.Ports {
		publicURL := port.PublicURL
		if publicURL == "" && resp.ExposureDomain != "" {
			publicURL = fmt.Sprintf("http://%d.%s", port.Port, resp.ExposureDomain)
		}
		_ = t.Append([]string{
			fmt.Sprintf("%d", port.Port),
			fmt.Sprintf("%v", port.Resume),
			publicURL,
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
