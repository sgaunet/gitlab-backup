package gitlab

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"github.com/sgaunet/gitlab-backup/pkg/constants"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// isRetryableStatusCode returns true for HTTP status codes that indicate a transient failure.
func isRetryableStatusCode(code int) bool {
	switch code {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// Exponential backoff constants.
const (
	backoffBase   = 2   // Multiplier for exponential growth.
	jitterPercent = 0.1 // ±10% jitter range.
)

// retryDelay calculates the delay before the next retry attempt using exponential backoff with jitter.
// The delay is base * 2^(attempt-1), capped at max, with ±10% jitter.
func retryDelay(attempt int) time.Duration {
	base := float64(constants.RetryBaseDelayMs)
	delay := base * math.Pow(backoffBase, float64(attempt-1))

	maxDelay := float64(constants.RetryMaxDelayMs)
	if delay > maxDelay {
		delay = maxDelay
	}

	// Add ±10% jitter to avoid thundering herd.
	jitter := delay * jitterPercent * (backoffBase*rand.Float64() - 1) //nolint:gosec // Jitter doesn't need crypto rand
	delay += jitter

	return time.Duration(delay) * time.Millisecond
}

// parseRetryAfter extracts the Retry-After header value from an HTTP response.
// Returns the duration to wait, or 0 if the header is absent or unparsable.
func parseRetryAfter(resp *gitlab.Response) time.Duration {
	if resp == nil || resp.Response == nil {
		return 0
	}

	header := resp.Header.Get("Retry-After")
	if header == "" {
		return 0
	}

	// Try parsing as seconds (integer).
	if seconds, err := strconv.Atoi(header); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP-date.
	if t, err := http.ParseTime(header); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}

	return 0
}

// retryWithResponse retries a function that returns (T, *gitlab.Response, error).
// It retries on transient failures (network errors, retryable HTTP status codes).
//
//nolint:ireturn // Generic return type is intentional for retry helper
func retryWithResponse[T any](
	ctx context.Context,
	operation string,
	fn func() (T, *gitlab.Response, error),
) (T, *gitlab.Response, error) {
	var zero T

	result, resp, err := fn()
	if err == nil {
		return result, resp, nil
	}

	if !shouldRetry(resp) {
		return zero, resp, err
	}

	for attempt := 1; attempt <= constants.RetryMaxAttempts; attempt++ {
		delay := chooseDelay(attempt, resp)

		select {
		case <-ctx.Done():
			return zero, resp, fmt.Errorf("%s: %w: %w", operation, ctx.Err(), err)
		case <-time.After(delay):
		}

		result, resp, err = fn()
		if err == nil {
			return result, resp, nil
		}

		if !shouldRetry(resp) {
			return zero, resp, err
		}
	}

	return zero, resp, fmt.Errorf("%s: %w: %w", operation, ErrRetryExhausted, err)
}

// retryResponseOnly retries a function that returns (*gitlab.Response, error).
// It retries on transient failures (network errors, retryable HTTP status codes).
func retryResponseOnly(
	ctx context.Context,
	operation string,
	fn func() (*gitlab.Response, error),
) (*gitlab.Response, error) {
	resp, err := fn()
	if err == nil {
		return resp, nil
	}

	if !shouldRetry(resp) {
		return resp, err
	}

	for attempt := 1; attempt <= constants.RetryMaxAttempts; attempt++ {
		delay := chooseDelay(attempt, resp)

		select {
		case <-ctx.Done():
			return resp, fmt.Errorf("%s: %w: %w", operation, ctx.Err(), err)
		case <-time.After(delay):
		}

		resp, err = fn()
		if err == nil {
			return resp, nil
		}

		if !shouldRetry(resp) {
			return resp, err
		}
	}

	return resp, fmt.Errorf("%s: %w: %w", operation, ErrRetryExhausted, err)
}

// shouldRetry determines if a failed request should be retried based on the response.
// A nil response (network error) or a retryable status code triggers a retry.
func shouldRetry(resp *gitlab.Response) bool {
	if resp == nil || resp.Response == nil {
		return true // Network error — no response received.
	}

	return isRetryableStatusCode(resp.StatusCode)
}

// chooseDelay returns the delay before the next retry, preferring Retry-After header if present.
func chooseDelay(attempt int, resp *gitlab.Response) time.Duration {
	if ra := parseRetryAfter(resp); ra > 0 {
		return ra
	}

	return retryDelay(attempt)
}
