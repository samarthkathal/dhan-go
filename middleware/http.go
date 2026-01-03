package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"sync"
	"time"
)

// RoundTripperFunc is an adapter to allow using functions as http.RoundTripper
type RoundTripperFunc func(*http.Request) (*http.Response, error)

func (f RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// ChainRoundTrippers composes multiple RoundTripper wrappers
// Wrappers are applied in order: first wrapper is outermost
func ChainRoundTrippers(transport http.RoundTripper, wrappers ...func(http.RoundTripper) http.RoundTripper) http.RoundTripper {
	result := transport
	// Apply in reverse order so first wrapper is outermost
	for i := len(wrappers) - 1; i >= 0; i-- {
		result = wrappers[i](result)
	}
	return result
}

// RateLimitRoundTripper implements token bucket rate limiting
func RateLimitRoundTripper(rate float64, burst int) func(http.RoundTripper) http.RoundTripper {
	limiter := &tokenBucketLimiter{
		rate:       rate,
		burst:      burst,
		tokens:     float64(burst),
		lastUpdate: time.Now(),
	}

	return func(next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if err := limiter.Wait(req.Context()); err != nil {
				return nil, err
			}
			return next.RoundTrip(req)
		})
	}
}

// tokenBucketLimiter implements token bucket rate limiting
type tokenBucketLimiter struct {
	mu         sync.Mutex
	rate       float64   // tokens per second
	burst      int       // maximum tokens
	tokens     float64   // current tokens
	lastUpdate time.Time // last update time
}

func (tb *tokenBucketLimiter) Wait(ctx context.Context) error {
	tb.mu.Lock()

	now := time.Now()
	elapsed := now.Sub(tb.lastUpdate).Seconds()
	tb.tokens = min(float64(tb.burst), tb.tokens+elapsed*tb.rate)
	tb.lastUpdate = now

	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		tb.mu.Unlock()
		return nil
	}

	waitTime := time.Duration((1.0-tb.tokens)/tb.rate*1000) * time.Millisecond
	tb.mu.Unlock()

	select {
	case <-time.After(waitTime):
		return tb.Wait(ctx)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// LoggingRoundTripper logs HTTP requests and responses
func LoggingRoundTripper(logger *log.Logger) func(http.RoundTripper) http.RoundTripper {
	if logger == nil {
		logger = log.Default()
	}

	return func(next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			start := time.Now()

			logger.Printf("[HTTP] --> %s %s", req.Method, req.URL.Path)

			resp, err := next.RoundTrip(req)

			duration := time.Since(start)

			if err != nil {
				logger.Printf("[HTTP] <-- %s %s [ERROR] %v (%v)", req.Method, req.URL.Path, err, duration)
			} else {
				logger.Printf("[HTTP] <-- %s %s [%d] (%v)", req.Method, req.URL.Path, resp.StatusCode, duration)
			}

			return resp, err
		})
	}
}

// RecoveryRoundTripper recovers from panics in HTTP requests
func RecoveryRoundTripper(logger *log.Logger) func(http.RoundTripper) http.RoundTripper {
	if logger == nil {
		logger = log.Default()
	}

	return func(next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(req *http.Request) (resp *http.Response, err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.Printf("[PANIC] Recovered from panic in HTTP request: %v\n%s", r, debug.Stack())
					err = fmt.Errorf("panic recovered: %v", r)
					resp = nil
				}
			}()

			return next.RoundTrip(req)
		})
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
