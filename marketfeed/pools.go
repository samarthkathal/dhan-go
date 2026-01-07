package marketfeed

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync"
)

// bytesToFloat32 converts 4 bytes to float32 (Little Endian) - zero allocation
func bytesToFloat32(b []byte) float32 {
	bits := binary.LittleEndian.Uint32(b)
	return math.Float32frombits(bits)
}

// Pools for struct reuse to reduce GC pressure in high-throughput scenarios
var (
	headerPool = sync.Pool{
		New: func() interface{} { return &MarketFeedHeader{} },
	}
	tickerPool = sync.Pool{
		New: func() interface{} { return &TickerData{} },
	}
	quotePool = sync.Pool{
		New: func() interface{} { return &QuoteData{} },
	}
	oiPool = sync.Pool{
		New: func() interface{} { return &OIData{} },
	}
	prevClosePool = sync.Pool{
		New: func() interface{} { return &PrevCloseData{} },
	}
	fullPool = sync.Pool{
		New: func() interface{} { return &FullData{} },
	}
)

// acquireHeader gets a header from the pool
func acquireHeader() *MarketFeedHeader {
	return headerPool.Get().(*MarketFeedHeader)
}

// releaseHeader returns a header to the pool
func releaseHeader(h *MarketFeedHeader) {
	if h == nil {
		return
	}
	*h = MarketFeedHeader{}
	headerPool.Put(h)
}

// acquireTicker gets a ticker from the pool
func acquireTicker() *TickerData {
	return tickerPool.Get().(*TickerData)
}

// releaseTicker returns a ticker to the pool
func releaseTicker(t *TickerData) {
	if t == nil {
		return
	}
	*t = TickerData{}
	tickerPool.Put(t)
}

// acquireQuote gets a quote from the pool
func acquireQuote() *QuoteData {
	return quotePool.Get().(*QuoteData)
}

// releaseQuote returns a quote to the pool
func releaseQuote(q *QuoteData) {
	if q == nil {
		return
	}
	*q = QuoteData{}
	quotePool.Put(q)
}

// acquireOI gets an OI data from the pool
func acquireOI() *OIData {
	return oiPool.Get().(*OIData)
}

// releaseOI returns an OI data to the pool
func releaseOI(o *OIData) {
	if o == nil {
		return
	}
	*o = OIData{}
	oiPool.Put(o)
}

// acquirePrevClose gets a prev close from the pool
func acquirePrevClose() *PrevCloseData {
	return prevClosePool.Get().(*PrevCloseData)
}

// releasePrevClose returns a prev close to the pool
func releasePrevClose(p *PrevCloseData) {
	if p == nil {
		return
	}
	*p = PrevCloseData{}
	prevClosePool.Put(p)
}

// acquireFull gets a full data from the pool
func acquireFull() *FullData {
	return fullPool.Get().(*FullData)
}

// releaseFull returns a full data to the pool
func releaseFull(f *FullData) {
	if f == nil {
		return
	}
	*f = FullData{}
	fullPool.Put(f)
}

// parseTickerDataPooled parses a ticker packet using pooled allocation
// Caller MUST call releaseTicker when done with the data
func parseTickerDataPooled(data []byte) (*TickerData, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("insufficient data for ticker: got %d bytes, need 16", len(data))
	}

	if data[0] != FeedCodeTicker {
		return nil, fmt.Errorf("invalid response code for ticker: %d", data[0])
	}

	ticker := acquireTicker()
	ticker.Header.ResponseCode = data[0]
	ticker.Header.MessageLength = int16(binary.LittleEndian.Uint16(data[1:3]))
	ticker.Header.ExchangeSegment = data[3]
	ticker.Header.SecurityID = int32(binary.LittleEndian.Uint32(data[4:8]))
	ticker.LastTradedPrice = bytesToFloat32(data[8:12])
	ticker.TradeTimeEpoch = int32(binary.LittleEndian.Uint32(data[12:16]))

	return ticker, nil
}

// parseQuoteDataPooled parses a quote packet using pooled allocation
// Caller MUST call releaseQuote when done with the data
func parseQuoteDataPooled(data []byte) (*QuoteData, error) {
	if len(data) < 50 {
		return nil, fmt.Errorf("insufficient data for quote: got %d bytes, need 50", len(data))
	}

	if data[0] != FeedCodeQuote {
		return nil, fmt.Errorf("invalid response code for quote: %d", data[0])
	}

	quote := acquireQuote()
	quote.Header.ResponseCode = data[0]
	quote.Header.MessageLength = int16(binary.LittleEndian.Uint16(data[1:3]))
	quote.Header.ExchangeSegment = data[3]
	quote.Header.SecurityID = int32(binary.LittleEndian.Uint32(data[4:8]))
	quote.LastTradedPrice = bytesToFloat32(data[8:12])
	quote.LastTradedQuantity = int16(binary.LittleEndian.Uint16(data[12:14]))
	quote.TradeTimeEpoch = int32(binary.LittleEndian.Uint32(data[14:18]))
	quote.AverageTradedPrice = bytesToFloat32(data[18:22])
	quote.Volume = int32(binary.LittleEndian.Uint32(data[22:26]))
	quote.TotalSellQuantity = int32(binary.LittleEndian.Uint32(data[26:30]))
	quote.TotalBuyQuantity = int32(binary.LittleEndian.Uint32(data[30:34]))
	quote.DayOpen = bytesToFloat32(data[34:38])
	quote.DayClose = bytesToFloat32(data[38:42])
	quote.DayHigh = bytesToFloat32(data[42:46])
	quote.DayLow = bytesToFloat32(data[46:50])

	return quote, nil
}

// parseFullDataPooled parses a full packet using pooled allocation
// Caller MUST call releaseFull when done with the data
func parseFullDataPooled(data []byte) (*FullData, error) {
	if len(data) < 162 {
		return nil, fmt.Errorf("insufficient data for full: got %d bytes, need 162", len(data))
	}

	if data[0] != FeedCodeFull {
		return nil, fmt.Errorf("invalid response code for full: %d", data[0])
	}

	full := acquireFull()
	full.Header.ResponseCode = data[0]
	full.Header.MessageLength = int16(binary.LittleEndian.Uint16(data[1:3]))
	full.Header.ExchangeSegment = data[3]
	full.Header.SecurityID = int32(binary.LittleEndian.Uint32(data[4:8]))
	full.LastTradedPrice = bytesToFloat32(data[8:12])
	full.LastTradedQuantity = int16(binary.LittleEndian.Uint16(data[12:14]))
	full.TradeTimeEpoch = int32(binary.LittleEndian.Uint32(data[14:18]))
	full.AverageTradedPrice = bytesToFloat32(data[18:22])
	full.Volume = int32(binary.LittleEndian.Uint32(data[22:26]))
	full.TotalSellQuantity = int32(binary.LittleEndian.Uint32(data[26:30]))
	full.TotalBuyQuantity = int32(binary.LittleEndian.Uint32(data[30:34]))
	full.OpenInterest = int32(binary.LittleEndian.Uint32(data[34:38]))
	full.HighestOI = int32(binary.LittleEndian.Uint32(data[38:42]))
	full.LowestOI = int32(binary.LittleEndian.Uint32(data[42:46]))
	full.DayOpen = bytesToFloat32(data[46:50])
	full.DayClose = bytesToFloat32(data[50:54])
	full.DayHigh = bytesToFloat32(data[54:58])
	full.DayLow = bytesToFloat32(data[58:62])

	// Parse 5 levels of market depth
	depthOffset := 62
	for i := 0; i < 5; i++ {
		offset := depthOffset + (i * 20)
		full.Depth[i].BidQuantity = int32(binary.LittleEndian.Uint32(data[offset : offset+4]))
		full.Depth[i].AskQuantity = int32(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		full.Depth[i].BidOrderCount = int16(binary.LittleEndian.Uint16(data[offset+8 : offset+10]))
		full.Depth[i].AskOrderCount = int16(binary.LittleEndian.Uint16(data[offset+10 : offset+12]))
		full.Depth[i].BidPrice = bytesToFloat32(data[offset+12 : offset+16])
		full.Depth[i].AskPrice = bytesToFloat32(data[offset+16 : offset+20])
	}

	return full, nil
}

// parseOIDataPooled parses an OI packet using pooled allocation
// Caller MUST call releaseOI when done with the data
func parseOIDataPooled(data []byte) (*OIData, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("insufficient data for OI: got %d bytes, need 12", len(data))
	}

	if data[0] != FeedCodeOI {
		return nil, fmt.Errorf("invalid response code for OI: %d", data[0])
	}

	oi := acquireOI()
	oi.Header.ResponseCode = data[0]
	oi.Header.MessageLength = int16(binary.LittleEndian.Uint16(data[1:3]))
	oi.Header.ExchangeSegment = data[3]
	oi.Header.SecurityID = int32(binary.LittleEndian.Uint32(data[4:8]))
	oi.OpenInterest = int32(binary.LittleEndian.Uint32(data[8:12]))

	return oi, nil
}

// parsePrevCloseDataPooled parses a prev close packet using pooled allocation
// Caller MUST call releasePrevClose when done with the data
func parsePrevCloseDataPooled(data []byte) (*PrevCloseData, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("insufficient data for prev close: got %d bytes, need 16", len(data))
	}

	if data[0] != FeedCodePrevClose {
		return nil, fmt.Errorf("invalid response code for prev close: %d", data[0])
	}

	prevClose := acquirePrevClose()
	prevClose.Header.ResponseCode = data[0]
	prevClose.Header.MessageLength = int16(binary.LittleEndian.Uint16(data[1:3]))
	prevClose.Header.ExchangeSegment = data[3]
	prevClose.Header.SecurityID = int32(binary.LittleEndian.Uint32(data[4:8]))
	prevClose.PreviousClosePrice = bytesToFloat32(data[8:12])
	prevClose.PreviousOpenInterest = int32(binary.LittleEndian.Uint32(data[12:16]))

	return prevClose, nil
}

// parseErrorData parses an error packet and returns structured information.
func parseErrorData(data []byte) (*ErrorData, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("insufficient data for error packet: got %d bytes, need 10", len(data))
	}

	if data[0] != FeedCodeError {
		return nil, fmt.Errorf("invalid response code for error packet: %d", data[0])
	}

	errData := &ErrorData{}
	errData.Header.ResponseCode = data[0]
	errData.Header.MessageLength = int16(binary.LittleEndian.Uint16(data[1:3]))
	errData.Header.ExchangeSegment = data[3]
	errData.Header.SecurityID = int32(binary.LittleEndian.Uint32(data[4:8]))
	errData.ErrorCode = int16(binary.LittleEndian.Uint16(data[8:10]))

	return errData, nil
}

// =============================================================================
// SAFE CALLBACK-BASED API (Recommended for library users)
// =============================================================================
// These functions automatically manage pool lifecycle - no manual Release needed.
// The pooled object is only valid within the callback scope.

// WithTickerData parses ticker data and calls fn with the result.
// The TickerData is automatically returned to the pool after fn returns.
// Data is only valid during callback. To retain: myTicker := *ticker
func WithTickerData(data []byte, fn func(*TickerData) error) error {
	ticker, err := parseTickerDataPooled(data)
	if err != nil {
		return err
	}
	defer releaseTicker(ticker)
	return fn(ticker)
}

// WithQuoteData parses quote data and calls fn with the result.
// The QuoteData is automatically returned to the pool after fn returns.
// Data is only valid during callback. To retain: myQuote := *quote
func WithQuoteData(data []byte, fn func(*QuoteData) error) error {
	quote, err := parseQuoteDataPooled(data)
	if err != nil {
		return err
	}
	defer releaseQuote(quote)
	return fn(quote)
}

// WithFullData parses full data and calls fn with the result.
// The FullData is automatically returned to the pool after fn returns.
// Data is only valid during callback. To retain: myFull := *full
func WithFullData(data []byte, fn func(*FullData) error) error {
	full, err := parseFullDataPooled(data)
	if err != nil {
		return err
	}
	defer releaseFull(full)
	return fn(full)
}

// WithOIData parses OI data and calls fn with the result.
// The OIData is automatically returned to the pool after fn returns.
// Data is only valid during callback. To retain: myOI := *oi
func WithOIData(data []byte, fn func(*OIData) error) error {
	oi, err := parseOIDataPooled(data)
	if err != nil {
		return err
	}
	defer releaseOI(oi)
	return fn(oi)
}

// WithPrevCloseData parses prev close data and calls fn with the result.
// The PrevCloseData is automatically returned to the pool after fn returns.
// Data is only valid during callback. To retain: myPrevClose := *prevClose
func WithPrevCloseData(data []byte, fn func(*PrevCloseData) error) error {
	prevClose, err := parsePrevCloseDataPooled(data)
	if err != nil {
		return err
	}
	defer releasePrevClose(prevClose)
	return fn(prevClose)
}
