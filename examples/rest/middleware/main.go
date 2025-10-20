package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/samarthkathal/dhan-go/rest/client"
	"github.com/samarthkathal/dhan-go/utils"
)

func main() {
	// 1. Create metrics collector
	metricsCollector := utils.NewMetricsCollector()

	// 2. Create HTTP client with high-throughput configuration
	httpClient := utils.HighThroughputHTTPClient()

	// 3. Add middleware to the HTTP client's transport
	httpClient = utils.WithMiddleware(
		httpClient,
		utils.RecoveryRoundTripper(log.Default()),        // Panic recovery (innermost)
		utils.RateLimitRoundTripper(100, 10),             // Rate limiting: 100 req/sec, burst 10
		utils.MetricsRoundTripper(metricsCollector),      // Metrics collection
		utils.LoggingRoundTripper(log.Default()),         // Request/response logging (outermost)
	)

	// 4. Create authentication middleware
	accessToken := "your-access-token"
	authMiddleware := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("access-token", accessToken)
		return nil
	}

	// 5. Create Dhan API client
	dhanClient, err := client.NewClientWithResponses(
		"https://api.dhan.co",
		client.WithHTTPClient(httpClient),
		client.WithRequestEditorFn(authMiddleware),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// 6. Make API calls
	fmt.Println("Fetching holdings...")
	holdings, err := dhanClient.GetholdingsWithResponse(ctx, nil)
	if err != nil {
		log.Fatalf("Error fetching holdings: %v", err)
	}

	if holdings.StatusCode() == 200 && holdings.JSON200 != nil {
		fmt.Printf("Holdings: %d holdings\n", len(*holdings.JSON200))
	}

	fmt.Println("\nFetching positions...")
	positions, err := dhanClient.GetpositionsWithResponse(ctx, nil)
	if err != nil {
		log.Fatalf("Error fetching positions: %v", err)
	}

	if positions.StatusCode() == 200 && positions.JSON200 != nil {
		fmt.Printf("Positions: %d positions\n", len(*positions.JSON200))
	}

	fmt.Println("\nFetching fund limits...")
	funds, err := dhanClient.FundlimitWithResponse(ctx, nil)
	if err != nil {
		log.Fatalf("Error fetching funds: %v", err)
	}

	if funds.StatusCode() == 200 && funds.JSON200 != nil {
		fmt.Println("Fund limits fetched successfully")
	}

	// 7. Get historical data
	fmt.Println("\nFetching historical data...")
	securityID := "1333"
	exchangeSegment := client.HistoricalChartsRequestExchangeSegmentNSEEQ
	instrument := client.HistoricalChartsRequestInstrumentEQUITY

	historicalReq := client.HistoricalchartsJSONRequestBody{
		SecurityId:      &securityID,
		ExchangeSegment: &exchangeSegment,
		Instrument:      &instrument,
	}

	historical, err := dhanClient.HistoricalchartsWithResponse(ctx, nil, historicalReq)
	if err != nil {
		log.Fatalf("Error fetching historical data: %v", err)
	}

	if historical.StatusCode() == 200 && historical.JSON200 != nil {
		fmt.Printf("Historical data: %d candles\n", len(*historical.JSON200.Close))
	}

	// 8. Print collected metrics
	fmt.Println("\n=== Collected Metrics ===")
	metrics := metricsCollector.GetMetrics()

	fmt.Printf("Total Requests: %v\n", metrics["total_requests"])
	fmt.Printf("Total Errors: %v\n", metrics["total_errors"])

	fmt.Println("\nRequest Counts by Endpoint:")
	if counts, ok := metrics["request_counts"].(map[string]int64); ok {
		for endpoint, count := range counts {
			fmt.Printf("  %s: %d\n", endpoint, count)
		}
	}

	fmt.Println("\nRequest Durations (ms) by Endpoint:")
	if durations, ok := metrics["request_durations_ms"].(map[string]int64); ok {
		for endpoint, duration := range durations {
			if counts, ok := metrics["request_counts"].(map[string]int64); ok {
				avgDuration := float64(duration) / float64(counts[endpoint])
				fmt.Printf("  %s: %.2f ms (avg)\n", endpoint, avgDuration)
			}
		}
	}

	fmt.Println("\nStatus Codes:")
	if statusCodes, ok := metrics["status_codes"].(map[int]int64); ok {
		for code, count := range statusCodes {
			fmt.Printf("  %d: %d\n", code, count)
		}
	}

	fmt.Println("\n=== All operations completed successfully! ===")
}
