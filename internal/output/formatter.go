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
	Format(w io.Writer, data interface{}) error
}

// NewFormatter creates a new formatter for the given format.
func NewFormatter(format Format) Formatter {
	switch format {
	case FormatJSON:
		return &JSONFormatter{}
	case FormatYAML:
		return &YAMLFormatter{}
	default:
		return &TableFormatter{}
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
