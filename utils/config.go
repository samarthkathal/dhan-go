package utils

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// HTTPClientConfig holds configuration for HTTP client
type HTTPClientConfig struct {
	// Connection pool settings
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	MaxConnsPerHost     int
	IdleConnTimeout     time.Duration

	// Timeout settings
	DialTimeout           time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	ExpectContinueTimeout time.Duration

	// Keep alive
	KeepAlive time.Duration

	// TLS
	InsecureSkipVerify bool
}

// DefaultConfig returns a balanced configuration suitable for most use cases
func DefaultConfig() *HTTPClientConfig {
	return &HTTPClientConfig{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       10,
		IdleConnTimeout:       90 * time.Second,
		DialTimeout:           30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		KeepAlive:             30 * time.Second,
		InsecureSkipVerify:    false,
	}
}

// LowLatencyConfig returns configuration optimized for low latency
func LowLatencyConfig() *HTTPClientConfig {
	return &HTTPClientConfig{
		MaxIdleConns:          50,
		MaxIdleConnsPerHost:   5,
		MaxConnsPerHost:       5,
		IdleConnTimeout:       30 * time.Second,
		DialTimeout:           5 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		ExpectContinueTimeout: 500 * time.Millisecond,
		KeepAlive:             15 * time.Second,
		InsecureSkipVerify:    false,
	}
}

// HighThroughputConfig returns configuration optimized for high throughput
func HighThroughputConfig() *HTTPClientConfig {
	return &HTTPClientConfig{
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   20,
		MaxConnsPerHost:       20,
		IdleConnTimeout:       120 * time.Second,
		DialTimeout:           30 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		KeepAlive:             60 * time.Second,
		InsecureSkipVerify:    false,
	}
}

// NewHTTPClient creates an HTTP client with the given configuration
func NewHTTPClient(config *HTTPClientConfig) *http.Client {
	if config == nil {
		config = DefaultConfig()
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   config.DialTimeout,
			KeepAlive: config.KeepAlive,
		}).DialContext,
		MaxIdleConns:          config.MaxIdleConns,
		MaxIdleConnsPerHost:   config.MaxIdleConnsPerHost,
		MaxConnsPerHost:       config.MaxConnsPerHost,
		IdleConnTimeout:       config.IdleConnTimeout,
		TLSHandshakeTimeout:   config.TLSHandshakeTimeout,
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,
		ExpectContinueTimeout: config.ExpectContinueTimeout,
		ForceAttemptHTTP2:     true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
		},
	}

	return &http.Client{
		Transport: transport,
	}
}

// DefaultHTTPClient returns an HTTP client with default configuration
func DefaultHTTPClient() *http.Client {
	return NewHTTPClient(DefaultConfig())
}

// LowLatencyHTTPClient returns an HTTP client optimized for low latency
func LowLatencyHTTPClient() *http.Client {
	return NewHTTPClient(LowLatencyConfig())
}

// HighThroughputHTTPClient returns an HTTP client optimized for high throughput
func HighThroughputHTTPClient() *http.Client {
	return NewHTTPClient(HighThroughputConfig())
}

// WithMiddleware wraps an HTTP client's transport with middleware
func WithMiddleware(client *http.Client, wrappers ...func(http.RoundTripper) http.RoundTripper) *http.Client {
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	client.Transport = ChainRoundTrippers(transport, wrappers...)
	return client
}
