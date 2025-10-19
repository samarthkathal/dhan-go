package orderupdate

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/samarthkathal/dhan-go/utils"
	"github.com/samarthkathal/dhan-go/websocket/actors"
	"github.com/samarthkathal/dhan-go/websocket/types"
)

// Client is the WebSocket client for Order Update feed
type Client struct {
	// Configuration
	config *utils.WSConfig
	logger utils.Logger

	// Actor system
	engine            *actor.Engine
	connectionPID     *actor.PID
	healthMonitorPID  *actor.PID
	supervisorPID     *actor.PID

	// Middleware and metrics
	middleware utils.WSMiddleware
	metrics    *utils.WSMetricsCollector

	// State
	connected bool
	mu        sync.RWMutex

	// Callbacks
	callbacks     []types.OrderUpdateCallback
	callbacksLock sync.RWMutex
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

// NewClient creates a new Order Update WebSocket client
func NewClient(opts ...ClientOption) *Client {
	client := &Client{
		config:    utils.DefaultWSConfig(),
		logger:    log.Default(),
		metrics:   utils.NewWSMetricsCollector(),
		callbacks: make([]types.OrderUpdateCallback, 0),
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	// Build default middleware chain if not provided
	if client.middleware == nil && client.config.EnableLogging {
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
		// Log error but continue - will fail on Connect if engine is nil
		if client.logger != nil {
			client.logger.Printf("[OrderUpdateClient] Failed to create actor engine: %v", err)
		}
		return client
	}
	client.engine = engine

	return client
}

// Connect establishes a connection to the Order Update WebSocket
func (c *Client) Connect(url string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return fmt.Errorf("already connected")
	}

	// Spawn connection actor
	c.connectionPID = c.engine.Spawn(func() actor.Receiver {
		return actors.NewConnectionActor(
			c.config,
			c.logger,
			c.metrics,
			c.middleware,
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
		// Cast to concrete type to access SetConnectionPID
		if hmActor, ok := hm.(*actors.HealthMonitorActor); ok {
			healthMonitorActor = hmActor
		}
		return hm
	}, "health_monitor")

	// Set connection reference in health monitor
	if healthMonitorActor != nil {
		healthMonitorActor.SetConnectionPID(c.connectionPID)
	}

	// Register any existing callbacks
	c.callbacksLock.RLock()
	for _, callback := range c.callbacks {
		c.engine.Send(c.connectionPID, &actors.RegisterCallbackMsg{
			Callback: callback,
		})
	}
	c.callbacksLock.RUnlock()

	// Send connect message
	c.engine.Send(c.connectionPID, &actors.ConnectMsg{
		URL: url,
	})

	// Wait a moment for connection to establish
	time.Sleep(100 * time.Millisecond)

	c.connected = true

	if c.logger != nil {
		c.logger.Printf("[OrderUpdateClient] Connecting to %s", url)
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
		c.logger.Printf("[OrderUpdateClient] Disconnecting...")
	}

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

// OnUpdate registers a callback function that will be called when an order update is received
func (c *Client) OnUpdate(callback types.OrderUpdateCallback) {
	c.callbacksLock.Lock()
	defer c.callbacksLock.Unlock()

	c.callbacks = append(c.callbacks, callback)

	// If already connected, register callback with actor
	c.mu.RLock()
	connected := c.connected
	c.mu.RUnlock()

	if connected && c.connectionPID != nil {
		c.engine.Send(c.connectionPID, &actors.RegisterCallbackMsg{
			Callback: callback,
		})
	}

	if c.logger != nil {
		c.logger.Printf("[OrderUpdateClient] Callback registered (total: %d)", len(c.callbacks))
	}
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

// Send sends a message to the WebSocket server (for subscription requests, etc.)
func (c *Client) Send(data []byte) error {
	c.mu.RLock()
	connected := c.connected
	connectionPID := c.connectionPID
	c.mu.RUnlock()

	if !connected || connectionPID == nil {
		return fmt.Errorf("not connected")
	}

	c.engine.Send(connectionPID, &actors.SendMsg{
		Data: data,
	})

	return nil
}

// Shutdown performs a graceful shutdown of the client
func (c *Client) Shutdown() error {
	if c.logger != nil {
		c.logger.Printf("[OrderUpdateClient] Shutting down...")
	}

	// Disconnect if still connected
	if err := c.Disconnect(); err != nil {
		return fmt.Errorf("disconnect failed: %w", err)
	}

	if c.logger != nil {
		c.logger.Printf("[OrderUpdateClient] Shutdown complete")
	}

	return nil
}
