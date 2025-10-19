package actors

import (
	"sync"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/samarthkathal/dhan-go/utils"
)

// HealthMonitorActor monitors connection health and triggers reconnection
type HealthMonitorActor struct {
	// Configuration
	config *utils.WSConfig
	logger utils.Logger

	// State
	lastPingTime  time.Time
	lastPongTime  time.Time
	reconnections int
	lastError     error
	stateLock     sync.RWMutex

	// References
	connectionPID *actor.PID
	metrics       *utils.WSMetricsCollector

	// Control
	stopChan chan struct{}
	stopOnce sync.Once
}

// NewHealthMonitorActor creates a new health monitor actor
func NewHealthMonitorActor(
	config *utils.WSConfig,
	logger utils.Logger,
	metrics *utils.WSMetricsCollector,
) actor.Receiver {
	return &HealthMonitorActor{
		config:        config,
		logger:        logger,
		metrics:       metrics,
		lastPongTime:  time.Now(),
		lastPingTime:  time.Now(),
		reconnections: 0,
		stopChan:      make(chan struct{}),
	}
}

// Receive implements the Hollywood actor.Receiver interface
func (h *HealthMonitorActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		h.handleStarted(ctx)
	case actor.Stopped:
		h.handleStopped(ctx)
	case *PingMsg:
		h.handlePing(ctx, msg)
	case *PongMsg:
		h.handlePong(ctx, msg)
	case *HealthCheckMsg:
		h.handleHealthCheck(ctx, msg)
	case *ConnectedMsg:
		h.handleConnected(ctx, msg)
	case *DisconnectMsg:
		h.handleDisconnect(ctx, msg)
	}
}

func (h *HealthMonitorActor) handleStarted(ctx *actor.Context) {
	if h.logger != nil {
		h.logger.Printf("[HealthMonitor] Actor started")
	}

	// Start health check loop
	go h.healthCheckLoop(ctx)
}

func (h *HealthMonitorActor) handleStopped(ctx *actor.Context) {
	h.stopOnce.Do(func() {
		close(h.stopChan)
	})

	if h.logger != nil {
		h.logger.Printf("[HealthMonitor] Actor stopped")
	}
}

func (h *HealthMonitorActor) handlePing(ctx *actor.Context, msg *PingMsg) {
	h.stateLock.Lock()
	defer h.stateLock.Unlock()

	h.lastPingTime = time.Now()

	if h.logger != nil {
		h.logger.Printf("[HealthMonitor] Ping sent")
	}
}

func (h *HealthMonitorActor) handlePong(ctx *actor.Context, msg *PongMsg) {
	h.stateLock.Lock()
	defer h.stateLock.Unlock()

	h.lastPongTime = time.Now()

	if h.logger != nil {
		h.logger.Printf("[HealthMonitor] Pong received")
	}
}

func (h *HealthMonitorActor) handleHealthCheck(ctx *actor.Context, msg *HealthCheckMsg) {
	h.stateLock.RLock()
	healthy := h.isHealthy()
	lastPongTime := h.lastPongTime.UnixMilli()
	lastError := h.lastError
	reconnections := h.reconnections
	h.stateLock.RUnlock()

	// Send status to parent
	ctx.Send(ctx.Parent(), &HealthStatusMsg{
		Healthy:       healthy,
		LastPongTime:  lastPongTime,
		LastError:     lastError,
		Reconnections: reconnections,
	})
}

func (h *HealthMonitorActor) handleConnected(ctx *actor.Context, msg *ConnectedMsg) {
	h.stateLock.Lock()
	defer h.stateLock.Unlock()

	// Reset state on successful connection
	h.lastPongTime = time.Now()
	h.lastPingTime = time.Now()
	h.lastError = nil

	if h.logger != nil {
		h.logger.Printf("[HealthMonitor] Connection established (reconnections: %d)", h.reconnections)
	}
}

func (h *HealthMonitorActor) handleDisconnect(ctx *actor.Context, msg *DisconnectMsg) {
	h.stateLock.Lock()
	defer h.stateLock.Unlock()

	h.lastError = msg.Error

	if h.logger != nil {
		h.logger.Printf("[HealthMonitor] Disconnected: %s", msg.Reason)
	}
}

func (h *HealthMonitorActor) healthCheckLoop(ctx *actor.Context) {
	ticker := time.NewTicker(h.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.stopChan:
			return
		case <-ticker.C:
			h.performHealthCheck(ctx)
		}
	}
}

func (h *HealthMonitorActor) performHealthCheck(ctx *actor.Context) {
	h.stateLock.RLock()
	healthy := h.isHealthy()
	timeSinceLastPong := time.Since(h.lastPongTime)
	reconnections := h.reconnections
	h.stateLock.RUnlock()

	if !healthy {
		if h.logger != nil {
			h.logger.Printf("[HealthMonitor] Connection unhealthy (no pong for %v)", timeSinceLastPong)
		}

		// Check if we should attempt reconnection
		if reconnections < h.config.MaxReconnectAttempts {
			h.stateLock.Lock()
			h.reconnections++
			currentReconnections := h.reconnections
			h.stateLock.Unlock()

			if h.metrics != nil {
				h.metrics.RecordReconnection()
			}

			if h.logger != nil {
				h.logger.Printf("[HealthMonitor] Triggering reconnection (attempt %d/%d)",
					currentReconnections, h.config.MaxReconnectAttempts)
			}

			// Send reconnect message to connection actor
			if h.connectionPID != nil {
				ctx.Engine().Send(h.connectionPID, &ReconnectMsg{})
			}
		} else {
			if h.logger != nil {
				h.logger.Printf("[HealthMonitor] Max reconnection attempts reached (%d)",
					h.config.MaxReconnectAttempts)
			}

			// Notify parent that reconnection failed
			ctx.Send(ctx.Parent(), &DisconnectMsg{
				Reason: "max_reconnections_reached",
			})
		}
	}
}

func (h *HealthMonitorActor) isHealthy() bool {
	// Connection is healthy if we received a pong within the configured timeout
	return time.Since(h.lastPongTime) < h.config.PongWait
}

// SetConnectionPID sets the reference to the connection actor
func (h *HealthMonitorActor) SetConnectionPID(pid *actor.PID) {
	h.stateLock.Lock()
	defer h.stateLock.Unlock()
	h.connectionPID = pid
}

// ResetReconnections resets the reconnection counter (called on successful reconnection)
func (h *HealthMonitorActor) ResetReconnections() {
	h.stateLock.Lock()
	defer h.stateLock.Unlock()
	h.reconnections = 0
}
