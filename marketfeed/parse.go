package marketfeed

import (
	"encoding/binary"
	"fmt"
	"math"
)

// ParseMarketFeedHeader parses the common 8-byte header
// Bytes 1: Response Code (byte)
// Bytes 2-3: Message Length (int16, little endian)
// Byte 4: Exchange Segment (byte)
// Bytes 5-8: Security ID (int32, little endian)
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
// Header: 8 bytes
// Bytes 9-12: Last Traded Price (float32)
// Bytes 13-16: Trade Time Epoch (int32)
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
// Header: 8 bytes
// Bytes 9-12: Latest Traded Price (float32)
// Bytes 13-14: Last Traded Quantity (int16)
// Bytes 15-18: Last Trade Time (int32) ← FIXED: was incorrectly data[16:18]
// Bytes 19-22: Average Trade Price (float32)
// Bytes 23-26: Volume (int32)
// Bytes 27-30: Total Sell Quantity (int32)
// Bytes 31-34: Total Buy Quantity (int32)
// Bytes 35-38: Day Open (float32)
// Bytes 39-42: Day Close (float32)
// Bytes 43-46: Day High (float32)
// Bytes 47-50: Day Low (float32)
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
		TradeTimeEpoch:     int32(binary.LittleEndian.Uint32(data[14:18])), // FIXED: was data[16:18]
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
// Header: 8 bytes
// Bytes 9-12: Open Interest (int32)
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
// Header: 8 bytes
// Bytes 9-12: Previous Close Price (float32)
// Bytes 13-16: Previous Open Interest (int32)
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

// ParseFullData parses a full packet with market depth (162 bytes total) ← FIXED: was 150 bytes
// Header: 8 bytes
// Bytes 9-12: Latest Traded Price (float32)
// Bytes 13-14: Last Traded Quantity (int16)
// Bytes 15-18: Last Trade Time (int32) ← FIXED: no padding
// Bytes 19-22: Average Trade Price (float32)
// Bytes 23-26: Volume (int32)
// Bytes 27-30: Total Sell Quantity (int32)
// Bytes 31-34: Total Buy Quantity (int32)
// Bytes 35-38: Open Interest (int32) ← FIXED: was missing
// Bytes 39-42: Highest OI (int32) ← FIXED: was missing
// Bytes 43-46: Lowest OI (int32) ← FIXED: was missing
// Bytes 47-50: Day Open (float32)
// Bytes 51-54: Day Close (float32)
// Bytes 55-58: Day High (float32)
// Bytes 59-62: Day Low (float32)
// Bytes 63-162: Market Depth (5 levels × 20 bytes = 100 bytes)
func ParseFullData(data []byte) (*FullData, error) {
	if len(data) < 162 { // FIXED: was 150
		return nil, fmt.Errorf("insufficient data for full: got %d bytes, need 162", len(data))
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
		TradeTimeEpoch:     int32(binary.LittleEndian.Uint32(data[14:18])), // FIXED: was data[16:18]
		AverageTradedPrice: bytesToFloat32(data[18:22]),
		Volume:             int32(binary.LittleEndian.Uint32(data[22:26])),
		TotalSellQuantity:  int32(binary.LittleEndian.Uint32(data[26:30])),
		TotalBuyQuantity:   int32(binary.LittleEndian.Uint32(data[30:34])),
		OpenInterest:       int32(binary.LittleEndian.Uint32(data[34:38])), // FIXED: was missing
		HighestOI:          int32(binary.LittleEndian.Uint32(data[38:42])), // FIXED: was missing
		LowestOI:           int32(binary.LittleEndian.Uint32(data[42:46])), // FIXED: was missing
		DayOpen:            bytesToFloat32(data[46:50]),                     // FIXED: offset changed
		DayClose:           bytesToFloat32(data[50:54]),                     // FIXED: offset changed
		DayHigh:            bytesToFloat32(data[54:58]),                     // FIXED: offset changed
		DayLow:             bytesToFloat32(data[58:62]),                     // FIXED: offset changed
	}

	// Parse 5 levels of market depth (bytes 63-162)
	depthOffset := 62 // FIXED: was 50
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
// Header: 8 bytes
// Bytes 9-10: Error Code (int16)
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

// bytesToFloat32 converts 4 bytes to float32 (Little Endian) - zero allocation
func bytesToFloat32(b []byte) float32 {
	bits := binary.LittleEndian.Uint32(b)
	return math.Float32frombits(bits)
}
