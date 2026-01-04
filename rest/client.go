// Package rest provides a clean wrapper around the auto-generated Dhan REST API client
package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/samarthkathal/dhan-go/internal/limiter"
	"github.com/samarthkathal/dhan-go/internal/restgen"
)

// Client provides a clean interface to the Dhan REST API
type Client struct {
	gen         *restgen.ClientWithResponses
	rateLimiter *limiter.HTTPRateLimiter
	httpClient  *http.Client
	baseURL     string
	accessToken string
}

// NewClient creates a new REST API client
func NewClient(baseURL, accessToken string, httpClient *http.Client, opts ...Option) (*Client, error) {
	// Use default HTTP client if none provided
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	// Apply options to build configuration
	cfg := &clientConfig{
		httpClient: httpClient,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Create auth middleware
	authMiddleware := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("access-token", accessToken)
		req.Header.Set("Content-Type", "application/json")
		return nil
	}

	// Create rate limiting middleware (if enabled)
	var rateLimitMiddleware restgen.RequestEditorFn
	if cfg.rateLimiter != nil {
		rateLimitMiddleware = func(ctx context.Context, req *http.Request) error {
			// Wait for rate limit before making request
			if err := cfg.rateLimiter.Wait(ctx, req.URL.Path); err != nil {
				return fmt.Errorf("rate limit: %w", err)
			}
			return nil
		}
	}

	// Combine all middleware (rate limit first, then auth, then user)
	reqEditors := []restgen.RequestEditorFn{}
	if rateLimitMiddleware != nil {
		reqEditors = append(reqEditors, rateLimitMiddleware)
	}
	reqEditors = append(reqEditors, authMiddleware)
	if cfg.requestEditor != nil {
		reqEditors = append(reqEditors, cfg.requestEditor)
	}

	// Create generated client
	genClient, err := restgen.NewClientWithResponses(
		baseURL,
		restgen.WithHTTPClient(cfg.httpClient),
		restgen.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
			for _, editor := range reqEditors {
				if err := editor(ctx, req); err != nil {
					return err
				}
			}
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create REST client: %w", err)
	}

	return &Client{
		gen:         genClient,
		rateLimiter: cfg.rateLimiter,

		// Some endpoints are not supported by the generated client
		// so we need to use the http client directly for those endpoints
		httpClient:  cfg.httpClient,
		baseURL:     baseURL,
		accessToken: accessToken,
	}, nil
}

// ============================================================================
// GENERATED CLIENT METHODS
// These methods use the auto-generated OpenAPI client (c.gen.*)
// ============================================================================

// ----------------------------------------------------------------------------
// Portfolio
// ----------------------------------------------------------------------------

// GetHoldings retrieves user's holdings
func (c *Client) GetHoldings(ctx context.Context) (*restgen.GetholdingsResult, error) {
	resp, err := c.gen.GetholdingsWithResponse(ctx, &restgen.GetholdingsParams{})
	if err != nil {
		return nil, fmt.Errorf("get holdings failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get holdings returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// GetPositions retrieves user's positions
func (c *Client) GetPositions(ctx context.Context) (*restgen.GetpositionsResult, error) {
	resp, err := c.gen.GetpositionsWithResponse(ctx, &restgen.GetpositionsParams{})
	if err != nil {
		return nil, fmt.Errorf("get positions failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get positions returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// ConvertPosition converts a position (e.g., intraday to CNC or vice versa)
func (c *Client) ConvertPosition(ctx context.Context, req restgen.ConvertpositionJSONRequestBody) (*restgen.ConvertpositionResult, error) {
	resp, err := c.gen.ConvertpositionWithResponse(ctx, &restgen.ConvertpositionParams{}, req)
	if err != nil {
		return nil, fmt.Errorf("convert position failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("convert position returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// ----------------------------------------------------------------------------
// Orders
// ----------------------------------------------------------------------------

// GetOrders retrieves user's orders
func (c *Client) GetOrders(ctx context.Context) (*restgen.GetordersResult, error) {
	resp, err := c.gen.GetordersWithResponse(ctx, &restgen.GetordersParams{})
	if err != nil {
		return nil, fmt.Errorf("get orders failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get orders returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// GetOrderByID retrieves a specific order by order ID
func (c *Client) GetOrderByID(ctx context.Context, orderID string) (*restgen.GetorderbyorderidResult, error) {
	resp, err := c.gen.GetorderbyorderidWithResponse(ctx, orderID, &restgen.GetorderbyorderidParams{})
	if err != nil {
		return nil, fmt.Errorf("get order by ID failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get order by ID returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// GetOrderByCorrelationID retrieves an order by correlation ID
func (c *Client) GetOrderByCorrelationID(ctx context.Context, correlationID string) (*restgen.GetorderbycorrelationidResult, error) {
	resp, err := c.gen.GetorderbycorrelationidWithResponse(ctx, correlationID, &restgen.GetorderbycorrelationidParams{})
	if err != nil {
		return nil, fmt.Errorf("get order by correlation ID failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get order by correlation ID returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// PlaceOrder places a new order
func (c *Client) PlaceOrder(ctx context.Context, req restgen.PlaceorderJSONRequestBody) (*restgen.PlaceorderResult, error) {
	resp, err := c.gen.PlaceorderWithResponse(ctx, &restgen.PlaceorderParams{}, req)
	if err != nil {
		return nil, fmt.Errorf("place order failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("place order returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// ModifyOrder modifies an existing order
func (c *Client) ModifyOrder(ctx context.Context, orderID string, req restgen.ModifyorderJSONRequestBody) (*restgen.ModifyorderResult, error) {
	resp, err := c.gen.ModifyorderWithResponse(ctx, orderID, &restgen.ModifyorderParams{}, req)
	if err != nil {
		return nil, fmt.Errorf("modify order failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("modify order returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// CancelOrder cancels an existing order
func (c *Client) CancelOrder(ctx context.Context, orderID string) (*restgen.CancelorderResult, error) {
	resp, err := c.gen.CancelorderWithResponse(ctx, orderID, &restgen.CancelorderParams{})
	if err != nil {
		return nil, fmt.Errorf("cancel order failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("cancel order returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// PlaceSliceOrder places a slice/basket order (splits large orders)
func (c *Client) PlaceSliceOrder(ctx context.Context, req restgen.PlacesliceorderJSONRequestBody) (*restgen.PlacesliceorderResult, error) {
	resp, err := c.gen.PlacesliceorderWithResponse(ctx, &restgen.PlacesliceorderParams{}, req)
	if err != nil {
		return nil, fmt.Errorf("place slice order failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("place slice order returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// ----------------------------------------------------------------------------
// Forever Orders (GTT - Good Till Triggered)
// ----------------------------------------------------------------------------

// GetForeverOrders retrieves all forever/GTT orders
func (c *Client) GetForeverOrders(ctx context.Context) (*restgen.GetforeverordersResult, error) {
	resp, err := c.gen.GetforeverordersWithResponse(ctx, &restgen.GetforeverordersParams{})
	if err != nil {
		return nil, fmt.Errorf("get forever orders failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get forever orders returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// PlaceForeverOrder places a new forever/GTT order
func (c *Client) PlaceForeverOrder(ctx context.Context, req restgen.PlaceforeverorderJSONRequestBody) (*restgen.PlaceforeverorderResult, error) {
	resp, err := c.gen.PlaceforeverorderWithResponse(ctx, &restgen.PlaceforeverorderParams{}, req)
	if err != nil {
		return nil, fmt.Errorf("place forever order failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("place forever order returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// ModifyForeverOrder modifies an existing forever/GTT order
func (c *Client) ModifyForeverOrder(ctx context.Context, orderID string, req restgen.ModifyforeverorderJSONRequestBody) (*restgen.ModifyforeverorderResult, error) {
	resp, err := c.gen.ModifyforeverorderWithResponse(ctx, orderID, &restgen.ModifyforeverorderParams{}, req)
	if err != nil {
		return nil, fmt.Errorf("modify forever order failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("modify forever order returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// CancelForeverOrder cancels an existing forever/GTT order
func (c *Client) CancelForeverOrder(ctx context.Context, orderID string) (*restgen.CancelforeverorderResult, error) {
	resp, err := c.gen.CancelforeverorderWithResponse(ctx, orderID, &restgen.CancelforeverorderParams{})
	if err != nil {
		return nil, fmt.Errorf("cancel forever order failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("cancel forever order returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// ----------------------------------------------------------------------------
// Alert Orders
// ----------------------------------------------------------------------------

// GetAllAlertOrders retrieves all alert orders
func (c *Client) GetAllAlertOrders(ctx context.Context) (*restgen.GetAllAlertOrdersResult, error) {
	resp, err := c.gen.GetAllAlertOrdersWithResponse(ctx, &restgen.GetAllAlertOrdersParams{})
	if err != nil {
		return nil, fmt.Errorf("get all alert orders failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get all alert orders returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// GetAlertOrder retrieves a specific alert order by ID
func (c *Client) GetAlertOrder(ctx context.Context, alertID string) (*restgen.GetAlertOrderResult, error) {
	resp, err := c.gen.GetAlertOrderWithResponse(ctx, alertID, &restgen.GetAlertOrderParams{})
	if err != nil {
		return nil, fmt.Errorf("get alert order failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get alert order returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// PlaceAlertOrder places a new alert order
func (c *Client) PlaceAlertOrder(ctx context.Context, req restgen.AlertOrderJSONRequestBody) (*restgen.AlertOrderResult, error) {
	resp, err := c.gen.AlertOrderWithResponse(ctx, &restgen.AlertOrderParams{}, req)
	if err != nil {
		return nil, fmt.Errorf("place alert order failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("place alert order returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// ModifyAlertOrder modifies an existing alert order
func (c *Client) ModifyAlertOrder(ctx context.Context, alertID string, req restgen.ModifyAlertOrderJSONRequestBody) (*restgen.ModifyAlertOrderResult, error) {
	resp, err := c.gen.ModifyAlertOrderWithResponse(ctx, alertID, &restgen.ModifyAlertOrderParams{}, req)
	if err != nil {
		return nil, fmt.Errorf("modify alert order failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("modify alert order returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// DeleteAlertOrder deletes an alert order
func (c *Client) DeleteAlertOrder(ctx context.Context, alertID string) (*restgen.DelAlertOrderResult, error) {
	resp, err := c.gen.DelAlertOrderWithResponse(ctx, alertID, &restgen.DelAlertOrderParams{})
	if err != nil {
		return nil, fmt.Errorf("delete alert order failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("delete alert order returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// ----------------------------------------------------------------------------
// Super Orders (Bracket/Cover Orders)
// ----------------------------------------------------------------------------

// GetSuperOrders retrieves all super/bracket orders
func (c *Client) GetSuperOrders(ctx context.Context) (*restgen.GetsuperordersResult, error) {
	resp, err := c.gen.GetsuperordersWithResponse(ctx, &restgen.GetsuperordersParams{})
	if err != nil {
		return nil, fmt.Errorf("get super orders failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get super orders returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// PlaceSuperOrder places a new super/bracket order
func (c *Client) PlaceSuperOrder(ctx context.Context, req restgen.PlacesuperorderJSONRequestBody) (*restgen.PlacesuperorderResult, error) {
	resp, err := c.gen.PlacesuperorderWithResponse(ctx, &restgen.PlacesuperorderParams{}, req)
	if err != nil {
		return nil, fmt.Errorf("place super order failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("place super order returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// ModifySuperOrder modifies an existing super/bracket order
func (c *Client) ModifySuperOrder(ctx context.Context, orderID string, req restgen.ModifysuperorderJSONRequestBody) (*restgen.ModifysuperorderResult, error) {
	resp, err := c.gen.ModifysuperorderWithResponse(ctx, orderID, &restgen.ModifysuperorderParams{}, req)
	if err != nil {
		return nil, fmt.Errorf("modify super order failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("modify super order returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// CancelSuperOrder cancels a super/bracket order
// orderLeg specifies which leg to cancel (e.g., "ENTRY_LEG", "TARGET_LEG", "STOP_LOSS_LEG")
func (c *Client) CancelSuperOrder(ctx context.Context, orderID string, orderLeg string) (*restgen.CancelsuperorderResult, error) {
	resp, err := c.gen.CancelsuperorderWithResponse(ctx, orderID, restgen.CancelsuperorderParamsOrderLeg(orderLeg), &restgen.CancelsuperorderParams{})
	if err != nil {
		return nil, fmt.Errorf("cancel super order failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("cancel super order returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// ----------------------------------------------------------------------------
// Trades
// ----------------------------------------------------------------------------

// GetAllTrades retrieves all trades for today
func (c *Client) GetAllTrades(ctx context.Context) (*restgen.GetalltradesResult, error) {
	resp, err := c.gen.GetalltradesWithResponse(ctx, &restgen.GetalltradesParams{})
	if err != nil {
		return nil, fmt.Errorf("get all trades failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get all trades returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// GetTradeHistory retrieves paginated trade history
func (c *Client) GetTradeHistory(ctx context.Context, fromDate, toDate string, pageNumber string) (*restgen.GettradehistoryResult, error) {
	resp, err := c.gen.GettradehistoryWithResponse(ctx, fromDate, toDate, pageNumber, &restgen.GettradehistoryParams{})
	if err != nil {
		return nil, fmt.Errorf("get trade history failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get trade history returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// GetTradesByOrderID retrieves trades for a specific order
func (c *Client) GetTradesByOrderID(ctx context.Context, orderID string) (*restgen.GettradebyorderidResult, error) {
	resp, err := c.gen.GettradebyorderidWithResponse(ctx, orderID, &restgen.GettradebyorderidParams{})
	if err != nil {
		return nil, fmt.Errorf("get trades by order ID failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get trades by order ID returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// ----------------------------------------------------------------------------
// Funds & Margin
// ----------------------------------------------------------------------------

// GetFundLimits retrieves fund limits
func (c *Client) GetFundLimits(ctx context.Context) (*restgen.FundlimitResult, error) {
	resp, err := c.gen.FundlimitWithResponse(ctx, &restgen.FundlimitParams{})
	if err != nil {
		return nil, fmt.Errorf("get fund limits failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get fund limits returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// GetLedger retrieves ledger/cash flow information
func (c *Client) GetLedger(ctx context.Context, fromDate, toDate string) (*restgen.LedgerResult, error) {
	resp, err := c.gen.LedgerWithResponse(ctx, &restgen.LedgerParams{
		FromDate: &fromDate,
		ToDate:   &toDate,
	})
	if err != nil {
		return nil, fmt.Errorf("get ledger failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get ledger returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// CalculateMargin calculates margin requirements for an order
func (c *Client) CalculateMargin(ctx context.Context, req restgen.MargincalculatorJSONRequestBody) (*restgen.MargincalculatorResult, error) {
	resp, err := c.gen.MargincalculatorWithResponse(ctx, &restgen.MargincalculatorParams{}, req)
	if err != nil {
		return nil, fmt.Errorf("calculate margin failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("calculate margin returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// ----------------------------------------------------------------------------
// Historical Data
// ----------------------------------------------------------------------------

// GetHistoricalData retrieves daily historical OHLC data for a security
func (c *Client) GetHistoricalData(ctx context.Context, req restgen.HistoricalchartsJSONRequestBody) (*restgen.HistoricalchartsResult, error) {
	resp, err := c.gen.HistoricalchartsWithResponse(ctx, &restgen.HistoricalchartsParams{}, req)
	if err != nil {
		return nil, fmt.Errorf("get historical data failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get historical data returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// GetIntradayData retrieves intraday OHLC data for a security
func (c *Client) GetIntradayData(ctx context.Context, req restgen.IntradaychartsJSONRequestBody) (*restgen.IntradaychartsResult, error) {
	resp, err := c.gen.IntradaychartsWithResponse(ctx, &restgen.IntradaychartsParams{}, req)
	if err != nil {
		return nil, fmt.Errorf("get intraday data failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get intraday data returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// GetExpiredOptionsData retrieves historical data for expired options on a rolling basis
func (c *Client) GetExpiredOptionsData(ctx context.Context, req restgen.OptionchartJSONRequestBody) (*restgen.OptionchartResult, error) {
	resp, err := c.gen.OptionchartWithResponse(ctx, &restgen.OptionchartParams{}, req)
	if err != nil {
		return nil, fmt.Errorf("get expired options data failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get expired options data returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// ----------------------------------------------------------------------------
// Kill Switch
// ----------------------------------------------------------------------------

// GetKillSwitchStatus retrieves the current kill switch status
func (c *Client) GetKillSwitchStatus(ctx context.Context) (*restgen.KillSwitchStatusResult, error) {
	resp, err := c.gen.KillSwitchStatusWithResponse(ctx, &restgen.KillSwitchStatusParams{})
	if err != nil {
		return nil, fmt.Errorf("get kill switch status failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get kill switch status returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// SetKillSwitch activates or deactivates the kill switch
// status should be "ACTIVATE" or "DEACTIVATE"
func (c *Client) SetKillSwitch(ctx context.Context, status string) (*restgen.KillswitchResult, error) {
	resp, err := c.gen.KillswitchWithResponse(ctx, &restgen.KillswitchParams{KillSwitchStatus: status})
	if err != nil {
		return nil, fmt.Errorf("set kill switch failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("set kill switch returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// ----------------------------------------------------------------------------
// EDIS (Electronic Delivery Instruction Slip)
// ----------------------------------------------------------------------------

// SubmitEDISForm submits an EDIS form for holdings sell
func (c *Client) SubmitEDISForm(ctx context.Context, req restgen.EdisformJSONRequestBody) (*restgen.EdisformResult, error) {
	resp, err := c.gen.EdisformWithResponse(ctx, &restgen.EdisformParams{}, req)
	if err != nil {
		return nil, fmt.Errorf("submit EDIS form failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("submit EDIS form returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// SubmitBulkEDISForm submits bulk EDIS forms
func (c *Client) SubmitBulkEDISForm(ctx context.Context, req restgen.BulkedisformJSONRequestBody) (*restgen.BulkedisformResult, error) {
	resp, err := c.gen.BulkedisformWithResponse(ctx, &restgen.BulkedisformParams{}, req)
	if err != nil {
		return nil, fmt.Errorf("submit bulk EDIS form failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("submit bulk EDIS form returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// GetEDISQuantityStatus retrieves EDIS quantity status for an ISIN
func (c *Client) GetEDISQuantityStatus(ctx context.Context, isin string) (*restgen.EdisqtystatusResult, error) {
	resp, err := c.gen.EdisqtystatusWithResponse(ctx, isin, &restgen.EdisqtystatusParams{})
	if err != nil {
		return nil, fmt.Errorf("get EDIS quantity status failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get EDIS quantity status returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// GetEDISTPIN retrieves EDIS T-PIN inquiry form
func (c *Client) GetEDISTPIN(ctx context.Context) (*restgen.EdistpinResult, error) {
	resp, err := c.gen.EdistpinWithResponse(ctx, &restgen.EdistpinParams{})
	if err != nil {
		return nil, fmt.Errorf("get EDIS TPIN failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get EDIS TPIN returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// ----------------------------------------------------------------------------
// IP Management
// ----------------------------------------------------------------------------

// GetIP retrieves registered IP addresses
func (c *Client) GetIP(ctx context.Context) (*restgen.GetIPResult, error) {
	resp, err := c.gen.GetIPWithResponse(ctx, &restgen.GetIPParams{})
	if err != nil {
		return nil, fmt.Errorf("get IP failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get IP returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// SetIP sets the primary and secondary IP addresses
func (c *Client) SetIP(ctx context.Context, req restgen.SetIPJSONRequestBody) (*restgen.SetIPResult, error) {
	resp, err := c.gen.SetIPWithResponse(ctx, &restgen.SetIPParams{}, req)
	if err != nil {
		return nil, fmt.Errorf("set IP failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("set IP returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// ModifyIP modifies the registered IP addresses
func (c *Client) ModifyIP(ctx context.Context, req restgen.ModifyIPJSONRequestBody) (*restgen.ModifyIPResult, error) {
	resp, err := c.gen.ModifyIPWithResponse(ctx, &restgen.ModifyIPParams{}, req)
	if err != nil {
		return nil, fmt.Errorf("modify IP failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("modify IP returned status %d", resp.StatusCode())
	}

	return resp, nil
}

// ----------------------------------------------------------------------------
// Rate Limiter Helpers
// ----------------------------------------------------------------------------

// GetRateLimiterStats returns current rate limiter statistics
// Returns nil if rate limiting is not enabled
func (c *Client) GetRateLimiterStats() map[string]interface{} {
	if c.rateLimiter == nil {
		return nil
	}
	return c.rateLimiter.GetStats()
}

// GetRateLimiter returns the underlying rate limiter
// Returns nil if rate limiting is not enabled
func (c *Client) GetRateLimiter() *limiter.HTTPRateLimiter {
	return c.rateLimiter
}

// ============================================================================
// MANUAL HTTP METHODS
// These endpoints are not in the OpenAPI spec, so we use direct HTTP calls.
// ============================================================================

// doRequest performs an HTTP request with authentication headers
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("access-token", c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Apply rate limiting if enabled
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, path); err != nil {
			return nil, fmt.Errorf("rate limit: %w", err)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// ----------------------------------------------------------------------------
// Market Quote (Manual HTTP)
// ----------------------------------------------------------------------------

// GetLTP retrieves last traded price for the specified securities.
// Request format: {"NSE_EQ": [11536], "NSE_FNO": [49081, 49082]}
func (c *Client) GetLTP(ctx context.Context, req MarketQuoteRequest) (*LTPResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/marketfeed/ltp", req)
	if err != nil {
		return nil, fmt.Errorf("get LTP failed: %w", err)
	}

	var result LTPResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse LTP response: %w", err)
	}

	return &result, nil
}

// GetOHLC retrieves OHLC data for the specified securities.
// Request format: {"NSE_EQ": [11536], "NSE_FNO": [49081, 49082]}
func (c *Client) GetOHLC(ctx context.Context, req MarketQuoteRequest) (*OHLCResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/marketfeed/ohlc", req)
	if err != nil {
		return nil, fmt.Errorf("get OHLC failed: %w", err)
	}

	var result OHLCResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse OHLC response: %w", err)
	}

	return &result, nil
}

// GetQuote retrieves full quote data including market depth for the specified securities.
// Request format: {"NSE_EQ": [11536], "NSE_FNO": [49081, 49082]}
func (c *Client) GetQuote(ctx context.Context, req MarketQuoteRequest) (*QuoteResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/marketfeed/quote", req)
	if err != nil {
		return nil, fmt.Errorf("get quote failed: %w", err)
	}

	var result QuoteResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse quote response: %w", err)
	}

	return &result, nil
}

// ----------------------------------------------------------------------------
// Option Chain (Manual HTTP)
// ----------------------------------------------------------------------------

// GetOptionChain retrieves the option chain for a specified underlying instrument.
func (c *Client) GetOptionChain(ctx context.Context, underlyingScrip int, underlyingSeg, expiry string) (*OptionChainResponse, error) {
	req := OptionChainRequest{
		UnderlyingScrip: underlyingScrip,
		UnderlyingSeg:   underlyingSeg,
		Expiry:          expiry,
	}

	respBody, err := c.doRequest(ctx, http.MethodPost, "/optionchain", req)
	if err != nil {
		return nil, fmt.Errorf("get option chain failed: %w", err)
	}

	var result OptionChainResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse option chain response: %w", err)
	}

	return &result, nil
}

// GetExpiryList retrieves the list of expiry dates for a specified underlying instrument.
func (c *Client) GetExpiryList(ctx context.Context, underlyingScrip int, underlyingSeg string) (*ExpiryListResponse, error) {
	req := ExpiryListRequest{
		UnderlyingScrip: underlyingScrip,
		UnderlyingSeg:   underlyingSeg,
	}

	respBody, err := c.doRequest(ctx, http.MethodPost, "/optionchain/expirylist", req)
	if err != nil {
		return nil, fmt.Errorf("get expiry list failed: %w", err)
	}

	var result ExpiryListResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse expiry list response: %w", err)
	}

	return &result, nil
}
