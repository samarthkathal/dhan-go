package types

// Order Status constants
const (
	OrderStatusTransit  = "TRANSIT"
	OrderStatusPending  = "PENDING"
	OrderStatusRejected = "REJECTED"
	OrderStatusCancelled = "CANCELLED"
	OrderStatusTraded   = "TRADED"
	OrderStatusExpired  = "EXPIRED"
)

// Transaction Type constants
const (
	TransactionTypeBuy  = "BUY"
	TransactionTypeSell = "SELL"
)

// Product Type constants
const (
	ProductTypeCNC     = "CNC"
	ProductTypeIntraday = "INTRADAY"
	ProductTypeMargin  = "MARGIN"
	ProductTypeCO      = "CO"
	ProductTypeBO      = "BO"
	ProductTypeMTF     = "MTF"
)

// Order Type constants
const (
	OrderTypeLimit        = "LIMIT"
	OrderTypeMarket       = "MARKET"
	OrderTypeStopLoss     = "STOP_LOSS"
	OrderTypeStopLossMarket = "STOP_LOSS_MARKET"
)

// Exchange Segment constants
const (
	ExchangeNSEEQ     = "NSE_EQ"
	ExchangeNSEFNO    = "NSE_FNO"
	ExchangeNSECurrency = "NSE_CURRENCY"
	ExchangeBSEEQ     = "BSE_EQ"
	ExchangeBSEFNO    = "BSE_FNO"
	ExchangeBSECurrency = "BSE_CURRENCY"
	ExchangeMCXComm   = "MCX_COMM"
	ExchangeIDXI      = "IDX_I"
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
