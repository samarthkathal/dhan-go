package fulldepth

// DepthLevel represents the depth level (20 or 200)
type DepthLevel int

const (
	// Depth20 is 20-level market depth
	Depth20 DepthLevel = 20
	// Depth200 is 200-level market depth (only for NSE/NSE_FNO)
	Depth200 DepthLevel = 200
)

// WebSocket URLs for Full Depth
const (
	// Depth20URL is the WebSocket URL for 20-level depth
	Depth20URL = "wss://depth-api-feed.dhan.co/twentydepth"
	// Depth200URL is the WebSocket URL for 200-level depth
	Depth200URL = "wss://full-depth-api.dhan.co/"
)

// Feed response codes
const (
	FeedCodeBid        byte = 41 // Bid data
	FeedCodeAsk        byte = 51 // Ask data
	FeedCodeDisconnect byte = 50 // Disconnection/Error
)

// Request codes
const (
	RequestCodeSubscribe   int = 23 // Subscribe to instruments
	RequestCodeDisconnect  int = 12 // Disconnect
)

// Exchange segment constants (same as marketfeed)
const (
	ExchangeNSEEQCode   byte = 1
	ExchangeNSEFNOCode  byte = 2
)

// Exchange segment names
const (
	ExchangeNSEEQ  = "NSE_EQ"
	ExchangeNSEFNO = "NSE_FNO"
)

// DepthHeader contains the 12-byte header for depth responses
type DepthHeader struct {
	MessageLength   int16  // Bytes 0-1: Message length
	ResponseCode    byte   // Byte 2: Response code (41=bid, 51=ask, 50=error)
	ExchangeSegment byte   // Byte 3: Exchange segment
	SecurityID      int32  // Bytes 4-7: Security ID
	NumRows         int32  // Bytes 8-11: Number of rows
}

// DepthEntry represents a single level in the market depth
type DepthEntry struct {
	Price    float64 // Price at this level
	Quantity int32   // Quantity at this level
	Orders   int32   // Number of orders at this level
}

// DepthData contains depth data for one side (bid or ask)
type DepthData struct {
	Header  DepthHeader
	IsBid   bool         // true for bid, false for ask
	Entries []DepthEntry // Market depth entries
}

// FullDepthData contains combined bid and ask depth for an instrument
type FullDepthData struct {
	ExchangeSegment byte
	SecurityID      int32
	Bids            []DepthEntry
	Asks            []DepthEntry
}

// Instrument represents an instrument to subscribe to
type Instrument struct {
	ExchangeSegment string // "NSE_EQ" or "NSE_FNO"
	SecurityID      int    // Security ID
}

// DepthCallback is the callback for receiving depth data
type DepthCallback func(*FullDepthData)

// ErrorCallback is the callback for errors
type ErrorCallback func(error)

// Error codes for disconnection
const (
	ErrorCodeMaxConnections   = 805 // No. of active websocket connections exceeded
	ErrorCodeNotSubscribed    = 806 // Subscribe to Data APIs to continue
	ErrorCodeTokenExpired     = 807 // Access Token is expired
	ErrorCodeInvalidClient    = 808 // Invalid Client ID
	ErrorCodeAuthFailed       = 809 // Authentication Failed
)

// Helper functions

// exchangeCodeToName converts exchange segment code to name
func exchangeCodeToName(code byte) string {
	switch code {
	case ExchangeNSEEQCode:
		return ExchangeNSEEQ
	case ExchangeNSEFNOCode:
		return ExchangeNSEFNO
	default:
		return "UNKNOWN"
	}
}

// exchangeNameToCode converts exchange segment name to code
func exchangeNameToCode(name string) byte {
	switch name {
	case ExchangeNSEEQ:
		return ExchangeNSEEQCode
	case ExchangeNSEFNO:
		return ExchangeNSEFNOCode
	default:
		return 0
	}
}

// GetExchangeName returns the exchange name for FullDepthData
func (f *FullDepthData) GetExchangeName() string {
	return exchangeCodeToName(f.ExchangeSegment)
}

// GetBestBid returns the best (highest) bid price and quantity
func (f *FullDepthData) GetBestBid() (price float64, quantity int32) {
	if len(f.Bids) == 0 {
		return 0, 0
	}
	// Bids should be sorted descending by price
	return f.Bids[0].Price, f.Bids[0].Quantity
}

// GetBestAsk returns the best (lowest) ask price and quantity
func (f *FullDepthData) GetBestAsk() (price float64, quantity int32) {
	if len(f.Asks) == 0 {
		return 0, 0
	}
	// Asks should be sorted ascending by price
	return f.Asks[0].Price, f.Asks[0].Quantity
}

// GetSpread returns the bid-ask spread
func (f *FullDepthData) GetSpread() float64 {
	askPrice, _ := f.GetBestAsk()
	bidPrice, _ := f.GetBestBid()
	return askPrice - bidPrice
}

// GetTotalBidQuantity returns the total quantity across all bid levels
func (f *FullDepthData) GetTotalBidQuantity() int64 {
	var total int64
	for _, entry := range f.Bids {
		total += int64(entry.Quantity)
	}
	return total
}

// GetTotalAskQuantity returns the total quantity across all ask levels
func (f *FullDepthData) GetTotalAskQuantity() int64 {
	var total int64
	for _, entry := range f.Asks {
		total += int64(entry.Quantity)
	}
	return total
}

// Copy returns a deep copy of DepthData.
// Use this when you need to retain data beyond the callback scope.
func (d *DepthData) Copy() DepthData {
	cp := *d
	cp.Entries = make([]DepthEntry, len(d.Entries))
	copy(cp.Entries, d.Entries)
	return cp
}

// Copy returns a deep copy of FullDepthData.
// Use this when you need to retain data beyond the callback scope.
func (f *FullDepthData) Copy() FullDepthData {
	cp := *f
	cp.Bids = make([]DepthEntry, len(f.Bids))
	copy(cp.Bids, f.Bids)
	cp.Asks = make([]DepthEntry, len(f.Asks))
	copy(cp.Asks, f.Asks)
	return cp
}
