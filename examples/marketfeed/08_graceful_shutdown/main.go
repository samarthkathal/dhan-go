// Package main demonstrates graceful shutdown for MarketFeed client.
//
// This example shows:
// - Proper signal handling (SIGINT, SIGTERM)
// - Unsubscribing before disconnect
// - Clean connection closure
// - Context cancellation patterns
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

	"github.com/samarthkathal/dhan-go/marketfeed"
)

func main() {
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("DHAN_ACCESS_TOKEN environment variable not set")
	}

	fmt.Println("MarketFeed Graceful Shutdown Example")
	fmt.Println()

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// WaitGroup to track message processing
	var wg sync.WaitGroup

	// Track subscribed instruments for cleanup
	var subscribedInstruments []marketfeed.Instrument
	var instrumentsMu sync.Mutex

	// Create client
	client, err := marketfeed.NewClient(
		accessToken,
		marketfeed.WithTickerCallback(func(data *marketfeed.TickerData) {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Check if shutdown in progress
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Process the data
				fmt.Printf("TICKER | ID: %d | LTP: %.2f | Time: %v\n",
					data.Header.SecurityID,
					data.LastTradedPrice,
					data.GetTradeTime().Format("15:04:05"))
			}()
		}),
		marketfeed.WithErrorCallback(func(err error) {
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

	// Subscribe
	subscribedInstruments = []marketfeed.Instrument{
		{SecurityID: "1333", ExchangeSegment: marketfeed.ExchangeNSEEQ},
		{SecurityID: "1594", ExchangeSegment: marketfeed.ExchangeNSEEQ},
	}

	instrumentsMu.Lock()
	fmt.Println("Subscribing to instruments:")
	for _, inst := range subscribedInstruments {
		fmt.Printf("  - %s:%s\n", inst.ExchangeSegment, inst.SecurityID)
	}
	instrumentsMu.Unlock()

	if err := client.Subscribe(context.Background(), subscribedInstruments); err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	fmt.Println("Subscribed")
	fmt.Println()

	fmt.Println("Receiving data... (Press Ctrl+C for graceful shutdown)")
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

	// Step 1: Signal shutdown to all goroutines
	fmt.Println("Step 1: Signaling shutdown to all goroutines...")
	cancel()

	// Step 2: Wait for in-flight message processing with timeout
	fmt.Println("Step 2: Waiting for in-flight messages...")
	processingDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(processingDone)
	}()

	select {
	case <-processingDone:
		fmt.Println("         All messages processed")
	case <-time.After(5 * time.Second):
		fmt.Println("         Timeout waiting for messages (forcing shutdown)")
	}

	// Step 3: Unsubscribe from all instruments
	fmt.Println("Step 3: Unsubscribing from instruments...")
	instrumentsMu.Lock()
	instToUnsub := subscribedInstruments
	instrumentsMu.Unlock()

	unsubCtx, unsubCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := client.Unsubscribe(unsubCtx, instToUnsub); err != nil {
		fmt.Printf("         Warning: unsubscribe error: %v\n", err)
	} else {
		fmt.Println("         Unsubscribed from all instruments")
	}
	unsubCancel()

	// Step 4: Disconnect
	fmt.Println("Step 4: Disconnecting...")
	if err := client.Disconnect(); err != nil {
		fmt.Printf("         Warning: disconnect error: %v\n", err)
	} else {
		fmt.Println("         Disconnected successfully")
	}

	fmt.Println()
	fmt.Println("=== SHUTDOWN COMPLETE ===")
	fmt.Println()
	fmt.Println("Graceful Shutdown Best Practices:")
	fmt.Println("  1. Cancel context to signal all goroutines")
	fmt.Println("  2. Wait for in-flight message processing")
	fmt.Println("  3. Unsubscribe from instruments")
	fmt.Println("  4. Disconnect the client")
	fmt.Println("  5. Use timeouts to prevent hanging")
}
