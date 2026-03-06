package output

import (
	"io"
)

// Format represents an output format.
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

// Formatter is the interface for output formatters.
type Formatter interface {
	// Format writes the formatted output to the writer.
	Format(w io.Writer, data any) error
}

// Options controls formatter behavior.
type Options struct {
	ShowSecrets bool
}

// NewFormatter creates a new formatter for the given format.
func NewFormatter(format Format) Formatter {
	return NewFormatterWithOptions(format, Options{})
}

// NewFormatterWithOptions creates a new formatter with options.
func NewFormatterWithOptions(format Format, opts Options) Formatter {
	switch format {
	case FormatJSON:
		return &JSONFormatter{showSecrets: opts.ShowSecrets}
	case FormatYAML:
		return &YAMLFormatter{showSecrets: opts.ShowSecrets}
	default:
		return &TableFormatter{showSecrets: opts.ShowSecrets}
	}
}

// ParseFormat parses a string into a Format.
func ParseFormat(s string) Format {
	switch s {
	case "json":
		return FormatJSON
	case "yaml":
		return FormatYAML
	default:
		return FormatTable
	}
}
