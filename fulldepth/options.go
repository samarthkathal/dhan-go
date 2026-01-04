package fulldepth

import (
	"time"
)

// Config holds configuration for the Full Depth client
type Config struct {
	DepthLevel       DepthLevel    // 20 or 200 depth levels
	ConnectTimeout   time.Duration // Connection timeout
	ReadTimeout      time.Duration // Read timeout
	WriteTimeout     time.Duration // Write timeout
	PingInterval     time.Duration // Ping interval for keepalive
	ReconnectDelay   time.Duration // Delay between reconnection attempts
	MaxReconnects    int           // Maximum reconnection attempts (0 = unlimited)
	ReadBufferSize   int           // WebSocket read buffer size
	WriteBufferSize  int           // WebSocket write buffer size
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		DepthLevel:      Depth20,
		ConnectTimeout:  30 * time.Second,
		ReadTimeout:     0, // No read timeout
		WriteTimeout:    10 * time.Second,
		PingInterval:    10 * time.Second,
		ReconnectDelay:  5 * time.Second,
		MaxReconnects:   0, // Unlimited
		ReadBufferSize:  16384,
		WriteBufferSize: 4096,
	}
}

// Option is a functional option for configuring the client
type Option func(*Client)

// WithConfig sets a custom configuration
func WithConfig(config *Config) Option {
	return func(c *Client) {
		c.config = config
	}
}

// WithDepthLevel sets the depth level (20 or 200)
func WithDepthLevel(level DepthLevel) Option {
	return func(c *Client) {
		c.config.DepthLevel = level
	}
}

// WithDepthCallback registers a callback for depth updates
func WithDepthCallback(cb DepthCallback) Option {
	return func(c *Client) {
		c.depthCallbacks = append(c.depthCallbacks, cb)
	}
}

// WithErrorCallback registers an error callback
func WithErrorCallback(cb ErrorCallback) Option {
	return func(c *Client) {
		c.errorCallbacks = append(c.errorCallbacks, cb)
	}
}

// WithConnectTimeout sets the connection timeout
func WithConnectTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.config.ConnectTimeout = timeout
	}
}

// WithReconnectDelay sets the reconnection delay
func WithReconnectDelay(delay time.Duration) Option {
	return func(c *Client) {
		c.config.ReconnectDelay = delay
	}
}

// WithMaxReconnects sets the maximum number of reconnection attempts
func WithMaxReconnects(max int) Option {
	return func(c *Client) {
		c.config.MaxReconnects = max
	}
}
