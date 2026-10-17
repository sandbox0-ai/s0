package output

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"

	"gopkg.in/yaml.v3"
)

// YAMLFormatter formats output as YAML.
type YAMLFormatter struct {
	showSecrets bool
}

// Format writes the data as YAML to the writer.
func (f *YAMLFormatter) Format(w io.Writer, data interface{}) error {
	data = redactSensitiveData(data, f.showSecrets)
	// First convert to JSON to handle Opt types correctly (they implement MarshalJSON)
	// Then convert JSON to a generic structure for YAML serialization
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	var generic interface{}
	decoder := json.NewDecoder(bytes.NewReader(jsonBytes))
	decoder.UseNumber()
	if err := decoder.Decode(&generic); err != nil {
		return err
	}

	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	return encoder.Encode(yamlExactNumbers(generic))
}

// Preserve runtime generations and other integer identities beyond float64's
// exact range when converting generated JSON models to YAML.
func yamlExactNumbers(value any) any {
	switch v := value.(type) {
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return n
		}
		if n, err := strconv.ParseUint(string(v), 10, 64); err == nil {
			return n
		}
		if n, err := v.Float64(); err == nil {
			return n
		}
		return string(v)
	case map[string]any:
		for k, item := range v {
			v[k] = yamlExactNumbers(item)
		}
	case []any:
		for i, item := range v {
			v[i] = yamlExactNumbers(item)
		}
	}
	return value
}
