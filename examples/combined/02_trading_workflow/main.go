// Package main demonstrates a complete trading workflow.
//
// This example shows:
// - Monitoring prices via MarketFeed
// - Placing orders via REST when conditions are met
// - Tracking order execution via OrderUpdate
// - Complete workflow orchestration
//
// IMPORTANT: This is a DEMO example. It does NOT place real orders.
// The order placement code is commented out for safety.
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

// TradingState tracks the current state of our trading logic
type TradingState struct {
	mu              sync.RWMutex
	lastPrice       float32
	targetBuyPrice  float32
	targetSellPrice float32
	pendingOrderID  string
	orderFilled     bool
}

func main() {
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("DHAN_ACCESS_TOKEN environment variable not set")
	}

	fmt.Println("Dhan Trading Workflow Example")
	fmt.Println()
	fmt.Println("IMPORTANT: This is a DEMO. No real orders are placed.")
	fmt.Println()
	fmt.Println("Workflow:")
	fmt.Println("  1. Connect to MarketFeed for price updates")
	fmt.Println("  2. Connect to OrderUpdate for order tracking")
	fmt.Println("  3. Monitor price for buy/sell signals")
	fmt.Println("  4. (DEMO) Simulate order placement")
	fmt.Println("  5. Track order execution")
	fmt.Println()

	// Initialize trading state
	state := &TradingState{
		targetBuyPrice:  3500.00, // Demo: Buy if price drops to this level
		targetSellPrice: 3600.00, // Demo: Sell if price rises to this level
	}

	// Context for shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ============================================
	// Create Clients
	// ============================================

	// REST client for order operations
	restClient, err := rest.NewClient(
		"https://api.dhan.co",
		accessToken,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to create REST client: %v", err)
	}
	fmt.Println("REST client ready")

	// MarketFeed client for price monitoring
	marketClient, err := marketfeed.NewClient(
		accessToken,
		marketfeed.WithTickerCallback(func(data *marketfeed.TickerData) {
			handlePriceUpdate(state, restClient, data)
		}),
		marketfeed.WithErrorCallback(func(err error) {
			log.Printf("[MARKET ERROR] %v", err)
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create MarketFeed client: %v", err)
	}
	fmt.Println("MarketFeed client ready")

	// OrderUpdate client for execution tracking
	orderClient, err := orderupdate.NewClient(
		accessToken,
		orderupdate.WithOrderUpdateCallback(func(alert *orderupdate.OrderAlert) {
			handleOrderUpdate(state, alert)
		}),
		orderupdate.WithErrorCallback(func(err error) {
			log.Printf("[ORDER ERROR] %v", err)
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create OrderUpdate client: %v", err)
	}
	fmt.Println("OrderUpdate client ready")
	fmt.Println()

	// ============================================
	// Connect WebSocket Clients
	// ============================================

	connectCtx, connectCancel := context.WithTimeout(ctx, 30*time.Second)

	fmt.Println("Connecting MarketFeed...")
	if err := marketClient.Connect(connectCtx); err != nil {
		log.Fatalf("MarketFeed connect failed: %v", err)
	}
	fmt.Println("MarketFeed connected")

	fmt.Println("Connecting OrderUpdate...")
	if err := orderClient.Connect(connectCtx); err != nil {
		log.Fatalf("OrderUpdate connect failed: %v", err)
	}
	fmt.Println("OrderUpdate connected")
	connectCancel()
	fmt.Println()

	// ============================================
	// Subscribe to Market Data
	// ============================================

	// Monitor TCS for demo
	instruments := []marketfeed.Instrument{
		{SecurityID: "1333", ExchangeSegment: marketfeed.ExchangeNSEEQ},
	}

	fmt.Printf("Subscribing to %d instrument(s)...\n", len(instruments))
	if err := marketClient.Subscribe(context.Background(), instruments); err != nil {
		log.Fatalf("Subscribe failed: %v", err)
	}
	fmt.Println("Subscribed")
	fmt.Println()

	// ============================================
	// Trading Logic Display
	// ============================================

	fmt.Println("=== TRADING PARAMETERS (DEMO) ===")
	fmt.Printf("Target Buy Price:  %.2f\n", state.targetBuyPrice)
	fmt.Printf("Target Sell Price: %.2f\n", state.targetSellPrice)
	fmt.Println("=================================")
	fmt.Println()
	fmt.Println("Monitoring prices... (Press Ctrl+C to stop)")
	fmt.Println()

	// Periodic status display
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				state.mu.RLock()
				fmt.Printf("[STATUS] Last Price: %.2f | Pending Order: %s | Filled: %v\n",
					state.lastPrice,
					state.pendingOrderID,
					state.orderFilled)
				state.mu.RUnlock()
			}
		}
	}()

	// ============================================
	// Wait for Shutdown
	// ============================================

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println()
	fmt.Println("Shutting down trading workflow...")

	cancel()

	if err := orderClient.Disconnect(); err != nil {
		log.Printf("OrderUpdate disconnect error: %v", err)
	}
	if err := marketClient.Disconnect(); err != nil {
		log.Printf("MarketFeed disconnect error: %v", err)
	}

	fmt.Println()
	fmt.Println("Trading workflow complete")
	fmt.Println()
	fmt.Println("In Production:")
	fmt.Println("  - Implement proper risk management")
	fmt.Println("  - Add order quantity/position limits")
	fmt.Println("  - Handle partial fills")
	fmt.Println("  - Implement stop-loss logic")
	fmt.Println("  - Add logging and monitoring")
}

// handlePriceUpdate processes incoming price data
func handlePriceUpdate(state *TradingState, _ *rest.Client, data *marketfeed.TickerData) {
	state.mu.Lock()
	defer state.mu.Unlock()

	state.lastPrice = data.LastTradedPrice

	fmt.Printf("[PRICE] Security: %d | LTP: %.2f | Time: %v\n",
		data.Header.SecurityID,
		data.LastTradedPrice,
		data.GetTradeTime().Format("15:04:05"))

	// Check if we already have a pending order
	if state.pendingOrderID != "" {
		return
	}

	// Demo: Check for trading signals
	if data.LastTradedPrice <= state.targetBuyPrice {
		fmt.Println()
		fmt.Println(">>> BUY SIGNAL DETECTED <<<")
		fmt.Printf("    Price %.2f <= Target %.2f\n", data.LastTradedPrice, state.targetBuyPrice)
		fmt.Println("    (DEMO: Order placement would happen here)")
		fmt.Println()

		// DEMO: In production, you would place the order here:
		//
		// orderReq := restgen.PlaceorderJSONRequestBody{
		//     SecurityId:      "1333",
		//     ExchangeSegment: restgen.OrderRequestExchangeSegmentNSEEQ,
		//     TransactionType: restgen.OrderRequestTransactionTypeBUY,
		//     OrderType:       restgen.OrderRequestOrderTypeMARKET,
		//     ProductType:     restgen.OrderRequestProductTypeCNC,
		//     Quantity:        pointerTo(int32(1)),
		// }
		// resp, err := restClient.PlaceOrder(ctx, orderReq)
		// if resp.JSON200 != nil {
		//     state.pendingOrderID = *resp.JSON200.OrderId
		// }

		// Simulate order ID for demo
		state.pendingOrderID = "DEMO-ORDER-123"

	} else if data.LastTradedPrice >= state.targetSellPrice {
		fmt.Println()
		fmt.Println(">>> SELL SIGNAL DETECTED <<<")
		fmt.Printf("    Price %.2f >= Target %.2f\n", data.LastTradedPrice, state.targetSellPrice)
		fmt.Println("    (DEMO: Order placement would happen here)")
		fmt.Println()

		// Simulate order ID for demo
		state.pendingOrderID = "DEMO-ORDER-456"
	}
}

// handleOrderUpdate processes incoming order updates
func handleOrderUpdate(state *TradingState, alert *orderupdate.OrderAlert) {
	if !alert.IsOrderAlert() {
		return
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	fmt.Printf("[ORDER UPDATE] ID: %s | Status: %s | Symbol: %s\n",
		alert.GetOrderID(),
		alert.GetStatus(),
		alert.Data.Symbol)

	// Check if this is our pending order
	if state.pendingOrderID != "" && alert.GetOrderID() == state.pendingOrderID {
		switch {
		case alert.IsFilled():
			fmt.Println()
			fmt.Println(">>> ORDER FILLED <<<")
			fmt.Printf("    Quantity: %d @ %.2f avg\n",
				alert.GetTradedQuantity(),
				alert.GetAvgTradedPrice())
			fmt.Println()
			state.orderFilled = true
			state.pendingOrderID = ""

		case alert.IsPartiallyFilled():
			fmt.Printf("    Partial fill: %d/%d\n",
				alert.GetTradedQuantity(),
				alert.Data.Quantity)

		case alert.IsRejected():
			fmt.Println()
			fmt.Println(">>> ORDER REJECTED <<<")
			fmt.Printf("    Reason: %s\n", alert.Data.ReasonDescription)
			fmt.Println()
			state.pendingOrderID = ""

		case alert.IsCancelled():
			fmt.Println()
			fmt.Println(">>> ORDER CANCELLED <<<")
			fmt.Println()
			state.pendingOrderID = ""
		}
	}
}
