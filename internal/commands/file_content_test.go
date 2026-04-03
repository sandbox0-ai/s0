package commands

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, errors.New("boom")
}

func TestReadCommandContent(t *testing.T) {
	t.Run("stdin", func(t *testing.T) {
		got, err := readCommandContent(true, "", strings.NewReader("hello\nworld"))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if string(got) != "hello\nworld" {
			t.Fatalf("unexpected content %q", string(got))
		}
	})

	t.Run("data", func(t *testing.T) {
		got, err := readCommandContent(false, "hello", io.NopCloser(strings.NewReader("")))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if string(got) != "hello" {
			t.Fatalf("unexpected content %q", string(got))
		}
	})

	t.Run("missing input", func(t *testing.T) {
		_, err := readCommandContent(false, "", strings.NewReader(""))
		if err == nil || err.Error() != "must specify either --stdin or --data" {
			t.Fatalf("unexpected error %v", err)
		}
	})

	t.Run("stdin read error", func(t *testing.T) {
		_, err := readCommandContent(true, "", errReader{})
		if err == nil || !strings.Contains(err.Error(), "read stdin") {
			t.Fatalf("unexpected error %v", err)
		}
	})
}
