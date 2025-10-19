# Dhan Go SDK - Complete Usage Guide

This guide covers everything you need to know about using the Dhan Go SDK, from basic usage to advanced production configurations.

## Table of Contents

1. [Installation](#installation)
2. [Quick Start](#quick-start)
3. [Code Generation](#code-generation)
4. [Authentication](#authentication)
5. [Basic Usage](#basic-usage)
6. [Middleware Configuration](#middleware-configuration)
7. [Connection Pooling](#connection-pooling)
8. [Graceful Shutdown](#graceful-shutdown)
9. [Error Handling](#error-handling)
10. [Common Patterns](#common-patterns)
11. [API Reference](#api-reference)
12. [Best Practices](#best-practices)

---

## Installation

```bash
go get github.com/samarthkathal/dhan-go
```

**Requirements:**
- Go 1.21 or higher
- Valid Dhan trading account with API access

---

## Quick Start

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

    // 3. Create API client
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
    positions, err := dhanClient.GetpositionsWithResponse(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }

    if positions.JSON200 != nil {
        log.Printf("Positions: %d", len(*positions.JSON200.Data))
    }
}
```

---

## Code Generation

The SDK uses [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) to generate type-safe client code from the Dhan v2 OpenAPI specification.

### Regenerating the Client

```bash
# Generate client from OpenAPI spec
go generate ./...
```

This reads `openapi.json` and generates `client/generated.go` with:
- All type definitions (request/response models)
- Client with all API methods
- Type-safe request builders

### When to Regenerate

- Dhan API adds new endpoints
- Dhan API changes request/response schemas
- You update the `openapi.json` file

### Verification After Generation

```bash
# Verify compilation
go build ./...

# Run tests (if any)
go test ./...
```

See [CODE_GENERATION.md](CODE_GENERATION.md) for detailed regeneration procedures.

---

## Authentication

Dhan API uses token-based authentication. You need an access token from your Dhan account.

### Method 1: Request Editor Function (Recommended)

```go
accessToken := "your-access-token"

authMiddleware := func(ctx context.Context, req *http.Request) error {
    req.Header.Set("access-token", accessToken)
    return nil
}

dhanClient, err := client.NewClientWithResponses(
    "https://api.dhan.co",
    client.WithRequestEditorFn(authMiddleware),
)
```

### Method 2: Environment Variable

```go
accessToken := os.Getenv("DHAN_ACCESS_TOKEN")

authMiddleware := func(ctx context.Context, req *http.Request) error {
    req.Header.Set("access-token", accessToken)
    return nil
}
```

### Method 3: Multiple Headers

```go
authMiddleware := func(ctx context.Context, req *http.Request) error {
    req.Header.Set("access-token", accessToken)
    req.Header.Set("User-Agent", "my-trading-bot/1.0")
    req.Header.Set("X-Custom-Header", "value")
    return nil
}
```

---

## Basic Usage

### Fetching Portfolio Data

```go
ctx := context.Background()

// Get holdings
holdings, err := dhanClient.GetholdingsWithResponse(ctx, nil)
if err != nil {
    log.Fatal(err)
}

if holdings.JSON200 != nil {
    for _, holding := range *holdings.JSON200.Data {
        fmt.Printf("Security: %s, Qty: %d\n",
            *holding.SecurityId, *holding.Quantity)
    }
}

// Get positions
positions, err := dhanClient.GetpositionsWithResponse(ctx, nil)
if err != nil {
    log.Fatal(err)
}

if positions.JSON200 != nil {
    for _, position := range *positions.JSON200.Data {
        fmt.Printf("Security: %s, PnL: %.2f\n",
            *position.SecurityId, *position.RealizedProfit)
    }
}

// Get fund limits
funds, err := dhanClient.GetfundlimitWithResponse(ctx, nil)
if err != nil {
    log.Fatal(err)
}

if funds.JSON200 != nil && funds.JSON200.Data != nil {
    fmt.Printf("Available: %.2f\n", *funds.JSON200.Data.AvailabelBalance)
}
```

### Placing Orders

```go
// Helper function for creating pointers
func ptr[T any](v T) *T {
    return &v
}

// Place market order
orderReq := client.PlaceorderJSONRequestBody{
    SecurityId:      ptr("1333"), // HDFC Bank
    ExchangeSegment: client.OrderRequestExchangeSegmentNSEEQ,
    TransactionType: client.OrderRequestTransactionTypeBUY,
    Quantity:        ptr(int32(1)),
    OrderType:       ptr(client.OrderRequestOrderTypeMARKET),
    ProductType:     ptr(client.OrderRequestProductTypeINTRADAY),
    Price:           ptr(float32(0)),
}

order, err := dhanClient.PlaceorderWithResponse(ctx, nil, orderReq)
if err != nil {
    log.Fatal(err)
}

if order.JSON200 != nil {
    fmt.Printf("Order ID: %s\n", *order.JSON200.Data.OrderId)
}
```

### Fetching Market Data

```go
// Historical data
securityID := "1333"
exchangeSegment := client.HistoricalChartsRequestExchangeSegmentNSEEQ
instrument := client.HistoricalChartsRequestInstrumentEQUITY

req := client.HistoricalchartsJSONRequestBody{
    SecurityId:      &securityID,
    ExchangeSegment: &exchangeSegment,
    Instrument:      &instrument,
}

historical, err := dhanClient.HistoricalchartsWithResponse(ctx, nil, req)
if err != nil {
    log.Fatal(err)
}

if historical.JSON200 != nil {
    fmt.Printf("Candles: %d\n", len(*historical.JSON200.Close))
}
```

---

## Middleware Configuration

The SDK supports middleware via `http.RoundTripper` wrappers. Middleware is applied to the HTTP client before passing it to the Dhan client.

### Available Middleware

1. **Logging** - Log all requests and responses
2. **Metrics** - Collect request statistics
3. **Rate Limiting** - Throttle requests (token bucket)
4. **Recovery** - Recover from panics

### Using Middleware

```go
import "github.com/samarthkathal/dhan-go/utils"

// Create metrics collector
metricsCollector := utils.NewMetricsCollector()

// Create HTTP client with middleware
httpClient := utils.DefaultHTTPClient()

httpClient = utils.WithMiddleware(
    httpClient,
    utils.RecoveryRoundTripper(log.Default()),        // Innermost
    utils.RateLimitRoundTripper(100, 10),             // 100 req/sec, burst 10
    utils.MetricsRoundTripper(metricsCollector),
    utils.LoggingRoundTripper(log.Default()),         // Outermost
)

// Use this HTTP client with Dhan client
dhanClient, err := client.NewClientWithResponses(
    "https://api.dhan.co",
    client.WithHTTPClient(httpClient),
    client.WithRequestEditorFn(authMiddleware),
)
```

### Middleware Order

Middleware is applied in the order specified. The first middleware is the **outermost** (processes request first, response last).

**Recommended order:**
1. Logging (outermost) - see all requests/responses
2. Metrics - measure performance
3. Rate Limiting - throttle before sending
4. Recovery (innermost) - catch panics

### Rate Limiting

```go
// 100 requests per second, burst of 10
utils.RateLimitRoundTripper(100, 10)

// Conservative: 50 req/sec
utils.RateLimitRoundTripper(50, 5)

// Aggressive: 200 req/sec
utils.RateLimitRoundTripper(200, 20)
```

### Logging

```go
// Default logger
utils.LoggingRoundTripper(log.Default())

// Custom logger
logger := log.New(os.Stdout, "[DHAN] ", log.LstdFlags)
utils.LoggingRoundTripper(logger)
```

### Metrics

```go
collector := utils.NewMetricsCollector()
utils.MetricsRoundTripper(collector)

// Later, get metrics
metrics := collector.GetMetrics()
fmt.Printf("Total requests: %v\n", metrics["total_requests"])
fmt.Printf("Total errors: %v\n", metrics["total_errors"])

// Request counts by endpoint
if counts, ok := metrics["request_counts"].(map[string]int64); ok {
    for endpoint, count := range counts {
        fmt.Printf("%s: %d\n", endpoint, count)
    }
}
```

---

## Connection Pooling

The SDK provides three connection pool presets optimized for different use cases.

### Presets

```go
import "github.com/samarthkathal/dhan-go/utils"

// Default (balanced)
httpClient := utils.DefaultHTTPClient()

// Low latency (faster response, fewer connections)
httpClient := utils.LowLatencyHTTPClient()

// High throughput (more connections, handle burst traffic)
httpClient := utils.HighThroughputHTTPClient()
```

### Custom Configuration

```go
config := &utils.HTTPClientConfig{
    MaxIdleConns:          100,
    MaxIdleConnsPerHost:   10,
    MaxConnsPerHost:       10,
    IdleConnTimeout:       90 * time.Second,
    DialTimeout:           30 * time.Second,
    TLSHandshakeTimeout:   10 * time.Second,
    ResponseHeaderTimeout: 10 * time.Second,
    ExpectContinueTimeout: 1 * time.Second,
    KeepAlive:             30 * time.Second,
    InsecureSkipVerify:    false,
}

httpClient := utils.NewHTTPClient(config)
```

### Choosing a Preset

- **Default**: Most use cases, balanced performance
- **Low Latency**: Latency-sensitive applications, real-time trading
- **High Throughput**: High-frequency trading, batch operations

---

## Graceful Shutdown

Use Go's `context.Context` for graceful shutdown. No custom tracking needed!

### Basic Pattern

```go
// Create cancellable context
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Handle signals
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

go func() {
    <-sigCh
    fmt.Println("Shutting down...")
    cancel() // Cancel all ongoing requests
}()

// All API calls use this context
positions, err := dhanClient.GetpositionsWithResponse(ctx, nil)
```

### With Timeout

```go
// Individual request timeout
requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
defer cancel()

positions, err := dhanClient.GetpositionsWithResponse(requestCtx, nil)
```

### Long-Running Application

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

ticker := time.NewTicker(5 * time.Second)
defer ticker.Stop()

for {
    select {
    case <-ctx.Done():
        fmt.Println("Shutdown complete")
        return

    case <-ticker.C:
        // Make periodic API calls
        positions, err := dhanClient.GetpositionsWithResponse(ctx, nil)
        if err != nil {
            if ctx.Err() == context.Canceled {
                return // Shutting down
            }
            log.Printf("Error: %v", err)
        }
    }
}
```

See [examples/03_graceful_shutdown](examples/03_graceful_shutdown/main.go) for a complete example.

---

## Error Handling

### Checking HTTP Status

```go
resp, err := dhanClient.GetpositionsWithResponse(ctx, nil)
if err != nil {
    log.Fatalf("Request failed: %v", err)
}

switch resp.StatusCode() {
case 200:
    // Success
    if resp.JSON200 != nil {
        // Process data
    }
case 401:
    log.Fatal("Authentication failed")
case 429:
    log.Fatal("Rate limit exceeded")
case 500:
    log.Fatal("Server error")
default:
    log.Fatalf("Unexpected status: %d", resp.StatusCode())
}
```

### Context Errors

```go
resp, err := dhanClient.GetpositionsWithResponse(ctx, nil)
if err != nil {
    if ctx.Err() == context.Canceled {
        log.Println("Request cancelled")
        return
    }
    if ctx.Err() == context.DeadlineExceeded {
        log.Println("Request timeout")
        return
    }
    log.Fatalf("Request failed: %v", err)
}
```

### Nil Checks

```go
resp, err := dhanClient.GetpositionsWithResponse(ctx, nil)
if err != nil {
    return err
}

if resp.JSON200 == nil {
    return fmt.Errorf("unexpected response format")
}

if resp.JSON200.Data == nil {
    return fmt.Errorf("no data in response")
}

// Safe to use
for _, position := range *resp.JSON200.Data {
    // Process position
}
```

---

## Common Patterns

### Retry Logic

```go
func fetchWithRetry(ctx context.Context, maxRetries int) error {
    for i := 0; i < maxRetries; i++ {
        resp, err := dhanClient.GetpositionsWithResponse(ctx, nil)
        if err == nil && resp.StatusCode() == 200 {
            // Success
            return nil
        }

        if i < maxRetries-1 {
            time.Sleep(time.Second * time.Duration(i+1)) // Exponential backoff
        }
    }
    return fmt.Errorf("failed after %d retries", maxRetries)
}
```

### Concurrent Requests

```go
var wg sync.WaitGroup
results := make(chan interface{}, 3)

// Fetch holdings
wg.Add(1)
go func() {
    defer wg.Done()
    holdings, err := dhanClient.GetholdingsWithResponse(ctx, nil)
    if err == nil {
        results <- holdings
    }
}()

// Fetch positions
wg.Add(1)
go func() {
    defer wg.Done()
    positions, err := dhanClient.GetpositionsWithResponse(ctx, nil)
    if err == nil {
        results <- positions
    }
}()

// Fetch funds
wg.Add(1)
go func() {
    defer wg.Done()
    funds, err := dhanClient.GetfundlimitWithResponse(ctx, nil)
    if err == nil {
        results <- funds
    }
}()

wg.Wait()
close(results)
```

### Request Timeout

```go
// Per-request timeout
requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

resp, err := dhanClient.GetpositionsWithResponse(requestCtx, nil)
```

### Custom Headers

```go
customHeaders := func(ctx context.Context, req *http.Request) error {
    req.Header.Set("access-token", accessToken)
    req.Header.Set("X-Request-ID", generateRequestID())
    req.Header.Set("X-Client-Version", "1.0.0")
    return nil
}

dhanClient, err := client.NewClientWithResponses(
    "https://api.dhan.co",
    client.WithRequestEditorFn(customHeaders),
)
```

---

## API Reference

### All Available Endpoints

The generated client provides methods for all 31 Dhan v2 REST API endpoints.

#### Portfolio APIs

```go
// Get holdings
dhanClient.GetholdingsWithResponse(ctx, params)

// Get positions
dhanClient.GetpositionsWithResponse(ctx, params)

// Convert position
dhanClient.ConvertpositionWithResponse(ctx, params, body)
```

#### Order APIs

```go
// Place order
dhanClient.PlaceorderWithResponse(ctx, params, body)

// Modify order
dhanClient.ModifyorderWithResponse(ctx, params, body)

// Cancel order
dhanClient.CancelorderWithResponse(ctx, params, body)

// Get order by ID
dhanClient.GetorderbyidWithResponse(ctx, orderId, params)

// Get order list
dhanClient.GetorderlistWithResponse(ctx, params)

// Get trade book
dhanClient.GettradebookWithResponse(ctx, params)

// Get trade history
dhanClient.GettradehistoryWithResponse(ctx, orderId, params)

// Place slice order
dhanClient.PlacesliceorderWithResponse(ctx, params, body)
```

#### Funds APIs

```go
// Get fund limits
dhanClient.GetfundlimitWithResponse(ctx, params)

// Calculate margin
dhanClient.MargincalculatorWithResponse(ctx, params, body)
```

#### Market Data APIs

```go
// Historical charts
dhanClient.HistoricalchartsWithResponse(ctx, params, body)

// Intraday charts
dhanClient.IntradaychartsWithResponse(ctx, params, body)

// Option chain
dhanClient.OptionchainWithResponse(ctx, params, body)
```

#### Forever Orders (GTT)

```go
// Place forever order
dhanClient.Placeorder_foreverorderWithResponse(ctx, params, body)

// Modify forever order
dhanClient.Modifyorder_foreverorderWithResponse(ctx, params, body)

// Cancel forever order
dhanClient.Cancelorder_foreverorderWithResponse(ctx, params, body)

// Get forever order list
dhanClient.Getorderlist_foreverorderWithResponse(ctx, params)
```

#### Super Orders (Bracket/Cover)

```go
// Place super order
dhanClient.PlacesuperorderWithResponse(ctx, params, body)

// Modify super order
dhanClient.ModifysuperorderWithResponse(ctx, params, body)

// Cancel super order
dhanClient.CancelsuperorderWithResponse(ctx, params, body)
```

#### Alert Orders

```go
// Place alert order
dhanClient.Placeorder_alertorderWithResponse(ctx, params, body)

// Modify alert order
dhanClient.Modifyorder_alertorderWithResponse(ctx, params, body)

// Cancel alert order
dhanClient.Cancelorder_alertorderWithResponse(ctx, params, body)

// Get alert order list
dhanClient.Getorderlist_alertorderWithResponse(ctx, params)
```

#### EDIS

```go
// Get EDIS form
dhanClient.GetedisformWithResponse(ctx, params, body)

// Inquiry EDIS transaction
dhanClient.InquiryedistranWithResponse(ctx, isinList, params)
```

#### Trader Controls

```go
// Kill switch status
dhanClient.KillswitchstatusWithResponse(ctx, params)

// Enable kill switch
dhanClient.EnablekillswitchWithResponse(ctx, params, body)

// Disable kill switch
dhanClient.DisablekillswitchWithResponse(ctx, params)
```

#### Statements

```go
// Get ledger report
dhanClient.GetledgerreportWithResponse(ctx, params, body)
```

### Finding Methods

```bash
# Search generated code for specific endpoint
grep -i "GetpositionsWithResponse" client/generated.go

# List all methods
grep "func (c \*ClientWithResponses)" client/generated.go
```

---

## Best Practices

### 1. Always Use Context

```go
// ✅ Good
ctx := context.Background()
resp, err := dhanClient.GetpositionsWithResponse(ctx, nil)

// ❌ Bad
resp, err := dhanClient.GetpositionsWithResponse(nil, nil) // Won't compile
```

### 2. Set Timeouts

```go
// ✅ Good - per-request timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// ✅ Good - HTTP client timeout
httpClient := &http.Client{
    Timeout: 30 * time.Second,
}
```

### 3. Handle Errors

```go
// ✅ Good
resp, err := dhanClient.GetpositionsWithResponse(ctx, nil)
if err != nil {
    return fmt.Errorf("failed to fetch positions: %w", err)
}

if resp.StatusCode() != 200 {
    return fmt.Errorf("API error: status %d", resp.StatusCode())
}

if resp.JSON200 == nil {
    return fmt.Errorf("unexpected response format")
}
```

### 4. Use Connection Pooling

```go
// ✅ Good - reuse HTTP client
httpClient := utils.DefaultHTTPClient()

dhanClient, _ := client.NewClientWithResponses(
    "https://api.dhan.co",
    client.WithHTTPClient(httpClient),
)

// Make many requests - connections are reused
```

### 5. Enable Middleware in Production

```go
// ✅ Good - comprehensive middleware
httpClient = utils.WithMiddleware(
    httpClient,
    utils.RecoveryRoundTripper(logger),
    utils.RateLimitRoundTripper(100, 10),
    utils.MetricsRoundTripper(collector),
    utils.LoggingRoundTripper(logger),
)
```

### 6. Secure Credentials

```go
// ✅ Good - from environment
accessToken := os.Getenv("DHAN_ACCESS_TOKEN")

// ❌ Bad - hardcoded
accessToken := "your-token-here"
```

### 7. Graceful Shutdown

```go
// ✅ Good - handle signals
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

go func() {
    <-sigCh
    cancel()
}()
```

---

## Examples

Complete working examples are available in the `examples/` directory:

1. **[01_basic](examples/01_basic/main.go)** - Basic usage with authentication
2. **[02_with_middleware](examples/02_with_middleware/main.go)** - Middleware configuration
3. **[03_graceful_shutdown](examples/03_graceful_shutdown/main.go)** - Context cancellation pattern
4. **[04_all_features](examples/04_all_features/main.go)** - Complete production setup

---

## Troubleshooting

### "401 Unauthorized"
- Check your access token
- Ensure `access-token` header is set correctly

### "429 Too Many Requests"
- Add rate limiting middleware
- Reduce request frequency

### Timeouts
- Increase HTTP client timeout
- Increase context timeout
- Check network connectivity

### Panic Errors
- Add recovery middleware
- Check for nil pointers in responses

---

## Additional Resources

- [README.md](README.md) - Project overview
- [CODE_GENERATION.md](CODE_GENERATION.md) - Code generation SOP
- [Dhan API Documentation](https://dhanhq.co/docs/v2/)
- [oapi-codegen Documentation](https://github.com/oapi-codegen/oapi-codegen)

---

**Happy Trading! 📈**
