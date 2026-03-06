package output

import (
	"encoding/json"
	"io"
)

// JSONFormatter formats output as JSON.
type JSONFormatter struct {
	showSecrets bool
}

// Format writes the data as JSON to the writer.
func (f *JSONFormatter) Format(w io.Writer, data interface{}) error {
	data = redactSensitiveData(data, f.showSecrets)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}
