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
