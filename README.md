# Dhan Go SDK

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-blue)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

A minimal, type-safe Go SDK for Dhan v2 Trading APIs. Built entirely on auto-generated code from OpenAPI spec with lightweight utilities for production use.

## Why This SDK?

- **Minimal**: ~500 lines of custom code, rest is generated
- **Type-Safe**: Auto-generated from official OpenAPI 3.0.1 spec
- **Production-Ready**: Rate limiting, logging, metrics, recovery, connection pooling
- **Simple**: No complex wrappers, use generated client directly
- **Flexible**: Easy to customize and extend

## Quick Start

### Installation

```bash
go get github.com/samarthkathal/dhan-go
```

### 5-Minute Example

```go
package main

import (
    "context"
    "log"
    "net/http"

    "github.com/samarthkathal/dhan-go/client"
)

func main() {
    // 1. Create HTTP client
    httpClient := &http.Client{}

    // 2. Authentication
    authMiddleware := func(ctx context.Context, req *http.Request) error {
        req.Header.Set("access-token", "your-access-token")
        return nil
    }

    // 3. Create Dhan API client
    dhanClient, err := client.NewClientWithResponses(
        "https://api.dhan.co",
        client.WithHTTPClient(httpClient),
        client.WithRequestEditorFn(authMiddleware),
    )
    if err != nil {
        log.Fatal(err)
    }

    // 4. Make API calls
    ctx := context.Background()

    // Get holdings
    holdings, err := dhanClient.GetholdingsWithResponse(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }

    if holdings.JSON200 != nil {
        log.Printf("Holdings: %d positions", len(*holdings.JSON200.Data))
    }

    // Get positions
    positions, err := dhanClient.GetpositionsWithResponse(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }

    if positions.JSON200 != nil {
        log.Printf("Positions: %d positions", len(*positions.JSON200.Data))
    }
}
```

### Production Setup (with Middleware)

```go
import (
    "github.com/samarthkathal/dhan-go/utils"
)

// Create metrics collector
metricsCollector := utils.NewMetricsCollector()

// Create HTTP client with high-throughput config
httpClient := utils.HighThroughputHTTPClient()

// Add middleware
httpClient = utils.WithMiddleware(
    httpClient,
    utils.RecoveryRoundTripper(log.Default()),        // Panic recovery
    utils.RateLimitRoundTripper(100, 10),             // 100 req/sec, burst 10
    utils.MetricsRoundTripper(metricsCollector),      // Collect metrics
    utils.LoggingRoundTripper(log.Default()),         // Log requests
)

// Create Dhan client with configured HTTP client
dhanClient, err := client.NewClientWithResponses(
    "https://api.dhan.co",
    client.WithHTTPClient(httpClient),
    client.WithRequestEditorFn(authMiddleware),
)
```

## Features

### Core Features

- ✅ **Type-Safe Client** - 8,941 lines auto-generated from OpenAPI spec
- ✅ **31 REST Endpoints** - All Dhan v2 APIs available
- ✅ **Context Support** - Full `context.Context` propagation
- ✅ **Zero Wrappers** - Use generated client directly, no abstractions

### Production Features

- ✅ **Rate Limiting** - Token bucket algorithm, configurable
- ✅ **Logging** - Request/response logging via `http.RoundTripper`
- ✅ **Metrics** - Collect request stats (counts, durations, errors)
- ✅ **Panic Recovery** - Automatic recovery with stack traces
- ✅ **Connection Pooling** - 3 presets (Default, LowLatency, HighThroughput)
- ✅ **Graceful Shutdown** - Via context cancellation (no custom tracking)

### Developer Experience

- ✅ **Simple** - No learning curve, standard Go patterns
- ✅ **Maintainable** - Regenerate client with `go generate ./...`
- ✅ **Documented** - Complete usage guide and examples
- ✅ **Flexible** - Easy to add custom middleware

## API Coverage

All 31 Dhan v2 REST API endpoints are available:

- **Portfolio**: Holdings, Positions, Position Conversion
- **Orders**: Place, Modify, Cancel, Order List, Trade Book
- **Funds**: Fund Limits, Margin Calculator
- **Market Data**: Historical Charts, Intraday Charts, Option Chain
- **Forever Orders (GTT)**: Place, Modify, Cancel, List
- **Super Orders**: Bracket/Cover Orders
- **Alert Orders**: Conditional Triggers
- **EDIS**: Electronic Delivery Authorization
- **Trader Controls**: Kill Switch
- **Statements**: Ledger Reports

## Architecture

```
┌─────────────────────────────────────────────────┐
│  Your Application                               │
└───────────────┬─────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────┐
│  Generated Client (client/generated.go)         │
│  - 31 API methods                               │
│  - Type-safe request/response                   │
│  - Context support                              │
└───────────────┬─────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────┐
│  Custom HTTP Client (utils/)                    │
│  - Rate limiting (RoundTripper)                 │
│  - Logging (RoundTripper)                       │
│  - Metrics (RoundTripper)                       │
│  - Recovery (RoundTripper)                      │
│  - Connection pooling (http.Transport)          │
└───────────────┬─────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────┐
│  Dhan API (https://api.dhan.co)                 │
└─────────────────────────────────────────────────┘
```

**Key Point**: No custom wrappers! Generated client + lightweight utilities.

## Project Structure

```
dhan-go/
├── client/
│   └── generated.go          # 8,941 lines - Auto-generated from OpenAPI
├── utils/
│   ├── transport.go          # ~230 lines - Middleware (RoundTrippers)
│   └── config.go             # ~130 lines - HTTP client presets
├── examples/
│   ├── 01_basic/             # Basic usage
│   ├── 02_with_middleware/   # Production middleware setup
│   ├── 03_graceful_shutdown/ # Context cancellation pattern
│   └── 04_all_features/      # Complete production example
├── openapi.json              # OpenAPI 3.0.1 spec
├── tools.go                  # Code generation directive
├── go.mod
├── README.md                 # This file
├── USAGE_GUIDE.md            # Complete usage guide
└── CODE_GENERATION.md        # Code generation SOP
```

**Total custom code**: ~360 lines (utils/)
**Generated code**: ~8,941 lines (client/)

## Documentation

- **[USAGE_GUIDE.md](USAGE_GUIDE.md)** - Complete usage guide with all features
- **[CODE_GENERATION.md](CODE_GENERATION.md)** - How to regenerate client from OpenAPI spec
- **[examples/](examples/)** - Working code examples

## Examples

All examples are in the `examples/` directory:

1. **[Basic Usage](examples/01_basic/main.go)** - Simple authentication and API calls
2. **[With Middleware](examples/02_with_middleware/main.go)** - Rate limiting, logging, metrics, recovery
3. **[Graceful Shutdown](examples/03_graceful_shutdown/main.go)** - Context cancellation pattern
4. **[All Features](examples/04_all_features/main.go)** - Complete production setup

Run any example:
```bash
cd examples/01_basic
go run main.go
```

## Code Generation

The SDK uses [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) to generate the client from Dhan's OpenAPI specification.

### Regenerate Client

```bash
# When Dhan API changes
go generate ./...

# Verify compilation
go build ./...
```

See [CODE_GENERATION.md](CODE_GENERATION.md) for detailed procedures.

## Middleware

All middleware is implemented as `http.RoundTripper` wrappers, applied to the HTTP client.

### Available Middleware

```go
// Rate limiting
utils.RateLimitRoundTripper(100, 10) // 100 req/sec, burst 10

// Logging
utils.LoggingRoundTripper(log.Default())

// Metrics
utils.MetricsRoundTripper(collector)

// Panic recovery
utils.RecoveryRoundTripper(log.Default())
```

### Chaining Middleware

```go
httpClient = utils.WithMiddleware(
    httpClient,
    utils.RecoveryRoundTripper(logger),        // Innermost
    utils.RateLimitRoundTripper(100, 10),
    utils.MetricsRoundTripper(collector),
    utils.LoggingRoundTripper(logger),         // Outermost
)
```

## Connection Pooling

Three presets for different use cases:

```go
// Default - balanced
httpClient := utils.DefaultHTTPClient()

// Low latency - optimized for speed
httpClient := utils.LowLatencyHTTPClient()

// High throughput - optimized for volume
httpClient := utils.HighThroughputHTTPClient()
```

Custom configuration:
```go
config := &utils.HTTPClientConfig{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    // ... more options
}
httpClient := utils.NewHTTPClient(config)
```

## Graceful Shutdown

Use Go's `context.Context` for graceful shutdown. No custom tracking needed!

```go
// Create cancellable context
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Handle signals
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

go func() {
    <-sigCh
    cancel() // Cancel all ongoing requests
}()

// All API calls use this context
positions, err := dhanClient.GetpositionsWithResponse(ctx, nil)
```

See [examples/03_graceful_shutdown](examples/03_graceful_shutdown/main.go) for complete example.

## API Methods

All methods follow the pattern: `<Operation>WithResponse(ctx, params, body)`

### Common Methods

```go
// Portfolio
dhanClient.GetholdingsWithResponse(ctx, nil)
dhanClient.GetpositionsWithResponse(ctx, nil)

// Orders
dhanClient.PlaceorderWithResponse(ctx, nil, orderReq)
dhanClient.ModifyorderWithResponse(ctx, nil, modifyReq)
dhanClient.CancelorderWithResponse(ctx, nil, cancelReq)
dhanClient.GetorderlistWithResponse(ctx, nil)

// Funds
dhanClient.GetfundlimitWithResponse(ctx, nil)
dhanClient.MargincalculatorWithResponse(ctx, nil, marginReq)

// Market Data
dhanClient.HistoricalchartsWithResponse(ctx, nil, histReq)
dhanClient.IntradaychartsWithResponse(ctx, nil, intradayReq)
dhanClient.OptionchainWithResponse(ctx, nil, optionReq)
```

### Finding Methods

```bash
# List all methods
grep "func (c \*ClientWithResponses)" client/generated.go

# Search for specific endpoint
grep -i "placeorder" client/generated.go
```

## Best Practices

1. **Always use context** - Every method accepts `context.Context`
2. **Set timeouts** - Use `context.WithTimeout` or HTTP client timeout
3. **Handle errors** - Check both `err` and `StatusCode()`
4. **Reuse HTTP client** - Create once, use for all requests (connection pooling)
5. **Enable middleware in production** - Logging, metrics, rate limiting, recovery
6. **Secure credentials** - Use environment variables, not hardcoded tokens
7. **Graceful shutdown** - Use context cancellation, handle signals

See [USAGE_GUIDE.md](USAGE_GUIDE.md) for more details.

## Requirements

- Go 1.21 or higher
- Valid Dhan trading account with API access
- Access token from Dhan

## Installation

```bash
go get github.com/samarthkathal/dhan-go
```

## Contributing

Contributions are welcome! Please:

1. Read the code generation guide: [CODE_GENERATION.md](CODE_GENERATION.md)
2. Follow Go best practices
3. Add tests for new features
4. Update documentation

## Resources

- [Dhan API Documentation](https://dhanhq.co/docs/v2/)
- [Dhan API Reference](https://api.dhan.co/v2/)
- [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen)

## License

MIT License - see [LICENSE](LICENSE) file for details

## Support

For issues and questions:
- Create an issue on GitHub
- See [USAGE_GUIDE.md](USAGE_GUIDE.md) for detailed documentation
- Check [examples/](examples/) for working code

---

**Built for simplicity and production-ready performance** 🚀
