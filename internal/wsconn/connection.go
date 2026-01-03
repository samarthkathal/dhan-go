package wsconn

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/samarthkathal/dhan-go/internal/limiter"
	"github.com/samarthkathal/dhan-go/middleware"
	"github.com/samarthkathal/dhan-go/metrics"
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
	EnableMetrics         bool
	EnableRecovery        bool
}

// MessageHandler is a function that processes incoming WebSocket messages
type MessageHandler func(ctx context.Context, messageType int, data []byte) error

// Connection represents a single WebSocket connection with goroutine-based lifecycle management
type Connection struct {
	id     string
	url    string
	config *WebSocketConfig

	// WebSocket connection
	connMu sync.RWMutex
	conn   *websocket.Conn

	// Channels for goroutine communication
	sendCh chan []byte
	stopCh chan struct{}
	doneCh chan struct{}

	// Message handling
	messageHandler middleware.WSMessageHandler
	middleware     middleware.WSMiddleware

	// Metrics and pooling
	metrics    *metrics.WSCollector
	bufferPool *pool.BufferPool
	limiter    *limiter.ConnectionLimiter

	// Health monitoring
	lastPingMu sync.RWMutex
	lastPing   time.Time
	lastPong   time.Time

	// State
	stateMu   sync.RWMutex
	connected bool
	ctx       context.Context
	cancel    context.CancelFunc
}

// ConnectionConfig holds configuration for creating a new connection
type ConnectionConfig struct {
	ID             string
	URL            string
	Config         *WebSocketConfig
	MessageHandler middleware.WSMessageHandler
	Middleware     middleware.WSMiddleware
	Metrics        *metrics.WSCollector
	BufferPool     *pool.BufferPool
	Limiter        *limiter.ConnectionLimiter
}

// NewConnection creates a new WebSocket connection (not yet connected)
func NewConnection(cfg ConnectionConfig) *Connection {
	if cfg.Config == nil {
		cfg.Config = defaultWebSocketConfig()
	}
	if cfg.BufferPool == nil {
		cfg.BufferPool = pool.NewBufferPool()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Connection{
		id:             cfg.ID,
		url:            cfg.URL,
		config:         cfg.Config,
		messageHandler: cfg.MessageHandler,
		middleware:     cfg.Middleware,
		metrics:        cfg.Metrics,
		bufferPool:     cfg.BufferPool,
		limiter:        cfg.Limiter,
		sendCh:         make(chan []byte, 256),
		stopCh:         make(chan struct{}),
		doneCh:         make(chan struct{}),
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Connect establishes the WebSocket connection and starts goroutines
func (c *Connection) Connect(ctx context.Context) error {
	c.stateMu.Lock()
	if c.connected {
		c.stateMu.Unlock()
		return fmt.Errorf("connection %s already connected", c.id)
	}
	c.stateMu.Unlock()

	// Check limiter if available
	if c.limiter != nil {
		if err := c.limiter.AcquireConnection(c.id); err != nil {
			return fmt.Errorf("failed to acquire connection slot: %w", err)
		}
	}

	// Connect with timeout
	connectCtx, cancel := context.WithTimeout(ctx, c.config.ConnectTimeout)
	defer cancel()

	dialer := websocket.Dialer{
		HandshakeTimeout: c.config.ConnectTimeout,
		ReadBufferSize:   c.config.ReadBufferSize,
		WriteBufferSize:  c.config.WriteBufferSize,
	}

	conn, _, err := dialer.DialContext(connectCtx, c.url, nil)
	if err != nil {
		if c.limiter != nil {
			c.limiter.ReleaseConnection(c.id)
		}
		return fmt.Errorf("failed to dial WebSocket: %w", err)
	}

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	c.stateMu.Lock()
	c.connected = true
	c.stateMu.Unlock()

	// Record metrics
	if c.metrics != nil {
		c.metrics.RecordConnection(true)
	}

	// Start goroutines
	go c.readLoop()
	go c.writeLoop()
	go c.healthLoop()

	return nil
}

// readLoop continuously reads messages from the WebSocket
func (c *Connection) readLoop() {
	defer func() {
		c.disconnect()
		c.doneCh <- struct{}{}
	}()

	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return
	}

	// Set read deadline based on pong wait
	if c.config.PongWait > 0 {
		conn.SetReadDeadline(time.Now().Add(c.config.PongWait))
	}

	// Set pong handler
	conn.SetPongHandler(func(string) error {
		c.lastPingMu.Lock()
		c.lastPong = time.Now()
		c.lastPingMu.Unlock()

		if c.config.PongWait > 0 {
			conn.SetReadDeadline(time.Now().Add(c.config.PongWait))
		}
		return nil
	})

	for {
		select {
		case <-c.stopCh:
			return
		case <-c.ctx.Done():
			return
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			if c.metrics != nil {
				c.metrics.RecordError()
			}
			return
		}

		// Process message through middleware and handler
		if c.messageHandler != nil {
			handler := c.messageHandler
			if c.middleware != nil {
				handler = c.middleware(handler)
			}

			if err := handler(c.ctx, message); err != nil {
				if c.metrics != nil {
					c.metrics.RecordError()
				}
				// Continue processing other messages
			}
		}
	}
}

// writeLoop continuously writes messages to the WebSocket
func (c *Connection) writeLoop() {
	ticker := time.NewTicker(c.config.PingInterval)
	defer ticker.Stop()

	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return
	}

	for {
		select {
		case <-c.stopCh:
			return
		case <-c.ctx.Done():
			return
		case message := <-c.sendCh:
			if c.config.WriteTimeout > 0 {
				conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
			}

			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				if c.metrics != nil {
					c.metrics.RecordError()
				}
				return
			}

			if c.metrics != nil {
				c.metrics.RecordMessageSent(len(message))
			}

		case <-ticker.C:
			// Send ping
			if c.config.WriteTimeout > 0 {
				conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
			}

			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				if c.metrics != nil {
					c.metrics.RecordError()
				}
				return
			}

			c.lastPingMu.Lock()
			c.lastPing = time.Now()
			c.lastPingMu.Unlock()
		}
	}
}

// healthLoop monitors connection health
func (c *Connection) healthLoop() {
	if c.config.PongWait == 0 {
		return // Health monitoring disabled
	}

	ticker := time.NewTicker(c.config.PingInterval * 2)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.lastPingMu.RLock()
			lastPing := c.lastPing
			lastPong := c.lastPong
			c.lastPingMu.RUnlock()

			// Check if we've sent a ping but haven't received a pong
			if !lastPing.IsZero() && lastPong.Before(lastPing) {
				elapsed := time.Since(lastPing)
				if elapsed > c.config.PongWait {
					// Connection appears dead
					c.disconnect()
					return
				}
			}
		}
	}
}

// Send sends a message through the WebSocket connection
func (c *Connection) Send(message []byte) error {
	c.stateMu.RLock()
	connected := c.connected
	c.stateMu.RUnlock()

	if !connected {
		return fmt.Errorf("connection %s not connected", c.id)
	}

	select {
	case c.sendCh <- message:
		return nil
	case <-c.ctx.Done():
		return fmt.Errorf("connection %s closed", c.id)
	default:
		return fmt.Errorf("send buffer full for connection %s", c.id)
	}
}

// disconnect closes the connection (internal)
func (c *Connection) disconnect() {
	c.stateMu.Lock()
	if !c.connected {
		c.stateMu.Unlock()
		return
	}
	c.connected = false
	c.stateMu.Unlock()

	c.connMu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connMu.Unlock()

	if c.metrics != nil {
		c.metrics.RecordConnection(false)
	}

	if c.limiter != nil {
		c.limiter.ReleaseConnection(c.id)
	}
}

// Close closes the connection and stops all goroutines
func (c *Connection) Close() error {
	c.stateMu.RLock()
	connected := c.connected
	c.stateMu.RUnlock()

	if !connected {
		return nil
	}

	// Signal stop
	close(c.stopCh)

	// Cancel context
	c.cancel()

	// Wait for goroutines to finish (with timeout)
	select {
	case <-c.doneCh:
	case <-time.After(5 * time.Second):
		// Force disconnect if goroutines don't finish
	}

	c.disconnect()

	return nil
}

// IsConnected returns whether the connection is currently connected
func (c *Connection) IsConnected() bool {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.connected
}

// ID returns the connection ID
func (c *Connection) ID() string {
	return c.id
}

// HealthStatus returns the health status of the connection
func (c *Connection) HealthStatus() HealthStatus {
	c.lastPingMu.RLock()
	defer c.lastPingMu.RUnlock()

	c.stateMu.RLock()
	connected := c.connected
	c.stateMu.RUnlock()

	return HealthStatus{
		Connected: connected,
		LastPing:  c.lastPing,
		LastPong:  c.lastPong,
	}
}

// HealthStatus contains health information about a connection
type HealthStatus struct {
	Connected bool
	LastPing  time.Time
	LastPong  time.Time
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
		EnableMetrics:         true,
		EnableRecovery:        true,
	}
}
