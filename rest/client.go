// Package rest provides a clean wrapper around the auto-generated Dhan REST API client
package rest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samarthkathal/dhan-go/internal/limiter"
	"github.com/samarthkathal/dhan-go/internal/restgen"
)

// Client provides a clean interface to the Dhan REST API
type Client struct {
	gen         *restgen.ClientWithResponses
	rateLimiter *limiter.HTTPRateLimiter
}

// NewClient creates a new REST API client
func NewClient(baseURL, accessToken string, httpClient *http.Client, opts ...Option) (*Client, error) {
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

// TODO: Add remaining REST API methods following the same pattern
// This is a foundation - additional methods can be added as needed:
// - GetForeverOrders
// - PlaceForeverOrder
// - ModifyForeverOrder
// - CancelForeverOrder
// - GetAlertOrders
// - PlaceAlertOrder
// - ModifyAlertOrder
// - CancelAlertOrder
// - GetHistoricalCharts
// - GetIntradayCharts
// - GetOptionChain
// - EDIS methods
// - etc.
