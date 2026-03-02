package output

import (
	"encoding/json"
	"io"

	"gopkg.in/yaml.v3"
)

// YAMLFormatter formats output as YAML.
type YAMLFormatter struct{}

// Format writes the data as YAML to the writer.
func (f *YAMLFormatter) Format(w io.Writer, data interface{}) error {
	// First convert to JSON to handle Opt types correctly (they implement MarshalJSON)
	// Then convert JSON to a generic structure for YAML serialization
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	var generic interface{}
	if err := json.Unmarshal(jsonBytes, &generic); err != nil {
		return err
	}

	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	return encoder.Encode(generic)
}
