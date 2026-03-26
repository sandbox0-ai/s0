package commands

import "testing"

func TestNeedsWorkerRecovery(t *testing.T) {
	tests := map[string]bool{
		"running":         false,
		"stopped":         false,
		"stale":           true,
		"error":           false,
		"reseed_required": false,
	}
	for input, want := range tests {
		if got := needsWorkerRecovery(input); got != want {
			t.Fatalf("needsWorkerRecovery(%q) = %t, want %t", input, got, want)
		}
	}
}
