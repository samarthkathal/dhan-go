package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/samarthkathal/dhan-go/client"
	"github.com/samarthkathal/dhan-go/utils"
)

func main() {
	// ========================================
	// 1. SETUP: Context for graceful shutdown
	// ========================================
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		fmt.Printf("\n[SHUTDOWN] Received signal: %v\n", sig)
		fmt.Println("[SHUTDOWN] Cancelling all contexts...")
		cancel()
	}()

	// ========================================
	// 2. SETUP: Metrics and logging
	// ========================================
	logger := log.New(os.Stdout, "[DHAN] ", log.LstdFlags)
	metricsCollector := utils.NewMetricsCollector()

	// ========================================
	// 3. SETUP: HTTP client with middleware
	// ========================================
	// Start with high-throughput configuration
	httpClient := utils.HighThroughputHTTPClient()

	// Add middleware stack (applied in order)
	httpClient = utils.WithMiddleware(
		httpClient,
		utils.RecoveryRoundTripper(logger),              // 1. Panic recovery (innermost)
		utils.RateLimitRoundTripper(100, 10),            // 2. Rate limit: 100 req/sec, burst 10
		utils.MetricsRoundTripper(metricsCollector),     // 3. Collect metrics
		utils.LoggingRoundTripper(logger),               // 4. Log requests (outermost)
	)

	// Set overall client timeout
	httpClient.Timeout = 30 * time.Second

	// ========================================
	// 4. SETUP: Authentication
	// ========================================
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		accessToken = "your-access-token"
	}

	authMiddleware := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("access-token", accessToken)
		req.Header.Set("User-Agent", "dhan-go-sdk/1.0")
		return nil
	}

	// ========================================
	// 5. SETUP: Create Dhan API client
	// ========================================
	dhanClient, err := client.NewClientWithResponses(
		"https://api.dhan.co",
		client.WithHTTPClient(httpClient),
		client.WithRequestEditorFn(authMiddleware),
	)
	if err != nil {
		log.Fatal(err)
	}

	logger.Println("=== Dhan Go SDK - All Features Example ===")
	logger.Println("Features enabled:")
	logger.Println("  ✓ High-throughput connection pool")
	logger.Println("  ✓ Rate limiting (100 req/sec, burst 10)")
	logger.Println("  ✓ Request/response logging")
	logger.Println("  ✓ Metrics collection")
	logger.Println("  ✓ Panic recovery")
	logger.Println("  ✓ Graceful shutdown (Press Ctrl+C)")
	logger.Println("")

	// ========================================
	// 6. EXAMPLE: Fetch portfolio data
	// ========================================
	if err := fetchPortfolioData(ctx, dhanClient); err != nil {
		if ctx.Err() == context.Canceled {
			logger.Println("[INFO] Portfolio fetch cancelled due to shutdown")
		} else {
			logger.Printf("[ERROR] Portfolio fetch failed: %v\n", err)
		}
	}

	// ========================================
	// 7. EXAMPLE: Fetch market data
	// ========================================
	if err := fetchMarketData(ctx, dhanClient); err != nil {
		if ctx.Err() == context.Canceled {
			logger.Println("[INFO] Market data fetch cancelled due to shutdown")
		} else {
			logger.Printf("[ERROR] Market data fetch failed: %v\n", err)
		}
	}

	// ========================================
	// 8. EXAMPLE: Long-running operations
	// ========================================
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	logger.Println("\n[INFO] Starting periodic position checks (every 10s)...")
	logger.Println("[INFO] Press Ctrl+C to shutdown gracefully")

	for {
		select {
		case <-ctx.Done():
			logger.Println("[SHUTDOWN] Context cancelled. Exiting...")
			printMetricsSummary(metricsCollector)
			return

		case <-ticker.C:
			if err := checkPositions(ctx, dhanClient); err != nil {
				if ctx.Err() == context.Canceled {
					logger.Println("[SHUTDOWN] Position check cancelled")
					printMetricsSummary(metricsCollector)
					return
				}
				logger.Printf("[ERROR] Position check failed: %v\n", err)
			}
		}
	}
}

// fetchPortfolioData demonstrates fetching portfolio data with timeout
func fetchPortfolioData(ctx context.Context, dhanClient *client.ClientWithResponses) error {
	fmt.Println("\n=== Fetching Portfolio Data ===")

	// Create timeout context for this operation
	requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Get holdings
	holdings, err := dhanClient.GetholdingsWithResponse(requestCtx, nil)
	if err != nil {
		return fmt.Errorf("holdings: %w", err)
	}

	if holdings.StatusCode() == 200 && holdings.JSON200 != nil {
		fmt.Printf("✓ Holdings: %d holdings\n", len(*holdings.JSON200))
	}

	// Get positions
	positions, err := dhanClient.GetpositionsWithResponse(requestCtx, nil)
	if err != nil {
		return fmt.Errorf("positions: %w", err)
	}

	if positions.StatusCode() == 200 && positions.JSON200 != nil {
		fmt.Printf("✓ Positions: %d positions\n", len(*positions.JSON200))
	}

	// Get fund limits
	funds, err := dhanClient.FundlimitWithResponse(requestCtx, nil)
	if err != nil {
		return fmt.Errorf("funds: %w", err)
	}

	if funds.StatusCode() == 200 && funds.JSON200 != nil {
		fmt.Println("✓ Fund limits fetched")
		if funds.JSON200.AvailabelBalance != nil {
			fmt.Printf("  Available balance: ₹%.2f\n", *funds.JSON200.AvailabelBalance)
		}
	}

	return nil
}

// fetchMarketData demonstrates fetching historical market data
func fetchMarketData(ctx context.Context, dhanClient *client.ClientWithResponses) error {
	fmt.Println("\n=== Fetching Market Data ===")

	requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	securityID := "1333" // HDFC Bank
	exchangeSegment := client.HistoricalChartsRequestExchangeSegmentNSEEQ
	instrument := client.HistoricalChartsRequestInstrumentEQUITY

	req := client.HistoricalchartsJSONRequestBody{
		SecurityId:      &securityID,
		ExchangeSegment: &exchangeSegment,
		Instrument:      &instrument,
	}

	historical, err := dhanClient.HistoricalchartsWithResponse(requestCtx, nil, req)
	if err != nil {
		return fmt.Errorf("historical data: %w", err)
	}

	if historical.StatusCode() == 200 && historical.JSON200 != nil {
		fmt.Printf("✓ Historical data: %d candles (HDFC Bank)\n", len(*historical.JSON200.Close))
	}

	return nil
}

// checkPositions demonstrates periodic position checks
func checkPositions(ctx context.Context, dhanClient *client.ClientWithResponses) error {
	requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	positions, err := dhanClient.GetpositionsWithResponse(requestCtx, nil)
	if err != nil {
		return err
	}

	if positions.StatusCode() == 200 && positions.JSON200 != nil {
		fmt.Printf("\n[CHECK] Current positions: %d\n", len(*positions.JSON200))
	}

	return nil
}

// printMetricsSummary prints collected metrics
func printMetricsSummary(collector *utils.MetricsCollector) {
	metrics := collector.GetMetrics()

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("=== METRICS SUMMARY ===")
	fmt.Println(strings.Repeat("=", 50))

	fmt.Printf("Total Requests: %v\n", metrics["total_requests"])
	fmt.Printf("Total Errors: %v\n", metrics["total_errors"])

	fmt.Println("\nRequest Counts by Endpoint:")
	if counts, ok := metrics["request_counts"].(map[string]int64); ok {
		for endpoint, count := range counts {
			fmt.Printf("  %s: %d\n", endpoint, count)
		}
	}

	fmt.Println("\nAverage Request Durations:")
	if durations, ok := metrics["request_durations_ms"].(map[string]int64); ok {
		if counts, ok := metrics["request_counts"].(map[string]int64); ok {
			for endpoint, duration := range durations {
				avgDuration := float64(duration) / float64(counts[endpoint])
				fmt.Printf("  %s: %.2f ms\n", endpoint, avgDuration)
			}
		}
	}

	fmt.Println("\nStatus Codes:")
	if statusCodes, ok := metrics["status_codes"].(map[int]int64); ok {
		for code, count := range statusCodes {
			fmt.Printf("  %d: %d requests\n", code, count)
		}
	}

	fmt.Println(strings.Repeat("=", 50))
}
