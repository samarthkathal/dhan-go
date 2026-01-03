// Package main demonstrates MarketFeed with custom WebSocket configuration.
//
// This example shows:
// - Custom connection timeouts
// - Ping/pong intervals for keep-alive
// - Reconnection settings
// - Buffer sizes
// - Feature flags (logging, recovery)
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

	fmt.Println("MarketFeed Custom Configuration Example")
	fmt.Println()

	// Create custom WebSocket configuration
	customConfig := &marketfeed.WebSocketConfig{
		// Connection limits
		MaxConnections:        1,    // Single connection mode
		MaxInstrumentsPerConn: 5000, // Max instruments per connection
		MaxBatchSize:          100,  // Max instruments per subscription message

		// Timeout settings
		ConnectTimeout: 15 * time.Second, // Connection establishment timeout
		ReadTimeout:    0,                // Disabled (use pong wait instead)
		WriteTimeout:   5 * time.Second,  // Message send timeout

		// Keep-alive settings
		PingInterval: 10 * time.Second, // Send ping every 10s
		PongWait:     30 * time.Second, // Expect pong within 30s

		// Reconnection settings
		ReconnectDelay:       3 * time.Second, // Wait before reconnect
		MaxReconnectAttempts: 5,               // Max retry attempts (0 = unlimited)

		// Buffer settings
		ReadBufferSize:  8192, // 8KB read buffer
		WriteBufferSize: 4096, // 4KB write buffer

		// Feature flags
		EnableLogging:  true, // Enable internal logging
		EnableRecovery: true, // Enable panic recovery
	}

	fmt.Println("Custom Configuration:")
	fmt.Printf("  Connection Limits:\n")
	fmt.Printf("    MaxConnections:        %d\n", customConfig.MaxConnections)
	fmt.Printf("    MaxInstrumentsPerConn: %d\n", customConfig.MaxInstrumentsPerConn)
	fmt.Printf("    MaxBatchSize:          %d\n", customConfig.MaxBatchSize)
	fmt.Println()
	fmt.Printf("  Timeouts:\n")
	fmt.Printf("    ConnectTimeout:        %v\n", customConfig.ConnectTimeout)
	fmt.Printf("    WriteTimeout:          %v\n", customConfig.WriteTimeout)
	fmt.Println()
	fmt.Printf("  Keep-Alive:\n")
	fmt.Printf("    PingInterval:          %v\n", customConfig.PingInterval)
	fmt.Printf("    PongWait:              %v\n", customConfig.PongWait)
	fmt.Println()
	fmt.Printf("  Reconnection:\n")
	fmt.Printf("    ReconnectDelay:        %v\n", customConfig.ReconnectDelay)
	fmt.Printf("    MaxReconnectAttempts:  %d\n", customConfig.MaxReconnectAttempts)
	fmt.Println()
	fmt.Printf("  Buffers:\n")
	fmt.Printf("    ReadBufferSize:        %d bytes\n", customConfig.ReadBufferSize)
	fmt.Printf("    WriteBufferSize:       %d bytes\n", customConfig.WriteBufferSize)
	fmt.Println()
	fmt.Printf("  Features:\n")
	fmt.Printf("    EnableLogging:         %v\n", customConfig.EnableLogging)
	fmt.Printf("    EnableRecovery:        %v\n", customConfig.EnableRecovery)
	fmt.Println()

	// Create client with custom config
	client, err := marketfeed.NewClient(
		accessToken,
		marketfeed.WithConfig(customConfig),
		marketfeed.WithTickerCallback(func(data *marketfeed.TickerData) {
			fmt.Printf("TICKER | ID: %d | LTP: %.2f\n",
				data.Header.SecurityID,
				data.LastTradedPrice)
		}),
		marketfeed.WithErrorCallback(func(err error) {
			log.Printf("ERROR | %v", err)
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	fmt.Println("Client created with custom configuration")
	fmt.Println()

	// Connect with custom timeout
	connectCtx, cancel := context.WithTimeout(context.Background(), customConfig.ConnectTimeout)
	defer cancel()

	fmt.Printf("Connecting (timeout: %v)...\n", customConfig.ConnectTimeout)
	startTime := time.Now()

	if err := client.Connect(connectCtx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	fmt.Printf("Connected in %v\n", time.Since(startTime))
	fmt.Println()

	// Subscribe
	instruments := []marketfeed.Instrument{
		{SecurityID: "1333", ExchangeSegment: marketfeed.ExchangeNSEEQ},
	}

	if err := client.Subscribe(context.Background(), instruments); err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	fmt.Println("Subscribed")
	fmt.Println()

	// Print stats periodically
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stats := client.GetStats()
			fmt.Printf("STATS | Connected: %v\n", stats.Connected)
		}
	}()

	fmt.Println("Receiving data... (Press Ctrl+C to stop)")
	fmt.Println()

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println()
	fmt.Println("Shutting down...")
	if err := client.Disconnect(); err != nil {
		log.Printf("Error: %v", err)
	}

	fmt.Println("Done")
	fmt.Println()
	fmt.Println("Configuration Tips:")
	fmt.Println("  - Lower timeouts = Faster failure detection")
	fmt.Println("  - Higher PingInterval = Less network overhead")
	fmt.Println("  - Lower ReconnectDelay = Faster recovery")
	fmt.Println("  - Larger buffers = Better for high-volume data")
}
