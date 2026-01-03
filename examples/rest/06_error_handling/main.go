// Package main demonstrates error handling patterns for REST API.
//
// This example shows:
// - Handling context timeouts
// - Handling context cancellation
// - Checking HTTP status codes
// - Implementing retry with exponential backoff
// - Graceful error recovery
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
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/samarthkathal/dhan-go/rest"
)

func main() {
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("DHAN_ACCESS_TOKEN environment variable not set")
	}

	// Create REST client
	client, err := rest.NewClient(
		"https://api.dhan.co",
		accessToken,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to create REST client: %v", err)
	}

	fmt.Println("REST API Error Handling Examples")
	fmt.Println()

	// Example 1: Timeout handling
	fmt.Println("1. Timeout Handling:")
	ctx1, cancel1 := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel1()

	_, err = client.GetHoldings(ctx1)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Println("   Timeout detected (context.DeadlineExceeded)")
		} else {
			fmt.Printf("   Error: %v\n", err)
		}
	}
	fmt.Println()

	// Example 2: Context cancellation
	fmt.Println("2. Context Cancellation:")
	ctx2, cancel2 := context.WithCancel(context.Background())
	cancel2() // Cancel immediately

	_, err = client.GetOrders(ctx2)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Println("   Cancellation detected (context.Canceled)")
		} else {
			fmt.Printf("   Error: %v\n", err)
		}
	}
	fmt.Println()

	// Example 3: Checking HTTP status codes
	fmt.Println("3. HTTP Status Code Checking:")
	ctx3 := context.Background()
	holdings, err := client.GetHoldings(ctx3)
	if err != nil {
		fmt.Printf("   Request error: %v\n", err)
	} else {
		statusCode := holdings.StatusCode()
		switch statusCode {
		case http.StatusOK:
			fmt.Println("   Status 200 OK - Request successful")
		case http.StatusUnauthorized:
			fmt.Println("   Status 401 Unauthorized - Invalid access token")
		case http.StatusTooManyRequests:
			fmt.Println("   Status 429 Too Many Requests - Rate limited")
		case http.StatusInternalServerError:
			fmt.Println("   Status 500 Internal Server Error - Server issue")
		default:
			fmt.Printf("   Status %d - %s\n", statusCode, http.StatusText(statusCode))
		}
	}
	fmt.Println()

	// Example 4: Retry with exponential backoff
	fmt.Println("4. Retry with Exponential Backoff:")
	err = retryWithBackoff(func() error {
		ctx := context.Background()
		_, err := client.GetPositions(ctx)
		return err
	}, 3, 100*time.Millisecond)

	if err != nil {
		fmt.Printf("   Failed after retries: %v\n", err)
	} else {
		fmt.Println("   Request succeeded")
	}
	fmt.Println()

	// Example 5: Safe wrapper with error recovery
	fmt.Println("5. Safe Wrapper with Error Recovery:")
	data, err := safeGetHoldings(client)
	if err != nil {
		fmt.Printf("   Error handled gracefully: %v\n", err)
		fmt.Println("   Application continues running with default/cached data...")
	} else {
		fmt.Printf("   Successfully fetched %d holdings\n", len(data))
	}
	fmt.Println()

	fmt.Println("Error Handling Best Practices:")
	fmt.Println("  1. Always use context with timeouts")
	fmt.Println("  2. Implement retry logic for transient failures")
	fmt.Println("  3. Check HTTP status codes for specific handling")
	fmt.Println("  4. Log errors with context for debugging")
	fmt.Println("  5. Gracefully degrade when possible")
}

// retryWithBackoff implements exponential backoff retry logic
func retryWithBackoff(fn func() error, maxRetries int, baseDelay time.Duration) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		// Don't retry on context cancellation
		if errors.Is(err, context.Canceled) {
			return err
		}

		if i < maxRetries-1 {
			delay := baseDelay * time.Duration(1<<uint(i)) // Exponential: 100ms, 200ms, 400ms
			fmt.Printf("   Retry %d/%d after %v...\n", i+1, maxRetries, delay)
			time.Sleep(delay)
		}
	}
	return fmt.Errorf("failed after %d retries: %w", maxRetries, err)
}

// safeGetHoldings demonstrates graceful error handling with recovery
func safeGetHoldings(client *rest.Client) ([]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	holdings, err := client.GetHoldings(ctx)
	if err != nil {
		log.Printf("Failed to fetch holdings: %v", err)
		return nil, err
	}

	if holdings.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", holdings.StatusCode())
	}

	if holdings.JSON200 == nil {
		return nil, fmt.Errorf("empty response")
	}

	// Convert to generic slice for the example
	result := make([]interface{}, len(*holdings.JSON200))
	for i, h := range *holdings.JSON200 {
		result[i] = h
	}

	return result, nil
}
