package benchmarks

import (
	"encoding/json"
	"testing"

	"github.com/samarthkathal/dhan-go/orderupdate"
)

// Sample OrderAlert JSON for benchmarking
var sampleOrderAlertJSON = []byte(`{
	"Type": "order_alert",
	"Data": {
		"orderNo": "1234567890",
		"ExchOrderNo": "EX123456789",
		"dhanClientId": "1000123456",
		"correlationId": "corr-123",
		"tradingSymbol": "TCS",
		"securityId": "11536",
		"exchangeSegment": "NSE_EQ",
		"productType": "CNC",
		"orderType": "LIMIT",
		"validity": "DAY",
		"transactionType": "BUY",
		"quantity": 100,
		"disclosedQuantity": 0,
		"price": 3456.75,
		"triggerPrice": 0,
		"TradedQty": 100,
		"TradedPrice": 3456.50,
		"AvgTradedPrice": 3456.50,
		"remainingQuantity": 0,
		"Status": "TRADED",
		"orderStatus": "TRADED",
		"ReasonCode": "",
		"ReasonDescription": "",
		"orderDateTime": "2024-01-15T10:30:00Z",
		"exchOrderTime": "2024-01-15T10:30:01Z",
		"lastUpdatedTime": "2024-01-15T10:30:02Z",
		"afterMarketOrder": false
	}
}`)

var sampleRejectedAlertJSON = []byte(`{
	"Type": "order_alert",
	"Data": {
		"orderNo": "1234567891",
		"ExchOrderNo": "",
		"dhanClientId": "1000123456",
		"correlationId": "corr-124",
		"tradingSymbol": "RELIANCE",
		"securityId": "2885",
		"exchangeSegment": "NSE_EQ",
		"productType": "INTRADAY",
		"orderType": "MARKET",
		"validity": "DAY",
		"transactionType": "BUY",
		"quantity": 500,
		"disclosedQuantity": 0,
		"price": 0,
		"triggerPrice": 0,
		"TradedQty": 0,
		"TradedPrice": 0,
		"AvgTradedPrice": 0,
		"remainingQuantity": 500,
		"Status": "REJECTED",
		"orderStatus": "REJECTED",
		"ReasonCode": "RMS:Rule: Max Price Breached",
		"ReasonDescription": "Order price exceeds circuit limits",
		"orderDateTime": "2024-01-15T10:35:00Z",
		"exchOrderTime": "",
		"lastUpdatedTime": "2024-01-15T10:35:00Z",
		"afterMarketOrder": false
	}
}`)

var sampleFNOAlertJSON = []byte(`{
	"Type": "order_alert",
	"Data": {
		"orderNo": "1234567892",
		"ExchOrderNo": "EX123456790",
		"dhanClientId": "1000123456",
		"correlationId": "corr-125",
		"tradingSymbol": "NIFTY25JAN21500CE",
		"securityId": "49081",
		"exchangeSegment": "NSE_FNO",
		"productType": "INTRADAY",
		"orderType": "LIMIT",
		"validity": "DAY",
		"transactionType": "BUY",
		"quantity": 50,
		"disclosedQuantity": 0,
		"price": 125.50,
		"triggerPrice": 0,
		"TradedQty": 50,
		"TradedPrice": 125.50,
		"AvgTradedPrice": 125.50,
		"remainingQuantity": 0,
		"Status": "TRADED",
		"orderStatus": "TRADED",
		"expiryDate": "2025-01-30",
		"strikePrice": 21500,
		"optionType": "CALL",
		"instrumentType": "OPTIDX",
		"orderDateTime": "2024-01-15T10:40:00Z",
		"exchOrderTime": "2024-01-15T10:40:01Z",
		"lastUpdatedTime": "2024-01-15T10:40:02Z",
		"afterMarketOrder": false
	}
}`)

// BenchmarkUnmarshalOrderAlert benchmarks single alert parsing
func BenchmarkUnmarshalOrderAlert(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var alert orderupdate.OrderAlert
		if err := json.Unmarshal(sampleOrderAlertJSON, &alert); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalRejectedAlert benchmarks rejected alert parsing
func BenchmarkUnmarshalRejectedAlert(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var alert orderupdate.OrderAlert
		if err := json.Unmarshal(sampleRejectedAlertJSON, &alert); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalFNOAlert benchmarks F&O alert parsing
func BenchmarkUnmarshalFNOAlert(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var alert orderupdate.OrderAlert
		if err := json.Unmarshal(sampleFNOAlertJSON, &alert); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkUnmarshalOrderAlertParallel benchmarks parallel alert parsing
func BenchmarkUnmarshalOrderAlertParallel(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var alert orderupdate.OrderAlert
			if err := json.Unmarshal(sampleOrderAlertJSON, &alert); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkOrderAlertHelpers benchmarks helper method calls
func BenchmarkOrderAlertHelpers(b *testing.B) {
	var alert orderupdate.OrderAlert
	if err := json.Unmarshal(sampleOrderAlertJSON, &alert); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = alert.IsOrderAlert()
		_ = alert.GetOrderID()
		_ = alert.GetStatus()
		_ = alert.IsFilled()
		_ = alert.IsPartiallyFilled()
		_ = alert.IsRejected()
		_ = alert.IsCancelled()
		_ = alert.GetTradedQuantity()
		_ = alert.GetAvgTradedPrice()
	}
}

// BenchmarkOrderAlertIsFilled benchmarks IsFilled check
func BenchmarkOrderAlertIsFilled(b *testing.B) {
	var alert orderupdate.OrderAlert
	if err := json.Unmarshal(sampleOrderAlertJSON, &alert); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = alert.IsFilled()
	}
}

// BenchmarkUnmarshalOrderAlertBatch benchmarks parsing multiple alerts
func BenchmarkUnmarshalOrderAlertBatch(b *testing.B) {
	// Create a batch of different alert types
	alerts := [][]byte{
		sampleOrderAlertJSON,
		sampleRejectedAlertJSON,
		sampleFNOAlertJSON,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, alertJSON := range alerts {
			var alert orderupdate.OrderAlert
			if err := json.Unmarshal(alertJSON, &alert); err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkOrderAlertParseAndCheck benchmarks full parse + check workflow
func BenchmarkOrderAlertParseAndCheck(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var alert orderupdate.OrderAlert
		if err := json.Unmarshal(sampleOrderAlertJSON, &alert); err != nil {
			b.Fatal(err)
		}

		// Simulate typical usage
		if alert.IsOrderAlert() {
			if alert.IsFilled() {
				_ = alert.GetAvgTradedPrice()
			} else if alert.IsRejected() {
				_ = alert.GetStatus()
			}
		}
	}
}

// BenchmarkHighVolumeAlertParsing simulates high-volume alert parsing
func BenchmarkHighVolumeAlertParsing(b *testing.B) {
	// Create 100 different alerts (simulating varied order updates)
	alerts := make([][]byte, 100)
	for i := 0; i < 100; i++ {
		switch i % 3 {
		case 0:
			alerts[i] = sampleOrderAlertJSON
		case 1:
			alerts[i] = sampleRejectedAlertJSON
		case 2:
			alerts[i] = sampleFNOAlertJSON
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, alertJSON := range alerts {
			var alert orderupdate.OrderAlert
			if err := json.Unmarshal(alertJSON, &alert); err != nil {
				b.Fatal(err)
			}

			// Check status
			if alert.IsFilled() || alert.IsRejected() || alert.IsCancelled() {
				_ = alert.GetOrderID()
			}
		}
	}
}
