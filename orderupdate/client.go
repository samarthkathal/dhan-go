// Package orderupdate provides a client for Dhan's order update WebSocket API
package orderupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

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
	// OrderUpdateURL is the WebSocket URL for order updates
	OrderUpdateURL = "wss://api-feed.dhan.co/v2/order-update"
)

// Client provides access to Dhan's order update WebSocket API.
// It manages a single WebSocket connection for receiving order updates.
type Client struct {
	accessToken string
	config      *WebSocketConfig
	conn        *wsconn.Connection

	// Callbacks
	mu                      sync.RWMutex
	orderUpdateCallbacks    []OrderUpdateCallback
	errorCallbacks          []ErrorCallback

	// Middleware
	middleware middleware.WSMiddleware

	// State
	connected bool
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewClient creates a new order update client.
// This client manages a single WebSocket connection for receiving order updates.
func NewClient(accessToken string, opts ...Option) (*Client, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		accessToken:          accessToken,
		config:               defaultWebSocketConfig(),
		orderUpdateCallbacks: make([]OrderUpdateCallback, 0),
		errorCallbacks:       make([]ErrorCallback, 0),
		ctx:                  ctx,
		cancel:               cancel,
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
		URL:            OrderUpdateURL,
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
func (c *Client) handleMessage(ctx context.Context, data []byte) error {
	var alert OrderAlert
	if err := json.Unmarshal(data, &alert); err != nil {
		c.notifyError(fmt.Errorf("failed to parse order alert: %w", err))
		return err
	}

	c.notifyOrderUpdate(&alert)
	return nil
}

// notifyOrderUpdate notifies all registered order update callbacks
func (c *Client) notifyOrderUpdate(alert *OrderAlert) {
	c.mu.RLock()
	callbacks := c.orderUpdateCallbacks
	c.mu.RUnlock()

	for _, cb := range callbacks {
		go cb(alert)
	}
}

// notifyError notifies all registered error callbacks
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
		ReadTimeout:           0,
		WriteTimeout:          10 * time.Second,
		PingInterval:          10 * time.Second,
		PongWait:              40 * time.Second,
		ReconnectDelay:        5 * time.Second,
		MaxReconnectAttempts:  0,
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
