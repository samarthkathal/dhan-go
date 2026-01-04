// Package main demonstrates data retention with Full Market Depth.
//
// This example shows:
// - How to safely retain depth data beyond callback scope using Copy()
// - Tracking order book snapshots over time
// - Comparing depth changes between updates
//
// IMPORTANT: Callback data pointers are only valid during callback execution.
// The SDK uses object pooling internally, so data may be reused after callback returns.
// For fulldepth structs (which contain slices), you MUST use Copy() method:
//
//	myDepth := data.Copy()  // Deep copy - required for slice data
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
	"sync"
	"syscall"
	"time"

	"github.com/samarthkathal/dhan-go/fulldepth"
)

// OrderBookTracker tracks depth snapshots and calculates changes
type OrderBookTracker struct {
	mu sync.RWMutex

	// Previous and current snapshots (retained from callbacks)
	previous map[int32]fulldepth.FullDepthData
	current  map[int32]fulldepth.FullDepthData

	// Update counter
	updateCount int64
}

func NewOrderBookTracker() *OrderBookTracker {
	return &OrderBookTracker{
		previous: make(map[int32]fulldepth.FullDepthData),
		current:  make(map[int32]fulldepth.FullDepthData),
	}
}

// RecordDepth stores depth data from callback
// IMPORTANT: We use Copy() since fulldepth structs contain slices
func (t *OrderBookTracker) RecordDepth(data *fulldepth.FullDepthData) {
	t.mu.Lock()
	defer t.mu.Unlock()

	secID := data.SecurityID

	// Move current to previous
	if curr, exists := t.current[secID]; exists {
		t.previous[secID] = curr
	}

	// IMPORTANT: Use Copy() for fulldepth structs!
	// Shallow copy (*data) would share the underlying slice memory,
	// which gets reused by the pool after callback returns.
	t.current[secID] = data.Copy()
	t.updateCount++
}

// GetCurrentDepth returns the latest depth snapshot
func (t *OrderBookTracker) GetCurrentDepth(securityID int32) (fulldepth.FullDepthData, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	data, ok := t.current[securityID]
	return data, ok
}

// GetPreviousDepth returns the previous depth snapshot
func (t *OrderBookTracker) GetPreviousDepth(securityID int32) (fulldepth.FullDepthData, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	data, ok := t.previous[securityID]
	return data, ok
}

// PrintChanges prints changes between previous and current depth
func (t *OrderBookTracker) PrintChanges(securityID int32) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	curr, hasCurr := t.current[securityID]
	prev, hasPrev := t.previous[securityID]

	if !hasCurr {
		fmt.Printf("No data for security %d\n", securityID)
		return
	}

	fmt.Printf("\n=== Depth Update #%d for Security %d ===\n", t.updateCount, securityID)

	// Current best bid/ask
	bidPrice, bidQty := curr.GetBestBid()
	askPrice, askQty := curr.GetBestAsk()
	fmt.Printf("Best Bid: %.2f x %d | Best Ask: %.2f x %d | Spread: %.2f\n",
		bidPrice, bidQty, askPrice, askQty, curr.GetSpread())

	// Compare with previous if available
	if hasPrev {
		prevBidPrice, _ := prev.GetBestBid()
		prevAskPrice, _ := prev.GetBestAsk()

		bidChange := bidPrice - prevBidPrice
		askChange := askPrice - prevAskPrice
		spreadChange := curr.GetSpread() - prev.GetSpread()

		fmt.Printf("Changes: Bid %+.2f | Ask %+.2f | Spread %+.2f\n",
			bidChange, askChange, spreadChange)

		// Volume changes
		currBidVol := curr.GetTotalBidQuantity()
		currAskVol := curr.GetTotalAskQuantity()
		prevBidVol := prev.GetTotalBidQuantity()
		prevAskVol := prev.GetTotalAskQuantity()

		fmt.Printf("Volume: Bid %d (%+d) | Ask %d (%+d)\n",
			currBidVol, currBidVol-prevBidVol,
			currAskVol, currAskVol-prevAskVol)
	}
}

func main() {
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("DHAN_ACCESS_TOKEN environment variable not set")
	}

	clientID := os.Getenv("DHAN_CLIENT_ID")
	if clientID == "" {
		log.Fatal("DHAN_CLIENT_ID environment variable not set")
	}

	fmt.Println("Full Depth Data Retention Example")
	fmt.Println()
	fmt.Println("This example demonstrates:")
	fmt.Println("  - Safe data retention from callbacks using Copy()")
	fmt.Println("  - Tracking order book changes between updates")
	fmt.Println()
	fmt.Println("IMPORTANT: For fulldepth structs, you MUST use Copy():")
	fmt.Println("  myDepth := data.Copy()  // Deep copy for slice data")
	fmt.Println()

	// Create order book tracker
	tracker := NewOrderBookTracker()

	// Security ID we're tracking (for change display)
	var trackSecurityID int32 = 11536 // TCS

	// Create Full Depth client
	client, err := fulldepth.NewClient(
		accessToken,
		clientID,
		fulldepth.WithDepthLevel(fulldepth.Depth20),

		// Depth callback - use Copy() for retention
		fulldepth.WithDepthCallback(func(data *fulldepth.FullDepthData) {
			// IMPORTANT: data pointer is only valid during this callback!
			// We must use Copy() because FullDepthData contains slices (Bids, Asks).
			// A shallow copy would share the underlying arrays, which get reused
			// by the pool after this callback returns.
			tracker.RecordDepth(data)

			// Print changes for our tracked security
			if data.SecurityID == trackSecurityID {
				tracker.PrintChanges(trackSecurityID)
			}
		}),

		fulldepth.WithErrorCallback(func(err error) {
			log.Printf("Error: %v", err)
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Connect
	ctx := context.Background()
	fmt.Println("Connecting...")
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	fmt.Println("Connected")
	fmt.Println()

	// Wait for connection to stabilize
	time.Sleep(1 * time.Second)

	// Subscribe to instruments
	instruments := []fulldepth.Instrument{
		{ExchangeSegment: fulldepth.ExchangeNSEEQ, SecurityID: int(trackSecurityID)},
	}

	fmt.Printf("Subscribing to security %d...\n", trackSecurityID)
	if err := client.Subscribe(ctx, instruments); err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	fmt.Println("Subscribed")
	fmt.Println()
	fmt.Println("Tracking depth changes... (Press Ctrl+C to stop)")

	// Wait for interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Print final snapshot
	fmt.Println("\n=== Final Snapshot ===")
	if depth, ok := tracker.GetCurrentDepth(trackSecurityID); ok {
		fmt.Printf("Security %d:\n", depth.SecurityID)
		fmt.Println("Top 5 Bids:")
		for i := 0; i < 5 && i < len(depth.Bids); i++ {
			b := depth.Bids[i]
			fmt.Printf("  %.2f x %d (%d orders)\n", b.Price, b.Quantity, b.Orders)
		}
		fmt.Println("Top 5 Asks:")
		for i := 0; i < 5 && i < len(depth.Asks); i++ {
			a := depth.Asks[i]
			fmt.Printf("  %.2f x %d (%d orders)\n", a.Price, a.Quantity, a.Orders)
		}
	}

	fmt.Println("\nDisconnecting...")
	if err := client.Disconnect(); err != nil {
		log.Printf("Error: %v", err)
	}
	fmt.Println("Done")
}
