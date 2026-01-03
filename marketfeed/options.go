package marketfeed

import (
	"github.com/samarthkathal/dhan-go/middleware"
)

// PooledOption is a functional option for configuring the pooled market feed client
type PooledOption func(*PooledClient)

// WithPooledConfig sets a custom WebSocket configuration for the pooled client
func WithPooledConfig(config *WebSocketConfig) PooledOption {
	return func(c *PooledClient) {
		c.config = config
	}
}

// WithPooledMiddleware sets custom WebSocket middleware for the pooled client
func WithPooledMiddleware(mw middleware.WSMiddleware) PooledOption {
	return func(c *PooledClient) {
		c.middleware = mw
	}
}

// WithPooledTickerCallback registers a ticker data callback for the pooled client
func WithPooledTickerCallback(cb TickerCallback) PooledOption {
	return func(c *PooledClient) {
		c.tickerCallbacks = append(c.tickerCallbacks, cb)
	}
}

// WithPooledQuoteCallback registers a quote data callback for the pooled client
func WithPooledQuoteCallback(cb QuoteCallback) PooledOption {
	return func(c *PooledClient) {
		c.quoteCallbacks = append(c.quoteCallbacks, cb)
	}
}

// WithPooledOICallback registers an open interest callback for the pooled client
func WithPooledOICallback(cb OICallback) PooledOption {
	return func(c *PooledClient) {
		c.oiCallbacks = append(c.oiCallbacks, cb)
	}
}

// WithPooledPrevCloseCallback registers a previous close callback for the pooled client
func WithPooledPrevCloseCallback(cb PrevCloseCallback) PooledOption {
	return func(c *PooledClient) {
		c.prevCloseCallbacks = append(c.prevCloseCallbacks, cb)
	}
}

// WithPooledFullCallback registers a full data callback for the pooled client
func WithPooledFullCallback(cb FullCallback) PooledOption {
	return func(c *PooledClient) {
		c.fullCallbacks = append(c.fullCallbacks, cb)
	}
}

// WithPooledErrorCallback registers an error callback for the pooled client
func WithPooledErrorCallback(cb ErrorCallback) PooledOption {
	return func(c *PooledClient) {
		c.errorCallbacks = append(c.errorCallbacks, cb)
	}
}

// Option is a functional option for configuring the single-connection market feed client
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

// WithTickerCallback registers a ticker data callback
func WithTickerCallback(cb TickerCallback) Option {
	return func(c *Client) {
		c.tickerCallbacks = append(c.tickerCallbacks, cb)
	}
}

// WithQuoteCallback registers a quote data callback
func WithQuoteCallback(cb QuoteCallback) Option {
	return func(c *Client) {
		c.quoteCallbacks = append(c.quoteCallbacks, cb)
	}
}

// WithOICallback registers an open interest callback
func WithOICallback(cb OICallback) Option {
	return func(c *Client) {
		c.oiCallbacks = append(c.oiCallbacks, cb)
	}
}

// WithPrevCloseCallback registers a previous close callback
func WithPrevCloseCallback(cb PrevCloseCallback) Option {
	return func(c *Client) {
		c.prevCloseCallbacks = append(c.prevCloseCallbacks, cb)
	}
}

// WithFullCallback registers a full data callback
func WithFullCallback(cb FullCallback) Option {
	return func(c *Client) {
		c.fullCallbacks = append(c.fullCallbacks, cb)
	}
}

// WithErrorCallback registers an error callback
func WithErrorCallback(cb ErrorCallback) Option {
	return func(c *Client) {
		c.errorCallbacks = append(c.errorCallbacks, cb)
	}
}
