package benchmarks

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/samarthkathal/dhan-go/marketfeed"
)

// createTickerPacket creates a mock ticker packet (16 bytes)
func createTickerPacket() []byte {
	data := make([]byte, 16)
	data[0] = marketfeed.FeedCodeTicker // Response code
	binary.LittleEndian.PutUint16(data[1:3], 16) // Message length
	data[3] = 1 // Exchange segment
	binary.LittleEndian.PutUint32(data[4:8], 11536) // Security ID
	binary.LittleEndian.PutUint32(data[8:12], math.Float32bits(1234.50)) // LTP
	binary.LittleEndian.PutUint32(data[12:16], 1609459200) // Epoch
	return data
}

// createQuotePacket creates a mock quote packet (50 bytes)
func createQuotePacket() []byte {
	data := make([]byte, 50)
	data[0] = marketfeed.FeedCodeQuote
	binary.LittleEndian.PutUint16(data[1:3], 50)
	data[3] = 1
	binary.LittleEndian.PutUint32(data[4:8], 11536)
	binary.LittleEndian.PutUint32(data[8:12], math.Float32bits(1234.50))
	binary.LittleEndian.PutUint16(data[12:14], 100)
	binary.LittleEndian.PutUint32(data[14:18], 1609459200)
	binary.LittleEndian.PutUint32(data[18:22], math.Float32bits(1230.25))
	binary.LittleEndian.PutUint32(data[22:26], 1000000)
	binary.LittleEndian.PutUint32(data[26:30], 500000)
	binary.LittleEndian.PutUint32(data[30:34], 600000)
	binary.LittleEndian.PutUint32(data[34:38], math.Float32bits(1220.00))
	binary.LittleEndian.PutUint32(data[38:42], math.Float32bits(1225.00))
	binary.LittleEndian.PutUint32(data[42:46], math.Float32bits(1240.00))
	binary.LittleEndian.PutUint32(data[46:50], math.Float32bits(1210.00))
	return data
}

// createFullPacket creates a mock full packet (162 bytes)
func createFullPacket() []byte {
	data := make([]byte, 162)
	data[0] = marketfeed.FeedCodeFull
	binary.LittleEndian.PutUint16(data[1:3], 162)
	data[3] = 1
	binary.LittleEndian.PutUint32(data[4:8], 11536)
	binary.LittleEndian.PutUint32(data[8:12], math.Float32bits(1234.50))
	binary.LittleEndian.PutUint16(data[12:14], 100)
	binary.LittleEndian.PutUint32(data[14:18], 1609459200)
	binary.LittleEndian.PutUint32(data[18:22], math.Float32bits(1230.25))
	binary.LittleEndian.PutUint32(data[22:26], 1000000)
	binary.LittleEndian.PutUint32(data[26:30], 500000)
	binary.LittleEndian.PutUint32(data[30:34], 600000)
	binary.LittleEndian.PutUint32(data[34:38], 50000)  // OI
	binary.LittleEndian.PutUint32(data[38:42], 60000)  // Highest OI
	binary.LittleEndian.PutUint32(data[42:46], 40000)  // Lowest OI
	binary.LittleEndian.PutUint32(data[46:50], math.Float32bits(1220.00))
	binary.LittleEndian.PutUint32(data[50:54], math.Float32bits(1225.00))
	binary.LittleEndian.PutUint32(data[54:58], math.Float32bits(1240.00))
	binary.LittleEndian.PutUint32(data[58:62], math.Float32bits(1210.00))

	// Market depth (5 levels x 20 bytes each)
	for i := 0; i < 5; i++ {
		offset := 62 + (i * 20)
		binary.LittleEndian.PutUint32(data[offset:offset+4], 1000)      // Bid qty
		binary.LittleEndian.PutUint32(data[offset+4:offset+8], 1200)    // Ask qty
		binary.LittleEndian.PutUint16(data[offset+8:offset+10], 10)     // Bid orders
		binary.LittleEndian.PutUint16(data[offset+10:offset+12], 12)    // Ask orders
		binary.LittleEndian.PutUint32(data[offset+12:offset+16], math.Float32bits(1234.00-float32(i)*0.05))
		binary.LittleEndian.PutUint32(data[offset+16:offset+20], math.Float32bits(1235.00+float32(i)*0.05))
	}
	return data
}

// createOIPacket creates a mock OI packet (12 bytes)
func createOIPacket() []byte {
	data := make([]byte, 12)
	data[0] = marketfeed.FeedCodeOI
	binary.LittleEndian.PutUint16(data[1:3], 12)
	data[3] = 1
	binary.LittleEndian.PutUint32(data[4:8], 11536)
	binary.LittleEndian.PutUint32(data[8:12], 50000)
	return data
}

// createPrevClosePacket creates a mock prev close packet (16 bytes)
func createPrevClosePacket() []byte {
	data := make([]byte, 16)
	data[0] = marketfeed.FeedCodePrevClose
	binary.LittleEndian.PutUint16(data[1:3], 16)
	data[3] = 1
	binary.LittleEndian.PutUint32(data[4:8], 11536)
	binary.LittleEndian.PutUint32(data[8:12], math.Float32bits(1225.00))
	binary.LittleEndian.PutUint32(data[12:16], 45000)
	return data
}

// BenchmarkParseTickerData benchmarks ticker parsing
func BenchmarkParseTickerData(b *testing.B) {
	data := createTickerPacket()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := marketfeed.ParseTickerData(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseQuoteData benchmarks quote parsing
func BenchmarkParseQuoteData(b *testing.B) {
	data := createQuotePacket()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := marketfeed.ParseQuoteData(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseFullData benchmarks full data parsing
func BenchmarkParseFullData(b *testing.B) {
	data := createFullPacket()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := marketfeed.ParseFullData(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseOIData benchmarks OI parsing
func BenchmarkParseOIData(b *testing.B) {
	data := createOIPacket()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := marketfeed.ParseOIData(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParsePrevCloseData benchmarks prev close parsing
func BenchmarkParsePrevCloseData(b *testing.B) {
	data := createPrevClosePacket()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := marketfeed.ParsePrevCloseData(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseMarketFeedHeader benchmarks header parsing
func BenchmarkParseMarketFeedHeader(b *testing.B) {
	data := createTickerPacket()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := marketfeed.ParseMarketFeedHeader(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseTickerDataParallel benchmarks parallel ticker parsing
func BenchmarkParseTickerDataParallel(b *testing.B) {
	data := createTickerPacket()
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := marketfeed.ParseTickerData(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkParseFullDataParallel benchmarks parallel full data parsing
func BenchmarkParseFullDataParallel(b *testing.B) {
	data := createFullPacket()
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := marketfeed.ParseFullData(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkHighVolumeParsing simulates high-volume parsing (1000 messages)
func BenchmarkHighVolumeParsing(b *testing.B) {
	// Create a mix of message types
	packets := make([][]byte, 1000)
	for i := 0; i < 1000; i++ {
		switch i % 5 {
		case 0:
			packets[i] = createTickerPacket()
		case 1:
			packets[i] = createQuotePacket()
		case 2:
			packets[i] = createOIPacket()
		case 3:
			packets[i] = createPrevClosePacket()
		case 4:
			packets[i] = createFullPacket()
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, data := range packets {
			header, err := marketfeed.ParseMarketFeedHeader(data)
			if err != nil {
				b.Fatal(err)
			}

			switch header.ResponseCode {
			case marketfeed.FeedCodeTicker:
				_, _ = marketfeed.ParseTickerData(data)
			case marketfeed.FeedCodeQuote:
				_, _ = marketfeed.ParseQuoteData(data)
			case marketfeed.FeedCodeOI:
				_, _ = marketfeed.ParseOIData(data)
			case marketfeed.FeedCodePrevClose:
				_, _ = marketfeed.ParsePrevCloseData(data)
			case marketfeed.FeedCodeFull:
				_, _ = marketfeed.ParseFullData(data)
			}
		}
	}
}
