package commands

import "testing"

func TestBuildFunctionInvokeRequest(t *testing.T) {
	resetFunctionFlags()
	functionMethod = "POST"
	functionPath = "/hello"
	functionHandler = "custom_handler"
	functionHeader = []string{"x-value=ok", "x-value=again"}
	functionQuery = []string{"name=ada"}
	functionBodyData = `{"ok":true}`
	functionTimeoutMS = 1500
	t.Cleanup(resetFunctionFlags)

	req, err := buildFunctionInvokeRequest()
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if method, ok := req.Method.Get(); !ok || method != "POST" {
		t.Fatalf("method = %q set=%v, want POST", method, ok)
	}
	if path, ok := req.Path.Get(); !ok || path != "/hello" {
		t.Fatalf("path = %q set=%v, want /hello", path, ok)
	}
	if handler, ok := req.Handler.Get(); !ok || handler != "custom_handler" {
		t.Fatalf("handler = %q set=%v, want custom_handler", handler, ok)
	}
	headers, ok := req.Headers.Get()
	if !ok || len(headers["x-value"]) != 2 {
		t.Fatalf("headers = %#v, want repeated x-value", headers)
	}
	query, ok := req.Query.Get()
	if !ok || len(query["name"]) != 1 || query["name"][0] != "ada" {
		t.Fatalf("query = %#v, want name=ada", query)
	}
	if body, ok := req.BodyBase64.Get(); !ok || body != "eyJvayI6dHJ1ZX0=" {
		t.Fatalf("body = %q set=%v, want encoded JSON", body, ok)
	}
	if timeout, ok := req.TimeoutMs.Get(); !ok || timeout != 1500 {
		t.Fatalf("timeout = %d set=%v, want 1500", timeout, ok)
	}
}

func TestBuildFunctionInvokeRequestRejectsConflictingBodyFlags(t *testing.T) {
	resetFunctionFlags()
	functionBodyData = "body"
	functionBodyBase64 = "Ym9keQ=="
	t.Cleanup(resetFunctionFlags)

	if _, err := buildFunctionInvokeRequest(); err == nil {
		t.Fatal("expected conflicting body flags error")
	}
}

func TestBuildFunctionInvokeRequestRejectsNegativeTimeout(t *testing.T) {
	resetFunctionFlags()
	functionTimeoutMS = -1
	t.Cleanup(resetFunctionFlags)

	if _, err := buildFunctionInvokeRequest(); err == nil {
		t.Fatal("expected negative timeout error")
	}
}

func resetFunctionFlags() {
	functionMethod = ""
	functionPath = ""
	functionHandler = ""
	functionHeader = nil
	functionQuery = nil
	functionBodyStdin = false
	functionBodyData = ""
	functionBodyBase64 = ""
	functionTimeoutMS = 0
}
