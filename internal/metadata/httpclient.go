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
	Client         *http.Client
	MaxRetries     int
	InitialBackoff time.Duration
	RateLimiter    *rate.Limiter
}

// Option is a functional option for configuring RetryClient.
type Option func(*RetryClient)

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(maxRetries int) Option {
	return func(rc *RetryClient) {
		rc.MaxRetries = maxRetries
	}
}

// WithInitialBackoff sets the initial backoff duration.
func WithInitialBackoff(backoff time.Duration) Option {
	return func(rc *RetryClient) {
		rc.InitialBackoff = backoff
	}
}

// WithRateLimiter sets the rate limiter.
func WithRateLimiter(limiter *rate.Limiter) Option {
	return func(rc *RetryClient) {
		rc.RateLimiter = limiter
	}
}

// WithHTTPClient sets the underlying HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(rc *RetryClient) {
		rc.Client = client
	}
}

// NewRetryClient creates a new RetryClient with the given options.
func NewRetryClient(opts ...Option) *RetryClient {
	rc := &RetryClient{
		Client:         &http.Client{},
		MaxRetries:     3,
		InitialBackoff: 2 * time.Second,
		RateLimiter:    nil,
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

// Do executes an HTTP request with retry logic, exponential backoff, and rate limiting.
// It retries on HTTP 429 (Too Many Requests) and 5xx errors.
// It respects the Retry-After header if present.
// It does not retry on timeout errors.
func (rc *RetryClient) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	backoff := rc.InitialBackoff

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

	for attempt := 0; attempt <= rc.MaxRetries; attempt++ {
		if rc.RateLimiter != nil {
			if err := rc.RateLimiter.Wait(ctx); err != nil {
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

		resp, err := rc.Client.Do(req)

		if err != nil && ctx.Err() != nil {
			return nil, err
		}

		if err == nil && !isRetryableStatus(resp.StatusCode) {
			return resp, nil
		}

		if err != nil {
			return resp, err
		}

		if attempt == rc.MaxRetries {
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

// isRetryableStatus returns true if the HTTP status code should trigger a retry.
func isRetryableStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || (statusCode >= 500 && statusCode < 600)
}
