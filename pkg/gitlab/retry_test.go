package gitlab

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sgaunet/gitlab-backup/pkg/constants"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func TestIsRetryableStatusCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code     int
		expected bool
	}{
		{http.StatusTooManyRequests, true},
		{http.StatusInternalServerError, true},
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusGatewayTimeout, true},
		{http.StatusOK, false},
		{http.StatusBadRequest, false},
		{http.StatusUnauthorized, false},
		{http.StatusForbidden, false},
		{http.StatusNotFound, false},
		{http.StatusConflict, false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, isRetryableStatusCode(tt.code), "status code %d", tt.code)
	}
}

func TestRetryDelay(t *testing.T) {
	t.Parallel()

	// Verify exponential progression with ±10% jitter tolerance.
	tests := []struct {
		attempt     int
		expectedMs  float64
		description string
	}{
		{1, 1000, "first retry ~1s"},
		{2, 2000, "second retry ~2s"},
		{3, 4000, "third retry ~4s"},
		{10, float64(constants.RetryMaxDelayMs), "capped at max"},
	}

	for _, tt := range tests {
		delay := retryDelay(tt.attempt)
		delayMs := float64(delay.Milliseconds())

		// Allow ±10% jitter.
		lower := tt.expectedMs * 0.89
		upper := tt.expectedMs * 1.11

		assert.GreaterOrEqual(t, delayMs, lower, "%s: delay %v too low", tt.description, delay)
		assert.LessOrEqual(t, delayMs, upper, "%s: delay %v too high", tt.description, delay)
	}
}

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()

	t.Run("nil response", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, time.Duration(0), parseRetryAfter(nil))
	})

	t.Run("no header", func(t *testing.T) {
		t.Parallel()
		resp := &gitlab.Response{Response: &http.Response{Header: http.Header{}}}
		assert.Equal(t, time.Duration(0), parseRetryAfter(resp))
	})

	t.Run("seconds value", func(t *testing.T) {
		t.Parallel()
		header := http.Header{}
		header.Set("Retry-After", "120")
		resp := &gitlab.Response{Response: &http.Response{Header: header}}
		assert.Equal(t, 120*time.Second, parseRetryAfter(resp))
	})
}

func makeResponse(statusCode int) *gitlab.Response {
	return &gitlab.Response{
		Response: &http.Response{
			StatusCode: statusCode,
			Header:     http.Header{},
		},
	}
}

func TestRetryWithResponse_SuccessFirstAttempt(t *testing.T) {
	t.Parallel()

	calls := 0
	result, _, err := retryWithResponse(context.Background(), "test", func() (string, *gitlab.Response, error) {
		calls++
		return "ok", makeResponse(http.StatusOK), nil
	})

	require.NoError(t, err)
	assert.Equal(t, "ok", result)
	assert.Equal(t, 1, calls)
}

func TestRetryWithResponse_SuccessAfterRetries(t *testing.T) {
	t.Parallel()

	calls := 0
	result, _, err := retryWithResponse(context.Background(), "test", func() (string, *gitlab.Response, error) {
		calls++
		if calls <= 2 {
			return "", makeResponse(http.StatusServiceUnavailable), errors.New("unavailable")
		}
		return "ok", makeResponse(http.StatusOK), nil
	})

	require.NoError(t, err)
	assert.Equal(t, "ok", result)
	assert.Equal(t, 3, calls)
}

func TestRetryWithResponse_NonRetryableError(t *testing.T) {
	t.Parallel()

	calls := 0
	_, _, err := retryWithResponse(context.Background(), "test", func() (string, *gitlab.Response, error) {
		calls++
		return "", makeResponse(http.StatusUnauthorized), errors.New("unauthorized")
	})

	require.Error(t, err)
	assert.Equal(t, 1, calls)
	assert.False(t, errors.Is(err, ErrRetryExhausted))
}

func TestRetryWithResponse_MaxRetriesExhausted(t *testing.T) {
	t.Parallel()

	calls := 0
	_, _, err := retryWithResponse(context.Background(), "test", func() (string, *gitlab.Response, error) {
		calls++
		return "", makeResponse(http.StatusInternalServerError), errors.New("server error")
	})

	require.Error(t, err)
	assert.Equal(t, 1+constants.RetryMaxAttempts, calls)
	assert.True(t, errors.Is(err, ErrRetryExhausted))
}

func TestRetryWithResponse_ContextCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	calls := 0
	_, _, err := retryWithResponse(ctx, "test", func() (string, *gitlab.Response, error) {
		calls++
		cancel() // Cancel after first call.
		return "", makeResponse(http.StatusServiceUnavailable), errors.New("unavailable")
	})

	require.Error(t, err)
	assert.Equal(t, 1, calls)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestRetryWithResponse_NilResponse(t *testing.T) {
	t.Parallel()

	calls := 0
	result, _, err := retryWithResponse(context.Background(), "test", func() (string, *gitlab.Response, error) {
		calls++
		if calls <= 2 {
			return "", nil, errors.New("network error")
		}
		return "ok", makeResponse(http.StatusOK), nil
	})

	require.NoError(t, err)
	assert.Equal(t, "ok", result)
	assert.Equal(t, 3, calls)
}

func TestRetryResponseOnly_SuccessFirstAttempt(t *testing.T) {
	t.Parallel()

	calls := 0
	_, err := retryResponseOnly(context.Background(), "test", func() (*gitlab.Response, error) {
		calls++
		return makeResponse(http.StatusOK), nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestRetryResponseOnly_SuccessAfterRetries(t *testing.T) {
	t.Parallel()

	calls := 0
	_, err := retryResponseOnly(context.Background(), "test", func() (*gitlab.Response, error) {
		calls++
		if calls <= 2 {
			return makeResponse(http.StatusBadGateway), errors.New("bad gateway")
		}
		return makeResponse(http.StatusOK), nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestRetryResponseOnly_NonRetryableError(t *testing.T) {
	t.Parallel()

	calls := 0
	_, err := retryResponseOnly(context.Background(), "test", func() (*gitlab.Response, error) {
		calls++
		return makeResponse(http.StatusForbidden), errors.New("forbidden")
	})

	require.Error(t, err)
	assert.Equal(t, 1, calls)
	assert.False(t, errors.Is(err, ErrRetryExhausted))
}

func TestRetryResponseOnly_MaxRetriesExhausted(t *testing.T) {
	t.Parallel()

	calls := 0
	_, err := retryResponseOnly(context.Background(), "test", func() (*gitlab.Response, error) {
		calls++
		return makeResponse(http.StatusGatewayTimeout), errors.New("timeout")
	})

	require.Error(t, err)
	assert.Equal(t, 1+constants.RetryMaxAttempts, calls)
	assert.True(t, errors.Is(err, ErrRetryExhausted))
}

func TestRetryResponseOnly_NilResponse(t *testing.T) {
	t.Parallel()

	calls := 0
	_, err := retryResponseOnly(context.Background(), "test", func() (*gitlab.Response, error) {
		calls++
		if calls <= 1 {
			return nil, errors.New("network error")
		}
		return makeResponse(http.StatusOK), nil
	})

	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestShouldRetry(t *testing.T) {
	t.Parallel()

	assert.True(t, shouldRetry(nil), "nil response should retry")
	assert.True(t, shouldRetry(&gitlab.Response{Response: nil}), "nil HTTP response should retry")
	assert.True(t, shouldRetry(makeResponse(http.StatusServiceUnavailable)), "503 should retry")
	assert.False(t, shouldRetry(makeResponse(http.StatusOK)), "200 should not retry")
	assert.False(t, shouldRetry(makeResponse(http.StatusNotFound)), "404 should not retry")
}
