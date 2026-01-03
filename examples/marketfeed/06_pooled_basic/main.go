// Package main demonstrates basic PooledClient usage for MarketFeed.
//
// This example shows:
// - Creating a PooledClient for connection pooling
// - Automatic connection management (up to 5 connections)
// - Instrument distribution across connections
// - Pool statistics
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
	"os/signal"
	"syscall"
	"time"

	"github.com/samarthkathal/dhan-go/marketfeed"
)

func main() {
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("DHAN_ACCESS_TOKEN environment variable not set")
	}

	fmt.Println("MarketFeed PooledClient Basic Example")
	fmt.Println()

	fmt.Println("PooledClient Features:")
	fmt.Println("  - Up to 5 concurrent WebSocket connections")
	fmt.Println("  - Max 5000 instruments per connection")
	fmt.Println("  - Max 100 instruments per subscription batch")
	fmt.Println("  - Automatic instrument distribution")
	fmt.Println("  - Connection health monitoring")
	fmt.Println()

	// Create pooled client
	client, err := marketfeed.NewPooledClient(
		accessToken,
		marketfeed.WithPooledTickerCallback(func(data *marketfeed.TickerData) {
			fmt.Printf("TICKER | ID: %d | Exchange: %s | LTP: %.2f\n",
				data.Header.SecurityID,
				data.GetExchangeName(),
				data.LastTradedPrice)
		}),
		marketfeed.WithPooledQuoteCallback(func(data *marketfeed.QuoteData) {
			fmt.Printf("QUOTE  | ID: %d | OHLC: %.2f/%.2f/%.2f/%.2f | Vol: %d\n",
				data.Header.SecurityID,
				data.DayOpen,
				data.DayHigh,
				data.DayLow,
				data.DayClose,
				data.Volume)
		}),
		marketfeed.WithPooledErrorCallback(func(err error) {
			log.Printf("ERROR | %v", err)
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create pooled client: %v", err)
	}

	fmt.Println("PooledClient created")
	fmt.Println()

	// Connect
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Connecting...")
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	fmt.Println("Connected")
	fmt.Println()

	// Subscribe to multiple instruments
	instruments := []marketfeed.Instrument{
		{SecurityID: "1333", ExchangeSegment: marketfeed.ExchangeNSEEQ},  // TCS
		{SecurityID: "1594", ExchangeSegment: marketfeed.ExchangeNSEEQ},  // Infosys
		{SecurityID: "11536", ExchangeSegment: marketfeed.ExchangeNSEEQ}, // Reliance
		{SecurityID: "2885", ExchangeSegment: marketfeed.ExchangeNSEEQ},  // HDFC Bank
	}

	fmt.Println("Subscribing to instruments:")
	for _, inst := range instruments {
		fmt.Printf("  - %s:%s\n", inst.ExchangeSegment, inst.SecurityID)
	}

	if err := client.Subscribe(context.Background(), instruments); err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	fmt.Println("Subscribed")
	fmt.Println()

	// Print pool stats periodically
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stats := client.GetStats()
			fmt.Println("=== POOL STATS ===")
			fmt.Printf("Total Connections:  %d\n", stats.TotalConnections)
			fmt.Printf("Active Connections: %d\n", stats.ActiveConnections)
			fmt.Printf("Total Instruments:  %d\n", stats.TotalInstruments)
			fmt.Println("==================")
			fmt.Println()
		}
	}()

	fmt.Println("Receiving data... (Press Ctrl+C to stop)")
	fmt.Println("Pool stats will be printed every 10 seconds")
	fmt.Println()

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println()
	fmt.Println("Shutting down...")

	// Print final stats
	finalStats := client.GetStats()
	fmt.Println()
	fmt.Println("=== FINAL POOL STATS ===")
	fmt.Printf("Total Connections:  %d\n", finalStats.TotalConnections)
	fmt.Printf("Active Connections: %d\n", finalStats.ActiveConnections)
	fmt.Printf("Total Instruments:  %d\n", finalStats.TotalInstruments)
	fmt.Println("========================")

	if err := client.Disconnect(); err != nil {
		log.Printf("Error: %v", err)
	}

	fmt.Println()
	fmt.Println("Done")
}
