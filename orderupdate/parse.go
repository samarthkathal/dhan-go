package orderupdate

import (
	"encoding/json"
	"fmt"
)

// ParseOrderAlert parses a raw JSON message into an OrderAlert
// The message structure is:
// {
//   "Type": "order_alert",
//   "Data": { ... order fields ... }
// }
func ParseOrderAlert(data []byte) (*OrderAlert, error) {
	var alert OrderAlert
	if err := json.Unmarshal(data, &alert); err != nil {
		return nil, fmt.Errorf("failed to parse order alert: %w", err)
	}

	// Validate message type
	if alert.Type != "order_alert" {
		return nil, fmt.Errorf("invalid message type: expected 'order_alert', got '%s'", alert.Type)
	}

	return &alert, nil
}
