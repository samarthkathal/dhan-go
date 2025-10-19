package marketfeed

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/samarthkathal/dhan-go/utils"
	"github.com/samarthkathal/dhan-go/websocket/actors"
	"github.com/samarthkathal/dhan-go/websocket/types"
)

// Client is the WebSocket client for Market Feed
type Client struct {
	// Configuration
	config *utils.WSConfig
	logger utils.Logger

	// Actor system
	engine           *actor.Engine
	connectionPID    *actor.PID
	healthMonitorPID *actor.PID

	// Middleware and metrics
	middleware utils.WSMiddleware
	metrics    *utils.WSMetricsCollector

	// State
	connected     bool
	subscriptions map[string]bool // Track subscribed instruments
	mu            sync.RWMutex

	// Callbacks for different feed types
	tickerCallbacks    []types.TickerCallback
	quoteCallbacks     []types.QuoteCallback
	oiCallbacks        []types.OICallback
	prevCloseCallbacks []types.PrevCloseCallback
	fullCallbacks      []types.FullCallback
	errorCallbacks     []types.ErrorCallback
	callbacksLock      sync.RWMutex
}

// ClientOption is a functional option for configuring the client
type ClientOption func(*Client)

// WithConfig sets a custom WebSocket configuration
func WithConfig(config *utils.WSConfig) ClientOption {
	return func(c *Client) {
		c.config = config
	}
}

// WithLogger sets a custom logger
func WithLogger(logger utils.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithMiddleware sets custom middleware chain
func WithMiddleware(middleware utils.WSMiddleware) ClientOption {
	return func(c *Client) {
		c.middleware = middleware
	}
}

// WithMetrics enables metrics collection
func WithMetrics(collector *utils.WSMetricsCollector) ClientOption {
	return func(c *Client) {
		c.metrics = collector
	}
}

// NewClient creates a new Market Feed WebSocket client
func NewClient(opts ...ClientOption) *Client {
	client := &Client{
		config:             utils.DefaultWSConfig(),
		logger:             log.Default(),
		metrics:            utils.NewWSMetricsCollector(),
		subscriptions:      make(map[string]bool),
		tickerCallbacks:    make([]types.TickerCallback, 0),
		quoteCallbacks:     make([]types.QuoteCallback, 0),
		oiCallbacks:        make([]types.OICallback, 0),
		prevCloseCallbacks: make([]types.PrevCloseCallback, 0),
		fullCallbacks:      make([]types.FullCallback, 0),
		errorCallbacks:     make([]types.ErrorCallback, 0),
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	// Build default middleware chain if not provided
	if client.middleware == nil {
		middlewares := []utils.WSMiddleware{}

		if client.config.EnableRecovery {
			middlewares = append(middlewares, utils.WSRecoveryMiddleware(client.logger))
		}

		if client.config.EnableMetrics && client.metrics != nil {
			middlewares = append(middlewares, utils.WSMetricsMiddleware(client.metrics))
		}

		if client.config.EnableLogging {
			middlewares = append(middlewares, utils.WSLoggingMiddleware(client.logger))
		}

		if len(middlewares) > 0 {
			client.middleware = utils.ChainWSMiddleware(middlewares...)
		}
	}

	// Initialize Hollywood actor system
	engine, err := actor.NewEngine(actor.EngineConfig{})
	if err != nil {
		if client.logger != nil {
			client.logger.Printf("[MarketFeedClient] Failed to create actor engine: %v", err)
		}
		return client
	}
	client.engine = engine

	return client
}

// Connect establishes a connection to the Market Feed WebSocket
// URL format: wss://api-feed.dhan.co?version=2&token={token}&clientId={clientId}&authType=2
func (c *Client) Connect(url string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return fmt.Errorf("already connected")
	}

	// Spawn connection actor with custom message handler
	c.connectionPID = c.engine.Spawn(func() actor.Receiver {
		return actors.NewConnectionActor(
			c.config,
			c.logger,
			c.metrics,
			c.createBinaryMessageMiddleware(),
		)
	}, "connection")

	// Spawn health monitor actor
	var healthMonitorActor *actors.HealthMonitorActor
	c.healthMonitorPID = c.engine.Spawn(func() actor.Receiver {
		hm := actors.NewHealthMonitorActor(
			c.config,
			c.logger,
			c.metrics,
		)
		if hmActor, ok := hm.(*actors.HealthMonitorActor); ok {
			healthMonitorActor = hmActor
		}
		return hm
	}, "health_monitor")

	// Set connection reference in health monitor
	if healthMonitorActor != nil {
		healthMonitorActor.SetConnectionPID(c.connectionPID)
	}

	// Send connect message
	c.engine.Send(c.connectionPID, &actors.ConnectMsg{
		URL: url,
	})

	// Wait a moment for connection to establish
	time.Sleep(100 * time.Millisecond)

	c.connected = true

	if c.logger != nil {
		c.logger.Printf("[MarketFeedClient] Connecting to %s", url)
	}

	return nil
}

// Disconnect closes the WebSocket connection gracefully
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	if c.logger != nil {
		c.logger.Printf("[MarketFeedClient] Disconnecting...")
	}

	// Send disconnect message to server
	disconnectReq := types.NewDisconnectRequest()
	data, _ := disconnectReq.ToJSON()
	c.engine.Send(c.connectionPID, &actors.SendMsg{Data: data})

	// Give server time to process
	time.Sleep(100 * time.Millisecond)

	// Send disconnect message to connection actor
	if c.connectionPID != nil {
		c.engine.Send(c.connectionPID, &actors.DisconnectMsg{
			Reason: "client_disconnect",
		})
	}

	// Stop actors
	if c.connectionPID != nil {
		c.engine.Poison(c.connectionPID)
	}
	if c.healthMonitorPID != nil {
		c.engine.Poison(c.healthMonitorPID)
	}

	// Give actors time to cleanup
	time.Sleep(100 * time.Millisecond)

	c.connected = false

	return nil
}

// Subscribe subscribes to instruments (automatically batches if > 100)
func (c *Client) Subscribe(instruments []types.Instrument) error {
	c.mu.RLock()
	connected := c.connected
	c.mu.RUnlock()

	if !connected {
		return fmt.Errorf("not connected")
	}

	// Batch instruments into groups of 100
	batches := types.BatchInstruments(instruments)

	for _, batch := range batches {
		subReq, err := types.NewSubscriptionRequest(batch)
		if err != nil {
			return fmt.Errorf("failed to create subscription request: %w", err)
		}

		data, err := subReq.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to marshal subscription request: %w", err)
		}

		// Send subscription message
		c.engine.Send(c.connectionPID, &actors.SendMsg{Data: data})

		// Track subscriptions
		c.mu.Lock()
		for _, inst := range batch {
			key := fmt.Sprintf("%s:%s", inst.ExchangeSegment, inst.SecurityID)
			c.subscriptions[key] = true
		}
		c.mu.Unlock()

		if c.logger != nil {
			c.logger.Printf("[MarketFeedClient] Subscribed to %d instruments", len(batch))
		}

		// Small delay between batches
		if len(batches) > 1 {
			time.Sleep(50 * time.Millisecond)
		}
	}

	return nil
}

// Unsubscribe unsubscribes from instruments
func (c *Client) Unsubscribe(instruments []types.Instrument) error {
	c.mu.RLock()
	connected := c.connected
	c.mu.RUnlock()

	if !connected {
		return fmt.Errorf("not connected")
	}

	// Batch instruments into groups of 100
	batches := types.BatchInstruments(instruments)

	for _, batch := range batches {
		unsubReq, err := types.NewUnsubscriptionRequest(batch)
		if err != nil {
			return fmt.Errorf("failed to create unsubscription request: %w", err)
		}

		data, err := unsubReq.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to marshal unsubscription request: %w", err)
		}

		// Send unsubscription message
		c.engine.Send(c.connectionPID, &actors.SendMsg{Data: data})

		// Remove from tracked subscriptions
		c.mu.Lock()
		for _, inst := range batch {
			key := fmt.Sprintf("%s:%s", inst.ExchangeSegment, inst.SecurityID)
			delete(c.subscriptions, key)
		}
		c.mu.Unlock()

		if c.logger != nil {
			c.logger.Printf("[MarketFeedClient] Unsubscribed from %d instruments", len(batch))
		}

		// Small delay between batches
		if len(batches) > 1 {
			time.Sleep(50 * time.Millisecond)
		}
	}

	return nil
}

// Callback registration methods
func (c *Client) OnTicker(callback types.TickerCallback) {
	c.callbacksLock.Lock()
	defer c.callbacksLock.Unlock()
	c.tickerCallbacks = append(c.tickerCallbacks, callback)
}

func (c *Client) OnQuote(callback types.QuoteCallback) {
	c.callbacksLock.Lock()
	defer c.callbacksLock.Unlock()
	c.quoteCallbacks = append(c.quoteCallbacks, callback)
}

func (c *Client) OnOI(callback types.OICallback) {
	c.callbacksLock.Lock()
	defer c.callbacksLock.Unlock()
	c.oiCallbacks = append(c.oiCallbacks, callback)
}

func (c *Client) OnPrevClose(callback types.PrevCloseCallback) {
	c.callbacksLock.Lock()
	defer c.callbacksLock.Unlock()
	c.prevCloseCallbacks = append(c.prevCloseCallbacks, callback)
}

func (c *Client) OnFull(callback types.FullCallback) {
	c.callbacksLock.Lock()
	defer c.callbacksLock.Unlock()
	c.fullCallbacks = append(c.fullCallbacks, callback)
}

func (c *Client) OnError(callback types.ErrorCallback) {
	c.callbacksLock.Lock()
	defer c.callbacksLock.Unlock()
	c.errorCallbacks = append(c.errorCallbacks, callback)
}

// GetMetrics returns the current metrics
func (c *Client) GetMetrics() map[string]interface{} {
	if c.metrics == nil {
		return make(map[string]interface{})
	}
	return c.metrics.GetMetrics()
}

// IsConnected returns whether the client is currently connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// GetSubscriptionCount returns the number of currently subscribed instruments
func (c *Client) GetSubscriptionCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.subscriptions)
}

// Shutdown performs a graceful shutdown of the client
func (c *Client) Shutdown() error {
	if c.logger != nil {
		c.logger.Printf("[MarketFeedClient] Shutting down...")
	}

	// Disconnect if still connected
	if err := c.Disconnect(); err != nil {
		return fmt.Errorf("disconnect failed: %w", err)
	}

	if c.logger != nil {
		c.logger.Printf("[MarketFeedClient] Shutdown complete")
	}

	return nil
}

// createBinaryMessageMiddleware creates middleware that parses binary messages and invokes callbacks
func (c *Client) createBinaryMessageMiddleware() utils.WSMiddleware {
	// Wrap user middleware with binary parser
	binaryParser := func(next utils.WSMessageHandler) utils.WSMessageHandler {
		return func(ctx context.Context, msg []byte) error {
			// Parse binary message and invoke appropriate callbacks
			if err := c.parseBinaryMessage(msg); err != nil {
				if c.logger != nil {
					c.logger.Printf("[MarketFeedClient] Binary parse error: %v", err)
				}
				return err
			}
			return next(ctx, msg)
		}
	}

	// Chain with user middleware
	if c.middleware != nil {
		return utils.ChainWSMiddleware(c.middleware, binaryParser)
	}
	return binaryParser
}

// parseBinaryMessage parses binary message and invokes appropriate callbacks
func (c *Client) parseBinaryMessage(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("message too short: %d bytes", len(data))
	}

	// Parse header to determine message type
	header, err := types.ParseMarketFeedHeader(data)
	if err != nil {
		return err
	}

	// Parse based on response code and invoke callbacks
	switch header.ResponseCode {
	case types.FeedCodeTicker:
		ticker, err := types.ParseTickerData(data)
		if err != nil {
			return err
		}
		c.invokeTickerCallbacks(ticker)

	case types.FeedCodeQuote:
		quote, err := types.ParseQuoteData(data)
		if err != nil {
			return err
		}
		c.invokeQuoteCallbacks(quote)

	case types.FeedCodeOI:
		oi, err := types.ParseOIData(data)
		if err != nil {
			return err
		}
		c.invokeOICallbacks(oi)

	case types.FeedCodePrevClose:
		prevClose, err := types.ParsePrevCloseData(data)
		if err != nil {
			return err
		}
		c.invokePrevCloseCallbacks(prevClose)

	case types.FeedCodeFull:
		full, err := types.ParseFullData(data)
		if err != nil {
			return err
		}
		c.invokeFullCallbacks(full)

	case types.FeedCodeError:
		errorData, err := types.ParseErrorData(data)
		if err != nil {
			return err
		}
		c.invokeErrorCallbacks(errorData)

	default:
		return fmt.Errorf("unknown response code: %d", header.ResponseCode)
	}

	return nil
}

// Callback invocation methods
func (c *Client) invokeTickerCallbacks(ticker *types.TickerData) {
	c.callbacksLock.RLock()
	callbacks := make([]types.TickerCallback, len(c.tickerCallbacks))
	copy(callbacks, c.tickerCallbacks)
	c.callbacksLock.RUnlock()

	for _, callback := range callbacks {
		if callback != nil {
			callback(ticker)
		}
	}
}

func (c *Client) invokeQuoteCallbacks(quote *types.QuoteData) {
	c.callbacksLock.RLock()
	callbacks := make([]types.QuoteCallback, len(c.quoteCallbacks))
	copy(callbacks, c.quoteCallbacks)
	c.callbacksLock.RUnlock()

	for _, callback := range callbacks {
		if callback != nil {
			callback(quote)
		}
	}
}

func (c *Client) invokeOICallbacks(oi *types.OIData) {
	c.callbacksLock.RLock()
	callbacks := make([]types.OICallback, len(c.oiCallbacks))
	copy(callbacks, c.oiCallbacks)
	c.callbacksLock.RUnlock()

	for _, callback := range callbacks {
		if callback != nil {
			callback(oi)
		}
	}
}

func (c *Client) invokePrevCloseCallbacks(prevClose *types.PrevCloseData) {
	c.callbacksLock.RLock()
	callbacks := make([]types.PrevCloseCallback, len(c.prevCloseCallbacks))
	copy(callbacks, c.prevCloseCallbacks)
	c.callbacksLock.RUnlock()

	for _, callback := range callbacks {
		if callback != nil {
			callback(prevClose)
		}
	}
}

func (c *Client) invokeFullCallbacks(full *types.FullData) {
	c.callbacksLock.RLock()
	callbacks := make([]types.FullCallback, len(c.fullCallbacks))
	copy(callbacks, c.fullCallbacks)
	c.callbacksLock.RUnlock()

	for _, callback := range callbacks {
		if callback != nil {
			callback(full)
		}
	}
}

func (c *Client) invokeErrorCallbacks(errorData *types.ErrorData) {
	c.callbacksLock.RLock()
	callbacks := make([]types.ErrorCallback, len(c.errorCallbacks))
	copy(callbacks, c.errorCallbacks)
	c.callbacksLock.RUnlock()

	for _, callback := range callbacks {
		if callback != nil {
			callback(errorData)
		}
	}
}
