package commands

import (
	"fmt"
	"io"
)

func readCommandContent(stdin bool, data string, r io.Reader) ([]byte, error) {
	if stdin {
		content, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		return content, nil
	}
	if data != "" {
		return []byte(data), nil
	}
	return nil, fmt.Errorf("must specify either --stdin or --data")
}
