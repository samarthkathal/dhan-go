# dhan-go

Go client library for the [Dhan](https://dhan.co) trading API.

## Installation

```bash
go get github.com/samarthkathal/dhan-go
```

## Features

- **REST API** - Orders, positions, holdings, funds, data APIs
- **Data APIs** - Market Quote, Historical Data, Option Chain
- **MarketFeed WebSocket** - Real-time market data (Ticker, Quote, OI, Full depth)
- **FullDepth WebSocket** - 20/200-level market depth
- **OrderUpdate WebSocket** - Real-time order status updates
- **Connection pooling** - Handle 25,000+ instruments across 5 connections
- **Rate limiting** - Built-in rate limiter for API compliance
- **Middleware** - Logging, recovery, custom middleware support

## Quick Start

### REST API

```go
import "github.com/samarthkathal/dhan-go/rest"

client, _ := rest.NewClient(
    "https://api.dhan.co/v2",
    "your-access-token",
    nil, // uses http.DefaultClient
)

// Get holdings
holdings, _ := client.GetHoldings(ctx)

// Get LTP for instruments
ltp, _ := client.GetLTP(ctx, rest.MarketQuoteRequest{
    "NSE_EQ": {11536, 1333}, // TCS, HDFC Bank
})

// Get option chain
chain, _ := client.GetOptionChain(ctx, 13, "IDX_I", "2025-01-30")
```

### MarketFeed WebSocket

```go
import "github.com/samarthkathal/dhan-go/marketfeed"

client, _ := marketfeed.NewClient(
    "your-access-token",
    marketfeed.WithTickerCallback(func(data *marketfeed.TickerData) {
        fmt.Printf("LTP: %.2f\n", data.LastTradedPrice)
    }),
)

client.Connect(ctx)
client.Subscribe(ctx, []marketfeed.Instrument{
    {SecurityID: "1333", ExchangeSegment: marketfeed.ExchangeNSEEQ},
})

// For high-volume (100+ instruments), use PooledClient
pooled, _ := marketfeed.NewPooledClient("token", opts...)
```

### OrderUpdate WebSocket

```go
import "github.com/samarthkathal/dhan-go/orderupdate"

client, _ := orderupdate.NewClient(
    "your-access-token",
    orderupdate.WithOrderUpdateCallback(func(alert *orderupdate.OrderAlert) {
        fmt.Printf("Order %s: %s\n", alert.GetOrderID(), alert.GetStatus())

        if alert.IsFilled() {
            fmt.Printf("Filled at %.2f\n", alert.GetAvgTradedPrice())
        }
    }),
)

client.Connect(ctx)
```

### FullDepth WebSocket

```go
import "github.com/samarthkathal/dhan-go/fulldepth"

client, _ := fulldepth.NewClient(
    "your-access-token",
    "your-client-id",
    fulldepth.WithDepthLevel(fulldepth.Depth20), // or Depth200
    fulldepth.WithDepthCallback(func(data *fulldepth.FullDepthData) {
        bidPrice, _ := data.GetBestBid()
        askPrice, _ := data.GetBestAsk()
        fmt.Printf("Bid: %.2f | Ask: %.2f | Spread: %.2f\n",
            bidPrice, askPrice, data.GetSpread())
    }),
)

client.Connect(ctx)
client.Subscribe(ctx, []fulldepth.Instrument{
    {ExchangeSegment: "NSE_EQ", SecurityID: 11536},
})
```

## Configuration

### Custom WebSocket Config

```go
client, _ := marketfeed.NewClient(
    token,
    marketfeed.WithConfig(&marketfeed.WebSocketConfig{
        ConnectTimeout:       15 * time.Second,
        PingInterval:         10 * time.Second,
        ReconnectDelay:       3 * time.Second,
        MaxReconnectAttempts: 5,
    }),
)
```

### Rate Limiting

```go
client, _ := rest.NewClient(
    baseURL, token, httpClient,
    rest.WithDefaultRateLimiter(),
)
```

### Middleware

```go
import "github.com/samarthkathal/dhan-go/middleware"

// WebSocket middleware
client, _ := marketfeed.NewClient(
    token,
    marketfeed.WithMiddleware(
        middleware.ChainWSMiddleware(
            middleware.WSLoggingMiddleware(logger),
            middleware.WSRecoveryMiddleware(logger),
        ),
    ),
)
```

## Examples

See the [examples](./examples) directory for complete working examples:

- `examples/rest/` - REST API usage (including data APIs)
- `examples/marketfeed/` - Market data streaming
- `examples/orderupdate/` - Order tracking
- `examples/fulldepth/` - Full market depth (20/200 levels)
- `examples/combined/` - Full trading workflows

## API Reference

### REST Endpoints

| Method | Description |
|--------|-------------|
| `GetHoldings()` | Get portfolio holdings |
| `GetPositions()` | Get open positions |
| `GetOrders()` | Get today's orders |
| `GetFundLimits()` | Get fund/margin limits |
| `PlaceOrder()` | Place new order |
| `ModifyOrder()` | Modify existing order |
| `CancelOrder()` | Cancel order |

### Data APIs

| Method | Description |
|--------|-------------|
| `GetLTP()` | Last traded price for instruments |
| `GetOHLC()` | OHLC data for instruments |
| `GetQuote()` | Full quote with market depth |
| `GetHistoricalData()` | Daily OHLC candles |
| `GetIntradayData()` | Minute OHLC candles |
| `GetExpiredOptionsData()` | Historical data for expired options |
| `GetOptionChain()` | Option chain with greeks |
| `GetExpiryList()` | List of expiry dates |

### MarketFeed Data Types

| Callback | Data |
|----------|------|
| `WithTickerCallback` | LTP, last traded time |
| `WithQuoteCallback` | OHLC, volume, bid/ask totals |
| `WithOICallback` | Open interest |
| `WithPrevCloseCallback` | Previous close price |
| `WithFullCallback` | All data + 5-level market depth |

### OrderAlert Helpers

| Method | Description |
|--------|-------------|
| `IsFilled()` | Order completely filled |
| `IsPartiallyFilled()` | Order partially filled |
| `IsRejected()` | Order rejected |
| `IsCancelled()` | Order cancelled |
| `GetAvgTradedPrice()` | Average fill price |

### FullDepth Helpers

| Method | Description |
|--------|-------------|
| `GetBestBid()` | Best bid price and quantity |
| `GetBestAsk()` | Best ask price and quantity |
| `GetSpread()` | Bid-ask spread |
| `GetTotalBidQuantity()` | Sum of all bid quantities |
| `GetTotalAskQuantity()` | Sum of all ask quantities |

## License

MIT License - see [LICENSE](./LICENSE)
