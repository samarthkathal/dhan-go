package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/samarthkathal/dhan-go/client"
)

func main() {
	// 1. Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// 3. Start goroutine to handle shutdown signals
	go func() {
		sig := <-sigCh
		fmt.Printf("\nReceived signal: %v\n", sig)
		fmt.Println("Initiating graceful shutdown...")

		// Cancel the context - this will stop all ongoing API calls
		cancel()

		fmt.Println("All contexts cancelled. Waiting for in-flight requests to complete...")
	}()

	// 4. Create HTTP client
	httpClient := &http.Client{
		Timeout: 30 * time.Second, // Overall request timeout
	}

	// 5. Create authentication middleware
	accessToken := "your-access-token"
	authMiddleware := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("access-token", accessToken)
		return nil
	}

	// 6. Create Dhan API client
	dhanClient, err := client.NewClientWithResponses(
		"https://api.dhan.co",
		client.WithHTTPClient(httpClient),
		client.WithRequestEditorFn(authMiddleware),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Starting application... (Press Ctrl+C for graceful shutdown)")

	// 7. Simulate long-running operations
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Context was cancelled (shutdown signal received)
			fmt.Println("Context cancelled. Shutting down...")
			return

		case <-ticker.C:
			// Make periodic API calls
			fmt.Println("\nFetching positions...")

			// Use a timeout context for each individual request
			requestCtx, requestCancel := context.WithTimeout(ctx, 10*time.Second)

			positions, err := dhanClient.GetpositionsWithResponse(requestCtx, nil)

			if err != nil {
				if ctx.Err() == context.Canceled {
					fmt.Println("Request cancelled due to shutdown")
					requestCancel()
					return
				}
				fmt.Printf("Error fetching positions: %v\n", err)
			} else if positions.StatusCode() == 200 && positions.JSON200 != nil {
				fmt.Printf("Positions: %d positions\n", len(*positions.JSON200))
			} else {
				fmt.Printf("API returned status: %d\n", positions.StatusCode())
			}

			requestCancel()
		}
	}
}

/*
Key concepts demonstrated in this example:

1. Context Cancellation:
   - Create a cancellable context using context.WithCancel
   - Cancel the context when shutdown signal is received
   - All API calls will be cancelled when parent context is cancelled

2. Signal Handling:
   - Listen for SIGINT (Ctrl+C) and SIGTERM
   - Trigger graceful shutdown when signal is received

3. Timeouts:
   - Set overall timeout on HTTP client
   - Use context.WithTimeout for individual requests
   - Each request respects both its timeout and parent context cancellation

4. Graceful Shutdown Flow:
   - Signal received → Cancel context → In-flight requests respect cancellation
   - No need for custom tracking of in-flight requests
   - Context cancellation propagates through all API calls

5. Best Practices:
   - Always use context for API calls
   - Set reasonable timeouts
   - Handle context.Canceled errors appropriately
   - Clean up resources (defer cancel, ticker.Stop)
*/
