// Package main demonstrates REST client with rate limiting.
//
// This example shows:
// - Using Dhan's default rate limiter
// - Understanding rate limit categories
// - Getting rate limiter statistics
//
// Dhan Rate Limits:
// - Order APIs: 25/sec, 250/min, 1000/hour, 7000/day
// - Data APIs: 5/sec, 100k/day
// - Quote APIs: 1/sec
// - Non-Trading APIs: 20/sec
//
// Prerequisites:
// - Set DHAN_ACCESS_TOKEN environment variable
//
// Run:
//
//	export DHAN_ACCESS_TOKEN="your-access-token-here"
//	go run main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/samarthkathal/dhan-go/rest"
)

func main() {
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("DHAN_ACCESS_TOKEN environment variable not set")
	}

	ctx := context.Background()

	// Create REST client with default rate limiter
	client, err := rest.NewClient(
		"https://api.dhan.co",
		accessToken,
		nil,
		rest.WithDefaultRateLimiter(), // Enable Dhan's default rate limits
	)
	if err != nil {
		log.Fatalf("Failed to create REST client: %v", err)
	}

	fmt.Println("REST client created with rate limiting enabled")
	fmt.Println()

	// Display rate limit information
	fmt.Println("Dhan API Rate Limits:")
	fmt.Println("  Order APIs:")
	fmt.Println("    - 25 requests/second")
	fmt.Println("    - 250 requests/minute")
	fmt.Println("    - 1,000 requests/hour")
	fmt.Println("    - 7,000 requests/day")
	fmt.Println()
	fmt.Println("  Data APIs:")
	fmt.Println("    - 5 requests/second")
	fmt.Println("    - 100,000 requests/day")
	fmt.Println()
	fmt.Println("  Quote APIs:")
	fmt.Println("    - 1 request/second")
	fmt.Println()
	fmt.Println("  Non-Trading APIs:")
	fmt.Println("    - 20 requests/second")
	fmt.Println()

	// Make multiple requests to test rate limiting
	fmt.Println("Making 10 rapid requests to test rate limiting...")
	fmt.Println("(Rate limiter will automatically throttle if needed)")
	fmt.Println()

	for i := 1; i <= 10; i++ {
		start := time.Now()

		_, err := client.GetHoldings(ctx)
		if err != nil {
			log.Printf("Request %d error: %v", i, err)
			continue
		}

		duration := time.Since(start)
		fmt.Printf("Request %d completed in %v\n", i, duration)
	}
	fmt.Println()

	// Get rate limiter statistics
	fmt.Println("Rate Limiter Statistics:")
	stats := client.GetRateLimiterStats()
	if stats != nil {
		if totalReqs, ok := stats["total_requests"]; ok {
			fmt.Printf("  Total requests: %v\n", totalReqs)
		}
		if totalWait, ok := stats["total_wait_time"]; ok {
			fmt.Printf("  Total wait time: %v\n", totalWait)
		}
	} else {
		fmt.Println("  No statistics available")
	}
	fmt.Println()

	// Access the underlying rate limiter for advanced usage
	rateLimiter := client.GetRateLimiter()
	if rateLimiter != nil {
		fmt.Println("Rate limiter is active")
	}

	fmt.Println()
	fmt.Println("Benefits of rate limiting:")
	fmt.Println("  - Prevents API throttling and blocks")
	fmt.Println("  - Automatic request queuing")
	fmt.Println("  - No manual delays needed")
	fmt.Println("  - Compliant with Dhan API limits")
}
