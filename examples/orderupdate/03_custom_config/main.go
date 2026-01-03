// Package main demonstrates OrderUpdate with custom WebSocket configuration.
//
// This example shows:
// - Custom connection timeouts
// - Custom ping/pong intervals
// - Reconnection settings
// - Buffer sizes
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

	"github.com/samarthkathal/dhan-go/orderupdate"
)

func main() {
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("DHAN_ACCESS_TOKEN environment variable not set")
	}

	fmt.Println("OrderUpdate Custom Configuration Example")
	fmt.Println()

	// Create custom WebSocket configuration
	customConfig := &orderupdate.WebSocketConfig{
		// Connection settings
		MaxConnections:        1, // Order updates typically need 1 connection
		MaxInstrumentsPerConn: 100,
		MaxBatchSize:          10,

		// Timeout settings
		ConnectTimeout: 15 * time.Second,
		ReadTimeout:    0, // Disabled (use pong wait)
		WriteTimeout:   5 * time.Second,

		// Keep-alive settings
		PingInterval: 10 * time.Second,
		PongWait:     30 * time.Second,

		// Reconnection settings
		ReconnectDelay:       3 * time.Second,
		MaxReconnectAttempts: 10, // Retry up to 10 times

		// Buffer settings
		ReadBufferSize:  4096,
		WriteBufferSize: 2048,

		// Feature flags
		EnableLogging:  true,
		EnableRecovery: true,
	}

	fmt.Println("Custom Configuration:")
	fmt.Printf("  ConnectTimeout:        %v\n", customConfig.ConnectTimeout)
	fmt.Printf("  WriteTimeout:          %v\n", customConfig.WriteTimeout)
	fmt.Printf("  PingInterval:          %v\n", customConfig.PingInterval)
	fmt.Printf("  PongWait:              %v\n", customConfig.PongWait)
	fmt.Printf("  ReconnectDelay:        %v\n", customConfig.ReconnectDelay)
	fmt.Printf("  MaxReconnectAttempts:  %d\n", customConfig.MaxReconnectAttempts)
	fmt.Printf("  ReadBufferSize:        %d bytes\n", customConfig.ReadBufferSize)
	fmt.Printf("  WriteBufferSize:       %d bytes\n", customConfig.WriteBufferSize)
	fmt.Println()

	// Create client with custom config
	client, err := orderupdate.NewClient(
		accessToken,
		orderupdate.WithConfig(customConfig),
		orderupdate.WithOrderUpdateCallback(func(alert *orderupdate.OrderAlert) {
			if !alert.IsOrderAlert() {
				return
			}
			fmt.Printf("ORDER | %s | %s | %s | %.2f\n",
				alert.GetOrderID(),
				alert.GetStatus(),
				alert.Data.Symbol,
				alert.Data.Price)
		}),
		orderupdate.WithErrorCallback(func(err error) {
			log.Printf("ERROR | %v", err)
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	fmt.Println("Client created with custom configuration")
	fmt.Println()

	// Connect
	connectCtx, cancel := context.WithTimeout(context.Background(), customConfig.ConnectTimeout)
	defer cancel()

	fmt.Printf("Connecting (timeout: %v)...\n", customConfig.ConnectTimeout)
	startTime := time.Now()

	if err := client.Connect(connectCtx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	fmt.Printf("Connected in %v\n", time.Since(startTime))
	fmt.Println()

	// Print connection stats periodically
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stats := client.GetStats()
			fmt.Printf("STATS | Connected: %v\n", stats.Connected)
		}
	}()

	fmt.Println("Listening for order updates... (Press Ctrl+C to stop)")
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
	fmt.Println("  - Lower ReconnectDelay for faster recovery")
	fmt.Println("  - Higher MaxReconnectAttempts for reliability")
	fmt.Println("  - Tune PingInterval based on network conditions")
}
