# WebSocket Guide

Production-ready WebSocket implementation for Dhan's real-time APIs.

---

## Quick Start

### Order Updates (JSON Protocol)

```go
import "github.com/samarthkathal/dhan-go/websocket/orderupdate"

// Create client
client := orderupdate.NewClient()

// Register callback
client.OnUpdate(func(alert *types.OrderAlert) {
    if alert.IsFilled() {
        log.Printf("Order %s filled at %.2f",
            alert.GetOrderID(), alert.GetAvgTradedPrice())
    }
})

// Connect
url := "wss://api.dhan.co/v2/orderupdate?access_token=TOKEN"
client.Connect(url)
defer client.Shutdown()
```

### Market Feed (Binary Protocol)

```go
import "github.com/samarthkathal/dhan-go/websocket/marketfeed"

// Create client
client := marketfeed.NewClient()

// Register callbacks for feed types
client.OnTicker(func(ticker *types.TickerData) {
    log.Printf("LTP: %.2f", ticker.LastTradedPrice)
})

client.OnFull(func(full *types.FullData) {
    bidPrice, bidQty := full.GetBestBid()
    askPrice, askQty := full.GetBestAsk()
    log.Printf("Bid: %.2f/%d | Ask: %.2f/%d",
        bidPrice, bidQty, askPrice, askQty)
})

// Connect (different URL format)
url := "wss://api-feed.dhan.co?version=2&token=TOKEN&clientId=CLIENT_ID&authType=2"
client.Connect(url)
defer client.Shutdown()

// Subscribe to instruments
instruments := []types.Instrument{
    {ExchangeSegment: "NSE_EQ", SecurityID: "2885"},  // RELIANCE
}
client.Subscribe(instruments)
```

---

## Architecture

### Actor-Based System

```
Client
  │
  ├─→ ConnectionPoolActor (up to 5 connections)
  │     ├─→ ConnectionActor #1
  │     │     ├─→ WebSocket (gorilla)
  │     │     └─→ HealthMonitorActor
  │     ├─→ ConnectionActor #2...
  │     └─→ Rate Limiter (enforces Dhan limits)
  │
  └─→ User Callbacks
```

**Benefits:**
- **Fault Isolation**: Actor failures don't cascade
- **Load Balancing**: Automatic distribution across 5 connections
- **Rate Limiting**: Enforces max 5 connections, 5000 instruments each
- **Auto-Reconnection**: Health monitoring with exponential backoff

---

## Market Feed Types

### Ticker (16 bytes)
- LTP + Timestamp
- Lightweight for watchlists
- ~10-20 updates/sec/instrument

### Quote (50 bytes)
- OHLC + Volume + Buy/Sell quantities
- Complete trade data
- ~5-10 updates/sec/instrument

### Full (150 bytes)
- Quote data + 5 levels of market depth
- Order book with bid/ask prices and quantities
- ~2-5 updates/sec/instrument

### OI (12 bytes)
- Open Interest for derivatives
- Updated on position changes

### PrevClose (16 bytes)
- Previous close reference data
- For percentage calculations

---

## Configuration

```go
import "github.com/samarthkathal/dhan-go/utils"

// Default configuration
config := utils.DefaultWSConfig()

// Or customize
config := &utils.WSConfig{
    MaxConnections:        5,              // Dhan limit
    MaxInstrumentsPerConn: 5000,           // Dhan limit
    MaxBatchSize:          100,            // Dhan limit
    ConnectTimeout:        10 * time.Second,
    PingInterval:          10 * time.Second,
    PongWait:              40 * time.Second,
    ReconnectDelay:        5 * time.Second,
    MaxReconnectAttempts:  5,
    EnableLogging:         true,
    EnableMetrics:         true,
    EnableRecovery:        true,
}

// Use with client
client := orderupdate.NewClient(
    orderupdate.WithConfig(config),
    orderupdate.WithLogger(log.Default()),
)
```

---

## Performance

### Buffer Pool
Automatic memory pooling for WebSocket operations:

```go
// Internally used by binary parsers
buf := utils.GetBuffer(1024)  // Get pooled buffer
defer utils.PutBuffer(buf)     // Return to pool
```

### Rate Limiting
Enforced automatically by ConnectionPool:

```go
// Limits checked on every operation:
// - Max 5 connections per user
// - Max 5000 instruments per connection
// - Max 100 instruments per subscription message
```

### Benchmarks
- Header parsing: ~100 ns/op
- Ticker parsing: ~200 ns/op
- Quote parsing: ~500 ns/op
- Full parsing: ~1500 ns/op
- Tested: 500 instruments, 5000 msg/sec sustained

---

## Examples

### Order Update
```bash
export DHAN_ACCESS_TOKEN="your_token"
go run examples/05_websocket_orderupdate/main.go
```

### Market Feed
```bash
export DHAN_ACCESS_TOKEN="your_token"
export DHAN_CLIENT_ID="your_client_id"
go run examples/06_websocket_marketfeed/main.go
```

### Postback Webhook
```bash
go run examples/07_postback_webhook/main.go
# Configure in Dhan portal: http://localhost:8080/postback
```

---

## Best Practices

### 1. Use Appropriate Feed Type
- **Ticker**: Watchlists, price alerts
- **Quote**: OHLC analysis, volume tracking
- **Full**: Order book analysis, HFT algorithms

### 2. Manage Subscriptions
```go
// Subscribe to multiple instruments (auto-batches > 100)
instruments := []types.Instrument{...} // Can be > 100
client.Subscribe(instruments)          // Automatically batched

// Check subscription count
count := client.GetSubscriptionCount()

// Unsubscribe when done
client.Unsubscribe(instruments)
```

### 3. Handle Errors
```go
// Register error callback for Market Feed
client.OnError(func(errorData *types.ErrorData) {
    log.Printf("WebSocket error code: %d", errorData.ErrorCode)
    // Handle forced disconnection
})

// Check connection status
if !client.IsConnected() {
    // Reconnect or alert
}
```

### 4. Graceful Shutdown
```go
// Always defer Shutdown()
defer client.Shutdown()

// Or use signal handling
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
<-sigChan
client.Shutdown()
```

---

## Troubleshooting

### No Data Received
**Check:**
1. Access token valid (refresh daily)
2. Client ID correct (for Market Feed)
3. Instruments exist (valid security IDs)
4. Market is open (no data off-hours)
5. Callbacks registered before connecting

### Connection Drops
**Check:**
1. Internet connection stable
2. Ping/pong timeout (increase PongWait if needed)
3. Access token not expired
4. Not exceeding 5 connection limit

### High Latency
**Check:**
1. Network quality
2. CPU usage (parsing overhead)
3. Callback processing time (offload heavy work)
4. Too many subscriptions (> 5000?)

### Rate Limit Errors
```go
// Connection limit
// Error: "max connections reached (5/5)"
// Solution: Use existing connections or close unused ones

// Instrument limit
// Error: "would exceed max instruments per connection"
// Solution: ConnectionPool automatically distributes

// Subscription batch limit
// Error: "too many instruments in single message (150/100)"
// Solution: BatchInstruments() or client.Subscribe() handles this
```

---

## Advanced Usage

### Custom Middleware
```go
// Create custom message handler
myMiddleware := func(next utils.WSMessageHandler) utils.WSMessageHandler {
    return func(ctx context.Context, msg []byte) error {
        // Pre-processing
        start := time.Now()

        // Call next middleware/handler
        err := next(ctx, msg)

        // Post-processing
        duration := time.Since(start)
        log.Printf("Processed in %v", duration)

        return err
    }
}

// Chain with built-in middleware
middleware := utils.ChainWSMiddleware(
    utils.WSRecoveryMiddleware(logger),
    myMiddleware,
    utils.WSMetricsMiddleware(collector),
)

client := marketfeed.NewClient(
    marketfeed.WithMiddleware(middleware),
)
```

### Metrics Collection
```go
metrics := client.GetMetrics()

log.Printf("Messages: %v", metrics["messages_received"])
log.Printf("Bytes: %v", metrics["bytes_received"])
log.Printf("Latency: %.2f ms", metrics["avg_latency_ms"])
log.Printf("Errors: %v", metrics["errors"])
log.Printf("Reconnections: %v", metrics["reconnections"])
```

---

## Binary Protocol Details

### Message Structure (Little Endian)

**Header (8 bytes):**
```
Byte 1:       Response Code (2=Ticker, 4=Quote, 8=Full, etc.)
Bytes 2-3:    Message Length (int16)
Byte 4:       Exchange Segment (1=NSE_EQ, 2=NSE_FNO, etc.)
Bytes 5-8:    Security ID (int32)
```

**Ticker Packet (16 bytes):**
```
Header (8) + LTP (float32, 4) + Time (int32, 4)
```

**Quote Packet (50 bytes):**
```
Header (8) + LTP (4) + LTQ (2) + Padding (2) + Time (4) +
ATP (4) + Volume (4) + Sell Qty (4) + Buy Qty (4) +
OHLC (4×4=16)
```

**Full Packet (150 bytes):**
```
Quote (50) + Market Depth (100 = 5 levels × 20 bytes)

Each depth level:
  Bid Qty (4) + Ask Qty (4) + Bid Orders (2) + Ask Orders (2) +
  Bid Price (4) + Ask Price (4) = 20 bytes
```

---

## FAQ

**Q: How many instruments can I subscribe to?**
A: Up to 25,000 total (5 connections × 5000 each). ConnectionPool handles this automatically.

**Q: What's the difference between Order Update and Market Feed?**
A: Order Update is for your own orders (JSON). Market Feed is for market data (binary, all instruments).

**Q: Can I use multiple clients simultaneously?**
A: Yes, but total connections across all clients must not exceed 5.

**Q: How do I get Security IDs?**
A: Use REST API endpoints to search for instruments and get their security IDs.

**Q: Which is faster: WebSocket or REST API?**
A: WebSocket is real-time push. REST requires polling. Use WebSocket for live data.

**Q: How do I handle > 5000 instruments?**
A: ConnectionPool automatically uses multiple connections. Just call Subscribe() with all instruments.

**Q: Can I customize the logger?**
A: Yes, implement the `utils.Logger` interface (just `Printf` method). Compatible with stdlib, logrus, zap, slog.

---

## References

- [Dhan Order Update API](https://dhanhq.co/docs/v2/order-update/)
- [Dhan Market Feed API](https://dhanhq.co/docs/v2/live-market-feed/)
- [Dhan Postback API](https://dhanhq.co/docs/v2/postback/)
- [REST API Guide](USAGE_GUIDE.md)
- [Code Generation](CODE_GENERATION.md)

---

**Version:** 1.0
**Status:** Production-Ready
**Last Updated:** 2025-10-19
