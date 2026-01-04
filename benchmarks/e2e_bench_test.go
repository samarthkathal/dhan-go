package benchmarks

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/samarthkathal/dhan-go/fulldepth"
	"github.com/samarthkathal/dhan-go/marketfeed"
	"github.com/samarthkathal/dhan-go/orderupdate"
)

// BenchmarkE2ETickerFlow simulates end-to-end ticker data flow
// Parse binary message -> route by type -> dispatch callback
func BenchmarkE2ETickerFlow(b *testing.B) {
	tickerData := createTickerPacket()
	var callbackCount atomic.Int64

	// Simulate callback
	callback := func(data *marketfeed.TickerData) {
		callbackCount.Add(1)
		_ = data.LastTradedPrice
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Parse header
		header, err := marketfeed.ParseMarketFeedHeader(tickerData)
		if err != nil {
			b.Fatal(err)
		}

		// Route by type
		if header.ResponseCode == marketfeed.FeedCodeTicker {
			ticker, err := marketfeed.ParseTickerData(tickerData)
			if err != nil {
				b.Fatal(err)
			}
			// Dispatch callback (inline, no goroutine for benchmark accuracy)
			callback(ticker)
		}
	}
}

// BenchmarkE2EQuoteFlow simulates end-to-end quote data flow
func BenchmarkE2EQuoteFlow(b *testing.B) {
	quoteData := createQuotePacket()
	var callbackCount atomic.Int64

	callback := func(data *marketfeed.QuoteData) {
		callbackCount.Add(1)
		_ = data.Volume
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		header, err := marketfeed.ParseMarketFeedHeader(quoteData)
		if err != nil {
			b.Fatal(err)
		}

		if header.ResponseCode == marketfeed.FeedCodeQuote {
			quote, err := marketfeed.ParseQuoteData(quoteData)
			if err != nil {
				b.Fatal(err)
			}
			callback(quote)
		}
	}
}

// BenchmarkE2EFullDepthFlow simulates full depth data flow
// Parse bid -> parse ask -> combine -> dispatch
func BenchmarkE2EFullDepthFlow(b *testing.B) {
	bidData := createDepth20BidPacket()
	askData := createDepth20AskPacket()
	var callbackCount atomic.Int64

	callback := func(data *fulldepth.FullDepthData) {
		callbackCount.Add(1)
		_, _ = data.GetBestBid()
		_, _ = data.GetBestAsk()
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Parse bid
		bid, _, err := fulldepth.ParseDepthData(bidData, fulldepth.Depth20)
		if err != nil {
			b.Fatal(err)
		}

		// Parse ask
		ask, _, err := fulldepth.ParseDepthData(askData, fulldepth.Depth20)
		if err != nil {
			b.Fatal(err)
		}

		// Combine
		combined := &fulldepth.FullDepthData{
			ExchangeSegment: bid.Header.ExchangeSegment,
			SecurityID:      bid.Header.SecurityID,
			Bids:            bid.Entries,
			Asks:            ask.Entries,
		}

		callback(combined)
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
				if _, err := marketfeed.ParseTickerData(msg.data); err == nil {
					processed.Add(1)
				}
			case marketfeed.FeedCodeQuote:
				if _, err := marketfeed.ParseQuoteData(msg.data); err == nil {
					processed.Add(1)
				}
			case marketfeed.FeedCodeOI:
				if _, err := marketfeed.ParseOIData(msg.data); err == nil {
					processed.Add(1)
				}
			case marketfeed.FeedCodePrevClose:
				if _, err := marketfeed.ParsePrevCloseData(msg.data); err == nil {
					processed.Add(1)
				}
			case marketfeed.FeedCodeFull:
				if _, err := marketfeed.ParseFullData(msg.data); err == nil {
					processed.Add(1)
				}
			}
		}
	}
}

// BenchmarkCallbackDispatchGoroutine benchmarks goroutine spawn overhead
func BenchmarkCallbackDispatchGoroutine(b *testing.B) {
	tickerData := createTickerPacket()
	var wg sync.WaitGroup

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ticker, _ := marketfeed.ParseTickerData(tickerData)

		wg.Add(1)
		go func(t *marketfeed.TickerData) {
			defer wg.Done()
			_ = t.LastTradedPrice
		}(ticker)
	}

	wg.Wait()
}

// BenchmarkCallbackDispatchInline benchmarks inline callback (no goroutine)
func BenchmarkCallbackDispatchInline(b *testing.B) {
	tickerData := createTickerPacket()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ticker, _ := marketfeed.ParseTickerData(tickerData)
		// Inline processing
		_ = ticker.LastTradedPrice
	}
}

// BenchmarkE2EParallelProcessing simulates parallel message processing
func BenchmarkE2EParallelProcessing(b *testing.B) {
	tickerData := createTickerPacket()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			header, _ := marketfeed.ParseMarketFeedHeader(tickerData)
			if header.ResponseCode == marketfeed.FeedCodeTicker {
				ticker, _ := marketfeed.ParseTickerData(tickerData)
				_ = ticker.LastTradedPrice
			}
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
			bid, _, _ := fulldepth.ParseDepthData(bidData, fulldepth.Depth20)
			ask, _, _ := fulldepth.ParseDepthData(askData, fulldepth.Depth20)

			combined := &fulldepth.FullDepthData{
				ExchangeSegment: bid.Header.ExchangeSegment,
				SecurityID:      bid.Header.SecurityID,
				Bids:            bid.Entries,
				Asks:            ask.Entries,
			}
			_ = combined.GetSpread()
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
		_, _ = marketfeed.ParseTickerData(tickerData)
	}

	// b.N iterations completed - ns/op tells us throughput
	// If ns/op is 21ns, then throughput is ~47 million messages/second per core
}

// Helper to create 200-depth ask packet (needed for e2e tests)
func createDepth200AskPacket() []byte {
	numRows := 200
	data := make([]byte, 12+numRows*16)

	binary.LittleEndian.PutUint16(data[0:2], uint16(len(data)))
	data[2] = fulldepth.FeedCodeAsk
	data[3] = 1
	binary.LittleEndian.PutUint32(data[4:8], 11536)
	binary.LittleEndian.PutUint32(data[8:12], uint32(numRows))

	for i := 0; i < numRows; i++ {
		offset := 12 + i*16
		price := 1235.00 + float64(i)*0.01
		binary.LittleEndian.PutUint64(data[offset:offset+8], math.Float64bits(price))
		binary.LittleEndian.PutUint32(data[offset+8:offset+12], uint32(100+i*10))
		binary.LittleEndian.PutUint32(data[offset+12:offset+16], uint32(1+i%50))
	}
	return data
}
