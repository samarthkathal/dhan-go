# Dhan Go SDK Examples

## Prerequisites

- Go 1.21+
- Dhan API access token

```bash
export DHAN_ACCESS_TOKEN="your-token"
```

## Quick Start

```bash
cd examples/rest/01_basic
go run main.go
```

## Callback Data Lifecycle

**Important**: In WebSocket callbacks, data pointers are only valid during callback execution. The SDK uses object pooling for zero-allocation parsing, so data may be reused after the callback returns.

**To retain data beyond the callback**, you must copy it:

```go
// marketfeed - shallow copy works (all value types)
marketfeed.WithTickerCallback(func(data *marketfeed.TickerData) {
    myTicker := *data  // Safe: all fields are value types
})

// fulldepth - use Copy() method (has slices)
fulldepth.WithDepthCallback(func(data *fulldepth.FullDepthData) {
    myDepth := data.Copy()  // Required: slices need deep copy
})
```

See `marketfeed/04_with_metrics` and `fulldepth/03_data_retention` for complete examples.

## Examples

### REST API (`rest/`)

| Example | Description |
|---------|-------------|
| 01_basic | Get holdings, positions, orders |
| 02_place_order | Place market/limit orders |
| 03_modify_cancel_order | Modify and cancel orders |
| 04_with_rate_limiting | Use rate limiter |
| 05_with_custom_http | Custom HTTP client config |
| 06_error_handling | Error handling patterns |
| 07_graceful_shutdown | Signal handling, cleanup |
| 08_data_apis | Historical data, market quote, option chain |

### MarketFeed WebSocket (`marketfeed/`)

| Example | Description |
|---------|-------------|
| 01_basic_ticker | Simple ticker subscription |
| 02_all_data_types | Ticker, Quote, OI, PrevClose, Full callbacks |
| 03_custom_config | Custom timeouts, buffers |
| 04_with_metrics | **Data retention pattern** - storing data from callbacks |
| 05_with_middleware | Logging, recovery middleware |
| 06_pooled_basic | PooledClient for multiple connections |
| 07_pooled_high_volume | Subscribe to 100+ instruments |
| 08_graceful_shutdown | Clean disconnection |

### OrderUpdate WebSocket (`orderupdate/`)

| Example | Description |
|---------|-------------|
| 01_basic | Receive order alerts |
| 02_status_helpers | IsFilled, IsRejected, IsCancelled helpers |
| 03_custom_config | Custom timeouts |
| 05_graceful_shutdown | Clean disconnection |

### FullDepth WebSocket (`fulldepth/`)

| Example | Description |
|---------|-------------|
| 01_basic | 20-level market depth |
| 02_200_depth | 200-level market depth (NSE only) |
| 03_data_retention | **Data retention with Copy()** - order book tracking |

### Combined (`combined/`)

| Example | Description |
|---------|-------------|
| 01_all_clients | REST + MarketFeed + OrderUpdate together |
| 02_trading_workflow | Monitor price, place order, track execution |
