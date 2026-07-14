package metadata

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// TestRetryOn429 verifies that the client retries on 429 (Too Many Requests)
func TestRetryOn429(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	client := NewRetryClient(
		WithMaxRetries(3),
		WithInitialBackoff(10*time.Millisecond),
	)

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if requestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", requestCount)
	}
}

// TestRetryOn500 verifies that the client retries on 5xx errors
func TestRetryOn500(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	client := NewRetryClient(
		WithMaxRetries(3),
		WithInitialBackoff(10*time.Millisecond),
	)

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if requestCount != 3 {
		t.Fatalf("expected 3 requests, got %d", requestCount)
	}
}

// TestExponentialBackoffTiming verifies that backoff delays increase exponentially
func TestExponentialBackoffTiming(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount < 4 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewRetryClient(
		WithMaxRetries(3),
		WithInitialBackoff(20*time.Millisecond),
	)

	start := time.Now()
	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)

	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Expected delays: 20ms + 40ms + 80ms = 140ms minimum
	// Allow some tolerance for system variance
	expectedMin := 140 * time.Millisecond
	if elapsed < expectedMin {
		t.Fatalf("expected elapsed >= %v, got %v", expectedMin, elapsed)
	}
}

// TestRetryAfterHeader verifies that Retry-After header is respected
func TestRetryAfterHeader(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewRetryClient(
		WithMaxRetries(3),
		WithInitialBackoff(10*time.Millisecond),
	)

	start := time.Now()
	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)

	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Retry-After: 1 second should be respected
	if elapsed < 1*time.Second {
		t.Fatalf("expected elapsed >= 1s (Retry-After), got %v", elapsed)
	}
}

// TestMaxRetriesExhausted verifies that error is returned after max retries
func TestMaxRetriesExhausted(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewRetryClient(
		WithMaxRetries(2),
		WithInitialBackoff(10*time.Millisecond),
	)

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, _ := client.Do(req)

	// Should return the last response (500)
	if resp == nil || resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 response, got %v", resp)
	}
	if requestCount != 3 {
		t.Fatalf("expected 3 requests (1 initial + 2 retries), got %d", requestCount)
	}
}

// TestRateLimiterThrottles verifies that rate limiter throttles requests
func TestRateLimiterThrottles(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a rate limiter: 2 requests per 100ms
	limiter := rate.NewLimiter(rate.Every(50*time.Millisecond), 1)

	client := NewRetryClient(
		WithRateLimiter(limiter),
	)

	start := time.Now()

	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest("GET", server.URL, nil)
		_, err := client.Do(req)
		if err != nil {
			t.Fatalf("request %d failed: %v", i+1, err)
		}
	}

	elapsed := time.Since(start)

	// With rate limiter at 50ms per request:
	// Request 1: immediate (0ms)
	// Request 2: wait 50ms
	// Request 3: wait 50ms
	// Total: ~100ms minimum
	expectedMin := 100 * time.Millisecond
	if elapsed < expectedMin {
		t.Fatalf("expected elapsed >= %v (rate limiting), got %v", expectedMin, elapsed)
	}
}

// TestTimeoutNoRetry verifies that requests exceeding timeout are not retried
func TestTimeoutNoRetry(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		// Simulate a slow response that will exceed timeout
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewRetryClient(
		WithMaxRetries(3),
		WithInitialBackoff(10*time.Millisecond),
		WithHTTPClient(&http.Client{
			Timeout: 50 * time.Millisecond,
		}),
	)

	req, _ := http.NewRequest("GET", server.URL, nil)
	_, err := client.Do(req)

	// Should fail with timeout error, not retry
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout error, got: %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("expected 1 request (no retries on timeout), got %d", requestCount)
	}
}

// TestRetryAfterParsingError verifies graceful handling of invalid Retry-After
func TestRetryAfterParsingError(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.Header().Set("Retry-After", "invalid")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewRetryClient(
		WithMaxRetries(3),
		WithInitialBackoff(10*time.Millisecond),
	)

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	// Should fall back to exponential backoff when Retry-After is invalid
	if requestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", requestCount)
	}
}

// TestNoRetryOn200 verifies that successful responses are not retried
func TestNoRetryOn200(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	client := NewRetryClient(
		WithMaxRetries(3),
		WithInitialBackoff(10*time.Millisecond),
	)

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if requestCount != 1 {
		t.Fatalf("expected 1 request (no retry on success), got %d", requestCount)
	}
}

// TestNewAniListClient verifies the AniList convenience constructor produces a usable client.
func TestNewAniListClient(t *testing.T) {
	t.Parallel()
	client := NewAniListClient()

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("expected successful request, got error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// TestContextCancellation verifies that context cancellation is respected
func TestContextCancellation(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewRetryClient(
		WithMaxRetries(3),
		WithInitialBackoff(10*time.Millisecond),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	_, err := client.Do(req)

	if err == nil {
		t.Fatalf("expected context deadline error, got nil")
	}
}

// TestRetryOn502 verifies retry on 502 Bad Gateway
func TestRetryOn502(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewRetryClient(
		WithMaxRetries(3),
		WithInitialBackoff(10*time.Millisecond),
	)

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if requestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", requestCount)
	}
}

// TestRetryOn503 verifies retry on 503 Service Unavailable
func TestRetryOn503(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewRetryClient(
		WithMaxRetries(3),
		WithInitialBackoff(10*time.Millisecond),
	)

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if requestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", requestCount)
	}
}

// TestPostRetryPreservesBody verifies that POST request bodies are preserved across retries
func TestPostRetryPreservesBody(t *testing.T) {
	t.Parallel()
	expectedBody := `{"query":"test query","variables":{"id":123}}`
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("request %d: failed to read body: %v", requestCount, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if string(body) != expectedBody {
			t.Errorf("request %d: expected body %q, got %q", requestCount, expectedBody, string(body))
		}
		if requestCount < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewRetryClient(
		WithMaxRetries(3),
		WithInitialBackoff(10*time.Millisecond),
	)

	req, _ := http.NewRequest("POST", server.URL, strings.NewReader(expectedBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if requestCount != 3 {
		t.Fatalf("expected 3 requests, got %d", requestCount)
	}
}

// TestNoRetryOn4xx verifies that 4xx errors (except 429) are not retried
func TestNoRetryOn4xx(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewRetryClient(
		WithMaxRetries(3),
		WithInitialBackoff(10*time.Millisecond),
	)

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if requestCount != 1 {
		t.Fatalf("expected 1 request (no retry on 4xx), got %d", requestCount)
	}
}

// TestRetryAfterWithZeroSeconds verifies Retry-After parsing with zero seconds
func TestRetryAfterWithZeroSeconds(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewRetryClient(
		WithMaxRetries(3),
		WithInitialBackoff(100*time.Millisecond),
	)

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if requestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", requestCount)
	}
}

// TestRateLimiterWithRetry verifies that rate limiter works with retries
func TestRateLimiterWithRetry(t *testing.T) {
	t.Parallel()
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	limiter := rate.NewLimiter(rate.Every(50*time.Millisecond), 1)
	client := NewRetryClient(
		WithMaxRetries(3),
		WithInitialBackoff(10*time.Millisecond),
		WithRateLimiter(limiter),
	)

	start := time.Now()
	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := client.Do(req)

	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Rate limiter should throttle both requests
	// Request 1: immediate (rate limiter allows)
	// Request 2: wait 50ms (rate limiter) + 10ms (backoff) = 60ms
	expectedMin := 50 * time.Millisecond
	if elapsed < expectedMin {
		t.Fatalf("expected elapsed >= %v (rate limiting), got %v", expectedMin, elapsed)
	}
}
