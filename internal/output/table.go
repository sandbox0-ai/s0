package output

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

// TableFormatter formats output as a table.
type TableFormatter struct{}

// Format writes the data as a table to the writer.
func (f *TableFormatter) Format(w io.Writer, data interface{}) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer tw.Flush()

	switch v := data.(type) {
	case []apispec.Template:
		return f.formatTemplates(tw, v)
	case *apispec.Template:
		return f.formatTemplate(w, v)
	case []apispec.SandboxVolume:
		return f.formatVolumes(tw, v)
	case *apispec.SandboxVolume:
		return f.formatVolume(w, v)
	case []apispec.Snapshot:
		return f.formatSnapshots(tw, v)
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
	case string:
		fmt.Fprintln(w, v)
		return nil
	default:
		// Fallback to simple printing
		fmt.Fprintf(w, "%v\n", data)
		return nil
	}
}

func (f *TableFormatter) formatTemplates(w io.Writer, templates []apispec.Template) error {
	fmt.Fprintln(w, "TEMPLATE_ID\tSCOPE\tCREATED_AT")
	for _, t := range templates {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			t.TemplateID,
			t.Scope,
			t.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}
	return nil
}

func (f *TableFormatter) formatTemplate(w io.Writer, t *apispec.Template) error {
	fmt.Fprintf(w, "Template ID:\t%s\n", t.TemplateID)
	fmt.Fprintf(w, "Scope:\t%s\n", t.Scope)
	if v, ok := t.TeamID.Get(); ok {
		fmt.Fprintf(w, "Team ID:\t%s\n", v)
	}
	if v, ok := t.UserID.Get(); ok {
		fmt.Fprintf(w, "User ID:\t%s\n", v)
	}
	fmt.Fprintf(w, "Created:\t%s\n", t.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Updated:\t%s\n", t.UpdatedAt.Format("2006-01-02 15:04:05"))
	return nil
}

func (f *TableFormatter) formatVolumes(w io.Writer, volumes []apispec.SandboxVolume) error {
	fmt.Fprintln(w, "ID\tTEAM_ID\tCACHE_SIZE\tCREATED")
	for _, v := range volumes {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			v.ID,
			v.TeamID,
			v.CacheSize,
			v.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}
	return nil
}

func (f *TableFormatter) formatVolume(w io.Writer, v *apispec.SandboxVolume) error {
	fmt.Fprintf(w, "ID:\t%s\n", v.ID)
	fmt.Fprintf(w, "Team ID:\t%s\n", v.TeamID)
	fmt.Fprintf(w, "User ID:\t%s\n", v.UserID)
	fmt.Fprintf(w, "Cache Size:\t%s\n", v.CacheSize)
	fmt.Fprintf(w, "Buffer Size:\t%s\n", v.BufferSize)
	fmt.Fprintf(w, "Created:\t%s\n", v.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Updated:\t%s\n", v.UpdatedAt.Format("2006-01-02 15:04:05"))
	return nil
}

func (f *TableFormatter) formatSnapshots(w io.Writer, snapshots []apispec.Snapshot) error {
	fmt.Fprintln(w, "ID\tNAME\tSIZE\tCREATED")
	for _, s := range snapshots {
		name := s.Name
		if name == "" {
			name = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%d bytes\t%s\n",
			s.ID,
			name,
			s.SizeBytes,
			s.CreatedAt,
		)
	}
	return nil
}

func (f *TableFormatter) formatSnapshot(w io.Writer, s *apispec.Snapshot) error {
	fmt.Fprintf(w, "ID:\t%s\n", s.ID)
	fmt.Fprintf(w, "Volume ID:\t%s\n", s.VolumeID)
	fmt.Fprintf(w, "Name:\t%s\n", s.Name)
	if v, ok := s.Description.Get(); ok {
		fmt.Fprintf(w, "Description:\t%s\n", v)
	}
	fmt.Fprintf(w, "Size:\t%d bytes\n", s.SizeBytes)
	fmt.Fprintf(w, "Created:\t%s\n", s.CreatedAt)
	return nil
}

func (f *TableFormatter) formatSandbox(w io.Writer, s *apispec.Sandbox) error {
	fmt.Fprintf(w, "ID:\t%s\n", s.ID)
	fmt.Fprintf(w, "Template ID:\t%s\n", s.TemplateID)
	fmt.Fprintf(w, "Team ID:\t%s\n", s.TeamID)
	if v, ok := s.UserID.Get(); ok {
		fmt.Fprintf(w, "User ID:\t%s\n", v)
	}
	fmt.Fprintf(w, "Status:\t%s\n", s.Status)
	fmt.Fprintf(w, "Paused:\t%v\n", s.Paused)
	fmt.Fprintf(w, "Pod Name:\t%s\n", s.PodName)
	fmt.Fprintf(w, "Claimed At:\t%s\n", s.ClaimedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "Expires At:\t%s\n", s.ExpiresAt.Format("2006-01-02 15:04:05"))
	return nil
}

func (f *TableFormatter) formatSandboxStatus(w io.Writer, s *apispec.SandboxStatus) error {
	if v, ok := s.Status.Get(); ok {
		fmt.Fprintf(w, "Status:\t%s\n", v)
	}
	if v, ok := s.ClaimedAt.Get(); ok {
		fmt.Fprintf(w, "Claimed At:\t%s\n", v)
	}
	if v, ok := s.ExpiresAt.Get(); ok {
		fmt.Fprintf(w, "Expires At:\t%s\n", v)
	}
	if v, ok := s.CreatedAt.Get(); ok {
		fmt.Fprintf(w, "Created At:\t%s\n", v)
	}
	return nil
}

func (f *TableFormatter) formatRefreshResponse(w io.Writer, r *apispec.RefreshResponse) error {
	fmt.Fprintf(w, "Sandbox ID:\t%s\n", r.SandboxID)
	fmt.Fprintf(w, "Expires At:\t%s\n", r.ExpiresAt.Format("2006-01-02 15:04:05"))
	return nil
}

func (f *TableFormatter) formatSuccessMessage(w io.Writer, r *apispec.SuccessMessageResponse) error {
	fmt.Fprintf(w, "Success:\t%v\n", r.Success)
	if v, ok := r.Data.Get(); ok {
		if msg, ok := v.Message.Get(); ok {
			fmt.Fprintf(w, "Message:\t%s\n", msg)
		}
	}
	return nil
}

func (f *TableFormatter) formatSuccessDeleted(w io.Writer, r *apispec.SuccessDeletedResponse) error {
	fmt.Fprintf(w, "Message:\tResource deleted successfully\n")
	return nil
}

// PrintTable is a helper for printing tabular data.
func PrintTable(headers []string, rows [][]string) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print headers
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(tw, "\t")
		}
		fmt.Fprint(tw, h)
	}
	fmt.Fprintln(tw)

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(tw, "\t")
			}
			fmt.Fprint(tw, cell)
		}
		fmt.Fprintln(tw)
	}

	tw.Flush()
}
