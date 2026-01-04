// Package main demonstrates 200-level Full Market Depth usage.
//
// This example shows:
// - Creating a Full Depth client with 200-level depth
// - Subscribing to a single instrument (200-depth supports only one at a time)
// - Analyzing deep order book data
//
// Prerequisites:
// - Set DHAN_ACCESS_TOKEN and DHAN_CLIENT_ID environment variables
// - Data API subscription required for 200-depth access
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

	// Create Full Depth client with 200-level depth
	// Note: 200-depth requires Data API subscription
	client, err := fulldepth.NewClient(
		accessToken,
		clientID,
		fulldepth.WithDepthLevel(fulldepth.Depth200),
		fulldepth.WithDepthCallback(handleDepth),
		fulldepth.WithErrorCallback(handleError),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Connect to WebSocket
	fmt.Println("Connecting to 200-Depth WebSocket...")
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	fmt.Println("Connected!")

	// Wait for connection to stabilize
	time.Sleep(1 * time.Second)

	// Subscribe to a single instrument (200-depth supports only one at a time)
	instruments := []fulldepth.Instrument{
		{ExchangeSegment: fulldepth.ExchangeNSEEQ, SecurityID: 11536}, // TCS
	}

	fmt.Println("Subscribing to TCS (200-depth)...")
	if err := client.Subscribe(ctx, instruments); err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	fmt.Println("Subscribed!")

	// Print stats
	stats := client.GetStats()
	fmt.Printf("Stats: Connected=%v, DepthLevel=%d, URL=%s\n",
		stats.Connected, stats.DepthLevel, stats.URL)
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
	fmt.Printf("\n=== 200-Depth Update: %s / %d ===\n",
		data.GetExchangeName(), data.SecurityID)

	// Best bid/ask
	bidPrice, bidQty := data.GetBestBid()
	askPrice, askQty := data.GetBestAsk()
	fmt.Printf("Best Bid: %.2f x %d | Best Ask: %.2f x %d | Spread: %.2f\n",
		bidPrice, bidQty, askPrice, askQty, data.GetSpread())

	// Depth summary
	fmt.Printf("Bid Levels: %d | Ask Levels: %d\n", len(data.Bids), len(data.Asks))
	fmt.Printf("Total Bid Qty: %d | Total Ask Qty: %d\n",
		data.GetTotalBidQuantity(), data.GetTotalAskQuantity())

	// Show top 10 levels for 200-depth
	fmt.Println("\nTop 10 Bids:")
	for i := 0; i < 10 && i < len(data.Bids); i++ {
		b := data.Bids[i]
		fmt.Printf("  %3d. %.2f x %d (%d orders)\n", i+1, b.Price, b.Quantity, b.Orders)
	}

	fmt.Println("\nTop 10 Asks:")
	for i := 0; i < 10 && i < len(data.Asks); i++ {
		a := data.Asks[i]
		fmt.Printf("  %3d. %.2f x %d (%d orders)\n", i+1, a.Price, a.Quantity, a.Orders)
	}

	// Calculate price range
	if len(data.Bids) > 0 && len(data.Asks) > 0 {
		highestBid := data.Bids[0].Price
		lowestBid := data.Bids[len(data.Bids)-1].Price
		lowestAsk := data.Asks[0].Price
		highestAsk := data.Asks[len(data.Asks)-1].Price

		fmt.Printf("\nPrice Range:\n")
		fmt.Printf("  Bid range: %.2f - %.2f (%.2f%% spread)\n",
			lowestBid, highestBid, (highestBid-lowestBid)/highestBid*100)
		fmt.Printf("  Ask range: %.2f - %.2f (%.2f%% spread)\n",
			lowestAsk, highestAsk, (highestAsk-lowestAsk)/lowestAsk*100)
	}
}

func handleError(err error) {
	log.Printf("Error: %v", err)
}
