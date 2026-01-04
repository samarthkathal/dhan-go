// Package main demonstrates basic Full Market Depth usage with the Dhan Go SDK.
//
// This example shows:
// - Creating a Full Depth client with 20-level depth
// - Subscribing to instruments
// - Receiving depth updates via callbacks
//
// Prerequisites:
// - Set DHAN_ACCESS_TOKEN and DHAN_CLIENT_ID environment variables
//
// Run:
//
//	export DHAN_ACCESS_TOKEN="your-access-token-here"
//	export DHAN_CLIENT_ID="your-client-id"
//	go run main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/samarthkathal/dhan-go/fulldepth"
)

func main() {
	// Get credentials from environment
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("DHAN_ACCESS_TOKEN environment variable not set")
	}

	clientID := os.Getenv("DHAN_CLIENT_ID")
	if clientID == "" {
		log.Fatal("DHAN_CLIENT_ID environment variable not set")
	}

	ctx := context.Background()

	// Create Full Depth client with 20-level depth
	client, err := fulldepth.NewClient(
		accessToken,
		clientID,
		fulldepth.WithDepthLevel(fulldepth.Depth20),
		fulldepth.WithDepthCallback(handleDepth),
		fulldepth.WithErrorCallback(handleError),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Connect to WebSocket
	fmt.Println("Connecting to Full Depth WebSocket...")
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	fmt.Println("Connected!")

	// Wait for connection to stabilize
	time.Sleep(1 * time.Second)

	// Subscribe to instruments
	instruments := []fulldepth.Instrument{
		{ExchangeSegment: fulldepth.ExchangeNSEEQ, SecurityID: 11536}, // TCS
		{ExchangeSegment: fulldepth.ExchangeNSEEQ, SecurityID: 1333},  // HDFC Bank
	}

	fmt.Printf("Subscribing to %d instruments...\n", len(instruments))
	if err := client.Subscribe(ctx, instruments); err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	fmt.Println("Subscribed!")

	// Print stats
	stats := client.GetStats()
	fmt.Printf("Stats: Connected=%v, DepthLevel=%d, Instruments=%d\n",
		stats.Connected, stats.DepthLevel, stats.InstrumentCount)
	fmt.Println()
	fmt.Println("Waiting for depth updates (press Ctrl+C to exit)...")
	fmt.Println()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Disconnect
	fmt.Println("\nDisconnecting...")
	if err := client.Disconnect(); err != nil {
		log.Printf("Error disconnecting: %v", err)
	}
	fmt.Println("Disconnected")
}

func handleDepth(data *fulldepth.FullDepthData) {
	fmt.Printf("\n=== Depth Update: %s / %d ===\n",
		data.GetExchangeName(), data.SecurityID)

	// Best bid/ask
	bidPrice, bidQty := data.GetBestBid()
	askPrice, askQty := data.GetBestAsk()
	fmt.Printf("Best Bid: %.2f x %d | Best Ask: %.2f x %d | Spread: %.2f\n",
		bidPrice, bidQty, askPrice, askQty, data.GetSpread())

	// Show top 5 levels
	fmt.Println("Top 5 Bids:")
	for i := 0; i < 5 && i < len(data.Bids); i++ {
		b := data.Bids[i]
		fmt.Printf("  %d. %.2f x %d (%d orders)\n", i+1, b.Price, b.Quantity, b.Orders)
	}

	fmt.Println("Top 5 Asks:")
	for i := 0; i < 5 && i < len(data.Asks); i++ {
		a := data.Asks[i]
		fmt.Printf("  %d. %.2f x %d (%d orders)\n", i+1, a.Price, a.Quantity, a.Orders)
	}

	fmt.Printf("Total Bid Qty: %d | Total Ask Qty: %d\n",
		data.GetTotalBidQuantity(), data.GetTotalAskQuantity())
}

func handleError(err error) {
	log.Printf("Error: %v", err)
}
