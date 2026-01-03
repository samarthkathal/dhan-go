package marketfeed

import (
	"encoding/json"
	"fmt"
)

// Instrument represents a single instrument to subscribe/unsubscribe
type Instrument struct {
	ExchangeSegment string `json:"ExchangeSegment"` // e.g., "NSE_EQ", "NSE_FNO"
	SecurityID      string `json:"SecurityId"`      // e.g., "1333"
}

// SubscriptionRequest represents a subscription/unsubscription request
type SubscriptionRequest struct {
	RequestCode       int          `json:"RequestCode"`       // 15 for subscribe, 16 for unsubscribe
	InstrumentCount   int          `json:"InstrumentCount"`   // Number of instruments
	InstrumentList    []Instrument `json:"InstrumentList"`    // List of instruments
}

// DisconnectRequest represents a disconnect request
type DisconnectRequest struct {
	RequestCode int `json:"RequestCode"` // 12 for disconnect
}

// NewSubscriptionRequest creates a new subscription request (max 100 instruments per message)
func NewSubscriptionRequest(instruments []Instrument) (*SubscriptionRequest, error) {
	if len(instruments) == 0 {
		return nil, fmt.Errorf("no instruments provided")
	}
	if len(instruments) > 100 {
		return nil, fmt.Errorf("too many instruments: %d (max 100 per message)", len(instruments))
	}

	return &SubscriptionRequest{
		RequestCode:     RequestCodeSubscribe,
		InstrumentCount: len(instruments),
		InstrumentList:  instruments,
	}, nil
}

// NewUnsubscriptionRequest creates a new unsubscription request
func NewUnsubscriptionRequest(instruments []Instrument) (*SubscriptionRequest, error) {
	if len(instruments) == 0 {
		return nil, fmt.Errorf("no instruments provided")
	}
	if len(instruments) > 100 {
		return nil, fmt.Errorf("too many instruments: %d (max 100 per message)", len(instruments))
	}

	return &SubscriptionRequest{
		RequestCode:     RequestCodeUnsubscribe,
		InstrumentCount: len(instruments),
		InstrumentList:  instruments,
	}, nil
}

// NewDisconnectRequest creates a new disconnect request
func NewDisconnectRequest() *DisconnectRequest {
	return &DisconnectRequest{
		RequestCode: RequestCodeDisconnect,
	}
}

// ToJSON converts the subscription request to JSON bytes
func (s *SubscriptionRequest) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// ToJSON converts the disconnect request to JSON bytes
func (d *DisconnectRequest) ToJSON() ([]byte, error) {
	return json.Marshal(d)
}

// BatchInstruments splits a large list of instruments into batches of 100
func BatchInstruments(instruments []Instrument) [][]Instrument {
	batches := [][]Instrument{}
	batchSize := 100

	for i := 0; i < len(instruments); i += batchSize {
		end := i + batchSize
		if end > len(instruments) {
			end = len(instruments)
		}
		batches = append(batches, instruments[i:end])
	}

	return batches
}
