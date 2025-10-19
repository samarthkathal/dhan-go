package actors

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/gorilla/websocket"
	"github.com/samarthkathal/dhan-go/utils"
	"github.com/samarthkathal/dhan-go/websocket/types"
)

// ConnectionActor manages a single WebSocket connection
type ConnectionActor struct {
	// Configuration
	config *utils.WSConfig
	logger utils.Logger

	// WebSocket connection
	conn     *websocket.Conn
	connLock sync.RWMutex

	// Actor system
	engine        *actor.Engine
	healthMonitor *actor.PID

	// Middleware and metrics
	middleware utils.WSMiddleware
	metrics    *utils.WSMetricsCollector

	// Callbacks
	callbacks     []types.OrderUpdateCallback
	callbacksLock sync.RWMutex

	// State
	connected   bool
	stopChan    chan struct{}
	stopOnce    sync.Once
	ctx         context.Context
	ctxCancel   context.CancelFunc
}

// NewConnectionActor creates a new connection actor
func NewConnectionActor(
	config *utils.WSConfig,
	logger utils.Logger,
	metrics *utils.WSMetricsCollector,
	middleware utils.WSMiddleware,
) actor.Receiver {
	ctx, cancel := context.WithCancel(context.Background())

	return &ConnectionActor{
		config:     config,
		logger:     logger,
		middleware: middleware,
		metrics:    metrics,
		callbacks:  make([]types.OrderUpdateCallback, 0),
		stopChan:   make(chan struct{}),
		ctx:        ctx,
		ctxCancel:  cancel,
	}
}

// Receive implements the Hollywood actor.Receiver interface
func (c *ConnectionActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		c.handleStarted(ctx)
	case actor.Stopped:
		c.handleStopped(ctx)
	case *ConnectMsg:
		c.handleConnect(ctx, msg)
	case *DisconnectMsg:
		c.handleDisconnect(ctx, msg)
	case *ReconnectMsg:
		c.handleReconnect(ctx, msg)
	case *SendMsg:
		c.handleSend(ctx, msg)
	case *RawMessageMsg:
		c.handleRawMessage(ctx, msg)
	case *RegisterCallbackMsg:
		c.handleRegisterCallback(ctx, msg)
	case *UnregisterCallbackMsg:
		c.handleUnregisterCallback(ctx, msg)
	case *MetricsRequestMsg:
		c.handleMetricsRequest(ctx, msg)
	}
}

func (c *ConnectionActor) handleStarted(ctx *actor.Context) {
	if c.logger != nil {
		c.logger.Printf("[ConnectionActor] Actor started")
	}
}

func (c *ConnectionActor) handleStopped(ctx *actor.Context) {
	c.stopOnce.Do(func() {
		close(c.stopChan)
		c.ctxCancel()
		c.closeConnection()
	})

	if c.logger != nil {
		c.logger.Printf("[ConnectionActor] Actor stopped")
	}
}

func (c *ConnectionActor) handleConnect(ctx *actor.Context, msg *ConnectMsg) {
	c.connLock.Lock()
	defer c.connLock.Unlock()

	if c.connected {
		if c.logger != nil {
			c.logger.Printf("[ConnectionActor] Already connected")
		}
		return
	}

	// Create WebSocket connection
	dialer := websocket.Dialer{
		HandshakeTimeout: c.config.ConnectTimeout,
	}

	conn, _, err := dialer.Dial(msg.URL, nil)
	if err != nil {
		if c.logger != nil {
			c.logger.Printf("[ConnectionActor] Connection failed: %v", err)
		}
		ctx.Send(ctx.Parent(), &DisconnectMsg{
			Reason: "connection_failed",
			Error:  err,
		})
		return
	}

	c.conn = conn
	c.connected = true

	if c.metrics != nil {
		c.metrics.RecordConnection(true)
	}

	if c.logger != nil {
		c.logger.Printf("[ConnectionActor] Connected to %s", msg.URL)
	}

	// Notify parent
	ctx.Send(ctx.Parent(), &ConnectedMsg{})

	// Start reader and writer goroutines
	go c.readLoop(ctx)
	go c.writeLoop(ctx)
}

func (c *ConnectionActor) handleDisconnect(ctx *actor.Context, msg *DisconnectMsg) {
	if c.logger != nil {
		c.logger.Printf("[ConnectionActor] Disconnecting: %s", msg.Reason)
	}

	c.closeConnection()
}

func (c *ConnectionActor) handleReconnect(ctx *actor.Context, msg *ReconnectMsg) {
	if c.logger != nil {
		c.logger.Printf("[ConnectionActor] Reconnecting...")
	}

	c.closeConnection()

	// Wait a bit before reconnecting
	time.Sleep(c.config.ReconnectDelay)

	// Trigger reconnection via parent
	ctx.Send(ctx.Parent(), msg)
}

func (c *ConnectionActor) handleSend(ctx *actor.Context, msg *SendMsg) {
	c.connLock.RLock()
	conn := c.conn
	connected := c.connected
	c.connLock.RUnlock()

	if !connected || conn == nil {
		if c.logger != nil {
			c.logger.Printf("[ConnectionActor] Cannot send: not connected")
		}
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, msg.Data); err != nil {
		if c.logger != nil {
			c.logger.Printf("[ConnectionActor] Send error: %v", err)
		}
		ctx.Send(ctx.PID(), &DisconnectMsg{
			Reason: "send_error",
			Error:  err,
		})
		return
	}

	if c.metrics != nil {
		c.metrics.RecordMessageSent(len(msg.Data))
	}
}

func (c *ConnectionActor) handleRawMessage(ctx *actor.Context, msg *RawMessageMsg) {
	// Apply middleware if configured
	if c.middleware != nil {
		handler := c.middleware(c.processMessage)
		if err := handler(c.ctx, msg.Data); err != nil {
			if c.logger != nil {
				c.logger.Printf("[ConnectionActor] Middleware error: %v", err)
			}
			return
		}
	} else {
		// No middleware, process directly
		if err := c.processMessage(c.ctx, msg.Data); err != nil {
			if c.logger != nil {
				c.logger.Printf("[ConnectionActor] Process error: %v", err)
			}
		}
	}
}

func (c *ConnectionActor) processMessage(ctx context.Context, data []byte) error {
	// Parse JSON message
	var baseMsg struct {
		Type string `json:"Type"`
	}

	if err := json.Unmarshal(data, &baseMsg); err != nil {
		return fmt.Errorf("failed to parse message type: %w", err)
	}

	// Handle order alert messages
	if baseMsg.Type == "order_alert" {
		alert, err := types.ParseOrderAlert(data)
		if err != nil {
			return fmt.Errorf("failed to parse order alert: %w", err)
		}

		// Invoke callbacks
		c.callbacksLock.RLock()
		callbacks := make([]types.OrderUpdateCallback, len(c.callbacks))
		copy(callbacks, c.callbacks)
		c.callbacksLock.RUnlock()

		for _, callback := range callbacks {
			if callback != nil {
				callback(alert)
			}
		}
	}

	return nil
}

func (c *ConnectionActor) handleRegisterCallback(ctx *actor.Context, msg *RegisterCallbackMsg) {
	c.callbacksLock.Lock()
	defer c.callbacksLock.Unlock()

	c.callbacks = append(c.callbacks, msg.Callback)

	if c.logger != nil {
		c.logger.Printf("[ConnectionActor] Callback registered (total: %d)", len(c.callbacks))
	}
}

func (c *ConnectionActor) handleUnregisterCallback(ctx *actor.Context, msg *UnregisterCallbackMsg) {
	c.callbacksLock.Lock()
	defer c.callbacksLock.Unlock()

	c.callbacks = make([]types.OrderUpdateCallback, 0)

	if c.logger != nil {
		c.logger.Printf("[ConnectionActor] All callbacks unregistered")
	}
}

func (c *ConnectionActor) handleMetricsRequest(ctx *actor.Context, msg *MetricsRequestMsg) {
	if c.metrics == nil {
		return
	}

	metrics := c.metrics.GetMetrics()
	if msg.ReplyTo != nil {
		c.engine.Send(msg.ReplyTo, &MetricsResponseMsg{
			Metrics: metrics,
		})
	}
}

func (c *ConnectionActor) readLoop(ctx *actor.Context) {
	defer func() {
		if r := recover(); r != nil {
			if c.logger != nil {
				c.logger.Printf("[ConnectionActor] readLoop panic: %v", r)
			}
		}
	}()

	for {
		select {
		case <-c.stopChan:
			return
		default:
		}

		c.connLock.RLock()
		conn := c.conn
		c.connLock.RUnlock()

		if conn == nil {
			return
		}

		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if c.logger != nil {
				c.logger.Printf("[ConnectionActor] Read error: %v", err)
			}
			ctx.Send(ctx.PID(), &DisconnectMsg{
				Reason: "read_error",
				Error:  err,
			})
			return
		}

		// Handle different message types
		switch messageType {
		case websocket.TextMessage:
			// Send raw message to self for processing
			ctx.Send(ctx.PID(), &RawMessageMsg{
				Data:      data,
				Timestamp: time.Now().UnixMilli(),
			})

		case websocket.PingMessage:
			// Respond with pong
			if err := conn.WriteMessage(websocket.PongMessage, nil); err != nil {
				if c.logger != nil {
					c.logger.Printf("[ConnectionActor] Pong error: %v", err)
				}
			}

		case websocket.PongMessage:
			// Notify health monitor
			if c.healthMonitor != nil {
				ctx.Engine().Send(c.healthMonitor, &PongMsg{})
			}

		case websocket.CloseMessage:
			ctx.Send(ctx.PID(), &DisconnectMsg{
				Reason: "close_message",
			})
			return
		}
	}
}

func (c *ConnectionActor) writeLoop(ctx *actor.Context) {
	ticker := time.NewTicker(c.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.connLock.RLock()
			conn := c.conn
			connected := c.connected
			c.connLock.RUnlock()

			if !connected || conn == nil {
				return
			}

			// Send ping
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				if c.logger != nil {
					c.logger.Printf("[ConnectionActor] Ping error: %v", err)
				}
				ctx.Send(ctx.PID(), &DisconnectMsg{
					Reason: "ping_error",
					Error:  err,
				})
				return
			}

			// Notify health monitor that ping was sent
			if c.healthMonitor != nil {
				ctx.Engine().Send(c.healthMonitor, &PingMsg{})
			}
		}
	}
}

func (c *ConnectionActor) closeConnection() {
	c.connLock.Lock()
	defer c.connLock.Unlock()

	if c.conn != nil {
		// Send close message
		_ = c.conn.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)

		// Close connection
		_ = c.conn.Close()
		c.conn = nil
	}

	if c.connected {
		c.connected = false
		if c.metrics != nil {
			c.metrics.RecordConnection(false)
		}
	}

	if c.logger != nil {
		c.logger.Printf("[ConnectionActor] Connection closed")
	}
}
