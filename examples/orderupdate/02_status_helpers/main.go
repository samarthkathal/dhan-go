// Package main demonstrates OrderUpdate status helper methods.
//
// This example shows:
// - Using IsFilled(), IsPartiallyFilled(), IsRejected(), IsCancelled()
// - Accessing order times with GetOrderTime()
// - Checking order status programmatically
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

	fmt.Println("OrderUpdate Status Helpers Example")
	fmt.Println()

	fmt.Println("Available Status Helper Methods:")
	fmt.Println("  - GetOrderID()        -> string")
	fmt.Println("  - GetStatus()         -> string")
	fmt.Println("  - GetTradedQuantity() -> int32")
	fmt.Println("  - GetAvgTradedPrice() -> float32")
	fmt.Println("  - IsFilled()          -> bool")
	fmt.Println("  - IsPartiallyFilled() -> bool")
	fmt.Println("  - IsRejected()        -> bool")
	fmt.Println("  - IsCancelled()       -> bool")
	fmt.Println("  - IsOrderAlert()      -> bool")
	fmt.Println("  - GetOrderTime()      -> (time.Time, error)")
	fmt.Println()

	fmt.Println("Status Constants:")
	fmt.Printf("  - TRANSIT:   %s\n", orderupdate.OrderStatusTransit)
	fmt.Printf("  - PENDING:   %s\n", orderupdate.OrderStatusPending)
	fmt.Printf("  - REJECTED:  %s\n", orderupdate.OrderStatusRejected)
	fmt.Printf("  - CANCELLED: %s\n", orderupdate.OrderStatusCancelled)
	fmt.Printf("  - TRADED:    %s\n", orderupdate.OrderStatusTraded)
	fmt.Printf("  - EXPIRED:   %s\n", orderupdate.OrderStatusExpired)
	fmt.Println()

	// Create client with status-aware callback
	client, err := orderupdate.NewClient(
		accessToken,
		orderupdate.WithOrderUpdateCallback(func(alert *orderupdate.OrderAlert) {
			if !alert.IsOrderAlert() {
				return
			}

			fmt.Printf("ORDER UPDATE | %s | ", alert.GetOrderID())

			// Use status helpers to determine action
			switch {
			case alert.IsFilled():
				fmt.Printf("FILLED | Qty: %d @ %.2f avg\n",
					alert.GetTradedQuantity(),
					alert.GetAvgTradedPrice())
				fmt.Println("         Action: Update positions, log trade")

			case alert.IsPartiallyFilled():
				fmt.Printf("PARTIAL | Traded: %d | Remaining: %d\n",
					alert.GetTradedQuantity(),
					alert.Data.RemainingQty)
				fmt.Println("         Action: Monitor for complete fill")

			case alert.IsRejected():
				fmt.Printf("REJECTED | Reason: %s\n",
					alert.Data.ReasonDescription)
				fmt.Println("         Action: Alert user, check reason")

			case alert.IsCancelled():
				fmt.Printf("CANCELLED | By: %s\n", alert.Data.ReasonCode)
				fmt.Println("         Action: Confirm cancellation")

			default:
				// Check raw status for other states
				switch alert.GetStatus() {
				case orderupdate.OrderStatusTransit:
					fmt.Println("IN TRANSIT")
					fmt.Println("         Action: Wait for exchange confirmation")

				case orderupdate.OrderStatusPending:
					fmt.Println("PENDING")
					fmt.Println("         Action: Order accepted, waiting for execution")

				case orderupdate.OrderStatusExpired:
					fmt.Println("EXPIRED")
					fmt.Println("         Action: Order validity ended")

				default:
					fmt.Printf("Status: %s\n", alert.GetStatus())
				}
			}

			// Parse order time
			if orderTime, err := alert.GetOrderTime(); err == nil {
				fmt.Printf("         Time: %s\n", orderTime.Format("15:04:05"))
			}
			fmt.Println()
		}),
		orderupdate.WithErrorCallback(func(err error) {
			log.Printf("ERROR | %v", err)
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
	fmt.Println("Listening for orders... (Press Ctrl+C to stop)")
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
