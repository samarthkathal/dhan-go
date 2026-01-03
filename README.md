# dhan-go

Go client library for the [Dhan](https://dhan.co) trading API.

## Installation

```bash
go get github.com/samarthkathal/dhan-go
```

## Features

- **REST API** - Orders, positions, holdings, funds
- **MarketFeed WebSocket** - Real-time market data (Ticker, Quote, OI, Full depth)
- **OrderUpdate WebSocket** - Real-time order status updates
- **Connection pooling** - Handle 25,000+ instruments across 5 connections
- **Rate limiting** - Built-in rate limiter for API compliance
- **Middleware** - Logging, recovery, custom middleware support

## Quick Start

### REST API

```go
import (
    "github.com/samarthkathal/dhan-go/rest"
    "net/http"
)

client, _ := rest.NewClient(
    "https://api.dhan.co",
    "your-access-token",
    http.DefaultClient,
)

// Get holdings
holdings, _ := client.GetHoldings(ctx)

// Place order
order, _ := client.PlaceOrder(ctx, restgen.PlaceorderJSONRequestBody{
    TransactionType: restgen.OrderRequestTransactionTypeBUY,
    ExchangeSegment: restgen.OrderRequestExchangeSegmentNSEEQ,
    ProductType:     restgen.OrderRequestProductTypeCNC,
    OrderType:       restgen.OrderRequestOrderTypeMARKET,
    SecurityId:      "1333",
    Quantity:        1,
})
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

- `examples/rest/` - REST API usage
- `examples/marketfeed/` - Market data streaming
- `examples/orderupdate/` - Order tracking
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

## License

MIT License - see [LICENSE](./LICENSE)
