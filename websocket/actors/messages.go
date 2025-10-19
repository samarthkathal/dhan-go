package actors

import (
	"github.com/anthdm/hollywood/actor"
	"github.com/samarthkathal/dhan-go/websocket/types"
)

// Connection lifecycle messages

// ConnectMsg requests a connection to WebSocket server
type ConnectMsg struct {
	URL string
}

// DisconnectMsg requests disconnection or notifies of disconnect
type DisconnectMsg struct {
	Reason string
	Error  error
}

// ReconnectMsg triggers a reconnection attempt
type ReconnectMsg struct{}

// ConnectedMsg notifies successful connection
type ConnectedMsg struct{}

// Data flow messages

// RawMessageMsg contains raw WebSocket message data
type RawMessageMsg struct {
	Data      []byte
	Timestamp int64 // Unix timestamp in milliseconds
}

// OrderUpdateMsg contains parsed order update
type OrderUpdateMsg struct {
	Alert *types.OrderAlert
}

// Send Message requests sending data to WebSocket
type SendMsg struct {
	Data []byte
}

// Health monitoring messages

// PingMsg triggers a ping to server
type PingMsg struct{}

// PongMsg received from server
type PongMsg struct{}

// HealthCheckMsg requests health check
type HealthCheckMsg struct{}

// HealthStatusMsg reports health status
type HealthStatusMsg struct {
	Healthy       bool
	LastPongTime  int64
	LastError     error
	Reconnections int
}

// Metrics messages

// MetricsRequestMsg requests current metrics
type MetricsRequestMsg struct {
	ReplyTo *actor.PID
}

// MetricsResponseMsg contains metrics data
type MetricsResponseMsg struct {
	Metrics map[string]interface{}
}

// Callback registration messages

// RegisterCallbackMsg registers a callback for order updates
type RegisterCallbackMsg struct {
	Callback types.OrderUpdateCallback
}

// UnregisterCallbackMsg removes a callback
type UnregisterCallbackMsg struct{}

// Connection Pool messages

// PoolConnectMsg requests adding a new connection to the pool
type PoolConnectMsg struct {
	ConnectionID string
	URL          string
	ReplyTo      *actor.PID
}

// PoolDisconnectMsg requests removing a connection from the pool
type PoolDisconnectMsg struct {
	ConnectionID string
}

// PoolSubscribeMsg requests subscribing to instruments via the pool
type PoolSubscribeMsg struct {
	Instruments []types.Instrument
	ReplyTo     *actor.PID
}

// PoolUnsubscribeMsg requests unsubscribing from instruments via the pool
type PoolUnsubscribeMsg struct {
	Instruments []types.Instrument
}

// PoolStatsMsg requests pool statistics
type PoolStatsMsg struct {
	ReplyTo *actor.PID
}

// PoolConnectedMsg notifies that a connection was added to the pool
type PoolConnectedMsg struct {
	ConnectionID string
}

// PoolSubscribedMsg notifies successful subscription via pool
type PoolSubscribedMsg struct {
	ConnectionID string
	Count        int
}

// PoolStatsResponseMsg contains pool statistics
type PoolStatsResponseMsg struct {
	Stats map[string]interface{}
}

// PoolErrorMsg contains an error from pool operations
type PoolErrorMsg struct {
	Error error
}
