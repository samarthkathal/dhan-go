package types

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// Feed response codes
const (
	FeedCodeTicker    byte = 2  // LTP + Last Traded Time
	FeedCodeQuote     byte = 4  // Complete trade data
	FeedCodeOI        byte = 5  // Open Interest
	FeedCodePrevClose byte = 6  // Previous close data
	FeedCodeFull      byte = 8  // Complete data + market depth
	FeedCodeError     byte = 50 // Forced disconnection error
)

// Exchange segment codes
const (
	ExchangeNSEEQCode     byte = 1
	ExchangeNSEFNOCode    byte = 2
	ExchangeNSECurrCode   byte = 3
	ExchangeBSEEQCode     byte = 4
	ExchangeBSEFNOCode    byte = 5
	ExchangeBSECurrCode   byte = 6
	ExchangeMCXCommCode   byte = 7
	ExchangeIDXICode      byte = 13
)

// Subscription request codes
const (
	RequestCodeSubscribe   int = 15
	RequestCodeUnsubscribe int = 16
	RequestCodeDisconnect  int = 12
)

// MarketFeedHeader contains the common 8-byte header for all responses
type MarketFeedHeader struct {
	ResponseCode    byte   // Byte 1: Feed response code
	MessageLength   int16  // Bytes 2-3: Message length
	ExchangeSegment byte   // Byte 4: Exchange segment
	SecurityID      int32  // Bytes 5-8: Security ID
}

// TickerData contains LTP and last traded time (Response code 2)
// Total: 8 header + 8 data = 16 bytes
type TickerData struct {
	Header           MarketFeedHeader
	LastTradedPrice  float32 // Bytes 9-12: LTP
	TradeTimeEpoch   int32   // Bytes 13-16: Trade time (Unix timestamp)
}

// QuoteData contains complete trade data (Response code 4)
// Total: 8 header + 42 data = 50 bytes
type QuoteData struct {
	Header              MarketFeedHeader
	LastTradedPrice     float32 // Bytes 9-12: Latest traded price
	LastTradedQuantity  int16   // Bytes 13-14: Last traded quantity
	_                   int16   // Bytes 15-16: Padding
	TradeTimeEpoch      int32   // Bytes 17-18: Trade time (Unix timestamp)
	AverageTradedPrice  float32 // Bytes 19-22: Average trade price
	Volume              int32   // Bytes 23-26: Total volume
	TotalSellQuantity   int32   // Bytes 27-30: Total sell quantity
	TotalBuyQuantity    int32   // Bytes 31-34: Total buy quantity
	DayOpen             float32 // Bytes 35-38: Day open price
	DayClose            float32 // Bytes 39-42: Day close price
	DayHigh             float32 // Bytes 43-46: Day high price
	DayLow              float32 // Bytes 47-50: Day low price
}

// OIData contains Open Interest data (Response code 5)
// Total: 8 header + 4 data = 12 bytes
type OIData struct {
	Header       MarketFeedHeader
	OpenInterest int32 // Bytes 9-12: Open Interest
}

// PrevCloseData contains previous day reference data (Response code 6)
// Total: 8 header + 8 data = 16 bytes
type PrevCloseData struct {
	Header              MarketFeedHeader
	PreviousClosePrice  float32 // Bytes 9-12: Previous close price
	PreviousOpenInterest int32   // Bytes 13-16: Previous open interest
}

// MarketDepth contains one level of market depth (20 bytes per level)
type MarketDepth struct {
	BidQuantity   int32   // Bytes 0-3: Bid quantity
	AskQuantity   int32   // Bytes 4-7: Ask quantity
	BidOrderCount int16   // Bytes 8-9: Bid orders count
	AskOrderCount int16   // Bytes 10-11: Ask orders count
	BidPrice      float32 // Bytes 12-15: Bid price
	AskPrice      float32 // Bytes 16-19: Ask price
}

// FullData contains complete data with 5 levels of market depth (Response code 8)
// Total: 8 header + 42 quote + 100 depth = 150 bytes
type FullData struct {
	Header              MarketFeedHeader
	// Quote data (same as QuoteData)
	LastTradedPrice     float32 // Bytes 9-12: Latest traded price
	LastTradedQuantity  int16   // Bytes 13-14: Last traded quantity
	_                   int16   // Bytes 15-16: Padding
	TradeTimeEpoch      int32   // Bytes 17-18: Trade time (Unix timestamp)
	AverageTradedPrice  float32 // Bytes 19-22: Average trade price
	Volume              int32   // Bytes 23-26: Total volume
	TotalSellQuantity   int32   // Bytes 27-30: Total sell quantity
	TotalBuyQuantity    int32   // Bytes 31-34: Total buy quantity
	DayOpen             float32 // Bytes 35-38: Day open price
	DayClose            float32 // Bytes 39-42: Day close price
	DayHigh             float32 // Bytes 43-46: Day high price
	DayLow              float32 // Bytes 47-50: Day low price
	// Market depth (5 levels × 20 bytes each = 100 bytes)
	Depth [5]MarketDepth // Bytes 51-150: Market depth levels
}

// ErrorData contains error information for forced disconnection (Response code 50)
type ErrorData struct {
	Header    MarketFeedHeader
	ErrorCode int16 // Bytes 9-10: Error code
}

// MarketFeedCallback is the function signature for market feed handlers
type TickerCallback func(*TickerData)
type QuoteCallback func(*QuoteData)
type OICallback func(*OIData)
type PrevCloseCallback func(*PrevCloseData)
type FullCallback func(*FullData)
type ErrorCallback func(*ErrorData)

// ParseMarketFeedHeader parses the common 8-byte header
func ParseMarketFeedHeader(data []byte) (*MarketFeedHeader, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("insufficient data for header: got %d bytes, need 8", len(data))
	}

	header := &MarketFeedHeader{
		ResponseCode:    data[0],
		MessageLength:   int16(binary.LittleEndian.Uint16(data[1:3])),
		ExchangeSegment: data[3],
		SecurityID:      int32(binary.LittleEndian.Uint32(data[4:8])),
	}

	return header, nil
}

// ParseTickerData parses a ticker packet (16 bytes total)
func ParseTickerData(data []byte) (*TickerData, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("insufficient data for ticker: got %d bytes, need 16", len(data))
	}

	header, err := ParseMarketFeedHeader(data)
	if err != nil {
		return nil, err
	}

	if header.ResponseCode != FeedCodeTicker {
		return nil, fmt.Errorf("invalid response code for ticker: %d", header.ResponseCode)
	}

	ticker := &TickerData{
		Header:          *header,
		LastTradedPrice: bytesToFloat32(data[8:12]),
		TradeTimeEpoch:  int32(binary.LittleEndian.Uint32(data[12:16])),
	}

	return ticker, nil
}

// ParseQuoteData parses a quote packet (50 bytes total)
func ParseQuoteData(data []byte) (*QuoteData, error) {
	if len(data) < 50 {
		return nil, fmt.Errorf("insufficient data for quote: got %d bytes, need 50", len(data))
	}

	header, err := ParseMarketFeedHeader(data)
	if err != nil {
		return nil, err
	}

	if header.ResponseCode != FeedCodeQuote {
		return nil, fmt.Errorf("invalid response code for quote: %d", header.ResponseCode)
	}

	quote := &QuoteData{
		Header:             *header,
		LastTradedPrice:    bytesToFloat32(data[8:12]),
		LastTradedQuantity: int16(binary.LittleEndian.Uint16(data[12:14])),
		TradeTimeEpoch:     int32(binary.LittleEndian.Uint32(data[16:18])),
		AverageTradedPrice: bytesToFloat32(data[18:22]),
		Volume:             int32(binary.LittleEndian.Uint32(data[22:26])),
		TotalSellQuantity:  int32(binary.LittleEndian.Uint32(data[26:30])),
		TotalBuyQuantity:   int32(binary.LittleEndian.Uint32(data[30:34])),
		DayOpen:            bytesToFloat32(data[34:38]),
		DayClose:           bytesToFloat32(data[38:42]),
		DayHigh:            bytesToFloat32(data[42:46]),
		DayLow:             bytesToFloat32(data[46:50]),
	}

	return quote, nil
}

// ParseOIData parses an open interest packet (12 bytes total)
func ParseOIData(data []byte) (*OIData, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("insufficient data for OI: got %d bytes, need 12", len(data))
	}

	header, err := ParseMarketFeedHeader(data)
	if err != nil {
		return nil, err
	}

	if header.ResponseCode != FeedCodeOI {
		return nil, fmt.Errorf("invalid response code for OI: %d", header.ResponseCode)
	}

	oi := &OIData{
		Header:       *header,
		OpenInterest: int32(binary.LittleEndian.Uint32(data[8:12])),
	}

	return oi, nil
}

// ParsePrevCloseData parses a previous close packet (16 bytes total)
func ParsePrevCloseData(data []byte) (*PrevCloseData, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("insufficient data for prev close: got %d bytes, need 16", len(data))
	}

	header, err := ParseMarketFeedHeader(data)
	if err != nil {
		return nil, err
	}

	if header.ResponseCode != FeedCodePrevClose {
		return nil, fmt.Errorf("invalid response code for prev close: %d", header.ResponseCode)
	}

	prevClose := &PrevCloseData{
		Header:               *header,
		PreviousClosePrice:   bytesToFloat32(data[8:12]),
		PreviousOpenInterest: int32(binary.LittleEndian.Uint32(data[12:16])),
	}

	return prevClose, nil
}

// ParseFullData parses a full packet with market depth (150 bytes total)
func ParseFullData(data []byte) (*FullData, error) {
	if len(data) < 150 {
		return nil, fmt.Errorf("insufficient data for full: got %d bytes, need 150", len(data))
	}

	header, err := ParseMarketFeedHeader(data)
	if err != nil {
		return nil, err
	}

	if header.ResponseCode != FeedCodeFull {
		return nil, fmt.Errorf("invalid response code for full: %d", header.ResponseCode)
	}

	full := &FullData{
		Header:             *header,
		LastTradedPrice:    bytesToFloat32(data[8:12]),
		LastTradedQuantity: int16(binary.LittleEndian.Uint16(data[12:14])),
		TradeTimeEpoch:     int32(binary.LittleEndian.Uint32(data[16:18])),
		AverageTradedPrice: bytesToFloat32(data[18:22]),
		Volume:             int32(binary.LittleEndian.Uint32(data[22:26])),
		TotalSellQuantity:  int32(binary.LittleEndian.Uint32(data[26:30])),
		TotalBuyQuantity:   int32(binary.LittleEndian.Uint32(data[30:34])),
		DayOpen:            bytesToFloat32(data[34:38]),
		DayClose:           bytesToFloat32(data[38:42]),
		DayHigh:            bytesToFloat32(data[42:46]),
		DayLow:             bytesToFloat32(data[46:50]),
	}

	// Parse 5 levels of market depth
	depthOffset := 50
	for i := 0; i < 5; i++ {
		offset := depthOffset + (i * 20)
		full.Depth[i] = MarketDepth{
			BidQuantity:   int32(binary.LittleEndian.Uint32(data[offset : offset+4])),
			AskQuantity:   int32(binary.LittleEndian.Uint32(data[offset+4 : offset+8])),
			BidOrderCount: int16(binary.LittleEndian.Uint16(data[offset+8 : offset+10])),
			AskOrderCount: int16(binary.LittleEndian.Uint16(data[offset+10 : offset+12])),
			BidPrice:      bytesToFloat32(data[offset+12 : offset+16]),
			AskPrice:      bytesToFloat32(data[offset+16 : offset+20]),
		}
	}

	return full, nil
}

// ParseErrorData parses an error packet (10 bytes minimum)
func ParseErrorData(data []byte) (*ErrorData, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("insufficient data for error: got %d bytes, need 10", len(data))
	}

	header, err := ParseMarketFeedHeader(data)
	if err != nil {
		return nil, err
	}

	if header.ResponseCode != FeedCodeError {
		return nil, fmt.Errorf("invalid response code for error: %d", header.ResponseCode)
	}

	errorData := &ErrorData{
		Header:    *header,
		ErrorCode: int16(binary.LittleEndian.Uint16(data[8:10])),
	}

	return errorData, nil
}

// Helper methods for TickerData
func (t *TickerData) GetTradeTime() time.Time {
	return time.Unix(int64(t.TradeTimeEpoch), 0)
}

func (t *TickerData) GetExchangeName() string {
	return exchangeCodeToName(t.Header.ExchangeSegment)
}

// Helper methods for OIData
func (o *OIData) GetExchangeName() string {
	return exchangeCodeToName(o.Header.ExchangeSegment)
}

// Helper methods for PrevCloseData
func (p *PrevCloseData) GetExchangeName() string {
	return exchangeCodeToName(p.Header.ExchangeSegment)
}

// Helper methods for QuoteData
func (q *QuoteData) GetTradeTime() time.Time {
	return time.Unix(int64(q.TradeTimeEpoch), 0)
}

func (q *QuoteData) GetExchangeName() string {
	return exchangeCodeToName(q.Header.ExchangeSegment)
}

func (q *QuoteData) GetDayChange() float32 {
	if q.DayClose == 0 {
		return 0
	}
	return q.LastTradedPrice - q.DayClose
}

func (q *QuoteData) GetDayChangePercent() float32 {
	if q.DayClose == 0 {
		return 0
	}
	return ((q.LastTradedPrice - q.DayClose) / q.DayClose) * 100
}

// Helper methods for FullData
func (f *FullData) GetTradeTime() time.Time {
	return time.Unix(int64(f.TradeTimeEpoch), 0)
}

func (f *FullData) GetExchangeName() string {
	return exchangeCodeToName(f.Header.ExchangeSegment)
}

func (f *FullData) GetDayChange() float32 {
	if f.DayClose == 0 {
		return 0
	}
	return f.LastTradedPrice - f.DayClose
}

func (f *FullData) GetDayChangePercent() float32 {
	if f.DayClose == 0 {
		return 0
	}
	return ((f.LastTradedPrice - f.DayClose) / f.DayClose) * 100
}

func (f *FullData) GetBestBid() (price float32, quantity int32) {
	return f.Depth[0].BidPrice, f.Depth[0].BidQuantity
}

func (f *FullData) GetBestAsk() (price float32, quantity int32) {
	return f.Depth[0].AskPrice, f.Depth[0].AskQuantity
}

func (f *FullData) GetSpread() float32 {
	askPrice, _ := f.GetBestAsk()
	bidPrice, _ := f.GetBestBid()
	return askPrice - bidPrice
}

// bytesToFloat32 converts 4 bytes to float32 (Little Endian) - zero allocation
func bytesToFloat32(b []byte) float32 {
	bits := binary.LittleEndian.Uint32(b)
	return math.Float32frombits(bits)
}

// exchangeCodeToName converts exchange segment code to name
func exchangeCodeToName(code byte) string {
	switch code {
	case ExchangeNSEEQCode:
		return ExchangeNSEEQ
	case ExchangeNSEFNOCode:
		return ExchangeNSEFNO
	case ExchangeNSECurrCode:
		return ExchangeNSECurrency
	case ExchangeBSEEQCode:
		return ExchangeBSEEQ
	case ExchangeBSEFNOCode:
		return ExchangeBSEFNO
	case ExchangeBSECurrCode:
		return ExchangeBSECurrency
	case ExchangeMCXCommCode:
		return ExchangeMCXComm
	case ExchangeIDXICode:
		return ExchangeIDXI
	default:
		return "UNKNOWN"
	}
}

// exchangeNameToCode converts exchange segment name to code
func ExchangeNameToCode(name string) byte {
	switch name {
	case ExchangeNSEEQ:
		return ExchangeNSEEQCode
	case ExchangeNSEFNO:
		return ExchangeNSEFNOCode
	case ExchangeNSECurrency:
		return ExchangeNSECurrCode
	case ExchangeBSEEQ:
		return ExchangeBSEEQCode
	case ExchangeBSEFNO:
		return ExchangeBSEFNOCode
	case ExchangeBSECurrency:
		return ExchangeBSECurrCode
	case ExchangeMCXComm:
		return ExchangeMCXCommCode
	case ExchangeIDXI:
		return ExchangeIDXICode
	default:
		return 0
	}
}
