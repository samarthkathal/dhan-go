package benchmarks

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/samarthkathal/dhan-go/fulldepth"
	"github.com/samarthkathal/dhan-go/marketfeed"
	"github.com/samarthkathal/dhan-go/orderupdate"
)

// BenchmarkE2ETickerFlow simulates end-to-end ticker data flow
// Parse binary message -> dispatch callback (using With* API)
func BenchmarkE2ETickerFlow(b *testing.B) {
	tickerData := createTickerPacket()
	var callbackCount atomic.Int64

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := marketfeed.WithTickerData(tickerData, func(t *marketfeed.TickerData) error {
			callbackCount.Add(1)
			_ = t.LastTradedPrice
			return nil
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkE2EQuoteFlow simulates end-to-end quote data flow
func BenchmarkE2EQuoteFlow(b *testing.B) {
	quoteData := createQuotePacket()
	var callbackCount atomic.Int64

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := marketfeed.WithQuoteData(quoteData, func(q *marketfeed.QuoteData) error {
			callbackCount.Add(1)
			_ = q.Volume
			return nil
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkE2EFullDepthFlow simulates full depth data flow
// Parse bid + ask -> combine -> dispatch (using WithFullDepthData)
func BenchmarkE2EFullDepthFlow(b *testing.B) {
	bidData := createDepth20BidPacket()
	askData := createDepth20AskPacket()
	var callbackCount atomic.Int64

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := fulldepth.WithFullDepthData(bidData, askData, fulldepth.Depth20, func(f *fulldepth.FullDepthData) error {
			callbackCount.Add(1)
			_, _ = f.GetBestBid()
			_, _ = f.GetBestAsk()
			return nil
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkE2EOrderUpdateFlow simulates order update data flow
func BenchmarkE2EOrderUpdateFlow(b *testing.B) {
	var callbackCount atomic.Int64

	callback := func(alert *orderupdate.OrderAlert) {
		callbackCount.Add(1)
		if alert.IsFilled() {
			_ = alert.GetAvgTradedPrice()
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var alert orderupdate.OrderAlert
		if err := json.Unmarshal(sampleOrderAlertJSON, &alert); err != nil {
			b.Fatal(err)
		}
		callback(&alert)
	}
}

// BenchmarkE2EHighVolumeFlow simulates processing 1000 mixed messages
func BenchmarkE2EHighVolumeFlow(b *testing.B) {
	// Create mixed message types
	messages := make([]struct {
		msgType byte
		data    []byte
	}, 1000)

	for i := 0; i < 1000; i++ {
		switch i % 5 {
		case 0:
			messages[i] = struct {
				msgType byte
				data    []byte
			}{marketfeed.FeedCodeTicker, createTickerPacket()}
		case 1:
			messages[i] = struct {
				msgType byte
				data    []byte
			}{marketfeed.FeedCodeQuote, createQuotePacket()}
		case 2:
			messages[i] = struct {
				msgType byte
				data    []byte
			}{marketfeed.FeedCodeOI, createOIPacket()}
		case 3:
			messages[i] = struct {
				msgType byte
				data    []byte
			}{marketfeed.FeedCodePrevClose, createPrevClosePacket()}
		case 4:
			messages[i] = struct {
				msgType byte
				data    []byte
			}{marketfeed.FeedCodeFull, createFullPacket()}
		}
	}

	var processed atomic.Int64

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, msg := range messages {
			switch msg.msgType {
			case marketfeed.FeedCodeTicker:
				_ = marketfeed.WithTickerData(msg.data, func(t *marketfeed.TickerData) error {
					processed.Add(1)
					return nil
				})
			case marketfeed.FeedCodeQuote:
				_ = marketfeed.WithQuoteData(msg.data, func(q *marketfeed.QuoteData) error {
					processed.Add(1)
					return nil
				})
			case marketfeed.FeedCodeOI:
				_ = marketfeed.WithOIData(msg.data, func(o *marketfeed.OIData) error {
					processed.Add(1)
					return nil
				})
			case marketfeed.FeedCodePrevClose:
				_ = marketfeed.WithPrevCloseData(msg.data, func(p *marketfeed.PrevCloseData) error {
					processed.Add(1)
					return nil
				})
			case marketfeed.FeedCodeFull:
				_ = marketfeed.WithFullData(msg.data, func(f *marketfeed.FullData) error {
					processed.Add(1)
					return nil
				})
			}
		}
	}
}

// BenchmarkCallbackDispatchGoroutine benchmarks goroutine spawn overhead
// Shows overhead of spawning goroutines vs inline processing
func BenchmarkCallbackDispatchGoroutine(b *testing.B) {
	tickerData := createTickerPacket()
	var wg sync.WaitGroup

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = marketfeed.WithTickerData(tickerData, func(t *marketfeed.TickerData) error {
			// Copy data since we're passing to goroutine (data only valid in callback)
			tickerCopy := *t
			wg.Add(1)
			go func(ticker marketfeed.TickerData) {
				defer wg.Done()
				_ = ticker.LastTradedPrice
			}(tickerCopy)
			return nil
		})
	}

	wg.Wait()
}

// BenchmarkCallbackDispatchInline benchmarks inline callback (no goroutine)
func BenchmarkCallbackDispatchInline(b *testing.B) {
	tickerData := createTickerPacket()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = marketfeed.WithTickerData(tickerData, func(t *marketfeed.TickerData) error {
			// Inline processing - no need to copy since we use it within callback
			_ = t.LastTradedPrice
			return nil
		})
	}
}

// BenchmarkE2EParallelProcessing simulates parallel message processing
func BenchmarkE2EParallelProcessing(b *testing.B) {
	tickerData := createTickerPacket()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = marketfeed.WithTickerData(tickerData, func(t *marketfeed.TickerData) error {
				_ = t.LastTradedPrice
				return nil
			})
		}
	})
}

// BenchmarkE2EDepthParallelProcessing simulates parallel depth processing
func BenchmarkE2EDepthParallelProcessing(b *testing.B) {
	bidData := createDepth20BidPacket()
	askData := createDepth20AskPacket()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = fulldepth.WithFullDepthData(bidData, askData, fulldepth.Depth20, func(f *fulldepth.FullDepthData) error {
				_ = f.GetSpread()
				return nil
			})
		}
	})
}

// BenchmarkThroughputEstimate estimates messages per second
func BenchmarkThroughputEstimate(b *testing.B) {
	tickerData := createTickerPacket()

	b.ResetTimer()
	b.ReportAllocs()

	// Process as many messages as possible
	for i := 0; i < b.N; i++ {
		_ = marketfeed.WithTickerData(tickerData, func(t *marketfeed.TickerData) error {
			_ = t.LastTradedPrice
			return nil
		})
	}

	// b.N iterations completed - ns/op tells us throughput
	// If ns/op is 21ns, then throughput is ~47 million messages/second per core
}

// BenchmarkE2EDataRetention benchmarks the Copy() pattern for data retention
func BenchmarkE2EDataRetention(b *testing.B) {
	bidData := createDepth20BidPacket()
	askData := createDepth20AskPacket()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var retained fulldepth.FullDepthData
		_ = fulldepth.WithFullDepthData(bidData, askData, fulldepth.Depth20, func(f *fulldepth.FullDepthData) error {
			// Use Copy() to retain data beyond callback scope
			retained = f.Copy()
			return nil
		})
		// Use retained data outside callback
		_ = retained.GetSpread()
	}
}
