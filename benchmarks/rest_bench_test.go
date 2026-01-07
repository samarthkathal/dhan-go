package benchmarks

import (
	"encoding/json"
	"testing"

	"github.com/samarthkathal/dhan-go/rest"
)

// Sample JSON responses for benchmarking
var sampleLTPResponseJSON = []byte(`{
	"status": "success",
	"data": {
		"NSE_EQ": {
			"11536": {"security_id": 11536, "last_price": 3456.75, "last_traded_time": "2024-01-15T10:30:00Z"},
			"1333": {"security_id": 1333, "last_price": 1567.25, "last_traded_time": "2024-01-15T10:30:01Z"}
		},
		"NSE_FNO": {
			"49081": {"security_id": 49081, "last_price": 245.50, "last_traded_time": "2024-01-15T10:30:02Z"}
		}
	}
}`)

var sampleOHLCResponseJSON = []byte(`{
	"status": "success",
	"data": {
		"NSE_EQ": {
			"11536": {"security_id": 11536, "last_price": 3456.75, "open": 3440.00, "high": 3480.00, "low": 3430.00, "close": 3450.00}
		}
	}
}`)

var sampleQuoteResponseJSON = []byte(`{
	"status": "success",
	"data": {
		"NSE_EQ": {
			"11536": {
				"security_id": 11536,
				"last_price": 3456.75,
				"last_traded_qty": 100,
				"last_traded_time": "2024-01-15T10:30:00Z",
				"open": 3440.00,
				"high": 3480.00,
				"low": 3430.00,
				"close": 3450.00,
				"volume": 1234567,
				"oi": 500000,
				"avg_price": 3455.50,
				"total_buy_qty": 100000,
				"total_sell_qty": 120000,
				"lower_circuit": 3100.00,
				"upper_circuit": 3800.00,
				"bid": [
					{"price": 3456.50, "quantity": 100, "orders": 5},
					{"price": 3456.25, "quantity": 200, "orders": 8},
					{"price": 3456.00, "quantity": 150, "orders": 6},
					{"price": 3455.75, "quantity": 300, "orders": 10},
					{"price": 3455.50, "quantity": 250, "orders": 7}
				],
				"ask": [
					{"price": 3457.00, "quantity": 100, "orders": 4},
					{"price": 3457.25, "quantity": 180, "orders": 7},
					{"price": 3457.50, "quantity": 220, "orders": 9},
					{"price": 3457.75, "quantity": 280, "orders": 11},
					{"price": 3458.00, "quantity": 320, "orders": 12}
				]
			}
		}
	}
}`)

var sampleOptionChainResponseJSON = []byte(`{
	"status": "success",
	"data": {
		"last_price": 21500.50,
		"oc": {
			"21000": {
				"ce": {
					"security_id": 49081,
					"last_price": 520.50,
					"oi": 100000,
					"volume": 50000,
					"prev_close": 510.00,
					"prev_volume": 45000,
					"implied_volatility": 0.18,
					"greeks": {"delta": 0.65, "theta": -5.2, "gamma": 0.002, "vega": 12.5},
					"top_bid_price": 520.00,
					"top_bid_quantity": 50,
					"top_ask_price": 521.00,
					"top_ask_quantity": 75
				},
				"pe": {
					"security_id": 49082,
					"last_price": 42.50,
					"oi": 80000,
					"volume": 30000,
					"prev_close": 45.00,
					"prev_volume": 28000,
					"implied_volatility": 0.20,
					"greeks": {"delta": -0.12, "theta": -3.8, "gamma": 0.001, "vega": 8.2},
					"top_bid_price": 42.00,
					"top_bid_quantity": 100,
					"top_ask_price": 43.00,
					"top_ask_quantity": 120
				}
			},
			"21500": {
				"ce": {
					"security_id": 49083,
					"last_price": 125.75,
					"oi": 150000,
					"volume": 80000,
					"prev_close": 130.00,
					"prev_volume": 75000,
					"implied_volatility": 0.16,
					"greeks": {"delta": 0.52, "theta": -6.5, "gamma": 0.003, "vega": 15.0},
					"top_bid_price": 125.50,
					"top_bid_quantity": 200,
					"top_ask_price": 126.00,
					"top_ask_quantity": 180
				},
				"pe": {
					"security_id": 49084,
					"last_price": 115.25,
					"oi": 140000,
					"volume": 70000,
					"prev_close": 118.00,
					"prev_volume": 65000,
					"implied_volatility": 0.17,
					"greeks": {"delta": -0.48, "theta": -6.0, "gamma": 0.003, "vega": 14.5},
					"top_bid_price": 115.00,
					"top_bid_quantity": 150,
					"top_ask_price": 115.50,
					"top_ask_quantity": 170
				}
			}
		}
	}
}`)

// BenchmarkMarshalMarketQuoteRequest benchmarks request serialization
func BenchmarkMarshalMarketQuoteRequest(b *testing.B) {
	req := rest.MarketQuoteRequest{
		"NSE_EQ":  {11536, 1333, 2885, 14366},
		"NSE_FNO": {49081, 49082, 49083},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalLTPResponse benchmarks LTP response parsing with easyjson
func BenchmarkUnmarshalLTPResponse(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var resp rest.LTPResponse
		if err := resp.UnmarshalJSON(sampleLTPResponseJSON); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalLTPResponseStdJSON benchmarks LTP response parsing with std json
func BenchmarkUnmarshalLTPResponseStdJSON(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var resp rest.LTPResponse
		if err := json.Unmarshal(sampleLTPResponseJSON, &resp); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalOHLCResponse benchmarks OHLC response parsing with easyjson
func BenchmarkUnmarshalOHLCResponse(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var resp rest.OHLCResponse
		if err := resp.UnmarshalJSON(sampleOHLCResponseJSON); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalOHLCResponseStdJSON benchmarks OHLC response parsing with std json
func BenchmarkUnmarshalOHLCResponseStdJSON(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var resp rest.OHLCResponse
		if err := json.Unmarshal(sampleOHLCResponseJSON, &resp); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalQuoteResponse benchmarks full quote response parsing with easyjson
func BenchmarkUnmarshalQuoteResponse(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var resp rest.QuoteResponse
		if err := resp.UnmarshalJSON(sampleQuoteResponseJSON); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalQuoteResponseStdJSON benchmarks full quote response parsing with std json
func BenchmarkUnmarshalQuoteResponseStdJSON(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var resp rest.QuoteResponse
		if err := json.Unmarshal(sampleQuoteResponseJSON, &resp); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalOptionChainResponse benchmarks option chain response parsing with easyjson
func BenchmarkUnmarshalOptionChainResponse(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var resp rest.OptionChainResponse
		if err := resp.UnmarshalJSON(sampleOptionChainResponseJSON); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalOptionChainResponseStdJSON benchmarks option chain response parsing with std json
func BenchmarkUnmarshalOptionChainResponseStdJSON(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var resp rest.OptionChainResponse
		if err := json.Unmarshal(sampleOptionChainResponseJSON, &resp); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalLTPResponseParallel benchmarks parallel LTP response parsing with easyjson
func BenchmarkUnmarshalLTPResponseParallel(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var resp rest.LTPResponse
			if err := resp.UnmarshalJSON(sampleLTPResponseJSON); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkUnmarshalQuoteResponseParallel benchmarks parallel quote response parsing with easyjson
func BenchmarkUnmarshalQuoteResponseParallel(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var resp rest.QuoteResponse
			if err := resp.UnmarshalJSON(sampleQuoteResponseJSON); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkSecureHTTPClientCreation benchmarks creating a secure HTTP client
func BenchmarkSecureHTTPClientCreation(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = rest.SecureHTTPClient()
	}
}

// BenchmarkJSONRoundTrip benchmarks a full JSON marshal/unmarshal cycle
func BenchmarkJSONRoundTrip(b *testing.B) {
	req := rest.MarketQuoteRequest{
		"NSE_EQ":  {11536, 1333},
		"NSE_FNO": {49081},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Marshal request
		data, err := json.Marshal(req)
		if err != nil {
			b.Fatal(err)
		}

		// Unmarshal response
		var resp rest.LTPResponse
		if err := json.Unmarshal(sampleLTPResponseJSON, &resp); err != nil {
			b.Fatal(err)
		}
		_ = data
	}
}
