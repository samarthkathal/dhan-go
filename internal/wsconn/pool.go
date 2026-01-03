package wsconn

import (
	"context"
	"fmt"
	"sync"

	"github.com/samarthkathal/dhan-go/internal/limiter"
	"github.com/samarthkathal/dhan-go/middleware"
	"github.com/samarthkathal/dhan-go/metrics"
	"github.com/samarthkathal/dhan-go/pool"
)

// Pool manages a pool of WebSocket connections
type Pool struct {
	urlTemplate    string // URL template with placeholder for connection index
	config         *WebSocketConfig
	messageHandler middleware.WSMessageHandler
	middleware     middleware.WSMiddleware
	metrics        *metrics.WSCollector
	bufferPool     *pool.BufferPool
	limiter        *limiter.ConnectionLimiter

	mu          sync.RWMutex
	connections map[string]*Connection
	instruments map[string]string // instrument ID -> connection ID

	nextConnIndex int
}

// PoolConfig holds configuration for creating a connection pool
type PoolConfig struct {
	URLTemplate    string
	Config         *WebSocketConfig
	MessageHandler middleware.WSMessageHandler
	Middleware     middleware.WSMiddleware
	Metrics        *metrics.WSCollector
	BufferPool     *pool.BufferPool
	Limiter        *limiter.ConnectionLimiter
}

// NewPool creates a new connection pool
func NewPool(cfg PoolConfig) *Pool {
	if cfg.Config == nil {
		cfg.Config = defaultWebSocketConfig()
	}
	if cfg.BufferPool == nil {
		cfg.BufferPool = pool.NewBufferPool()
	}
	if cfg.Limiter == nil {
		cfg.Limiter = limiter.NewConnectionLimiter()
	}

	return &Pool{
		urlTemplate:    cfg.URLTemplate,
		config:         cfg.Config,
		messageHandler: cfg.MessageHandler,
		middleware:     cfg.Middleware,
		metrics:        cfg.Metrics,
		bufferPool:     cfg.BufferPool,
		limiter:        cfg.Limiter,
		connections:    make(map[string]*Connection),
		instruments:    make(map[string]string),
	}
}

// GetOrCreateConnection gets an existing connection or creates a new one
func (p *Pool) GetOrCreateConnection(ctx context.Context) (*Connection, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Try to find a connection with capacity
	for _, conn := range p.connections {
		if conn.IsConnected() {
			// Check if this connection has capacity for more instruments
			instCount := p.limiter.GetInstrumentCount(conn.ID())
			if instCount < p.config.MaxInstrumentsPerConn {
				return conn, nil
			}
		}
	}

	// Need to create a new connection
	if len(p.connections) >= p.config.MaxConnections {
		return nil, fmt.Errorf("max connections reached (%d)", p.config.MaxConnections)
	}

	connID := fmt.Sprintf("conn-%d", p.nextConnIndex)
	p.nextConnIndex++

	conn := NewConnection(ConnectionConfig{
		ID:             connID,
		URL:            p.urlTemplate,
		Config:         p.config,
		MessageHandler: p.messageHandler,
		Middleware:     p.middleware,
		Metrics:        p.metrics,
		BufferPool:     p.bufferPool,
		Limiter:        p.limiter,
	})

	if err := conn.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	p.connections[connID] = conn
	return conn, nil
}

// GetConnectionForInstrument gets the connection handling a specific instrument
func (p *Pool) GetConnectionForInstrument(instrumentID string) (*Connection, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	connID, exists := p.instruments[instrumentID]
	if !exists {
		return nil, false
	}

	conn, exists := p.connections[connID]
	return conn, exists
}

// AssignInstrumentToConnection assigns an instrument to a connection
func (p *Pool) AssignInstrumentToConnection(instrumentID, connectionID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	conn, exists := p.connections[connectionID]
	if !exists {
		return fmt.Errorf("connection %s not found", connectionID)
	}

	if !conn.IsConnected() {
		return fmt.Errorf("connection %s not connected", connectionID)
	}

	// Check limiter
	if err := p.limiter.CanSubscribe(connectionID, 1); err != nil {
		return err
	}

	// Add to limiter
	if err := p.limiter.AddInstruments(connectionID, 1); err != nil {
		return err
	}

	p.instruments[instrumentID] = connectionID
	return nil
}

// UnassignInstrument removes an instrument assignment
func (p *Pool) UnassignInstrument(instrumentID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	connID, exists := p.instruments[instrumentID]
	if !exists {
		return fmt.Errorf("instrument %s not assigned", instrumentID)
	}

	delete(p.instruments, instrumentID)
	p.limiter.RemoveInstruments(connID, 1)
	return nil
}

// Subscribe subscribes to instruments (distributes across connections)
func (p *Pool) Subscribe(ctx context.Context, instruments []string, subscribeMsg func(connID string, instruments []string) ([]byte, error)) error {
	if len(instruments) == 0 {
		return nil
	}

	// Group instruments by connection (for batch subscription)
	p.mu.Lock()
	connectionInstruments := make(map[string][]string)

	for _, inst := range instruments {
		// Find a connection for this instrument
		var connID string
		var conn *Connection

		// Try to find existing connection with capacity
		for cid, c := range p.connections {
			if c.IsConnected() {
				instCount := p.limiter.GetInstrumentCount(cid)
				if instCount < p.config.MaxInstrumentsPerConn {
					connID = cid
					conn = c
					break
				}
			}
		}

		// Need new connection?
		if conn == nil {
			if len(p.connections) >= p.config.MaxConnections {
				p.mu.Unlock()
				return fmt.Errorf("max connections reached, cannot subscribe to more instruments")
			}

			connID = fmt.Sprintf("conn-%d", p.nextConnIndex)
			p.nextConnIndex++

			newConn := NewConnection(ConnectionConfig{
				ID:             connID,
				URL:            p.urlTemplate,
				Config:         p.config,
				MessageHandler: p.messageHandler,
				Middleware:     p.middleware,
				Metrics:        p.metrics,
				BufferPool:     p.bufferPool,
				Limiter:        p.limiter,
			})

			p.mu.Unlock()
			if err := newConn.Connect(ctx); err != nil {
				return fmt.Errorf("failed to connect: %w", err)
			}
			p.mu.Lock()

			p.connections[connID] = newConn
			conn = newConn
		}

		// Assign instrument to connection
		p.instruments[inst] = connID
		connectionInstruments[connID] = append(connectionInstruments[connID], inst)
	}
	p.mu.Unlock()

	// Send subscription messages
	for connID, instList := range connectionInstruments {
		// Batch into groups of MaxBatchSize
		for i := 0; i < len(instList); i += p.config.MaxBatchSize {
			end := i + p.config.MaxBatchSize
			if end > len(instList) {
				end = len(instList)
			}
			batch := instList[i:end]

			// Add to limiter
			if err := p.limiter.AddInstruments(connID, len(batch)); err != nil {
				return fmt.Errorf("failed to add instruments to limiter: %w", err)
			}

			// Generate subscription message
			msg, err := subscribeMsg(connID, batch)
			if err != nil {
				return fmt.Errorf("failed to generate subscription message: %w", err)
			}

			// Send message
			p.mu.RLock()
			conn := p.connections[connID]
			p.mu.RUnlock()

			if err := conn.Send(msg); err != nil {
				return fmt.Errorf("failed to send subscription: %w", err)
			}
		}
	}

	return nil
}

// Unsubscribe unsubscribes from instruments
func (p *Pool) Unsubscribe(ctx context.Context, instruments []string, unsubscribeMsg func(connID string, instruments []string) ([]byte, error)) error {
	if len(instruments) == 0 {
		return nil
	}

	p.mu.Lock()
	connectionInstruments := make(map[string][]string)

	for _, inst := range instruments {
		connID, exists := p.instruments[inst]
		if !exists {
			continue // Not subscribed
		}

		connectionInstruments[connID] = append(connectionInstruments[connID], inst)
		delete(p.instruments, inst)
	}
	p.mu.Unlock()

	// Send unsubscription messages
	for connID, instList := range connectionInstruments {
		// Remove from limiter
		p.limiter.RemoveInstruments(connID, len(instList))

		// Generate unsubscription message
		msg, err := unsubscribeMsg(connID, instList)
		if err != nil {
			return fmt.Errorf("failed to generate unsubscription message: %w", err)
		}

		// Send message
		p.mu.RLock()
		conn, exists := p.connections[connID]
		p.mu.RUnlock()

		if !exists || !conn.IsConnected() {
			continue
		}

		if err := conn.Send(msg); err != nil {
			return fmt.Errorf("failed to send unsubscription: %w", err)
		}
	}

	return nil
}

// CloseAll closes all connections in the pool
func (p *Pool) CloseAll() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for _, conn := range p.connections {
		if err := conn.Close(); err != nil {
			lastErr = err
		}
	}

	p.connections = make(map[string]*Connection)
	p.instruments = make(map[string]string)
	p.limiter.Reset()

	return lastErr
}

// GetStats returns pool statistics
func (p *Pool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := PoolStats{
		TotalConnections:  len(p.connections),
		ActiveConnections: 0,
		TotalInstruments:  len(p.instruments),
		ConnectionStats:   make(map[string]ConnectionStats),
	}

	for connID, conn := range p.connections {
		if conn.IsConnected() {
			stats.ActiveConnections++
		}

		stats.ConnectionStats[connID] = ConnectionStats{
			Connected:       conn.IsConnected(),
			InstrumentCount: p.limiter.GetInstrumentCount(connID),
			Health:          conn.HealthStatus(),
		}
	}

	return stats
}

// PoolStats contains statistics about the connection pool
type PoolStats struct {
	TotalConnections  int
	ActiveConnections int
	TotalInstruments  int
	ConnectionStats   map[string]ConnectionStats
}

// ConnectionStats contains statistics about a single connection
type ConnectionStats struct {
	Connected       bool
	InstrumentCount int
	Health          HealthStatus
}
