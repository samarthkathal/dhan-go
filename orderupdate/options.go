package orderupdate

import (
	"github.com/samarthkathal/dhan-go/middleware"
	"github.com/samarthkathal/dhan-go/metrics"
)

// PooledOption is a functional option for configuring the pooled order update client
type PooledOption func(*PooledClient)

// WithPooledConfig sets a custom WebSocket configuration for the pooled client
func WithPooledConfig(config *WebSocketConfig) PooledOption {
	return func(c *PooledClient) {
		c.config = config
	}
}

// WithPooledMetrics sets a custom metrics collector for the pooled client
func WithPooledMetrics(collector *metrics.WSCollector) PooledOption {
	return func(c *PooledClient) {
		c.metrics = collector
	}
}

// WithPooledMiddleware sets custom WebSocket middleware for the pooled client
func WithPooledMiddleware(mw middleware.WSMiddleware) PooledOption {
	return func(c *PooledClient) {
		c.middleware = mw
	}
}

// WithPooledOrderUpdateCallback registers an order update callback for the pooled client
func WithPooledOrderUpdateCallback(cb OrderUpdateCallback) PooledOption {
	return func(c *PooledClient) {
		c.orderUpdateCallbacks = append(c.orderUpdateCallbacks, cb)
	}
}

// WithPooledErrorCallback registers an error callback for the pooled client
func WithPooledErrorCallback(cb ErrorCallback) PooledOption {
	return func(c *PooledClient) {
		c.errorCallbacks = append(c.errorCallbacks, cb)
	}
}

// Option is a functional option for configuring the single-connection order update client
type Option func(*Client)

// WithConfig sets a custom WebSocket configuration
func WithConfig(config *WebSocketConfig) Option {
	return func(c *Client) {
		c.config = config
	}
}

// WithMetrics sets a custom metrics collector
func WithMetrics(collector *metrics.WSCollector) Option {
	return func(c *Client) {
		c.metrics = collector
	}
}

// WithMiddleware sets custom WebSocket middleware
func WithMiddleware(mw middleware.WSMiddleware) Option {
	return func(c *Client) {
		c.middleware = mw
	}
}

// WithOrderUpdateCallback registers an order update callback
func WithOrderUpdateCallback(cb OrderUpdateCallback) Option {
	return func(c *Client) {
		c.orderUpdateCallbacks = append(c.orderUpdateCallbacks, cb)
	}
}

// WithErrorCallback registers an error callback
func WithErrorCallback(cb ErrorCallback) Option {
	return func(c *Client) {
		c.errorCallbacks = append(c.errorCallbacks, cb)
	}
}
