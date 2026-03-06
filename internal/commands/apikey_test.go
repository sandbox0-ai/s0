package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sandbox0-ai/sdk-go/pkg/apispec"
)

func TestPrintCreatedAPIKeyRaw(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var out bytes.Buffer
		data := &apispec.CreateAPIKeyResponse{
			Key: apispec.NewOptString("s0_test_secret"),
		}

		if err := printCreatedAPIKeyRaw(&out, data); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got := out.String(); got != "s0_test_secret\n" {
			t.Fatalf("unexpected output %q", got)
		}
	})

	t.Run("missing key", func(t *testing.T) {
		var out bytes.Buffer
		data := &apispec.CreateAPIKeyResponse{}

		err := printCreatedAPIKeyRaw(&out, data)
		if err == nil || !strings.Contains(err.Error(), "not returned") {
			t.Fatalf("expected missing key error, got %v", err)
		}
	})

	t.Run("blank key", func(t *testing.T) {
		var out bytes.Buffer
		data := &apispec.CreateAPIKeyResponse{
			Key: apispec.NewOptString("   "),
		}

		err := printCreatedAPIKeyRaw(&out, data)
		if err == nil || !strings.Contains(err.Error(), "not returned") {
			t.Fatalf("expected blank key error, got %v", err)
		}
	})

	t.Run("nil data", func(t *testing.T) {
		var out bytes.Buffer
		err := printCreatedAPIKeyRaw(&out, nil)
		if err == nil || !strings.Contains(err.Error(), "missing API key data") {
			t.Fatalf("expected nil data error, got %v", err)
		}
	})
}
