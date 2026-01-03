package limiter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// API Category Rate Limits (from https://dhanhq.co/docs/v2/)
const (
	// Order APIs: 25/sec, 250/min, 1000/hour, 7000/day
	OrderAPIsPerSecond = 25
	OrderAPIsPerMinute = 250
	OrderAPIsPerHour   = 1000
	OrderAPIsPerDay    = 7000

	// Data APIs: 5/sec, unlimited/min, unlimited/hour, 100k/day
	DataAPIsPerSecond = 5
	DataAPIsPerDay    = 100000

	// Quote APIs: 1/sec, unlimited after
	QuoteAPIsPerSecond = 1

	// Non-Trading APIs: 20/sec, unlimited after
	NonTradingAPIsPerSecond = 20
)

// EndpointCategory represents the category of an API endpoint
type EndpointCategory int

const (
	CategoryOrder EndpointCategory = iota
	CategoryData
	CategoryQuote
	CategoryNonTrading
)

// String returns the string representation of the category
func (c EndpointCategory) String() string {
	switch c {
	case CategoryOrder:
		return "Order"
	case CategoryData:
		return "Data"
	case CategoryQuote:
		return "Quote"
	case CategoryNonTrading:
		return "NonTrading"
	default:
		return "Unknown"
	}
}

// HTTPRateLimiter enforces Dhan's REST API rate limits
type HTTPRateLimiter struct {
	// Per-category rate limiters
	orderLimiters      *multiWindowLimiter
	dataLimiters       *multiWindowLimiter
	quoteLimiter       *rate.Limiter
	nonTradingLimiter  *rate.Limiter

	// Endpoint categorization
	endpointCategories map[string]EndpointCategory
	mu                 sync.RWMutex
}

// multiWindowLimiter handles rate limiting across multiple time windows
type multiWindowLimiter struct {
	perSecond *rate.Limiter
	perMinute *slidingWindowCounter
	perHour   *slidingWindowCounter
	perDay    *slidingWindowCounter
}

// slidingWindowCounter implements a sliding window counter for rate limiting
type slidingWindowCounter struct {
	limit    int
	window   time.Duration
	requests []time.Time
	mu       sync.Mutex
}

// NewHTTPRateLimiter creates a new HTTP rate limiter with Dhan's default limits
func NewHTTPRateLimiter() *HTTPRateLimiter {
	rl := &HTTPRateLimiter{
		// Order APIs: multiple windows
		orderLimiters: &multiWindowLimiter{
			perSecond: rate.NewLimiter(rate.Limit(OrderAPIsPerSecond), OrderAPIsPerSecond),
			perMinute: newSlidingWindowCounter(OrderAPIsPerMinute, time.Minute),
			perHour:   newSlidingWindowCounter(OrderAPIsPerHour, time.Hour),
			perDay:    newSlidingWindowCounter(OrderAPIsPerDay, 24*time.Hour),
		},
		// Data APIs: per-second and per-day only
		dataLimiters: &multiWindowLimiter{
			perSecond: rate.NewLimiter(rate.Limit(DataAPIsPerSecond), DataAPIsPerSecond),
			perDay:    newSlidingWindowCounter(DataAPIsPerDay, 24*time.Hour),
		},
		// Quote APIs: 1/sec
		quoteLimiter: rate.NewLimiter(rate.Limit(QuoteAPIsPerSecond), QuoteAPIsPerSecond),
		// Non-Trading APIs: 20/sec
		nonTradingLimiter: rate.NewLimiter(rate.Limit(NonTradingAPIsPerSecond), NonTradingAPIsPerSecond),

		endpointCategories: make(map[string]EndpointCategory),
	}

	// Initialize default endpoint categorizations
	rl.initializeEndpointCategories()

	return rl
}

// initializeEndpointCategories sets up the default endpoint-to-category mappings
func (rl *HTTPRateLimiter) initializeEndpointCategories() {
	// Order APIs
	orderEndpoints := []string{
		"/orders",           // Place order
		"/orders/",          // Modify/cancel order
		"/orders/slm",       // SL-M order
		"/orders/modify",    // Modify order
		"/orders/cancel",    // Cancel order
	}
	for _, ep := range orderEndpoints {
		rl.endpointCategories[ep] = CategoryOrder
	}

	// Data APIs
	dataEndpoints := []string{
		"/holdings",
		"/positions",
		"/funds",
		"/tradebook",
	}
	for _, ep := range dataEndpoints {
		rl.endpointCategories[ep] = CategoryData
	}

	// Quote APIs
	quoteEndpoints := []string{
		"/quotes",
		"/ltp",
		"/ohlc",
	}
	for _, ep := range quoteEndpoints {
		rl.endpointCategories[ep] = CategoryQuote
	}

	// Non-Trading APIs (default for everything else)
	nonTradingEndpoints := []string{
		"/edis",
		"/ledger",
		"/statements",
	}
	for _, ep := range nonTradingEndpoints {
		rl.endpointCategories[ep] = CategoryNonTrading
	}
}

// SetEndpointCategory allows customizing the category for an endpoint
func (rl *HTTPRateLimiter) SetEndpointCategory(endpoint string, category EndpointCategory) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.endpointCategories[endpoint] = category
}

// Wait blocks until the request is allowed under rate limits
// Returns error if context is cancelled
func (rl *HTTPRateLimiter) Wait(ctx context.Context, endpoint string) error {
	category := rl.categorizeEndpoint(endpoint)

	switch category {
	case CategoryOrder:
		return rl.waitOrderAPI(ctx)
	case CategoryData:
		return rl.waitDataAPI(ctx)
	case CategoryQuote:
		return rl.quoteLimiter.Wait(ctx)
	case CategoryNonTrading:
		return rl.nonTradingLimiter.Wait(ctx)
	default:
		// Default to non-trading limits if unknown
		return rl.nonTradingLimiter.Wait(ctx)
	}
}

// Allow checks if a request is allowed without blocking
func (rl *HTTPRateLimiter) Allow(endpoint string) error {
	category := rl.categorizeEndpoint(endpoint)

	switch category {
	case CategoryOrder:
		return rl.allowOrderAPI()
	case CategoryData:
		return rl.allowDataAPI()
	case CategoryQuote:
		if !rl.quoteLimiter.Allow() {
			return fmt.Errorf("quote API rate limit exceeded (1 req/sec)")
		}
		return nil
	case CategoryNonTrading:
		if !rl.nonTradingLimiter.Allow() {
			return fmt.Errorf("non-trading API rate limit exceeded (20 req/sec)")
		}
		return nil
	default:
		if !rl.nonTradingLimiter.Allow() {
			return fmt.Errorf("API rate limit exceeded")
		}
		return nil
	}
}

// categorizeEndpoint returns the category for an endpoint
func (rl *HTTPRateLimiter) categorizeEndpoint(endpoint string) EndpointCategory {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	// Check exact matches first
	if category, exists := rl.endpointCategories[endpoint]; exists {
		return category
	}

	// Check prefix matches for paths like /orders/{id}
	for pattern, category := range rl.endpointCategories {
		if len(pattern) > 0 && pattern[len(pattern)-1] == '/' {
			// Pattern ends with /, check prefix
			if len(endpoint) >= len(pattern) && endpoint[:len(pattern)] == pattern {
				return category
			}
		}
	}

	// Default to non-trading
	return CategoryNonTrading
}

// waitOrderAPI waits for order API rate limits
func (rl *HTTPRateLimiter) waitOrderAPI(ctx context.Context) error {
	// Check per-second limit (token bucket)
	if err := rl.orderLimiters.perSecond.Wait(ctx); err != nil {
		return fmt.Errorf("order API rate limit (per-second): %w", err)
	}

	// Check other windows (non-blocking, just tracking)
	if !rl.orderLimiters.perMinute.allow() {
		return fmt.Errorf("order API rate limit exceeded (250 req/min)")
	}
	if !rl.orderLimiters.perHour.allow() {
		return fmt.Errorf("order API rate limit exceeded (1000 req/hour)")
	}
	if !rl.orderLimiters.perDay.allow() {
		return fmt.Errorf("order API rate limit exceeded (7000 req/day)")
	}

	return nil
}

// allowOrderAPI checks order API rate limits without blocking
func (rl *HTTPRateLimiter) allowOrderAPI() error {
	if !rl.orderLimiters.perSecond.Allow() {
		return fmt.Errorf("order API rate limit exceeded (25 req/sec)")
	}
	if !rl.orderLimiters.perMinute.allow() {
		return fmt.Errorf("order API rate limit exceeded (250 req/min)")
	}
	if !rl.orderLimiters.perHour.allow() {
		return fmt.Errorf("order API rate limit exceeded (1000 req/hour)")
	}
	if !rl.orderLimiters.perDay.allow() {
		return fmt.Errorf("order API rate limit exceeded (7000 req/day)")
	}
	return nil
}

// waitDataAPI waits for data API rate limits
func (rl *HTTPRateLimiter) waitDataAPI(ctx context.Context) error {
	if err := rl.dataLimiters.perSecond.Wait(ctx); err != nil {
		return fmt.Errorf("data API rate limit (per-second): %w", err)
	}
	if !rl.dataLimiters.perDay.allow() {
		return fmt.Errorf("data API rate limit exceeded (100k req/day)")
	}
	return nil
}

// allowDataAPI checks data API rate limits without blocking
func (rl *HTTPRateLimiter) allowDataAPI() error {
	if !rl.dataLimiters.perSecond.Allow() {
		return fmt.Errorf("data API rate limit exceeded (5 req/sec)")
	}
	if !rl.dataLimiters.perDay.allow() {
		return fmt.Errorf("data API rate limit exceeded (100k req/day)")
	}
	return nil
}

// GetStats returns current rate limiter statistics
func (rl *HTTPRateLimiter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"order_apis": map[string]interface{}{
			"per_second_limit": OrderAPIsPerSecond,
			"per_minute_limit": OrderAPIsPerMinute,
			"per_hour_limit":   OrderAPIsPerHour,
			"per_day_limit":    OrderAPIsPerDay,
			"per_minute_used":  rl.orderLimiters.perMinute.count(),
			"per_hour_used":    rl.orderLimiters.perHour.count(),
			"per_day_used":     rl.orderLimiters.perDay.count(),
		},
		"data_apis": map[string]interface{}{
			"per_second_limit": DataAPIsPerSecond,
			"per_day_limit":    DataAPIsPerDay,
			"per_day_used":     rl.dataLimiters.perDay.count(),
		},
		"quote_apis": map[string]interface{}{
			"per_second_limit": QuoteAPIsPerSecond,
		},
		"non_trading_apis": map[string]interface{}{
			"per_second_limit": NonTradingAPIsPerSecond,
		},
	}
}

// newSlidingWindowCounter creates a new sliding window counter
func newSlidingWindowCounter(limit int, window time.Duration) *slidingWindowCounter {
	return &slidingWindowCounter{
		limit:    limit,
		window:   window,
		requests: make([]time.Time, 0, limit),
	}
}

// allow checks if a new request is allowed and records it if so
func (swc *slidingWindowCounter) allow() bool {
	swc.mu.Lock()
	defer swc.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-swc.window)

	// Remove expired requests
	validIdx := 0
	for i, reqTime := range swc.requests {
		if reqTime.After(windowStart) {
			validIdx = i
			break
		}
	}
	swc.requests = swc.requests[validIdx:]

	// Check if we're under the limit
	if len(swc.requests) >= swc.limit {
		return false
	}

	// Record this request
	swc.requests = append(swc.requests, now)
	return true
}

// count returns the current number of requests in the window
func (swc *slidingWindowCounter) count() int {
	swc.mu.Lock()
	defer swc.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-swc.window)

	// Count valid requests
	count := 0
	for _, reqTime := range swc.requests {
		if reqTime.After(windowStart) {
			count++
		}
	}

	return count
}
