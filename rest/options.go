package rest

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/samarthkathal/dhan-go/internal/limiter"
	"github.com/samarthkathal/dhan-go/internal/restgen"
)

// clientConfig holds configuration for the REST client
type clientConfig struct {
	httpClient    *http.Client
	requestEditor restgen.RequestEditorFn
	rateLimiter   *limiter.HTTPRateLimiter
	logger        *zerolog.Logger
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

// SecureHTTPClient creates an HTTP client with secure defaults:
// - TLS 1.2 minimum
// - 30 second timeout
// - Connection pooling with sensible limits
// - Keep-alive enabled
func SecureHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			ForceAttemptHTTP2:   true,
		},
	}
}

// WithSecureDefaults configures the client with secure HTTP settings
func WithSecureDefaults() Option {
	return func(cfg *clientConfig) {
		cfg.httpClient = SecureHTTPClient()
	}
}

// WithLogger sets a zerolog logger for debug logging of API responses
// When enabled, logs each response with function name, status code, and body
func WithLogger(logger *zerolog.Logger) Option {
	return func(cfg *clientConfig) {
		cfg.logger = logger
	}
}
