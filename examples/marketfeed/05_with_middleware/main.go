// Package main demonstrates MarketFeed with middleware for logging and recovery.
//
// This example shows:
// - Using WSLoggingMiddleware for message logging
// - Using WSRecoveryMiddleware for panic recovery
// - Using WSTimeoutMiddleware for message processing timeout
// - Chaining multiple middleware functions
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
	"github.com/samarthkathal/dhan-go/middleware"
)

func main() {
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("DHAN_ACCESS_TOKEN environment variable not set")
	}

	fmt.Println("MarketFeed with Middleware Example")
	fmt.Println()

	// Create a logger (uses stdlib log.Logger which implements middleware.Logger interface)
	logger := log.New(os.Stdout, "[MW] ", log.LstdFlags|log.Lmicroseconds)

	// Create individual middleware
	loggingMW := middleware.WSLoggingMiddleware(logger)
	recoveryMW := middleware.WSRecoveryMiddleware(logger)
	timeoutMW := middleware.WSTimeoutMiddleware(5 * time.Second)

	// Chain middleware (first middleware is outermost)
	// Order: Recovery -> Timeout -> Logging
	// This means:
	// 1. Recovery catches panics from all inner handlers
	// 2. Timeout applies to logging and message handling
	// 3. Logging runs closest to actual message handling
	chainedMiddleware := middleware.ChainWSMiddleware(
		recoveryMW,
		timeoutMW,
		loggingMW,
	)

	fmt.Println("Middleware Chain created:")
	fmt.Println("  1. RecoveryMiddleware - Catches panics")
	fmt.Println("  2. TimeoutMiddleware  - 5s timeout")
	fmt.Println("  3. LoggingMiddleware  - Logs messages")
	fmt.Println()

	// Create client with middleware
	client, err := marketfeed.NewClient(
		accessToken,
		marketfeed.WithMiddleware(chainedMiddleware),
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

	fmt.Println("Client created with middleware enabled")
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

	// Subscribe
	instruments := []marketfeed.Instrument{
		{SecurityID: "1333", ExchangeSegment: marketfeed.ExchangeNSEEQ},
	}

	if err := client.Subscribe(context.Background(), instruments); err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	fmt.Println("Subscribed")
	fmt.Println()
	fmt.Println("Receiving data with middleware logging... (Press Ctrl+C to stop)")
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
	fmt.Println("Middleware Benefits:")
	fmt.Println("  - LoggingMiddleware: Debug message flow")
	fmt.Println("  - RecoveryMiddleware: Graceful panic handling")
	fmt.Println("  - TimeoutMiddleware: Prevent stuck handlers")
	fmt.Println("  - Chainable: Compose multiple behaviors")
}
