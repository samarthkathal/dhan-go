package middleware

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"
)

// Logger is a generic interface for logging that's compatible with stdlib log.Logger
// and can be easily adapted to work with other logging frameworks (logrus, zap, slog, etc.)
type Logger interface {
	Printf(format string, v ...interface{})
}

// WSMessageHandler handles a WebSocket message
type WSMessageHandler func(ctx context.Context, msg []byte) error

// WSMiddleware wraps a WebSocket message handler
type WSMiddleware func(WSMessageHandler) WSMessageHandler

// WSMetricsCollector defines the interface for collecting WebSocket metrics
type WSMetricsCollector interface {
	RecordMessageReceived(bytes int, latency time.Duration)
	RecordError()
}

// ChainWSMiddleware composes multiple middleware functions
// Middleware is applied in order: first middleware is outermost
func ChainWSMiddleware(middlewares ...WSMiddleware) WSMiddleware {
	return func(handler WSMessageHandler) WSMessageHandler {
		// Apply in reverse order so first middleware is outermost
		for i := len(middlewares) - 1; i >= 0; i-- {
			handler = middlewares[i](handler)
		}
		return handler
	}
}

// WSLoggingMiddleware logs WebSocket messages
func WSLoggingMiddleware(logger Logger) WSMiddleware {
	if logger == nil {
		return func(next WSMessageHandler) WSMessageHandler {
			return next // No-op if no logger
		}
	}

	return func(next WSMessageHandler) WSMessageHandler {
		return func(ctx context.Context, msg []byte) error {
			start := time.Now()

			logger.Printf("[WS] --> Message received (%d bytes)", len(msg))

			err := next(ctx, msg)

			duration := time.Since(start)

			if err != nil {
				logger.Printf("[WS] <-- Error processing message: %v (%v)", err, duration)
			} else {
				logger.Printf("[WS] <-- Message processed successfully (%v)", duration)
			}

			return err
		}
	}
}

// WSMetricsMiddleware collects metrics for WebSocket messages
func WSMetricsMiddleware(collector WSMetricsCollector) WSMiddleware {
	if collector == nil {
		return func(next WSMessageHandler) WSMessageHandler {
			return next // No-op if no collector
		}
	}

	return func(next WSMessageHandler) WSMessageHandler {
		return func(ctx context.Context, msg []byte) error {
			start := time.Now()

			err := next(ctx, msg)

			duration := time.Since(start)
			collector.RecordMessageReceived(len(msg), duration)

			if err != nil {
				collector.RecordError()
			}

			return err
		}
	}
}

// WSRecoveryMiddleware recovers from panics in message handling
func WSRecoveryMiddleware(logger Logger) WSMiddleware {
	if logger == nil {
		return func(next WSMessageHandler) WSMessageHandler {
			return next // No-op if no logger
		}
	}

	return func(next WSMessageHandler) WSMessageHandler {
		return func(ctx context.Context, msg []byte) (err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.Printf("[WS PANIC] Recovered from panic: %v\n%s", r, debug.Stack())
					err = fmt.Errorf("panic recovered: %v", r)
				}
			}()

			return next(ctx, msg)
		}
	}
}

// WSContextMiddleware adds context values to the handler
func WSContextMiddleware(key, value interface{}) WSMiddleware {
	return func(next WSMessageHandler) WSMessageHandler {
		return func(ctx context.Context, msg []byte) error {
			ctx = context.WithValue(ctx, key, value)
			return next(ctx, msg)
		}
	}
}

// WSTimeoutMiddleware adds a timeout to message processing
func WSTimeoutMiddleware(timeout time.Duration) WSMiddleware {
	return func(next WSMessageHandler) WSMessageHandler {
		return func(ctx context.Context, msg []byte) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			done := make(chan error, 1)

			go func() {
				done <- next(ctx, msg)
			}()

			select {
			case err := <-done:
				return err
			case <-ctx.Done():
				return fmt.Errorf("message processing timeout: %w", ctx.Err())
			}
		}
	}
}
