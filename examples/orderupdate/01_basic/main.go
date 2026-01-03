// Package main demonstrates basic OrderUpdate WebSocket usage.
//
// This example shows:
// - Creating an OrderUpdate client
// - Receiving real-time order updates
// - Accessing order alert fields
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

	fmt.Println("OrderUpdate Basic Example")
	fmt.Println()

	// Create OrderUpdate client with callback
	client, err := orderupdate.NewClient(
		accessToken,
		orderupdate.WithOrderUpdateCallback(func(alert *orderupdate.OrderAlert) {
			// Check if it's an order alert
			if !alert.IsOrderAlert() {
				return
			}

			// Access order details using helper methods
			fmt.Println("=== ORDER UPDATE ===")
			fmt.Printf("Order ID:     %s\n", alert.GetOrderID())
			fmt.Printf("Status:       %s\n", alert.GetStatus())
			fmt.Printf("Symbol:       %s\n", alert.Data.Symbol)
			fmt.Printf("Exchange:     %s\n", alert.Data.Exchange)
			fmt.Printf("Type:         %s\n", alert.Data.TransactionType)
			fmt.Printf("Product:      %s\n", alert.Data.ProductType)
			fmt.Printf("Order Type:   %s\n", alert.Data.OrderType)
			fmt.Printf("Quantity:     %d\n", alert.Data.Quantity)
			fmt.Printf("Price:        %.2f\n", alert.Data.Price)
			fmt.Printf("Traded Qty:   %d\n", alert.GetTradedQuantity())
			fmt.Printf("Avg Price:    %.2f\n", alert.GetAvgTradedPrice())
			fmt.Printf("Remaining:    %d\n", alert.Data.RemainingQty)
			fmt.Println("====================")
			fmt.Println()
		}),
		orderupdate.WithErrorCallback(func(err error) {
			log.Printf("ERROR | %v", err)
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	fmt.Println("OrderUpdate client created")
	fmt.Println()

	// Connect
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Connecting to OrderUpdate WebSocket...")
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	fmt.Println("Connected")
	fmt.Println()

	fmt.Println("Listening for order updates... (Press Ctrl+C to stop)")
	fmt.Println("Place an order from your Dhan account to see updates here.")
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
}
