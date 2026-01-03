// Package main demonstrates PooledClient with high-volume multi-instrument subscriptions.
//
// This example shows:
// - Subscribing to many instruments (100+)
// - Automatic load distribution across connections
// - Batch subscription handling
// - Connection pool scaling
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
	"sync/atomic"
	"syscall"
	"time"

	"github.com/samarthkathal/dhan-go/marketfeed"
)

func main() {
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("DHAN_ACCESS_TOKEN environment variable not set")
	}

	fmt.Println("MarketFeed PooledClient High Volume Example")
	fmt.Println()

	fmt.Println("Configuration Limits:")
	fmt.Println("  - Max 5 connections per client")
	fmt.Println("  - Max 5000 instruments per connection")
	fmt.Println("  - Max 100 instruments per batch message")
	fmt.Println("  - Total capacity: 25,000 instruments")
	fmt.Println()

	// Counter for messages received
	var messageCount uint64

	// Create pooled client
	client, err := marketfeed.NewPooledClient(
		accessToken,
		marketfeed.WithPooledTickerCallback(func(data *marketfeed.TickerData) {
			atomic.AddUint64(&messageCount, 1)
			// Only print every 100th message to avoid flooding console
			if atomic.LoadUint64(&messageCount)%100 == 0 {
				fmt.Printf("TICKER [%d msgs] | ID: %d | LTP: %.2f\n",
					atomic.LoadUint64(&messageCount),
					data.Header.SecurityID,
					data.LastTradedPrice)
			}
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

	// Generate a list of 50 instruments for demo
	// In production, you would have actual security IDs
	nifty50SecurityIDs := []string{
		"1333",  // TCS
		"1594",  // Infosys
		"11536", // Reliance
		"2885",  // HDFC Bank
		"1348",  // Tech Mahindra
		"17388", // Wipro
		"5258",  // HCL Tech
		"11723", // Kotak Bank
		"16675", // ICICI Bank
		"694",   // Axis Bank
		"21614", // Asian Paints
		"1922",  // Bharti Airtel
		"3351",  // Bajaj Finance
		"3350",  // Bajaj Finserv
		"5097",  // Nestle
		"11703", // Britannia
		"1023",  // Hindustan Unilever
		"6191",  // Maruti
		"2031",  // ITC
		"17963", // UltraTech Cement
	}

	// Create instruments list
	instruments := make([]marketfeed.Instrument, 0, len(nifty50SecurityIDs))
	for _, secID := range nifty50SecurityIDs {
		instruments = append(instruments, marketfeed.Instrument{
			SecurityID:      secID,
			ExchangeSegment: marketfeed.ExchangeNSEEQ,
		})
	}

	fmt.Printf("Subscribing to %d instruments...\n", len(instruments))
	fmt.Println("(Instruments will be automatically distributed across connections)")
	fmt.Println()

	subscribeStart := time.Now()
	if err := client.Subscribe(context.Background(), instruments); err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	fmt.Printf("Subscribed to %d instruments in %v\n", len(instruments), time.Since(subscribeStart))
	fmt.Println()

	// Print stats periodically
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		startTime := time.Now()
		for range ticker.C {
			elapsed := time.Since(startTime)
			count := atomic.LoadUint64(&messageCount)
			rate := float64(count) / elapsed.Seconds()

			stats := client.GetStats()
			fmt.Println("=== HIGH VOLUME STATS ===")
			fmt.Printf("Runtime:            %v\n", elapsed.Round(time.Second))
			fmt.Printf("Messages Received:  %d\n", count)
			fmt.Printf("Message Rate:       %.2f msg/sec\n", rate)
			fmt.Printf("Active Connections: %d\n", stats.ActiveConnections)
			fmt.Printf("Total Instruments:  %d\n", stats.TotalInstruments)
			fmt.Println("=========================")
			fmt.Println()
		}
	}()

	fmt.Println("Receiving high-volume data... (Press Ctrl+C to stop)")
	fmt.Println("Stats will be printed every 5 seconds")
	fmt.Println()

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println()
	fmt.Println("Shutting down...")

	// Print final summary
	finalCount := atomic.LoadUint64(&messageCount)
	fmt.Println()
	fmt.Println("=== FINAL SUMMARY ===")
	fmt.Printf("Total Messages: %d\n", finalCount)
	fmt.Printf("Instruments:    %d\n", len(instruments))
	fmt.Println("=====================")

	if err := client.Disconnect(); err != nil {
		log.Printf("Error: %v", err)
	}

	fmt.Println()
	fmt.Println("Done")
	fmt.Println()
	fmt.Println("High Volume Tips:")
	fmt.Println("  - Use PooledClient for 100+ instruments")
	fmt.Println("  - Monitor connection health via GetStats()")
	fmt.Println("  - Process messages asynchronously")
	fmt.Println("  - Use buffered channels for heavy processing")
}
