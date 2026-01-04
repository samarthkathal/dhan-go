package rest

// MarketQuoteRequest represents a request for market quote data.
// Keys are exchange segments (e.g., "NSE_EQ", "NSE_FNO"), values are lists of security IDs.
// Example: {"NSE_EQ": [11536], "NSE_FNO": [49081, 49082]}
type MarketQuoteRequest map[string][]int

// LTPData represents last traded price data for a single security
type LTPData struct {
	SecurityID        int     `json:"security_id"`
	LastTradedPrice   float64 `json:"last_price"`
	LastTradedTime    string  `json:"last_traded_time,omitempty"`
}

// LTPResponse represents the response from the LTP API
type LTPResponse struct {
	Status string                        `json:"status"`
	Data   map[string]map[string]LTPData `json:"data"` // segment -> security_id -> data
}

// OHLCData represents OHLC data for a single security
type OHLCData struct {
	SecurityID      int     `json:"security_id"`
	LastTradedPrice float64 `json:"last_price"`
	Open            float64 `json:"open"`
	High            float64 `json:"high"`
	Low             float64 `json:"low"`
	Close           float64 `json:"close"`
}

// OHLCResponse represents the response from the OHLC API
type OHLCResponse struct {
	Status string                         `json:"status"`
	Data   map[string]map[string]OHLCData `json:"data"` // segment -> security_id -> data
}

// MarketDepthEntry represents a single level in the market depth
type MarketDepthEntry struct {
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
	Orders   int     `json:"orders"`
}

// QuoteData represents full quote data for a single security
type QuoteData struct {
	SecurityID      int                `json:"security_id"`
	LastTradedPrice float64            `json:"last_price"`
	LastTradedQty   int                `json:"last_traded_qty"`
	LastTradedTime  string             `json:"last_traded_time"`
	Open            float64            `json:"open"`
	High            float64            `json:"high"`
	Low             float64            `json:"low"`
	Close           float64            `json:"close"`
	Volume          int64              `json:"volume"`
	OpenInterest    int64              `json:"oi"`
	AvgTradedPrice  float64            `json:"avg_price"`
	TotalBuyQty     int64              `json:"total_buy_qty"`
	TotalSellQty    int64              `json:"total_sell_qty"`
	LowerCircuit    float64            `json:"lower_circuit"`
	UpperCircuit    float64            `json:"upper_circuit"`
	Bid             []MarketDepthEntry `json:"bid"`
	Ask             []MarketDepthEntry `json:"ask"`
}

// QuoteResponse represents the response from the Quote API
type QuoteResponse struct {
	Status string                          `json:"status"`
	Data   map[string]map[string]QuoteData `json:"data"` // segment -> security_id -> data
}

// OptionChainRequest represents a request for option chain data
type OptionChainRequest struct {
	UnderlyingScrip int    `json:"UnderlyingScrip"`          // Security ID of underlying
	UnderlyingSeg   string `json:"UnderlyingSeg,omitempty"`  // Exchange segment (e.g., "NSE_EQ", "IDX_I")
	Expiry          string `json:"Expiry,omitempty"`         // Expiry date in YYYY-MM-DD format
}

// ExpiryListRequest represents a request for expiry list
type ExpiryListRequest struct {
	UnderlyingScrip int    `json:"UnderlyingScrip"`         // Security ID of underlying
	UnderlyingSeg   string `json:"UnderlyingSeg,omitempty"` // Exchange segment
}

// OptionGreeks represents option greeks
type OptionGreeks struct {
	Delta float64 `json:"delta"`
	Theta float64 `json:"theta"`
	Gamma float64 `json:"gamma"`
	Vega  float64 `json:"vega"`
}

// OptionData represents data for a single option contract
type OptionData struct {
	SecurityID        int          `json:"security_id"`
	LastPrice         float64      `json:"last_price"`
	OpenInterest      int64        `json:"oi"`
	Volume            int64        `json:"volume"`
	PrevClose         float64      `json:"prev_close"`
	PrevVolume        int64        `json:"prev_volume"`
	ImpliedVolatility float64      `json:"implied_volatility"`
	Greeks            OptionGreeks `json:"greeks"`
	TopBidPrice       float64      `json:"top_bid_price"`
	TopBidQty         int          `json:"top_bid_quantity"`
	TopAskPrice       float64      `json:"top_ask_price"`
	TopAskQty         int          `json:"top_ask_quantity"`
}

// OptionStrikeData represents option data for a specific strike
type OptionStrikeData struct {
	CE *OptionData `json:"ce,omitempty"` // Call option data
	PE *OptionData `json:"pe,omitempty"` // Put option data
}

// OptionChainData represents the option chain data
type OptionChainData struct {
	LastPrice float64                     `json:"last_price"` // Underlying last price
	OC        map[string]OptionStrikeData `json:"oc"`         // Strike -> option data
}

// OptionChainResponse represents the response from the Option Chain API
type OptionChainResponse struct {
	Status string          `json:"status"`
	Data   OptionChainData `json:"data"`
}

// ExpiryListResponse represents the response from the Expiry List API
type ExpiryListResponse struct {
	Status string   `json:"status"`
	Data   []string `json:"data"` // List of expiry dates in YYYY-MM-DD format
}
