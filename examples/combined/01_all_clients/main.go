// Package main demonstrates using all Dhan clients together.
//
// This example shows:
// - REST client for account operations
// - MarketFeed client for real-time prices
// - OrderUpdate client for order tracking
// - Concurrent client management
// - Proper shutdown sequence
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
	"github.com/samarthkathal/dhan-go/orderupdate"
	"github.com/samarthkathal/dhan-go/rest"
)

func main() {
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("DHAN_ACCESS_TOKEN environment variable not set")
	}

	fmt.Println("Dhan All Clients Example")
	fmt.Println()
	fmt.Println("This example demonstrates using all three client types:")
	fmt.Println("  1. REST Client    - For account/order operations")
	fmt.Println("  2. MarketFeed     - For real-time market data")
	fmt.Println("  3. OrderUpdate    - For real-time order updates")
	fmt.Println()

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// ============================================
	// 1. Create REST Client
	// ============================================
	fmt.Println("Creating REST client...")
	restClient, err := rest.NewClient(
		"https://api.dhan.co",
		accessToken,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to create REST client: %v", err)
	}
	fmt.Println("REST client created")

	// Fetch initial account info
	fmt.Println()
	fmt.Println("Fetching account information...")

	holdingsCtx, holdingsCancel := context.WithTimeout(ctx, 10*time.Second)
	holdings, err := restClient.GetHoldings(holdingsCtx)
	holdingsCancel()

	if err != nil {
		log.Printf("Warning: Failed to fetch holdings: %v", err)
	} else if holdings.JSON200 != nil {
		fmt.Printf("Holdings: %d securities\n", len(*holdings.JSON200))
	}

	positionsCtx, positionsCancel := context.WithTimeout(ctx, 10*time.Second)
	positions, err := restClient.GetPositions(positionsCtx)
	positionsCancel()

	if err != nil {
		log.Printf("Warning: Failed to fetch positions: %v", err)
	} else if positions.JSON200 != nil {
		fmt.Printf("Positions: %d open\n", len(*positions.JSON200))
	}
	fmt.Println()

	// ============================================
	// 2. Create MarketFeed Client
	// ============================================
	fmt.Println("Creating MarketFeed client...")
	marketClient, err := marketfeed.NewClient(
		accessToken,
		marketfeed.WithTickerCallback(func(data *marketfeed.TickerData) {
			fmt.Printf("[MARKET] ID: %d | LTP: %.2f\n",
				data.Header.SecurityID,
				data.LastTradedPrice)
		}),
		marketfeed.WithErrorCallback(func(err error) {
			log.Printf("[MARKET ERROR] %v", err)
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create MarketFeed client: %v", err)
	}
	fmt.Println("MarketFeed client created")

	// Connect MarketFeed
	marketCtx, marketCancel := context.WithTimeout(ctx, 30*time.Second)
	if err := marketClient.Connect(marketCtx); err != nil {
		log.Fatalf("Failed to connect MarketFeed: %v", err)
	}
	marketCancel()
	fmt.Println("MarketFeed connected")
	fmt.Println()

	// ============================================
	// 3. Create OrderUpdate Client
	// ============================================
	fmt.Println("Creating OrderUpdate client...")
	orderClient, err := orderupdate.NewClient(
		accessToken,
		orderupdate.WithOrderUpdateCallback(func(alert *orderupdate.OrderAlert) {
			if !alert.IsOrderAlert() {
				return
			}
			fmt.Printf("[ORDER] %s | %s | %s | %.2f\n",
				alert.GetOrderID(),
				alert.GetStatus(),
				alert.Data.Symbol,
				alert.Data.Price)
		}),
		orderupdate.WithErrorCallback(func(err error) {
			log.Printf("[ORDER ERROR] %v", err)
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create OrderUpdate client: %v", err)
	}
	fmt.Println("OrderUpdate client created")

	// Connect OrderUpdate
	orderCtx, orderCancel := context.WithTimeout(ctx, 30*time.Second)
	if err := orderClient.Connect(orderCtx); err != nil {
		log.Fatalf("Failed to connect OrderUpdate: %v", err)
	}
	orderCancel()
	fmt.Println("OrderUpdate connected")
	fmt.Println()

	// ============================================
	// 4. Subscribe to Market Data
	// ============================================
	instruments := []marketfeed.Instrument{
		{SecurityID: "1333", ExchangeSegment: marketfeed.ExchangeNSEEQ}, // TCS
		{SecurityID: "1594", ExchangeSegment: marketfeed.ExchangeNSEEQ}, // Infosys
	}

	fmt.Println("Subscribing to market data...")
	if err := marketClient.Subscribe(context.Background(), instruments); err != nil {
		log.Printf("Warning: Failed to subscribe: %v", err)
	} else {
		fmt.Printf("Subscribed to %d instruments\n", len(instruments))
	}
	fmt.Println()

	// ============================================
	// 5. Periodic Status Check
	// ============================================
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fmt.Println()
				fmt.Println("=== STATUS CHECK ===")
				marketStats := marketClient.GetStats()
				orderStats := orderClient.GetStats()
				fmt.Printf("MarketFeed: Connected=%v\n", marketStats.Connected)
				fmt.Printf("OrderUpdate: Connected=%v\n", orderStats.Connected)
				fmt.Println("====================")
				fmt.Println()
			}
		}
	}()

	fmt.Println("All clients active. Receiving data...")
	fmt.Println("(Press Ctrl+C to stop)")
	fmt.Println()

	// ============================================
	// 6. Wait for Shutdown Signal
	// ============================================
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println()
	fmt.Println("=== SHUTDOWN SEQUENCE ===")
	fmt.Println()

	// Signal shutdown
	cancel()

	// Wait for goroutines
	fmt.Println("Waiting for background tasks...")
	wg.Wait()

	// Shutdown in reverse order of startup
	fmt.Println("Disconnecting OrderUpdate...")
	if err := orderClient.Disconnect(); err != nil {
		log.Printf("OrderUpdate disconnect error: %v", err)
	}

	fmt.Println("Disconnecting MarketFeed...")
	if err := marketClient.Disconnect(); err != nil {
		log.Printf("MarketFeed disconnect error: %v", err)
	}

	fmt.Println()
	fmt.Println("=== SHUTDOWN COMPLETE ===")
	fmt.Println()
	fmt.Println("Client Usage Summary:")
	fmt.Println("  REST       - Account queries, order placement")
	fmt.Println("  MarketFeed - Real-time price streaming")
	fmt.Println("  OrderUpdate - Real-time order status")
}
