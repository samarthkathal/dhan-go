package actors

import (
	"context"
	"fmt"
	"sync"

	"github.com/anthdm/hollywood/actor"
	"github.com/samarthkathal/dhan-go/utils"
	"github.com/samarthkathal/dhan-go/websocket/types"
)

// ConnectionInfo tracks a single connection in the pool
type ConnectionInfo struct {
	ID                string
	PID               *actor.PID
	HealthMonitorPID  *actor.PID
	Instruments       map[string]bool // Map of "exchange:securityID" -> true
	InstrumentCount   int
	URL               string
	Connected         bool
}

// ConnectionPoolActor manages multiple WebSocket connections for load balancing
type ConnectionPoolActor struct {
	config    *utils.WSConfig
	logger    utils.Logger
	limiter   *utils.ConnectionLimiter
	metrics   *utils.WSMetricsCollector
	middleware utils.WSMiddleware

	// Connection pool
	connections map[string]*ConnectionInfo // connectionID -> info
	connLock    sync.RWMutex

	// Engine reference
	engine *actor.Engine

	// Context for cancellation
	ctx       context.Context
	ctxCancel context.CancelFunc
}

// NewConnectionPoolActor creates a new connection pool actor
func NewConnectionPoolActor(
	config *utils.WSConfig,
	logger utils.Logger,
	metrics *utils.WSMetricsCollector,
	middleware utils.WSMiddleware,
) actor.Receiver {
	ctx, cancel := context.WithCancel(context.Background())

	return &ConnectionPoolActor{
		config:      config,
		logger:      logger,
		limiter:     utils.NewConnectionLimiter(),
		metrics:     metrics,
		middleware:  middleware,
		connections: make(map[string]*ConnectionInfo),
		ctx:         ctx,
		ctxCancel:   cancel,
	}
}

// Receive implements the Hollywood actor.Receiver interface
func (cp *ConnectionPoolActor) Receive(actorCtx *actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case actor.Started:
		cp.handleStarted(actorCtx)
	case actor.Stopped:
		cp.handleStopped(actorCtx)
	case *PoolConnectMsg:
		cp.handlePoolConnect(actorCtx, msg)
	case *PoolDisconnectMsg:
		cp.handlePoolDisconnect(actorCtx, msg)
	case *PoolSubscribeMsg:
		cp.handlePoolSubscribe(actorCtx, msg)
	case *PoolUnsubscribeMsg:
		cp.handlePoolUnsubscribe(actorCtx, msg)
	case *PoolStatsMsg:
		cp.handlePoolStats(actorCtx, msg)
	case *DisconnectMsg:
		// Connection actor notified us of disconnection
		cp.handleConnectionDisconnect(actorCtx, msg)
	}
}

func (cp *ConnectionPoolActor) handleStarted(actorCtx *actor.Context) {
	cp.engine = actorCtx.Engine()

	if cp.logger != nil {
		cp.logger.Printf("[ConnectionPool] Actor started")
	}
}

func (cp *ConnectionPoolActor) handleStopped(actorCtx *actor.Context) {
	cp.ctxCancel()

	if cp.logger != nil {
		cp.logger.Printf("[ConnectionPool] Actor stopped")
	}
}

func (cp *ConnectionPoolActor) handlePoolConnect(actorCtx *actor.Context, msg *PoolConnectMsg) {
	// Check if we can add another connection
	if err := cp.limiter.AcquireConnection(msg.ConnectionID); err != nil {
		if cp.logger != nil {
			cp.logger.Printf("[ConnectionPool] Cannot add connection: %v", err)
		}
		if msg.ReplyTo != nil {
			actorCtx.Engine().Send(msg.ReplyTo, &PoolErrorMsg{Error: err})
		}
		return
	}

	// Spawn connection actor
	connectionPID := actorCtx.Engine().Spawn(func() actor.Receiver {
		return NewConnectionActor(
			cp.config,
			cp.logger,
			cp.metrics,
			cp.middleware,
		)
	}, fmt.Sprintf("connection-%s", msg.ConnectionID))

	// Spawn health monitor for this connection
	var healthMonitorActor *HealthMonitorActor
	healthMonitorPID := actorCtx.Engine().Spawn(func() actor.Receiver {
		hm := NewHealthMonitorActor(
			cp.config,
			cp.logger,
			cp.metrics,
		)
		if hmActor, ok := hm.(*HealthMonitorActor); ok {
			healthMonitorActor = hmActor
		}
		return hm
	}, fmt.Sprintf("health-monitor-%s", msg.ConnectionID))

	// Set connection reference in health monitor
	if healthMonitorActor != nil {
		healthMonitorActor.SetConnectionPID(connectionPID)
	}

	// Store connection info
	cp.connLock.Lock()
	cp.connections[msg.ConnectionID] = &ConnectionInfo{
		ID:               msg.ConnectionID,
		PID:              connectionPID,
		HealthMonitorPID: healthMonitorPID,
		Instruments:      make(map[string]bool),
		InstrumentCount:  0,
		URL:              msg.URL,
		Connected:        false,
	}
	cp.connLock.Unlock()

	// Send connect message to connection actor
	actorCtx.Engine().Send(connectionPID, &ConnectMsg{URL: msg.URL})

	if cp.logger != nil {
		cp.logger.Printf("[ConnectionPool] Connection %s added", msg.ConnectionID)
	}

	if msg.ReplyTo != nil {
		actorCtx.Engine().Send(msg.ReplyTo, &PoolConnectedMsg{ConnectionID: msg.ConnectionID})
	}
}

func (cp *ConnectionPoolActor) handlePoolDisconnect(actorCtx *actor.Context, msg *PoolDisconnectMsg) {
	cp.connLock.Lock()
	info, exists := cp.connections[msg.ConnectionID]
	if !exists {
		cp.connLock.Unlock()
		if cp.logger != nil {
			cp.logger.Printf("[ConnectionPool] Connection %s not found", msg.ConnectionID)
		}
		return
	}

	// Remove from map first
	delete(cp.connections, msg.ConnectionID)
	cp.connLock.Unlock()

	// Disconnect and stop actors
	if info.PID != nil {
		actorCtx.Engine().Send(info.PID, &DisconnectMsg{Reason: "pool_disconnect"})
		actorCtx.Engine().Poison(info.PID)
	}
	if info.HealthMonitorPID != nil {
		actorCtx.Engine().Poison(info.HealthMonitorPID)
	}

	// Release limiter
	cp.limiter.ReleaseConnection(msg.ConnectionID)

	if cp.logger != nil {
		cp.logger.Printf("[ConnectionPool] Connection %s removed", msg.ConnectionID)
	}
}

func (cp *ConnectionPoolActor) handlePoolSubscribe(actorCtx *actor.Context, msg *PoolSubscribeMsg) {
	// Find connection with fewest instruments (load balancing)
	connectionID, err := cp.findBestConnection()
	if err != nil {
		if cp.logger != nil {
			cp.logger.Printf("[ConnectionPool] No available connection: %v", err)
		}
		if msg.ReplyTo != nil {
			actorCtx.Engine().Send(msg.ReplyTo, &PoolErrorMsg{Error: err})
		}
		return
	}

	// Check rate limits
	if err := cp.limiter.CanSubscribe(connectionID, len(msg.Instruments)); err != nil {
		if cp.logger != nil {
			cp.logger.Printf("[ConnectionPool] Rate limit check failed: %v", err)
		}
		if msg.ReplyTo != nil {
			actorCtx.Engine().Send(msg.ReplyTo, &PoolErrorMsg{Error: err})
		}
		return
	}

	// Get connection info
	cp.connLock.RLock()
	info := cp.connections[connectionID]
	cp.connLock.RUnlock()

	// Create subscription request
	subReq, err := types.NewSubscriptionRequest(msg.Instruments)
	if err != nil {
		if cp.logger != nil {
			cp.logger.Printf("[ConnectionPool] Failed to create subscription request: %v", err)
		}
		if msg.ReplyTo != nil {
			actorCtx.Engine().Send(msg.ReplyTo, &PoolErrorMsg{Error: err})
		}
		return
	}

	data, err := subReq.ToJSON()
	if err != nil {
		if cp.logger != nil {
			cp.logger.Printf("[ConnectionPool] Failed to marshal subscription: %v", err)
		}
		if msg.ReplyTo != nil {
			actorCtx.Engine().Send(msg.ReplyTo, &PoolErrorMsg{Error: err})
		}
		return
	}

	// Send to connection actor
	actorCtx.Engine().Send(info.PID, &SendMsg{Data: data})

	// Update tracking
	cp.connLock.Lock()
	for _, inst := range msg.Instruments {
		key := fmt.Sprintf("%s:%s", inst.ExchangeSegment, inst.SecurityID)
		info.Instruments[key] = true
	}
	info.InstrumentCount = len(info.Instruments)
	cp.connLock.Unlock()

	// Update limiter
	cp.limiter.AddInstruments(connectionID, len(msg.Instruments))

	if cp.logger != nil {
		cp.logger.Printf("[ConnectionPool] Subscribed %d instruments on connection %s",
			len(msg.Instruments), connectionID)
	}

	if msg.ReplyTo != nil {
		actorCtx.Engine().Send(msg.ReplyTo, &PoolSubscribedMsg{
			ConnectionID: connectionID,
			Count:        len(msg.Instruments),
		})
	}
}

func (cp *ConnectionPoolActor) handlePoolUnsubscribe(actorCtx *actor.Context, msg *PoolUnsubscribeMsg) {
	// Find which connection has these instruments
	connectionID := cp.findConnectionForInstruments(msg.Instruments)
	if connectionID == "" {
		if cp.logger != nil {
			cp.logger.Printf("[ConnectionPool] Instruments not found in any connection")
		}
		return
	}

	cp.connLock.RLock()
	info := cp.connections[connectionID]
	cp.connLock.RUnlock()

	// Create unsubscription request
	unsubReq, err := types.NewUnsubscriptionRequest(msg.Instruments)
	if err != nil {
		if cp.logger != nil {
			cp.logger.Printf("[ConnectionPool] Failed to create unsubscription request: %v", err)
		}
		return
	}

	data, err := unsubReq.ToJSON()
	if err != nil {
		if cp.logger != nil {
			cp.logger.Printf("[ConnectionPool] Failed to marshal unsubscription: %v", err)
		}
		return
	}

	// Send to connection actor
	actorCtx.Engine().Send(info.PID, &SendMsg{Data: data})

	// Update tracking
	cp.connLock.Lock()
	for _, inst := range msg.Instruments {
		key := fmt.Sprintf("%s:%s", inst.ExchangeSegment, inst.SecurityID)
		delete(info.Instruments, key)
	}
	info.InstrumentCount = len(info.Instruments)
	cp.connLock.Unlock()

	// Update limiter
	cp.limiter.RemoveInstruments(connectionID, len(msg.Instruments))

	if cp.logger != nil {
		cp.logger.Printf("[ConnectionPool] Unsubscribed %d instruments from connection %s",
			len(msg.Instruments), connectionID)
	}
}

func (cp *ConnectionPoolActor) handlePoolStats(actorCtx *actor.Context, msg *PoolStatsMsg) {
	cp.connLock.RLock()
	defer cp.connLock.RUnlock()

	stats := make(map[string]interface{})
	stats["connection_count"] = len(cp.connections)
	stats["limiter"] = cp.limiter.GetStats()

	connections := make([]map[string]interface{}, 0, len(cp.connections))
	for _, info := range cp.connections {
		connections = append(connections, map[string]interface{}{
			"id":               info.ID,
			"instrument_count": info.InstrumentCount,
			"connected":        info.Connected,
		})
	}
	stats["connections"] = connections

	if msg.ReplyTo != nil {
		actorCtx.Engine().Send(msg.ReplyTo, &PoolStatsResponseMsg{Stats: stats})
	}
}

func (cp *ConnectionPoolActor) handleConnectionDisconnect(actorCtx *actor.Context, msg *DisconnectMsg) {
	// A connection actor notified us it disconnected
	// We could implement failover here if needed
	if cp.logger != nil {
		cp.logger.Printf("[ConnectionPool] Connection disconnected: %s", msg.Reason)
	}
}

// findBestConnection returns the connection ID with the fewest instruments
func (cp *ConnectionPoolActor) findBestConnection() (string, error) {
	cp.connLock.RLock()
	defer cp.connLock.RUnlock()

	if len(cp.connections) == 0 {
		return "", fmt.Errorf("no connections available")
	}

	var bestID string
	minInstruments := int(^uint(0) >> 1) // Max int

	for id, info := range cp.connections {
		if info.InstrumentCount < minInstruments {
			minInstruments = info.InstrumentCount
			bestID = id
		}
	}

	return bestID, nil
}

// findConnectionForInstruments finds which connection has the given instruments
func (cp *ConnectionPoolActor) findConnectionForInstruments(instruments []types.Instrument) string {
	cp.connLock.RLock()
	defer cp.connLock.RUnlock()

	if len(instruments) == 0 {
		return ""
	}

	// Check first instrument to determine connection
	firstKey := fmt.Sprintf("%s:%s", instruments[0].ExchangeSegment, instruments[0].SecurityID)

	for id, info := range cp.connections {
		if info.Instruments[firstKey] {
			return id
		}
	}

	return ""
}
