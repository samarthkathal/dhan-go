package benchmarks

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/samarthkathal/dhan-go/fulldepth"
)

// createDepth20BidPacket creates a mock 20-depth bid packet
func createDepth20BidPacket() []byte {
	// Header (12 bytes) + 20 entries (16 bytes each) = 332 bytes
	numRows := 20
	data := make([]byte, 12+numRows*16)

	binary.LittleEndian.PutUint16(data[0:2], uint16(len(data))) // Message length
	data[2] = fulldepth.FeedCodeBid // Response code (bid)
	data[3] = 1 // Exchange segment (NSE_EQ)
	binary.LittleEndian.PutUint32(data[4:8], 11536) // Security ID
	binary.LittleEndian.PutUint32(data[8:12], uint32(numRows)) // Num rows

	// Create 20 depth entries
	for i := 0; i < numRows; i++ {
		offset := 12 + i*16
		price := 1234.50 - float64(i)*0.05
		binary.LittleEndian.PutUint64(data[offset:offset+8], math.Float64bits(price))
		binary.LittleEndian.PutUint32(data[offset+8:offset+12], uint32(1000+i*100)) // Quantity
		binary.LittleEndian.PutUint32(data[offset+12:offset+16], uint32(10+i))      // Orders
	}
	return data
}

// createDepth20AskPacket creates a mock 20-depth ask packet
func createDepth20AskPacket() []byte {
	numRows := 20
	data := make([]byte, 12+numRows*16)

	binary.LittleEndian.PutUint16(data[0:2], uint16(len(data)))
	data[2] = fulldepth.FeedCodeAsk
	data[3] = 1
	binary.LittleEndian.PutUint32(data[4:8], 11536)
	binary.LittleEndian.PutUint32(data[8:12], uint32(numRows))

	for i := 0; i < numRows; i++ {
		offset := 12 + i*16
		price := 1235.00 + float64(i)*0.05
		binary.LittleEndian.PutUint64(data[offset:offset+8], math.Float64bits(price))
		binary.LittleEndian.PutUint32(data[offset+8:offset+12], uint32(1200+i*100))
		binary.LittleEndian.PutUint32(data[offset+12:offset+16], uint32(12+i))
	}
	return data
}

// createDepth200BidPacket creates a mock 200-depth bid packet
func createDepth200BidPacket() []byte {
	numRows := 200
	data := make([]byte, 12+numRows*16)

	binary.LittleEndian.PutUint16(data[0:2], uint16(len(data)))
	data[2] = fulldepth.FeedCodeBid
	data[3] = 1
	binary.LittleEndian.PutUint32(data[4:8], 11536)
	binary.LittleEndian.PutUint32(data[8:12], uint32(numRows))

	for i := 0; i < numRows; i++ {
		offset := 12 + i*16
		price := 1234.50 - float64(i)*0.01
		binary.LittleEndian.PutUint64(data[offset:offset+8], math.Float64bits(price))
		binary.LittleEndian.PutUint32(data[offset+8:offset+12], uint32(100+i*10))
		binary.LittleEndian.PutUint32(data[offset+12:offset+16], uint32(1+i%50))
	}
	return data
}

// BenchmarkParseDepthHeader benchmarks depth header parsing
func BenchmarkParseDepthHeader(b *testing.B) {
	data := createDepth20BidPacket()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := fulldepth.ParseDepthHeader(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseDepth20Bid benchmarks 20-depth bid parsing
func BenchmarkParseDepth20Bid(b *testing.B) {
	data := createDepth20BidPacket()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, err := fulldepth.ParseDepthData(data, fulldepth.Depth20)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseDepth20Ask benchmarks 20-depth ask parsing
func BenchmarkParseDepth20Ask(b *testing.B) {
	data := createDepth20AskPacket()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, err := fulldepth.ParseDepthData(data, fulldepth.Depth20)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseDepth200 benchmarks 200-depth parsing
func BenchmarkParseDepth200(b *testing.B) {
	data := createDepth200BidPacket()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, err := fulldepth.ParseDepthData(data, fulldepth.Depth200)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseDepth20Parallel benchmarks parallel 20-depth parsing
func BenchmarkParseDepth20Parallel(b *testing.B) {
	data := createDepth20BidPacket()
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, err := fulldepth.ParseDepthData(data, fulldepth.Depth20)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkParseDepth200Parallel benchmarks parallel 200-depth parsing
func BenchmarkParseDepth200Parallel(b *testing.B) {
	data := createDepth200BidPacket()
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, err := fulldepth.ParseDepthData(data, fulldepth.Depth200)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkFullDepthRoundTrip simulates receiving bid+ask and combining
func BenchmarkFullDepthRoundTrip(b *testing.B) {
	bidData := createDepth20BidPacket()
	askData := createDepth20AskPacket()
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

		// Simulate combining (like processDepthData does)
		_ = &fulldepth.FullDepthData{
			ExchangeSegment: bid.Header.ExchangeSegment,
			SecurityID:      bid.Header.SecurityID,
			Bids:            bid.Entries,
			Asks:            ask.Entries,
		}
	}
}
