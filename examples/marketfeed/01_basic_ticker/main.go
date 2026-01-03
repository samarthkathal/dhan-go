// Package main demonstrates basic MarketFeed usage for ticker data.
//
// This example shows:
// - Creating a single-connection MarketFeed client
// - Subscribing to instruments for ticker data
// - Receiving real-time LTP updates via callbacks
// - Proper field access (Header.SecurityID, LastTradedPrice, etc.)
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

	fmt.Println("MarketFeed Basic Ticker Example")
	fmt.Println()

	// Create MarketFeed client with ticker callback
	client, err := marketfeed.NewClient(
		accessToken,
		marketfeed.WithTickerCallback(func(data *marketfeed.TickerData) {
			// Correct field access:
			// - data.Header.SecurityID for security ID
			// - data.LastTradedPrice for LTP
			// - data.GetTradeTime() for trade time
			// - data.GetExchangeName() for exchange name
			fmt.Printf("TICKER | Security: %d | Exchange: %s | LTP: %.2f | Time: %v\n",
				data.Header.SecurityID,
				data.GetExchangeName(),
				data.LastTradedPrice,
				data.GetTradeTime().Format("15:04:05"))
		}),
		marketfeed.WithErrorCallback(func(err error) {
			log.Printf("Error: %v", err)
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create MarketFeed client: %v", err)
	}

	fmt.Println("MarketFeed client created (single connection)")
	fmt.Println()

	// Connect to WebSocket
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Connecting to MarketFeed WebSocket...")
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	fmt.Println("Connected successfully")
	fmt.Println()

	// Subscribe to instruments
	// Example: TCS (1333), Infosys (1594) on NSE
	instruments := []marketfeed.Instrument{
		{
			SecurityID:      "1333", // TCS
			ExchangeSegment: marketfeed.ExchangeNSEEQ,
		},
		{
			SecurityID:      "1594", // Infosys
			ExchangeSegment: marketfeed.ExchangeNSEEQ,
		},
	}

	fmt.Println("Subscribing to instruments:")
	for _, inst := range instruments {
		fmt.Printf("  - %s:%s\n", inst.ExchangeSegment, inst.SecurityID)
	}

	if err := client.Subscribe(context.Background(), instruments); err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	fmt.Println("Subscribed successfully")
	fmt.Println()
	fmt.Println("Receiving market data... (Press Ctrl+C to stop)")
	fmt.Println()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println()
	fmt.Println("Shutting down...")

	// Graceful disconnect
	if err := client.Disconnect(); err != nil {
		log.Printf("Error during disconnect: %v", err)
	}

	fmt.Println("Disconnected successfully")
}
