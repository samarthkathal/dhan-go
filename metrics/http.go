package metrics

import (
	"fmt"
	"sync"
	"time"
)

// HTTPCollector collects HTTP request metrics
type HTTPCollector struct {
	mu               sync.RWMutex
	requestCounts    map[string]int64
	requestDurations map[string]int64 // in milliseconds
	errorCounts      map[string]int64
	statusCodes      map[int]int64
}

// NewHTTPCollector creates a new HTTP metrics collector
func NewHTTPCollector() *HTTPCollector {
	return &HTTPCollector{
		requestCounts:    make(map[string]int64),
		requestDurations: make(map[string]int64),
		errorCounts:      make(map[string]int64),
		statusCodes:      make(map[int]int64),
	}
}

// RecordRequest records metrics for an HTTP request
func (m *HTTPCollector) RecordRequest(method, path string, statusCode int, duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	endpoint := fmt.Sprintf("%s %s", method, path)
	m.requestCounts[endpoint]++
	m.requestDurations[endpoint] += duration.Milliseconds()

	if err != nil {
		m.errorCounts[endpoint]++
	}

	if statusCode > 0 {
		m.statusCodes[statusCode]++
	}
}

// GetMetrics returns the current metrics as a map
func (m *HTTPCollector) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := make(map[string]interface{})
	metrics["request_counts"] = copyMap(m.requestCounts)
	metrics["request_durations_ms"] = copyMap(m.requestDurations)
	metrics["error_counts"] = copyMap(m.errorCounts)
	metrics["status_codes"] = copyMapInt(m.statusCodes)

	// Calculate total requests
	var totalRequests int64
	for _, count := range m.requestCounts {
		totalRequests += count
	}
	metrics["total_requests"] = totalRequests

	// Calculate total errors
	var totalErrors int64
	for _, count := range m.errorCounts {
		totalErrors += count
	}
	metrics["total_errors"] = totalErrors

	return metrics
}

// Reset resets all metrics to zero
func (m *HTTPCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestCounts = make(map[string]int64)
	m.requestDurations = make(map[string]int64)
	m.errorCounts = make(map[string]int64)
	m.statusCodes = make(map[int]int64)
}

func copyMap(m map[string]int64) map[string]int64 {
	result := make(map[string]int64, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func copyMapInt(m map[int]int64) map[int]int64 {
	result := make(map[int]int64, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
