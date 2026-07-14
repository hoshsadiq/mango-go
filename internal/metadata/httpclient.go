package metadata

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

// RetryClient wraps an HTTP client with retry logic, exponential backoff, and rate limiting.
type RetryClient struct {
	client         *http.Client
	maxRetries     int
	initialBackoff time.Duration
	rateLimiter    *rate.Limiter
}

type Option func(*RetryClient)

func WithMaxRetries(maxRetries int) Option {
	return func(rc *RetryClient) {
		rc.maxRetries = maxRetries
	}
}

func WithInitialBackoff(backoff time.Duration) Option {
	return func(rc *RetryClient) {
		rc.initialBackoff = backoff
	}
}

func WithRateLimiter(limiter *rate.Limiter) Option {
	return func(rc *RetryClient) {
		rc.rateLimiter = limiter
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(rc *RetryClient) {
		rc.client = client
	}
}

func NewRetryClient(opts ...Option) *RetryClient {
	rc := &RetryClient{
		client:         &http.Client{},
		maxRetries:     3,
		initialBackoff: 2 * time.Second,
		rateLimiter:    nil,
	}

	for _, opt := range opts {
		opt(rc)
	}

	return rc
}

// NewAniListClient creates a RetryClient configured for AniList API.
// It has MaxRetries=3, InitialBackoff=2s, and a rate limiter of 15 requests per 10 seconds.
func NewAniListClient() *RetryClient {
	// 15 requests per 10 seconds = 1 request per 667ms
	limiter := rate.NewLimiter(rate.Every(667*time.Millisecond), 15)

	return NewRetryClient(
		WithMaxRetries(3),
		WithInitialBackoff(2*time.Second),
		WithRateLimiter(limiter),
	)
}

// Do executes the request with retries on 429 and 5xx, exponential backoff,
// and Retry-After support. Timeout errors are not retried.
func (rc *RetryClient) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	backoff := rc.initialBackoff

	// Buffer the request body so it can be replayed on retries.
	// http.Client.Do consumes req.Body, so without this POST retries send an empty body.
	if req.Body != nil && req.GetBody == nil {
		bodyBytes, err := io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
		req.Body, err = req.GetBody()
		if err != nil {
			return nil, fmt.Errorf("failed to reset request body: %w", err)
		}
	}

	for attempt := 0; attempt <= rc.maxRetries; attempt++ {
		if rc.rateLimiter != nil {
			if err := rc.rateLimiter.Wait(ctx); err != nil {
				return nil, fmt.Errorf("rate limiter error: %w", err)
			}
		}

		if attempt > 0 && req.GetBody != nil {
			var err error
			req.Body, err = req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("failed to reset request body for retry: %w", err)
			}
		}

		resp, err := rc.client.Do(req)

		if err != nil && ctx.Err() != nil {
			return nil, err
		}

		if err == nil && !isRetryableStatus(resp.StatusCode) {
			return resp, nil
		}

		if err != nil {
			return resp, err
		}

		if attempt == rc.maxRetries {
			return resp, nil
		}

		waitDuration := backoff

		if resp.StatusCode == http.StatusTooManyRequests {
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, err := strconv.Atoi(retryAfter); err == nil {
					waitDuration = time.Duration(seconds) * time.Second
				}
			}
		}

		deadline, ok := ctx.Deadline()
		if ok {
			timeRemaining := time.Until(deadline)
			if timeRemaining <= waitDuration {
				return resp, nil
			}
		}

		// Drain and close the response body before retry to prevent leaking connections
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		select {
		case <-time.After(waitDuration):
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		backoff *= 2
	}

	return nil, fmt.Errorf("unexpected: exhausted retries without returning")
}

func isRetryableStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || (statusCode >= 500 && statusCode < 600)
}
