// Package main demonstrates graceful shutdown for OrderUpdate client.
//
// This example shows:
// - Proper signal handling
// - Context cancellation
// - Clean connection closure
// - Waiting for in-flight processing
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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/samarthkathal/dhan-go/orderupdate"
)

func main() {
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("DHAN_ACCESS_TOKEN environment variable not set")
	}

	fmt.Println("OrderUpdate Graceful Shutdown Example")
	fmt.Println()

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// WaitGroup to track in-flight order processing
	var wg sync.WaitGroup

	// Counter for processed orders
	var orderCount uint64

	// Create client
	client, err := orderupdate.NewClient(
		accessToken,
		orderupdate.WithOrderUpdateCallback(func(alert *orderupdate.OrderAlert) {
			// Track this processing task
			wg.Add(1)
			defer wg.Done()

			// Check if shutdown is in progress
			select {
			case <-ctx.Done():
				fmt.Println("Skipping order processing - shutdown in progress")
				return
			default:
			}

			if !alert.IsOrderAlert() {
				return
			}

			// Increment counter
			atomic.AddUint64(&orderCount, 1)

			// Simulate some processing time
			processOrder(alert)
		}),
		orderupdate.WithErrorCallback(func(err error) {
			log.Printf("ERROR | %v", err)
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	fmt.Println("Client created")
	fmt.Println()

	// Connect with timeout
	connectCtx, connectCancel := context.WithTimeout(ctx, 30*time.Second)
	defer connectCancel()

	fmt.Println("Connecting...")
	if err := client.Connect(connectCtx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	fmt.Println("Connected")
	fmt.Println()

	fmt.Println("Listening for order updates... (Press Ctrl+C for graceful shutdown)")
	fmt.Println()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	sig := <-sigChan
	fmt.Println()
	fmt.Printf("Received signal: %v\n", sig)
	fmt.Println()
	fmt.Println("=== GRACEFUL SHUTDOWN SEQUENCE ===")
	fmt.Println()

	// Step 1: Signal shutdown to prevent new processing
	fmt.Println("Step 1: Signaling shutdown to handlers...")
	cancel()
	fmt.Println("         Done")

	// Step 2: Wait for in-flight order processing with timeout
	fmt.Println("Step 2: Waiting for in-flight order processing...")
	processingDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(processingDone)
	}()

	select {
	case <-processingDone:
		fmt.Println("         All order processing completed")
	case <-time.After(10 * time.Second):
		fmt.Println("         Timeout waiting for processing (forcing shutdown)")
	}

	// Step 3: Disconnect
	fmt.Println("Step 3: Disconnecting from WebSocket...")
	if err := client.Disconnect(); err != nil {
		fmt.Printf("         Warning: disconnect error: %v\n", err)
	} else {
		fmt.Println("         Disconnected successfully")
	}

	fmt.Println()
	fmt.Println("=== SHUTDOWN SUMMARY ===")
	fmt.Printf("Orders Processed: %d\n", atomic.LoadUint64(&orderCount))
	fmt.Println("========================")
	fmt.Println()
	fmt.Println("Graceful Shutdown Best Practices:")
	fmt.Println("  1. Cancel context to signal handlers")
	fmt.Println("  2. Wait for in-flight processing")
	fmt.Println("  3. Set reasonable timeout")
	fmt.Println("  4. Disconnect cleanly")
	fmt.Println("  5. Log final statistics")
}

// processOrder simulates processing an order update
func processOrder(alert *orderupdate.OrderAlert) {
	fmt.Printf("Processing Order | %s | %s | %s\n",
		alert.GetOrderID(),
		alert.GetStatus(),
		alert.Data.Symbol)

	// Determine action based on status
	switch {
	case alert.IsFilled():
		fmt.Printf("  -> Order filled at %.2f\n", alert.GetAvgTradedPrice())
		// In production: Update positions, log trade, etc.

	case alert.IsPartiallyFilled():
		fmt.Printf("  -> Partial fill: %d/%d\n",
			alert.GetTradedQuantity(),
			alert.Data.Quantity)
		// In production: Update pending quantity, etc.

	case alert.IsRejected():
		fmt.Printf("  -> Order rejected: %s\n", alert.Data.ReasonDescription)
		// In production: Alert user, log reason, etc.

	case alert.IsCancelled():
		fmt.Printf("  -> Order cancelled\n")
		// In production: Confirm cancellation, etc.

	default:
		fmt.Printf("  -> Status: %s\n", alert.GetStatus())
	}

	fmt.Println()
}
