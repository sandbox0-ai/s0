package commands

import (
	"errors"
	"fmt"

	sandbox0 "github.com/sandbox0-ai/sdk-go"
)

func formatSandboxCreateError(err error) string {
	if err == nil {
		return ""
	}
	if !sandbox0.IsClaimStartThrottled(err) {
		return err.Error()
	}

	message := fmt.Sprintf("%v", err)
	var apiErr *sandbox0.APIError
	if errors.As(err, &apiErr) && apiErr.RetryAfterSeconds > 0 {
		return fmt.Sprintf("%s\nHint: sandbox claim/start capacity is temporarily throttled. Retry after %d %s.", message, apiErr.RetryAfterSeconds, pluralizeSecond(apiErr.RetryAfterSeconds))
	}
	return message + "\nHint: sandbox claim/start capacity is temporarily throttled. Retry the request shortly."
}

func pluralizeSecond(seconds int) string {
	if seconds == 1 {
		return "second"
	}
	return "seconds"
}
