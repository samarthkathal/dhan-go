// Package main demonstrates graceful shutdown for REST API client.
//
// This example shows:
// - Using context for cancellation
// - Handling OS signals (SIGINT, SIGTERM)
// - Completing in-flight requests before shutdown
// - Implementing shutdown timeout
//
// Prerequisites:
// - Set DHAN_ACCESS_TOKEN environment variable
//
// Run:
//
//	export DHAN_ACCESS_TOKEN="your-access-token-here"
//	go run main.go
//	(Press Ctrl+C to trigger graceful shutdown)
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
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

	fmt.Println("REST client created")
	fmt.Println("Application is running... (Press Ctrl+C to shutdown gracefully)")
	fmt.Println()

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// WaitGroup to track in-flight requests
	var wg sync.WaitGroup

	// Start a background worker that makes periodic API calls
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		requestNum := 0
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Background worker received shutdown signal")
				return
			case <-ticker.C:
				requestNum++
				fmt.Printf("Making periodic request #%d...\n", requestNum)

				// Use context for the request so it can be cancelled
				reqCtx, reqCancel := context.WithTimeout(ctx, 5*time.Second)
				holdings, err := client.GetHoldings(reqCtx)
				reqCancel()

				if err != nil {
					if ctx.Err() != nil {
						// Parent context was cancelled, this is expected during shutdown
						return
					}
					log.Printf("Request error: %v", err)
				} else if holdings.JSON200 != nil {
					fmt.Printf("  Request #%d completed: %d holdings\n", requestNum, len(*holdings.JSON200))
				}
			}
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	fmt.Println()
	fmt.Printf("Received signal: %v\n", sig)
	fmt.Println("Initiating graceful shutdown...")
	fmt.Println()

	// Cancel context to signal all goroutines to stop
	cancel()

	// Wait for in-flight requests to complete with timeout
	shutdownComplete := make(chan struct{})
	go func() {
		wg.Wait()
		close(shutdownComplete)
	}()

	shutdownTimeout := 10 * time.Second
	fmt.Printf("Waiting up to %v for in-flight requests...\n", shutdownTimeout)

	select {
	case <-shutdownComplete:
		fmt.Println("All in-flight requests completed")
	case <-time.After(shutdownTimeout):
		fmt.Println("Shutdown timeout exceeded, forcing shutdown")
	}

	// Cleanup phase
	fmt.Println()
	fmt.Println("Cleaning up resources...")
	time.Sleep(100 * time.Millisecond) // Simulate cleanup

	fmt.Println()
	fmt.Println("Graceful shutdown complete")
	fmt.Println()
	fmt.Println("Shutdown Best Practices:")
	fmt.Println("  1. Use context.WithCancel for propagating cancellation")
	fmt.Println("  2. Handle SIGINT and SIGTERM signals")
	fmt.Println("  3. Use sync.WaitGroup to track in-flight operations")
	fmt.Println("  4. Set a reasonable shutdown timeout")
	fmt.Println("  5. Clean up resources before exit")
}
