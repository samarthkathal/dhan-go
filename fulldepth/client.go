// Package fulldepth provides a client for Dhan's Full Market Depth WebSocket API
package fulldepth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client provides access to Dhan's Full Market Depth WebSocket API.
// It supports both 20-depth and 200-depth levels.
type Client struct {
	accessToken string
	clientID    string
	config      *Config

	// WebSocket connection
	conn     *websocket.Conn
	connLock sync.Mutex

	// Callbacks - protected by mu
	mu             sync.RWMutex
	depthCallbacks []DepthCallback
	errorCallbacks []ErrorCallback

	// State - connected protected by connLock
	connected bool
	ctx       context.Context
	cancel    context.CancelFunc

	// Instruments - protected by instrLock
	instrLock   sync.RWMutex
	instruments map[string]Instrument // key: "exchange:securityID"

	// Pending depth data (for combining bid/ask) - protected by pendingLock
	pendingDepth map[int32]*FullDepthData // key: securityID
	pendingLock  sync.Mutex
}

// NewClient creates a new Full Depth client.
// accessToken is the Dhan API access token.
// clientID is the Dhan client ID.
func NewClient(accessToken, clientID string, opts ...Option) (*Client, error) {
	if accessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}
	if clientID == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		accessToken:    accessToken,
		clientID:       clientID,
		config:         DefaultConfig(),
		depthCallbacks: make([]DepthCallback, 0),
		errorCallbacks: make([]ErrorCallback, 0),
		instruments:    make(map[string]Instrument),
		pendingDepth:   make(map[int32]*FullDepthData),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// Connect establishes the WebSocket connection
func (c *Client) Connect(ctx context.Context) error {
	c.connLock.Lock()
	defer c.connLock.Unlock()

	if c.connected {
		return fmt.Errorf("already connected")
	}

	// Select URL based on depth level
	baseURL := Depth20URL
	if c.config.DepthLevel == Depth200 {
		baseURL = Depth200URL
	}

	// Build connection URL with authentication
	u, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	q := u.Query()
	q.Set("token", c.accessToken)
	q.Set("clientId", c.clientID)
	q.Set("authType", "2")
	u.RawQuery = q.Encode()

	// Configure dialer
	dialer := websocket.Dialer{
		ReadBufferSize:  c.config.ReadBufferSize,
		WriteBufferSize: c.config.WriteBufferSize,
		HandshakeTimeout: c.config.ConnectTimeout,
	}

	// Connect
	conn, _, err := dialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.conn = conn
	c.connected = true

	// Start reading messages
	go c.readLoop()

	return nil
}

// Disconnect closes the WebSocket connection
func (c *Client) Disconnect() error {
	c.connLock.Lock()
	defer c.connLock.Unlock()

	if !c.connected {
		return nil
	}

	c.cancel()
	c.connected = false

	if c.conn != nil {
		// Send disconnect message
		msg := map[string]int{"RequestCode": RequestCodeDisconnect}
		_ = c.conn.WriteJSON(msg)
		return c.conn.Close()
	}

	return nil
}

// Subscribe subscribes to market depth for the specified instruments.
// Note: For 200-depth, only one instrument can be subscribed at a time.
func (c *Client) Subscribe(ctx context.Context, instruments []Instrument) error {
	c.connLock.Lock()
	if !c.connected {
		c.connLock.Unlock()
		return fmt.Errorf("not connected")
	}

	// Validate instruments for 200-depth
	if c.config.DepthLevel == Depth200 && len(instruments) > 1 {
		c.connLock.Unlock()
		return fmt.Errorf("200-depth only supports one instrument at a time")
	}

	// Validate exchange segments (only NSE_EQ and NSE_FNO supported)
	for _, inst := range instruments {
		if inst.ExchangeSegment != ExchangeNSEEQ && inst.ExchangeSegment != ExchangeNSEFNO {
			c.connLock.Unlock()
			return fmt.Errorf("only NSE_EQ and NSE_FNO are supported for full depth, got: %s", inst.ExchangeSegment)
		}
	}

	// Build subscription message based on depth level
	var msg interface{}
	if c.config.DepthLevel == Depth200 {
		// 200-depth: single instrument
		inst := instruments[0]
		msg = map[string]interface{}{
			"RequestCode":     RequestCodeSubscribe,
			"ExchangeSegment": inst.ExchangeSegment,
			"SecurityId":      inst.SecurityID,
		}
	} else {
		// 20-depth: batch subscription
		instList := make([]map[string]interface{}, len(instruments))
		for i, inst := range instruments {
			instList[i] = map[string]interface{}{
				"ExchangeSegment": inst.ExchangeSegment,
				"SecurityId":      inst.SecurityID,
			}
		}
		msg = map[string]interface{}{
			"RequestCode":     RequestCodeSubscribe,
			"InstrumentCount": len(instruments),
			"InstrumentList":  instList,
		}
	}

	// Send subscription while holding lock
	err := c.conn.WriteJSON(msg)
	c.connLock.Unlock()

	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// Track subscribed instruments with proper locking
	c.instrLock.Lock()
	for _, inst := range instruments {
		key := fmt.Sprintf("%s:%d", inst.ExchangeSegment, inst.SecurityID)
		c.instruments[key] = inst
	}
	c.instrLock.Unlock()

	return nil
}

// IsConnected returns true if the client is connected
func (c *Client) IsConnected() bool {
	c.connLock.Lock()
	defer c.connLock.Unlock()
	return c.connected
}

// readLoop reads messages from the WebSocket connection
func (c *Client) readLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			c.connLock.Lock()
			if !c.connected {
				c.connLock.Unlock()
				return
			}
			conn := c.conn
			c.connLock.Unlock()

			_, data, err := conn.ReadMessage()
			if err != nil {
				c.connLock.Lock()
				connected := c.connected
				c.connLock.Unlock()
				if connected {
					c.notifyError(fmt.Errorf("read error: %w", err))
				}
				return
			}

			c.handleMessage(data)
		}
	}
}

// handleMessage processes a WebSocket message
// Data pointers passed to callbacks are deep copied for safety.
// Users receiving FullDepthData can safely store and use it.
func (c *Client) handleMessage(data []byte) {
	remaining := data

	for len(remaining) > 0 {
		depthData, next, err := parseDepthDataPooled(remaining, c.config.DepthLevel)
		if err != nil {
			c.notifyError(err)
			return
		}

		c.processDepthData(depthData)
		// Note: processDepthData keeps a reference to entries slice in pending,
		// so we just release the struct (entries slice is set to nil, not returned to pool)
		releaseDepthData(depthData)
		remaining = next
	}
}

// processDepthData processes parsed depth data
func (c *Client) processDepthData(data *DepthData) {
	c.pendingLock.Lock()
	defer c.pendingLock.Unlock()

	secID := data.Header.SecurityID

	// Get or create pending data for this security
	pending, exists := c.pendingDepth[secID]
	if !exists {
		pending = &FullDepthData{
			ExchangeSegment: data.Header.ExchangeSegment,
			SecurityID:      secID,
		}
		c.pendingDepth[secID] = pending
	}

	// Add entries to appropriate side
	if data.IsBid {
		pending.Bids = data.Entries
		// Sort bids descending by price
		sort.Slice(pending.Bids, func(i, j int) bool {
			return pending.Bids[i].Price > pending.Bids[j].Price
		})
	} else {
		pending.Asks = data.Entries
		// Sort asks ascending by price
		sort.Slice(pending.Asks, func(i, j int) bool {
			return pending.Asks[i].Price < pending.Asks[j].Price
		})
	}

	// If we have both bid and ask, notify callbacks with a deep copy
	if len(pending.Bids) > 0 && len(pending.Asks) > 0 {
		// Create a deep copy for callbacks to avoid race conditions
		notifyData := &FullDepthData{
			ExchangeSegment: pending.ExchangeSegment,
			SecurityID:      pending.SecurityID,
			Bids:            make([]DepthEntry, len(pending.Bids)),
			Asks:            make([]DepthEntry, len(pending.Asks)),
		}
		copy(notifyData.Bids, pending.Bids)
		copy(notifyData.Asks, pending.Asks)

		c.notifyDepth(notifyData)

		// Reset pending for next update
		c.pendingDepth[secID] = &FullDepthData{
			ExchangeSegment: data.Header.ExchangeSegment,
			SecurityID:      secID,
		}
	}
}

// notifyDepth notifies all registered depth callbacks
func (c *Client) notifyDepth(data *FullDepthData) {
	c.mu.RLock()
	callbacks := c.depthCallbacks
	c.mu.RUnlock()

	for _, cb := range callbacks {
		go cb(data)
	}
}

// notifyError notifies all registered error callbacks
func (c *Client) notifyError(err error) {
	c.mu.RLock()
	callbacks := c.errorCallbacks
	c.mu.RUnlock()

	for _, cb := range callbacks {
		go cb(err)
	}
}

// Stats returns connection statistics
type Stats struct {
	Connected        bool
	DepthLevel       DepthLevel
	InstrumentCount  int
	URL              string
}

// GetStats returns current connection statistics
func (c *Client) GetStats() Stats {
	c.connLock.Lock()
	connected := c.connected
	c.connLock.Unlock()

	c.instrLock.RLock()
	instrCount := len(c.instruments)
	c.instrLock.RUnlock()

	baseURL := Depth20URL
	if c.config.DepthLevel == Depth200 {
		baseURL = Depth200URL
	}

	return Stats{
		Connected:       connected,
		DepthLevel:      c.config.DepthLevel,
		InstrumentCount: instrCount,
		URL:             baseURL,
	}
}

// SubscribeJSON is a helper to subscribe using JSON string
// Example: `{"NSE_EQ": [11536], "NSE_FNO": [49081]}`
func (c *Client) SubscribeJSON(ctx context.Context, jsonData string) error {
	var data map[string][]int
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	var instruments []Instrument
	for exchange, secIDs := range data {
		for _, secID := range secIDs {
			instruments = append(instruments, Instrument{
				ExchangeSegment: exchange,
				SecurityID:      secID,
			})
		}
	}

	return c.Subscribe(ctx, instruments)
}

// WaitForConnection waits until connected or timeout
func (c *Client) WaitForConnection(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if c.IsConnected() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("connection timeout")
}
