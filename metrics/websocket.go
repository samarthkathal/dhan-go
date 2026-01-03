package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// WSCollector collects WebSocket connection metrics
type WSCollector struct {
	// Message counters
	messagesReceived atomic.Int64
	messagesSent     atomic.Int64
	bytesReceived    atomic.Int64
	bytesSent        atomic.Int64
	errors           atomic.Int64

	// Connection state
	activeConnections atomic.Int32
	totalConnections  atomic.Int64
	reconnections     atomic.Int64

	// Latency tracking
	mu              sync.RWMutex
	latencies       []time.Duration
	maxLatencyCount int
}

// NewWSCollector creates a new WebSocket metrics collector
func NewWSCollector() *WSCollector {
	return &WSCollector{
		maxLatencyCount: 1000, // Keep last 1000 latency samples
		latencies:       make([]time.Duration, 0, 1000),
	}
}

// RecordMessageReceived records a received message
func (w *WSCollector) RecordMessageReceived(bytes int, latency time.Duration) {
	w.messagesReceived.Add(1)
	w.bytesReceived.Add(int64(bytes))
	w.recordLatency(latency)
}

// RecordMessageSent records a sent message
func (w *WSCollector) RecordMessageSent(bytes int) {
	w.messagesSent.Add(1)
	w.bytesSent.Add(int64(bytes))
}

// RecordError records an error
func (w *WSCollector) RecordError() {
	w.errors.Add(1)
}

// RecordConnection records a connection state change
func (w *WSCollector) RecordConnection(connected bool) {
	if connected {
		w.activeConnections.Add(1)
		w.totalConnections.Add(1)
	} else {
		w.activeConnections.Add(-1)
	}
}

// RecordReconnection records a reconnection attempt
func (w *WSCollector) RecordReconnection() {
	w.reconnections.Add(1)
}

func (w *WSCollector) recordLatency(latency time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.latencies) >= w.maxLatencyCount {
		// Remove oldest entry
		w.latencies = w.latencies[1:]
	}
	w.latencies = append(w.latencies, latency)
}

// GetMetrics returns current metrics as a map
func (w *WSCollector) GetMetrics() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	metrics := make(map[string]interface{})

	// Message counts
	metrics["messages_received"] = w.messagesReceived.Load()
	metrics["messages_sent"] = w.messagesSent.Load()
	metrics["bytes_received"] = w.bytesReceived.Load()
	metrics["bytes_sent"] = w.bytesSent.Load()
	metrics["errors"] = w.errors.Load()

	// Connection state
	metrics["active_connections"] = w.activeConnections.Load()
	metrics["total_connections"] = w.totalConnections.Load()
	metrics["reconnections"] = w.reconnections.Load()

	// Latency stats
	if len(w.latencies) > 0 {
		var sum time.Duration
		min := w.latencies[0]
		max := w.latencies[0]

		for _, lat := range w.latencies {
			sum += lat
			if lat < min {
				min = lat
			}
			if lat > max {
				max = lat
			}
		}

		metrics["avg_latency_ms"] = float64(sum.Milliseconds()) / float64(len(w.latencies))
		metrics["min_latency_ms"] = min.Milliseconds()
		metrics["max_latency_ms"] = max.Milliseconds()
		metrics["latency_samples"] = len(w.latencies)
	}

	return metrics
}

// Reset resets all metrics to zero
func (w *WSCollector) Reset() {
	w.messagesReceived.Store(0)
	w.messagesSent.Store(0)
	w.bytesReceived.Store(0)
	w.bytesSent.Store(0)
	w.errors.Store(0)
	w.totalConnections.Store(0)
	w.reconnections.Store(0)

	w.mu.Lock()
	w.latencies = make([]time.Duration, 0, w.maxLatencyCount)
	w.mu.Unlock()
}
