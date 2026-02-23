package output

import (
	"fmt"
	"io"
	"os"

	"github.com/sandbox0-ai/s0/internal/client"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	sandbox0 "github.com/sandbox0-ai/sdk-go"
	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

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
	_ = t.Append([]string{"Claimed At:", s.ClaimedAt.Format("2006-01-02 15:04:05")})
	_ = t.Append([]string{"Expires At:", s.ExpiresAt.Format("2006-01-02 15:04:05")})
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
		_ = t.Append([]string{"Expires At:", v})
	}
	if v, ok := s.CreatedAt.Get(); ok {
		_ = t.Append([]string{"Created At:", v})
	}
	return t.Render()
}

func (f *TableFormatter) formatRefreshResponse(w io.Writer, r *apispec.RefreshResponse) error {
	t := newTable(w)
	_ = t.Append([]string{"Sandbox ID:", r.SandboxID})
	_ = t.Append([]string{"Expires At:", r.ExpiresAt.Format("2006-01-02 15:04:05")})
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
	t.Header([]string{"ID", "TEMPLATE ID", "STATUS", "PAUSED", "CREATED AT", "EXPIRES AT"})

	for _, s := range r.Sandboxes {
		_ = t.Append([]string{
			s.ID,
			s.TemplateID,
			string(s.Status),
			fmt.Sprintf("%v", s.Paused),
			s.CreatedAt.Format("2006-01-02 15:04:05"),
			s.ExpiresAt.Format("2006-01-02 15:04:05"),
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
