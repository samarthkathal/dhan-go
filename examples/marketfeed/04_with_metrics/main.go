// Package main demonstrates data retention and metrics collection with MarketFeed.
//
// This example shows:
// - How to safely retain callback data beyond callback scope
// - Collecting metrics (update counts, latest prices) from market data
// - Using shallow copy for marketfeed data (all value types)
//
// IMPORTANT: Callback data pointers are only valid during callback execution.
// The SDK uses object pooling internally, so data may be reused after callback returns.
// To retain data: myTicker := *ticker (shallow copy works for marketfeed structs)
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
	"sync"
	"syscall"
	"time"

	"github.com/samarthkathal/dhan-go/marketfeed"
)

// Metrics tracks market data statistics
type Metrics struct {
	mu sync.RWMutex

	// Latest data (retained from callbacks)
	latestTickers map[int32]marketfeed.TickerData
	latestQuotes  map[int32]marketfeed.QuoteData

	// Counters
	tickerUpdates int64
	quoteUpdates  int64
}

func NewMetrics() *Metrics {
	return &Metrics{
		latestTickers: make(map[int32]marketfeed.TickerData),
		latestQuotes:  make(map[int32]marketfeed.QuoteData),
	}
}

// RecordTicker stores ticker data from callback
// IMPORTANT: We copy the data since the pointer is only valid during callback
func (m *Metrics) RecordTicker(data *marketfeed.TickerData) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Shallow copy is safe for marketfeed structs (all value types)
	m.latestTickers[data.Header.SecurityID] = *data
	m.tickerUpdates++
}

// RecordQuote stores quote data from callback
func (m *Metrics) RecordQuote(data *marketfeed.QuoteData) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Shallow copy is safe for marketfeed structs (all value types)
	m.latestQuotes[data.Header.SecurityID] = *data
	m.quoteUpdates++
}

// GetLatestTicker returns the last recorded ticker for a security
func (m *Metrics) GetLatestTicker(securityID int32) (marketfeed.TickerData, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.latestTickers[securityID]
	return data, ok
}

// GetLatestQuote returns the last recorded quote for a security
func (m *Metrics) GetLatestQuote(securityID int32) (marketfeed.QuoteData, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	data, ok := m.latestQuotes[securityID]
	return data, ok
}

// PrintSummary prints current metrics summary
func (m *Metrics) PrintSummary() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fmt.Println("\n=== Metrics Summary ===")
	fmt.Printf("Ticker updates: %d\n", m.tickerUpdates)
	fmt.Printf("Quote updates:  %d\n", m.quoteUpdates)

	fmt.Println("\nLatest Ticker Prices:")
	for secID, ticker := range m.latestTickers {
		fmt.Printf("  Security %d: LTP=%.2f at %v\n",
			secID, ticker.LastTradedPrice, ticker.GetTradeTime().Format("15:04:05"))
	}

	fmt.Println("\nLatest Quote Data:")
	for secID, quote := range m.latestQuotes {
		fmt.Printf("  Security %d: O=%.2f H=%.2f L=%.2f C=%.2f Vol=%d\n",
			secID, quote.DayOpen, quote.DayHigh, quote.DayLow, quote.DayClose, quote.Volume)
	}
}

func main() {
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("DHAN_ACCESS_TOKEN environment variable not set")
	}

	fmt.Println("MarketFeed Metrics Example")
	fmt.Println()
	fmt.Println("This example demonstrates:")
	fmt.Println("  - Safe data retention from callbacks using shallow copy")
	fmt.Println("  - Collecting metrics from real-time market data")
	fmt.Println()

	// Create metrics collector
	metrics := NewMetrics()

	// Create MarketFeed client
	client, err := marketfeed.NewClient(
		accessToken,

		// Ticker callback - copy data for retention
		marketfeed.WithTickerCallback(func(data *marketfeed.TickerData) {
			// IMPORTANT: data pointer is only valid during this callback!
			// We must copy it to use outside the callback.
			metrics.RecordTicker(data)

			fmt.Printf("TICKER | ID: %d | LTP: %.2f\n",
				data.Header.SecurityID, data.LastTradedPrice)
		}),

		// Quote callback - copy data for retention
		marketfeed.WithQuoteCallback(func(data *marketfeed.QuoteData) {
			// IMPORTANT: data pointer is only valid during this callback!
			metrics.RecordQuote(data)

			fmt.Printf("QUOTE  | ID: %d | LTP: %.2f | Vol: %d\n",
				data.Header.SecurityID, data.LastTradedPrice, data.Volume)
		}),

		marketfeed.WithErrorCallback(func(err error) {
			log.Printf("Error: %v", err)
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Connect
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Connecting...")
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	fmt.Println("Connected")
	fmt.Println()

	// Subscribe to instruments
	instruments := []marketfeed.Instrument{
		{SecurityID: "1333", ExchangeSegment: marketfeed.ExchangeNSEEQ},  // TCS
		{SecurityID: "11536", ExchangeSegment: marketfeed.ExchangeNSEEQ}, // Infosys
	}

	fmt.Println("Subscribing to instruments...")
	if err := client.Subscribe(context.Background(), instruments); err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	fmt.Println("Subscribed")
	fmt.Println()

	// Print metrics summary every 10 seconds
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			metrics.PrintSummary()
		}
	}()

	fmt.Println("Receiving data... (Press Ctrl+C to stop)")
	fmt.Println("Metrics summary will be printed every 10 seconds")
	fmt.Println()

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// Final summary
	metrics.PrintSummary()

	fmt.Println("\nShutting down...")
	if err := client.Disconnect(); err != nil {
		log.Printf("Error: %v", err)
	}
	fmt.Println("Done")
}
