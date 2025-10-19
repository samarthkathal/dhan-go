package utils

import "time"

// WSConfig holds configuration for WebSocket connections
type WSConfig struct {
	// Connection limits
	MaxConnections        int // Max concurrent connections (5 for Dhan)
	MaxInstrumentsPerConn int // Max instruments per connection (5000 for Dhan)
	MaxBatchSize          int // Max instruments per subscription message (100 for Dhan)

	// Timeouts
	ConnectTimeout time.Duration // Timeout for initial connection
	ReadTimeout    time.Duration // Timeout for reading messages
	WriteTimeout   time.Duration // Timeout for writing messages
	PingInterval   time.Duration // Interval for ping messages (10s for Dhan)
	PongWait       time.Duration // Time to wait for pong response (40s for Dhan)

	// Reconnection
	ReconnectDelay       time.Duration // Delay before reconnection attempt
	MaxReconnectAttempts int           // Maximum reconnection attempts (0 = infinite)

	// Buffer sizes
	ReadBufferSize  int // WebSocket read buffer size
	WriteBufferSize int // WebSocket write buffer size

	// Middleware
	EnableLogging  bool
	EnableMetrics  bool
	EnableRecovery bool
}

// DefaultWSConfig returns default WebSocket configuration optimized for Dhan
func DefaultWSConfig() *WSConfig {
	return &WSConfig{
		MaxConnections:        5,
		MaxInstrumentsPerConn: 5000,
		MaxBatchSize:          100,
		ConnectTimeout:        30 * time.Second,
		ReadTimeout:           0, // No timeout (handled by ping/pong)
		WriteTimeout:          10 * time.Second,
		PingInterval:          10 * time.Second, // Dhan keepalive
		PongWait:              40 * time.Second, // Dhan disconnect threshold
		ReconnectDelay:        5 * time.Second,
		MaxReconnectAttempts:  0, // Infinite
		ReadBufferSize:        4096,
		WriteBufferSize:       4096,
		EnableLogging:         true,
		EnableMetrics:         true,
		EnableRecovery:        true,
	}
}

// HighFrequencyWSConfig returns configuration optimized for high-frequency trading
func HighFrequencyWSConfig() *WSConfig {
	config := DefaultWSConfig()
	config.ReadBufferSize = 8192
	config.WriteBufferSize = 8192
	config.ReconnectDelay = 1 * time.Second
	return config
}

// LowLatencyWSConfig returns configuration optimized for low latency
func LowLatencyWSConfig() *WSConfig {
	config := DefaultWSConfig()
	config.ReadBufferSize = 16384
	config.WriteBufferSize = 16384
	config.PingInterval = 5 * time.Second
	return config
}
