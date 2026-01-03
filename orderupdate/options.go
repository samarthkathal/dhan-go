package orderupdate

import (
	"github.com/samarthkathal/dhan-go/middleware"
)

// Option is a functional option for configuring the order update client
type Option func(*Client)

// WithConfig sets a custom WebSocket configuration
func WithConfig(config *WebSocketConfig) Option {
	return func(c *Client) {
		c.config = config
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
