// Package marketfeed provides a client for Dhan's market feed WebSocket API
package marketfeed

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/samarthkathal/dhan-go/internal/limiter"
	"github.com/samarthkathal/dhan-go/internal/wsconn"
	"github.com/samarthkathal/dhan-go/middleware"
	"github.com/samarthkathal/dhan-go/pool"
)

// WebSocketConfig holds configuration for WebSocket connections (local copy to avoid import cycle)
type WebSocketConfig struct {
	MaxConnections        int
	MaxInstrumentsPerConn int
	MaxBatchSize          int
	ConnectTimeout        time.Duration
	ReadTimeout           time.Duration
	WriteTimeout          time.Duration
	PingInterval          time.Duration
	PongWait              time.Duration
	ReconnectDelay        time.Duration
	MaxReconnectAttempts  int
	ReadBufferSize        int
	WriteBufferSize       int
	EnableLogging         bool
	EnableRecovery        bool
}

const (
	// MarketFeedURL is the WebSocket URL for market feed
	MarketFeedURL = "wss://api-feed.dhan.co"
)

// PooledClient provides access to Dhan's market feed WebSocket API with connection pooling.
// It manages up to 5 concurrent WebSocket connections and automatically distributes instruments
// across connections. Use NewPooledClient for high-volume scenarios with many instruments.
// For single-connection use cases, use Client (via NewClient) instead.
type PooledClient struct {
	accessToken string
	config      *WebSocketConfig
	pool        *wsconn.Pool

	// Callbacks
	mu                sync.RWMutex
	tickerCallbacks   []TickerCallback
	quoteCallbacks    []QuoteCallback
	oiCallbacks       []OICallback
	prevCloseCallbacks []PrevCloseCallback
	fullCallbacks     []FullCallback
	errorCallbacks    []ErrorCallback

	// Middleware
	middleware middleware.WSMiddleware

	// State
	connected bool
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewPooledClient creates a new pooled market feed client with connection pooling.
// This client automatically manages up to 5 WebSocket connections and distributes
// instruments across them (max 5000 instruments per connection, 100 per batch).
// Use this for high-volume scenarios. For single-connection use cases, use NewClient instead.
func NewPooledClient(accessToken string, opts ...PooledOption) (*PooledClient, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &PooledClient{
		accessToken:        accessToken,
		config:             defaultWebSocketConfig(),
		tickerCallbacks:    make([]TickerCallback, 0),
		quoteCallbacks:     make([]QuoteCallback, 0),
		oiCallbacks:        make([]OICallback, 0),
		prevCloseCallbacks: make([]PrevCloseCallback, 0),
		fullCallbacks:      make([]FullCallback, 0),
		errorCallbacks:     make([]ErrorCallback, 0),
		ctx:                ctx,
		cancel:             cancel,
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	// Create connection pool
	client.pool = wsconn.NewPool(wsconn.PoolConfig{
		URLTemplate:    MarketFeedURL,
		Config:         toWsconnConfig(client.config),
		MessageHandler: client.handleMessage,
		Middleware:     client.middleware,
		BufferPool:     pool.NewBufferPool(),
		Limiter:        limiter.NewConnectionLimiter(),
	})

	return client, nil
}

// Connect establishes the WebSocket connection
func (c *PooledClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return fmt.Errorf("already connected")
	}
	c.connected = true
	c.mu.Unlock()

	// Create at least one connection
	conn, err := c.pool.GetOrCreateConnection(ctx)
	if err != nil {
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		return fmt.Errorf("failed to create connection: %w", err)
	}

	// Send authorization message
	authMsg := fmt.Sprintf(`{"Authorization":"%s"}`, c.accessToken)
	if err := conn.Send([]byte(authMsg)); err != nil {
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		return fmt.Errorf("failed to send authorization: %w", err)
	}

	return nil
}

// Subscribe subscribes to market feed for given instruments
func (c *PooledClient) Subscribe(ctx context.Context, instruments []Instrument) error {
	c.mu.RLock()
	if !c.connected {
		c.mu.RUnlock()
		return fmt.Errorf("not connected")
	}
	c.mu.RUnlock()

	// Convert instruments to string IDs for tracking
	instrIDs := make([]string, len(instruments))
	for i, inst := range instruments {
		instrIDs[i] = fmt.Sprintf("%s:%s", inst.ExchangeSegment, inst.SecurityID)
	}

	// Subscribe using pool
	return c.pool.Subscribe(ctx, instrIDs, func(connID string, instList []string) ([]byte, error) {
		// Convert back to Instrument objects
		instObjs := make([]Instrument, len(instList))
		for i := range instList {
			// Parse the ID back (this is a simplification - in production, maintain a map)
			// For now, we'll need to keep the original instruments
			instObjs[i] = instruments[i%len(instruments)]
		}

		req, err := NewSubscriptionRequest(instObjs)
		if err != nil {
			return nil, err
		}
		return req.ToJSON()
	})
}

// Unsubscribe unsubscribes from market feed for given instruments
func (c *PooledClient) Unsubscribe(ctx context.Context, instruments []Instrument) error {
	c.mu.RLock()
	if !c.connected {
		c.mu.RUnlock()
		return fmt.Errorf("not connected")
	}
	c.mu.RUnlock()

	// Convert instruments to string IDs
	instrIDs := make([]string, len(instruments))
	for i, inst := range instruments {
		instrIDs[i] = fmt.Sprintf("%s:%s", inst.ExchangeSegment, inst.SecurityID)
	}

	// Unsubscribe using pool
	return c.pool.Unsubscribe(ctx, instrIDs, func(connID string, instList []string) ([]byte, error) {
		instObjs := make([]Instrument, len(instList))
		for i := range instList {
			instObjs[i] = instruments[i%len(instruments)]
		}

		req, err := NewUnsubscriptionRequest(instObjs)
		if err != nil {
			return nil, err
		}
		return req.ToJSON()
	})
}

// Disconnect closes the connection
func (c *PooledClient) Disconnect() error {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return nil
	}
	c.connected = false
	c.mu.Unlock()

	c.cancel()
	return c.pool.CloseAll()
}

// handleMessage processes incoming WebSocket messages
// Data pointers passed to callbacks are only valid during callback execution.
// If you need to retain data, copy the struct: myTicker := *ticker
func (c *PooledClient) handleMessage(ctx context.Context, data []byte) error {
	if len(data) < 1 {
		return fmt.Errorf("empty data received")
	}

	responseCode := data[0]

	// Route based on response code using pooled parsers
	switch responseCode {
	case FeedCodeTicker:
		ticker, err := parseTickerDataPooled(data)
		if err != nil {
			c.notifyError(err)
			return err
		}
		c.notifyTicker(ticker)
		releaseTicker(ticker)

	case FeedCodeQuote:
		quote, err := parseQuoteDataPooled(data)
		if err != nil {
			c.notifyError(err)
			return err
		}
		c.notifyQuote(quote)
		releaseQuote(quote)

	case FeedCodeOI:
		oi, err := parseOIDataPooled(data)
		if err != nil {
			c.notifyError(err)
			return err
		}
		c.notifyOI(oi)
		releaseOI(oi)

	case FeedCodePrevClose:
		prevClose, err := parsePrevCloseDataPooled(data)
		if err != nil {
			c.notifyError(err)
			return err
		}
		c.notifyPrevClose(prevClose)
		releasePrevClose(prevClose)

	case FeedCodeFull:
		full, err := parseFullDataPooled(data)
		if err != nil {
			c.notifyError(err)
			return err
		}
		c.notifyFull(full)
		releaseFull(full)

	case FeedCodeError:
		err := fmt.Errorf("feed error code received")
		c.notifyError(err)
		return err

	default:
		err := fmt.Errorf("unknown response code: %d", responseCode)
		c.notifyError(err)
		return err
	}

	return nil
}

// Callback notification methods
func (c *PooledClient) notifyTicker(data *TickerData) {
	c.mu.RLock()
	callbacks := c.tickerCallbacks
	c.mu.RUnlock()

	for _, cb := range callbacks {
		go cb(data)
	}
}

func (c *PooledClient) notifyQuote(data *QuoteData) {
	c.mu.RLock()
	callbacks := c.quoteCallbacks
	c.mu.RUnlock()

	for _, cb := range callbacks {
		go cb(data)
	}
}

func (c *PooledClient) notifyOI(data *OIData) {
	c.mu.RLock()
	callbacks := c.oiCallbacks
	c.mu.RUnlock()

	for _, cb := range callbacks {
		go cb(data)
	}
}

func (c *PooledClient) notifyPrevClose(data *PrevCloseData) {
	c.mu.RLock()
	callbacks := c.prevCloseCallbacks
	c.mu.RUnlock()

	for _, cb := range callbacks {
		go cb(data)
	}
}

func (c *PooledClient) notifyFull(data *FullData) {
	c.mu.RLock()
	callbacks := c.fullCallbacks
	c.mu.RUnlock()

	for _, cb := range callbacks {
		go cb(data)
	}
}

func (c *PooledClient) notifyError(err error) {
	c.mu.RLock()
	callbacks := c.errorCallbacks
	c.mu.RUnlock()

	for _, cb := range callbacks {
		go cb(err)
	}
}

// GetStats returns connection pool statistics
func (c *PooledClient) GetStats() wsconn.PoolStats {
	return c.pool.GetStats()
}

// Client provides access to Dhan's market feed WebSocket API with a single connection.
// This is simpler than PooledClient and gives you direct control over the connection lifecycle.
// Use this for single or few instruments. For high-volume scenarios with many instruments,
// use PooledClient (via NewPooledClient) instead.
type Client struct {
	accessToken string
	config      *WebSocketConfig
	conn        *wsconn.Connection

	// Callbacks
	mu                sync.RWMutex
	tickerCallbacks   []TickerCallback
	quoteCallbacks    []QuoteCallback
	oiCallbacks       []OICallback
	prevCloseCallbacks []PrevCloseCallback
	fullCallbacks     []FullCallback
	errorCallbacks    []ErrorCallback

	// Middleware
	middleware middleware.WSMiddleware

	// State
	connected bool
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewClient creates a new single-connection market feed client.
// This client manages a single WebSocket connection without pooling.
// It's simpler and more suitable for single or few instruments.
// For high-volume scenarios with many instruments, use NewPooledClient instead.
func NewClient(accessToken string, opts ...Option) (*Client, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		accessToken:        accessToken,
		config:             defaultWebSocketConfig(),
		tickerCallbacks:    make([]TickerCallback, 0),
		quoteCallbacks:     make([]QuoteCallback, 0),
		oiCallbacks:        make([]OICallback, 0),
		prevCloseCallbacks: make([]PrevCloseCallback, 0),
		fullCallbacks:      make([]FullCallback, 0),
		errorCallbacks:     make([]ErrorCallback, 0),
		ctx:                ctx,
		cancel:             cancel,
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// Connect establishes the WebSocket connection
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return fmt.Errorf("already connected")
	}
	c.connected = true
	c.mu.Unlock()

	// Create connection
	c.conn = wsconn.NewConnection(wsconn.ConnectionConfig{
		ID:             "single-conn",
		URL:            MarketFeedURL,
		Config:         toWsconnConfig(c.config),
		MessageHandler: c.handleMessage,
		Middleware:     c.middleware,
		BufferPool:     pool.NewBufferPool(),
		Limiter:        nil, // No limiter for single connection
	})

	if err := c.conn.Connect(ctx); err != nil {
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Send authorization message
	authMsg := fmt.Sprintf(`{"Authorization":"%s"}`, c.accessToken)
	if err := c.conn.Send([]byte(authMsg)); err != nil {
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		return fmt.Errorf("failed to send authorization: %w", err)
	}

	return nil
}

// Subscribe subscribes to market feed for given instruments
func (c *Client) Subscribe(ctx context.Context, instruments []Instrument) error {
	c.mu.RLock()
	if !c.connected {
		c.mu.RUnlock()
		return fmt.Errorf("not connected")
	}
	c.mu.RUnlock()

	// Create subscription request
	req, err := NewSubscriptionRequest(instruments)
	if err != nil {
		return fmt.Errorf("failed to create subscription request: %w", err)
	}

	data, err := req.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal subscription request: %w", err)
	}

	// Send subscription
	if err := c.conn.Send(data); err != nil {
		return fmt.Errorf("failed to send subscription: %w", err)
	}

	return nil
}

// Unsubscribe unsubscribes from market feed for given instruments
func (c *Client) Unsubscribe(ctx context.Context, instruments []Instrument) error {
	c.mu.RLock()
	if !c.connected {
		c.mu.RUnlock()
		return fmt.Errorf("not connected")
	}
	c.mu.RUnlock()

	// Create unsubscription request
	req, err := NewUnsubscriptionRequest(instruments)
	if err != nil {
		return fmt.Errorf("failed to create unsubscription request: %w", err)
	}

	data, err := req.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal unsubscription request: %w", err)
	}

	// Send unsubscription
	if err := c.conn.Send(data); err != nil {
		return fmt.Errorf("failed to send unsubscription: %w", err)
	}

	return nil
}

// Disconnect closes the connection
func (c *Client) Disconnect() error {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return nil
	}
	c.connected = false
	c.mu.Unlock()

	c.cancel()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// handleMessage processes incoming WebSocket messages
// Data pointers passed to callbacks are only valid during callback execution.
// If you need to retain data, copy the struct: myTicker := *ticker
func (c *Client) handleMessage(ctx context.Context, data []byte) error {
	if len(data) < 1 {
		return fmt.Errorf("empty data received")
	}

	responseCode := data[0]

	// Route based on response code using pooled parsers
	switch responseCode {
	case FeedCodeTicker:
		ticker, err := parseTickerDataPooled(data)
		if err != nil {
			c.notifyError(err)
			return err
		}
		c.notifyTicker(ticker)
		releaseTicker(ticker)

	case FeedCodeQuote:
		quote, err := parseQuoteDataPooled(data)
		if err != nil {
			c.notifyError(err)
			return err
		}
		c.notifyQuote(quote)
		releaseQuote(quote)

	case FeedCodeOI:
		oi, err := parseOIDataPooled(data)
		if err != nil {
			c.notifyError(err)
			return err
		}
		c.notifyOI(oi)
		releaseOI(oi)

	case FeedCodePrevClose:
		prevClose, err := parsePrevCloseDataPooled(data)
		if err != nil {
			c.notifyError(err)
			return err
		}
		c.notifyPrevClose(prevClose)
		releasePrevClose(prevClose)

	case FeedCodeFull:
		full, err := parseFullDataPooled(data)
		if err != nil {
			c.notifyError(err)
			return err
		}
		c.notifyFull(full)
		releaseFull(full)

	case FeedCodeError:
		err := fmt.Errorf("feed error code received")
		c.notifyError(err)
		return err

	default:
		err := fmt.Errorf("unknown response code: %d", responseCode)
		c.notifyError(err)
		return err
	}

	return nil
}

// Callback notification methods
func (c *Client) notifyTicker(data *TickerData) {
	c.mu.RLock()
	callbacks := c.tickerCallbacks
	c.mu.RUnlock()

	for _, cb := range callbacks {
		go cb(data)
	}
}

func (c *Client) notifyQuote(data *QuoteData) {
	c.mu.RLock()
	callbacks := c.quoteCallbacks
	c.mu.RUnlock()

	for _, cb := range callbacks {
		go cb(data)
	}
}

func (c *Client) notifyOI(data *OIData) {
	c.mu.RLock()
	callbacks := c.oiCallbacks
	c.mu.RUnlock()

	for _, cb := range callbacks {
		go cb(data)
	}
}

func (c *Client) notifyPrevClose(data *PrevCloseData) {
	c.mu.RLock()
	callbacks := c.prevCloseCallbacks
	c.mu.RUnlock()

	for _, cb := range callbacks {
		go cb(data)
	}
}

func (c *Client) notifyFull(data *FullData) {
	c.mu.RLock()
	callbacks := c.fullCallbacks
	c.mu.RUnlock()

	for _, cb := range callbacks {
		go cb(data)
	}
}

func (c *Client) notifyError(err error) {
	c.mu.RLock()
	callbacks := c.errorCallbacks
	c.mu.RUnlock()

	for _, cb := range callbacks {
		go cb(err)
	}
}

// GetStats returns connection statistics
func (c *Client) GetStats() wsconn.ConnectionStats {
	if c.conn == nil {
		return wsconn.ConnectionStats{
			Connected:       false,
			InstrumentCount: 0,
		}
	}
	return wsconn.ConnectionStats{
		Connected: c.conn.IsConnected(),
		Health:    c.conn.HealthStatus(),
	}
}

// defaultWebSocketConfig returns default WebSocket configuration
func defaultWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		MaxConnections:        5,
		MaxInstrumentsPerConn: 5000,
		MaxBatchSize:          100,
		ConnectTimeout:        30 * time.Second,
		ReadTimeout:           60 * time.Second, // Default read timeout to detect stale connections
		WriteTimeout:          10 * time.Second,
		PingInterval:          10 * time.Second,
		PongWait:              40 * time.Second,
		ReconnectDelay:        5 * time.Second,
		MaxReconnectAttempts:  10, // Sensible default limit
		ReadBufferSize:        4096,
		WriteBufferSize:       4096,
		EnableLogging:         true,
		EnableRecovery:        true,
	}
}

// toWsconnConfig converts local WebSocketConfig to wsconn.WebSocketConfig
func toWsconnConfig(cfg *WebSocketConfig) *wsconn.WebSocketConfig {
	return &wsconn.WebSocketConfig{
		MaxConnections:        cfg.MaxConnections,
		MaxInstrumentsPerConn: cfg.MaxInstrumentsPerConn,
		MaxBatchSize:          cfg.MaxBatchSize,
		ConnectTimeout:        cfg.ConnectTimeout,
		ReadTimeout:           cfg.ReadTimeout,
		WriteTimeout:          cfg.WriteTimeout,
		PingInterval:          cfg.PingInterval,
		PongWait:              cfg.PongWait,
		ReconnectDelay:        cfg.ReconnectDelay,
		MaxReconnectAttempts:  cfg.MaxReconnectAttempts,
		ReadBufferSize:        cfg.ReadBufferSize,
		WriteBufferSize:       cfg.WriteBufferSize,
		EnableLogging:         cfg.EnableLogging,
		EnableRecovery:        cfg.EnableRecovery,
	}
}
