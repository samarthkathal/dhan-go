package limiter

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Dhan API limits
const (
	MaxConnections             = 5    // Max WebSocket connections per user
	MaxInstrumentsPerConnection = 5000 // Max instruments per connection
	MaxInstrumentsPerMessage    = 100  // Max instruments per subscription message
)

// ConnectionLimiter enforces Dhan's connection and subscription limits
type ConnectionLimiter struct {
	maxConnections           int
	maxInstrumentsPerConn    int
	maxInstrumentsPerMessage int

	activeConnections        atomic.Int32
	instrumentsPerConnection map[string]*atomic.Int32 // connection ID -> count
	mu                       sync.RWMutex
}

// NewConnectionLimiter creates a new connection limiter with Dhan's default limits
func NewConnectionLimiter() *ConnectionLimiter {
	return &ConnectionLimiter{
		maxConnections:           MaxConnections,
		maxInstrumentsPerConn:    MaxInstrumentsPerConnection,
		maxInstrumentsPerMessage: MaxInstrumentsPerMessage,
		instrumentsPerConnection: make(map[string]*atomic.Int32),
	}
}

// NewConnectionLimiterWithLimits creates a limiter with custom limits
func NewConnectionLimiterWithLimits(maxConns, maxInstsPerConn, maxInstsPerMsg int) *ConnectionLimiter {
	return &ConnectionLimiter{
		maxConnections:           maxConns,
		maxInstrumentsPerConn:    maxInstsPerConn,
		maxInstrumentsPerMessage: maxInstsPerMsg,
		instrumentsPerConnection: make(map[string]*atomic.Int32),
	}
}

// AcquireConnection attempts to acquire a connection slot
// Returns error if max connections reached
func (cl *ConnectionLimiter) AcquireConnection(connectionID string) error {
	current := cl.activeConnections.Load()
	if current >= int32(cl.maxConnections) {
		return fmt.Errorf("max connections reached (%d/%d)", current, cl.maxConnections)
	}

	cl.mu.Lock()
	defer cl.mu.Unlock()

	// Double-check after acquiring lock
	current = cl.activeConnections.Load()
	if current >= int32(cl.maxConnections) {
		return fmt.Errorf("max connections reached (%d/%d)", current, cl.maxConnections)
	}

	cl.activeConnections.Add(1)
	cl.instrumentsPerConnection[connectionID] = &atomic.Int32{}

	return nil
}

// ReleaseConnection releases a connection slot
func (cl *ConnectionLimiter) ReleaseConnection(connectionID string) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	if _, exists := cl.instrumentsPerConnection[connectionID]; exists {
		delete(cl.instrumentsPerConnection, connectionID)
		cl.activeConnections.Add(-1)
	}
}

// CanSubscribe checks if adding instruments to a connection would exceed limits
// Returns error if limits would be exceeded
func (cl *ConnectionLimiter) CanSubscribe(connectionID string, instrumentCount int) error {
	if instrumentCount > cl.maxInstrumentsPerMessage {
		return fmt.Errorf("too many instruments in single message (%d/%d)",
			instrumentCount, cl.maxInstrumentsPerMessage)
	}

	cl.mu.RLock()
	counter, exists := cl.instrumentsPerConnection[connectionID]
	cl.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection %s not registered", connectionID)
	}

	current := counter.Load()
	if current+int32(instrumentCount) > int32(cl.maxInstrumentsPerConn) {
		return fmt.Errorf("would exceed max instruments per connection (%d + %d > %d)",
			current, instrumentCount, cl.maxInstrumentsPerConn)
	}

	return nil
}

// AddInstruments adds instruments to a connection's count
func (cl *ConnectionLimiter) AddInstruments(connectionID string, count int) error {
	cl.mu.RLock()
	counter, exists := cl.instrumentsPerConnection[connectionID]
	cl.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection %s not registered", connectionID)
	}

	newCount := counter.Add(int32(count))
	if newCount > int32(cl.maxInstrumentsPerConn) {
		// Rollback
		counter.Add(-int32(count))
		return fmt.Errorf("exceeded max instruments per connection (%d/%d)",
			newCount, cl.maxInstrumentsPerConn)
	}

	return nil
}

// RemoveInstruments removes instruments from a connection's count
func (cl *ConnectionLimiter) RemoveInstruments(connectionID string, count int) {
	cl.mu.RLock()
	counter, exists := cl.instrumentsPerConnection[connectionID]
	cl.mu.RUnlock()

	if exists {
		counter.Add(-int32(count))
	}
}

// GetConnectionCount returns the current number of active connections
func (cl *ConnectionLimiter) GetConnectionCount() int {
	return int(cl.activeConnections.Load())
}

// GetInstrumentCount returns the number of instruments for a connection
func (cl *ConnectionLimiter) GetInstrumentCount(connectionID string) int {
	cl.mu.RLock()
	counter, exists := cl.instrumentsPerConnection[connectionID]
	cl.mu.RUnlock()

	if !exists {
		return 0
	}

	return int(counter.Load())
}

// GetTotalInstruments returns the total number of instruments across all connections
func (cl *ConnectionLimiter) GetTotalInstruments() int {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	total := 0
	for _, counter := range cl.instrumentsPerConnection {
		total += int(counter.Load())
	}

	return total
}

// GetStats returns current limiter statistics
func (cl *ConnectionLimiter) GetStats() map[string]interface{} {
	cl.mu.RLock()
	defer cl.mu.RUnlock()

	perConnectionCounts := make(map[string]int)
	for connID, counter := range cl.instrumentsPerConnection {
		perConnectionCounts[connID] = int(counter.Load())
	}

	return map[string]interface{}{
		"active_connections":         int(cl.activeConnections.Load()),
		"max_connections":            cl.maxConnections,
		"total_instruments":          cl.GetTotalInstruments(),
		"max_instruments_per_conn":   cl.maxInstrumentsPerConn,
		"instruments_per_connection": perConnectionCounts,
	}
}

// Reset clears all limits (useful for testing)
func (cl *ConnectionLimiter) Reset() {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.activeConnections.Store(0)
	cl.instrumentsPerConnection = make(map[string]*atomic.Int32)
}
