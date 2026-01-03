package rest

import (
	"context"
	"net/http"

	"github.com/samarthkathal/dhan-go/internal/limiter"
	"github.com/samarthkathal/dhan-go/internal/restgen"
)

// clientConfig holds configuration for the REST client
type clientConfig struct {
	httpClient    *http.Client
	requestEditor restgen.RequestEditorFn
	rateLimiter   *limiter.HTTPRateLimiter
}

// Option is a functional option for configuring the REST client
type Option func(*clientConfig)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) Option {
	return func(cfg *clientConfig) {
		cfg.httpClient = client
	}
}

// WithRequestEditor sets a custom request editor for adding headers, logging, etc.
func WithRequestEditor(editor func(ctx context.Context, req *http.Request) error) Option {
	return func(cfg *clientConfig) {
		cfg.requestEditor = editor
	}
}

// WithRateLimiter enables rate limiting with a custom rate limiter
// If nil is passed, creates a new rate limiter with default Dhan limits
func WithRateLimiter(rateLimiter *limiter.HTTPRateLimiter) Option {
	return func(cfg *clientConfig) {
		if rateLimiter == nil {
			cfg.rateLimiter = limiter.NewHTTPRateLimiter()
		} else {
			cfg.rateLimiter = rateLimiter
		}
	}
}

// WithDefaultRateLimiter enables rate limiting with Dhan's default limits
func WithDefaultRateLimiter() Option {
	return WithRateLimiter(nil)
}
