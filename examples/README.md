# Dhan Go SDK - Examples

This directory contains comprehensive examples demonstrating all features and capabilities of the Dhan Go SDK.

## 📋 Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Example Categories](#example-categories)
  - [REST API Examples](#rest-api-examples)
  - [WebSocket MarketFeed Examples](#websocket-marketfeed-examples)
  - [WebSocket OrderUpdate Examples](#websocket-orderupdate-examples)
  - [Combined Examples](#combined-examples)
  - [Configuration Examples](#configuration-examples)
  - [Error Handling Examples](#error-handling-examples)
  - [Graceful Shutdown Examples](#graceful-shutdown-examples)
- [Example Structure](#example-structure)
- [Running Examples](#running-examples)

## Prerequisites

Before running any example, ensure you have:

1. **Go 1.21 or higher** installed
2. **Dhan Trading Account** with API access
3. **Access Token** from Dhan platform

Set your access token as an environment variable:

```bash
export DHAN_ACCESS_TOKEN="your-access-token-here"
```

## Quick Start

To run any example:

```bash
cd examples/<category>/<example-name>
export DHAN_ACCESS_TOKEN="your-token"
go run main.go
```

Example:
```bash
cd examples/rest/01_basic
export DHAN_ACCESS_TOKEN="your-token"
go run main.go
```

## Example Categories

### REST API Examples

Located in `rest/`

Examples demonstrating synchronous HTTP API operations for order management and portfolio queries.

| # | Example | Description | Key Concepts |
|---|---------|-------------|--------------|
| 01 | **basic** | Simple REST client usage | Creating client, fetching data, placing orders |
| 02 | **with_default_config** | DefaultConfig usage | HTTP connection pool, timeouts |
| 03 | **with_low_latency_config** | LowLatencyConfig for speed | Reduced connections, faster failures |
| 04 | **with_high_throughput_config** | HighThroughputConfig for volume | Larger pools, longer timeouts |
| 05 | **with_hft_config** | HighFrequencyConfig for HFT | Ultra-low latency settings |
| 06 | **with_rate_limiting** | Default rate limiter | Automatic quota enforcement |
| 07 | **with_custom_rate_limiter** | Custom rate limits | Per-endpoint rate limiting |
| 08 | **with_middleware** | Logging & metrics middleware | Request/response interception |
| 09 | **error_handling** | Comprehensive error patterns | Retries, timeouts, graceful degradation |
| 10 | **graceful_shutdown** | Proper cleanup | Signal handling, resource release |

### WebSocket MarketFeed Examples

#### Single Connection (`websocket/marketfeed_single/`)

Examples using single-connection client for simple use cases.

| # | Example | Description | Key Concepts |
|---|---------|-------------|--------------|
| 01 | **basic** | Simple WebSocket connection | Connect, subscribe, callbacks |
| 02 | **all_callbacks** | All 5 data types | Ticker, Quote, OI, PrevClose, Full |
| 03 | **with_default_config** | Default WebSocket config | Standard timeout settings |
| 04 | **with_custom_timeouts** | Custom timeout configuration | Connect, write, ping/pong tuning |
| 05 | **with_custom_buffers** | Buffer size optimization | Read/write buffer tuning |
| 06 | **with_ping_pong_config** | Keep-alive configuration | Ping interval, pong wait |
| 07 | **with_reconnection** | Auto-reconnect settings | Reconnect delay, max attempts |
| 08 | **with_middleware** | WebSocket middleware | Logging, metrics, recovery |
| 09 | **with_metrics** | Metrics collection | Connection stats, latency tracking |
| 10 | **error_handling** | Error callbacks & handling | Connection errors, parse errors |
| 11 | **graceful_shutdown** | Clean disconnection | Context cancellation, cleanup |

#### Pooled Connection (`websocket/marketfeed_pooled/`)

Examples using connection pool for high-volume scenarios.

| # | Example | Description | Key Concepts |
|---|---------|-------------|--------------|
| 01 | **basic** | Basic pooled usage | Auto connection management |
| 02 | **with_default_config** | Default pool config | Pool limits, batching |
| 03 | **with_custom_pool_limits** | Custom pool sizing | MaxConnections, MaxInstruments |
| 04 | **with_custom_batch_size** | Batch size tuning | Subscription batching |
| 05 | **multi_connection** | Multi-connection demo | 5 connections, 1000s of instruments |
| 06 | **with_middleware** | Pool-wide middleware | Middleware across connections |
| 07 | **with_metrics** | Pool statistics | Per-connection stats |
| 08 | **all_callbacks** | All callbacks with pooling | Combined data types |
| 09 | **error_handling** | Pool error handling | Connection failures, recovery |
| 10 | **graceful_shutdown** | Close all connections | Pool cleanup |

### WebSocket OrderUpdate Examples

#### Single Connection (`websocket/orderupdate_single/`)

| # | Example | Description | Key Concepts |
|---|---------|-------------|--------------|
| 01 | **basic** | Basic order updates | Connect, receive order alerts |
| 02 | **with_default_config** | Default configuration | Standard timeout settings |
| 03 | **with_custom_timeouts** | Custom timeouts | Connect, write, ping/pong |
| 04 | **with_ping_pong_config** | Keep-alive tuning | Health monitoring |
| 05 | **with_middleware** | Middleware stack | Logging, metrics, recovery |
| 06 | **with_metrics** | Metrics collection | Connection health, latency |
| 07 | **order_status_helpers** | Helper methods | IsFilled, IsRejected, IsCancelled |
| 08 | **error_handling** | Error handling patterns | Connection errors, parse errors |
| 09 | **graceful_shutdown** | Clean disconnection | Proper cleanup |

#### Pooled Connection (`websocket/orderupdate_pooled/`)

| # | Example | Description | Key Concepts |
|---|---------|-------------|--------------|
| 01 | **basic** | Basic pooled usage | Pool management |
| 02 | **with_custom_config** | Custom pool config | Pool limits, timeouts |
| 03 | **with_middleware** | Pool middleware | Across connections |
| 04 | **with_metrics** | Pool metrics | Statistics |
| 05 | **error_handling** | Pool error handling | Failures, recovery |
| 06 | **graceful_shutdown** | Pool cleanup | Close all connections |

### Combined Examples

Located in `combined/`

Examples showing how to use multiple client types together.

| # | Example | Description | Use Case |
|---|---------|-------------|----------|
| 01 | **unified_client_basic** | Main dhan.Client usage | Unified API access |
| 02 | **unified_with_custom_configs** | Custom configs for all | HTTP + WS configuration |
| 03 | **rest_and_single_ws** | REST + single WS | Simple trading app |
| 04 | **rest_and_pooled_ws** | REST + pooled WS | High-volume trading |
| 05 | **all_clients_together** | All clients combined | Full-featured application |
| 06 | **shared_metrics** | Unified metrics collection | Observability |
| 07 | **trading_workflow** | Complete trading flow | Place order → track execution |
| 08 | **error_handling_combined** | Cross-client error handling | Comprehensive patterns |
| 09 | **graceful_shutdown_combined** | Shutdown all clients | Production-ready cleanup |

### Configuration Examples

Located in `configs/`

Deep dive into configuration profiles.

| # | Example | Description | When to Use |
|---|---------|-------------|-------------|
| 01 | **default_config** | DefaultConfig explained | General-purpose apps |
| 02 | **low_latency_config** | LowLatencyConfig explained | Speed-critical apps |
| 03 | **high_throughput_config** | HighThroughputConfig explained | High-volume apps |
| 04 | **hft_config** | HighFrequencyConfig explained | HFT applications |
| 05 | **custom_http_config** | Build custom HTTP config | Specific requirements |
| 06 | **custom_websocket_config** | Build custom WS config | Advanced tuning |

### Error Handling Examples

Located in `error_handling/`

Comprehensive error handling patterns.

| # | Example | Description | Errors Covered |
|---|---------|-------------|----------------|
| 01 | **rest_errors** | REST API error patterns | HTTP errors, API errors |
| 02 | **ws_connection_errors** | WebSocket connection errors | Connect failures, timeouts |
| 03 | **ws_message_errors** | Message parsing errors | Invalid data, protocol errors |
| 04 | **rate_limit_errors** | Rate limit handling | Quota exceeded, backoff |
| 05 | **timeout_errors** | Timeout handling | Context timeouts, deadlines |
| 06 | **retry_patterns** | Retry with backoff | Exponential backoff, jitter |
| 07 | **error_callbacks** | Using error callbacks | WebSocket error handling |
| 08 | **panic_recovery** | Panic recovery middleware | Stability patterns |

### Graceful Shutdown Examples

Located in `graceful_shutdown/`

Production-ready shutdown patterns.

| # | Example | Description | Key Concepts |
|---|---------|-------------|--------------|
| 01 | **rest_shutdown** | REST client cleanup | In-flight requests |
| 02 | **single_ws_shutdown** | Single WS cleanup | Connection closure |
| 03 | **pooled_ws_shutdown** | Pool cleanup | Multiple connections |
| 04 | **combined_shutdown** | All clients shutdown | Coordinated cleanup |
| 05 | **signal_handling** | OS signal handling | SIGINT, SIGTERM |
| 06 | **context_cancellation** | Context-based cleanup | Propagating cancellation |
| 07 | **timeout_shutdown** | Shutdown with timeout | Force shutdown |
| 08 | **resource_cleanup** | Complete resource cleanup | Goroutines, connections |

## Example Structure

Each example follows this structure:

```
<category>/<example-name>/
└── main.go          # Complete, runnable example
```

Every `main.go` includes:

- **Header comment**: What the example demonstrates
- **Prerequisites**: What you need to run it
- **Code**: Fully commented, production-ready code
- **Best practices**: Guidelines in comments
- **Error handling**: Proper error checking
- **Cleanup**: Graceful resource release

## Running Examples

### Basic Usage

```bash
# Navigate to example
cd examples/rest/01_basic

# Set access token
export DHAN_ACCESS_TOKEN="your-token-here"

# Run
go run main.go
```

### With Custom Environment

```bash
# Create .env file (example)
echo 'DHAN_ACCESS_TOKEN=your-token' > .env

# Source it
source .env

# Run example
go run main.go
```

### Building Examples

```bash
# Build specific example
cd examples/rest/01_basic
go build -o basic main.go
./basic

# Build all examples
for dir in examples/*/*/; do
  (cd "$dir" && go build -o $(basename $(dirname "$dir")) main.go)
done
```

## Learning Path

### Beginner

1. Start with `rest/01_basic` - Understand REST client
2. Try `websocket/marketfeed_single/01_basic` - Learn WebSocket basics
3. Explore `websocket/orderupdate_single/01_basic` - Order tracking
4. Check `combined/05_all_clients_together` - See it all together

### Intermediate

1. Explore configuration examples in `configs/`
2. Learn error handling patterns in `error_handling/`
3. Study middleware examples
4. Understand graceful shutdown patterns

### Advanced

1. Pool management examples
2. Custom configuration tuning
3. Performance optimization
4. Production deployment patterns

## Common Patterns

### Creating Clients

```go
// REST client
restClient, err := rest.NewClient(baseURL, token, httpClient)

// MarketFeed single connection
mfClient, err := marketfeed.NewClient(token, options...)

// MarketFeed pooled
mfPooled, err := marketfeed.NewPooledClient(token, options...)

// OrderUpdate single
ouClient, err := orderupdate.NewClient(token, options...)

// OrderUpdate pooled
ouPooled, err := orderupdate.NewPooledClient(token, options...)
```

### Using Callbacks

```go
client, err := marketfeed.NewClient(
    token,
    marketfeed.WithTickerCallback(func(data *marketfeed.TickerData) {
        // Handle ticker data
    }),
    marketfeed.WithErrorCallback(func(err error) {
        // Handle errors
    }),
)
```

### Graceful Shutdown

```go
// Setup signal handling
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

// Wait for signal
<-sigChan

// Disconnect clients
client.Disconnect()
```

## Troubleshooting

### Common Issues

**"Access token not set"**
```bash
export DHAN_ACCESS_TOKEN="your-token"
```

**"Connection timeout"**
- Check internet connectivity
- Verify Dhan API status
- Try increasing timeout in config

**"Rate limit exceeded"**
- Use rate limiting examples
- Add delays between requests
- Check Dhan API limits

**"Import errors"**
```bash
go mod tidy
go get github.com/samarthkathal/dhan-go@latest
```

## Additional Resources

- **API Documentation**: https://api.dhan.co
- **SDK Documentation**: See main README.md
- **GitHub**: https://github.com/samarthkathal/dhan-go
- **Issues**: Report at GitHub Issues

## Contributing Examples

Have a useful example? Contributions are welcome!

1. Follow the existing structure
2. Add comprehensive comments
3. Include error handling
4. Test thoroughly
5. Submit a PR

## License

All examples are provided under the same license as the main SDK (see LICENSE file in root directory).

---

**Happy Trading with Dhan Go SDK! 📈🚀**
