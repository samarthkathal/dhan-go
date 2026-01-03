package orderupdate

import (
	"time"
)

// Order Status constants
const (
	OrderStatusTransit   = "TRANSIT"
	OrderStatusPending   = "PENDING"
	OrderStatusRejected  = "REJECTED"
	OrderStatusCancelled = "CANCELLED"
	OrderStatusTraded    = "TRADED"
	OrderStatusExpired   = "EXPIRED"
)

// Transaction Type constants
const (
	TransactionTypeBuy  = "BUY"
	TransactionTypeSell = "SELL"
)

// Product Type constants
const (
	ProductTypeCNC      = "CNC"
	ProductTypeIntraday = "INTRADAY"
	ProductTypeMargin   = "MARGIN"
	ProductTypeCO       = "CO"
	ProductTypeBO       = "BO"
	ProductTypeMTF      = "MTF"
)

// Order Type constants
const (
	OrderTypeLimit          = "LIMIT"
	OrderTypeMarket         = "MARKET"
	OrderTypeStopLoss       = "STOP_LOSS"
	OrderTypeStopLossMarket = "STOP_LOSS_MARKET"
)

// Exchange Segment constants
const (
	ExchangeNSEEQ       = "NSE_EQ"
	ExchangeNSEFNO      = "NSE_FNO"
	ExchangeNSECurrency = "NSE_CURRENCY"
	ExchangeBSEEQ       = "BSE_EQ"
	ExchangeBSEFNO      = "BSE_FNO"
	ExchangeBSECurrency = "BSE_CURRENCY"
	ExchangeMCXComm     = "MCX_COMM"
	ExchangeIDXI        = "IDX_I"
)

// Validity constants
const (
	ValidityDay = "DAY"
	ValidityIOC = "IOC"
)

// Option Type constants
const (
	OptionTypeCall = "CALL"
	OptionTypePut  = "PUT"
)

// OrderAlert represents a real-time order update message
type OrderAlert struct {
	Type string `json:"Type"` // "order_alert"
	Data OrderAlertData `json:"Data"`
}

// OrderAlertData contains the order update information
type OrderAlertData struct {
	// Order identifiers
	OrderID         string `json:"orderNo"`
	ExchangeOrderID string `json:"ExchOrderNo"`
	ClientID        string `json:"dhanClientId"`
	CorrelationID   string `json:"correlationId,omitempty"`

	// Order details
	Symbol          string `json:"tradingSymbol"`
	SecurityID      string `json:"securityId"`
	Exchange        string `json:"exchangeSegment"`
	ProductType     string `json:"productType"`
	OrderType       string `json:"orderType"`
	Validity        string `json:"validity"`
	TransactionType string `json:"transactionType"`

	// Quantities and prices
	Quantity         int32   `json:"quantity"`
	DisclosedQty     int32   `json:"disclosedQuantity,omitempty"`
	Price            float32 `json:"price"`
	TriggerPrice     float32 `json:"triggerPrice,omitempty"`
	TradedQuantity   int32   `json:"TradedQty,omitempty"`
	TradedPrice      float32 `json:"TradedPrice,omitempty"`
	AvgTradedPrice   float32 `json:"AvgTradedPrice,omitempty"`
	RemainingQty     int32   `json:"remainingQuantity,omitempty"`

	// Status and reason
	Status            string `json:"Status"`
	OrderStatus       string `json:"orderStatus"`
	ReasonCode        string `json:"ReasonCode,omitempty"`
	ReasonDescription string `json:"ReasonDescription,omitempty"`

	// Derivatives details (for F&O)
	ExpiryDate     string  `json:"expiryDate,omitempty"`
	StrikePrice    float32 `json:"strikePrice,omitempty"`
	OptionType     string  `json:"optionType,omitempty"`
	InstrumentType string  `json:"instrumentType,omitempty"`

	// Timestamps
	OrderDateTime    string `json:"orderDateTime"`
	ExchangeTime     string `json:"exchOrderTime,omitempty"`
	LastUpdatedTime  string `json:"lastUpdatedTime,omitempty"`

	// Bracket/Cover order details
	BOProfitValue     float32 `json:"boProfitValue,omitempty"`
	BOStopLossValue   float32 `json:"boStopLossValue,omitempty"`
	LegName           string  `json:"legName,omitempty"`

	// Additional flags
	AfterMarketOrder bool   `json:"afterMarketOrder,omitempty"`
	AmoTime          string `json:"amoTime,omitempty"`
}

// OrderUpdateCallback is the function signature for order update handlers
type OrderUpdateCallback func(*OrderAlert)

// ErrorCallback is the function signature for error handlers
type ErrorCallback func(error)

// IsOrderAlert checks if the message type is an order alert
func (o *OrderAlert) IsOrderAlert() bool {
	return o.Type == "order_alert"
}

// GetOrderID returns the order ID
func (o *OrderAlert) GetOrderID() string {
	return o.Data.OrderID
}

// GetStatus returns the order status
func (o *OrderAlert) GetStatus() string {
	return o.Data.Status
}

// GetTradedQuantity returns the traded quantity
func (o *OrderAlert) GetTradedQuantity() int32 {
	return o.Data.TradedQuantity
}

// GetAvgTradedPrice returns the average traded price
func (o *OrderAlert) GetAvgTradedPrice() float32 {
	return o.Data.AvgTradedPrice
}

// IsFilled returns true if the order is completely filled
func (o *OrderAlert) IsFilled() bool {
	return o.Data.Status == "TRADED" && o.Data.RemainingQty == 0
}

// IsPartiallyFilled returns true if the order is partially filled
func (o *OrderAlert) IsPartiallyFilled() bool {
	return o.Data.TradedQuantity > 0 && o.Data.RemainingQty > 0
}

// IsRejected returns true if the order is rejected
func (o *OrderAlert) IsRejected() bool {
	return o.Data.Status == "REJECTED"
}

// IsCancelled returns true if the order is cancelled
func (o *OrderAlert) IsCancelled() bool {
	return o.Data.Status == "CANCELLED"
}

// GetOrderTime parses and returns the order time
func (o *OrderAlert) GetOrderTime() (time.Time, error) {
	return time.Parse(time.RFC3339, o.Data.OrderDateTime)
}
