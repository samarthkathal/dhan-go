// Package main demonstrates order modification and cancellation with the Dhan Go SDK.
//
// This example shows:
// - Modifying an existing order (price, quantity, order type)
// - Canceling an order
// - Using the correct request types
//
// WARNING: This example contains code that can modify/cancel REAL orders.
// The actual operations are commented out for safety.
//
// Prerequisites:
// - Set DHAN_ACCESS_TOKEN environment variable
// - Have existing order IDs to modify/cancel
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

	"github.com/samarthkathal/dhan-go/internal/restgen"
	"github.com/samarthkathal/dhan-go/rest"
)

// Helper function for creating pointers
func ptr[T any](v T) *T {
	return &v
}

func main() {
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("DHAN_ACCESS_TOKEN environment variable not set")
	}

	ctx := context.Background()

	// Create REST client
	client, err := rest.NewClient(
		"https://api.dhan.co",
		accessToken,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to create REST client: %v", err)
	}

	fmt.Println("REST client created successfully")
	fmt.Println()

	// First, let's fetch existing orders to get order IDs
	fmt.Println("Fetching existing orders...")
	orders, err := client.GetOrders(ctx)
	if err != nil {
		log.Printf("Error fetching orders: %v", err)
	} else if orders.JSON200 != nil {
		fmt.Printf("Found %d orders\n", len(*orders.JSON200))
		for i, o := range *orders.JSON200 {
			if i >= 5 {
				break
			}
			if o.OrderId != nil {
				status := "unknown"
				if o.OrderStatus != nil {
					status = string(*o.OrderStatus)
				}
				symbol := "unknown"
				if o.TradingSymbol != nil {
					symbol = *o.TradingSymbol
				}
				fmt.Printf("  - Order ID: %s | Symbol: %s | Status: %s\n",
					*o.OrderId, symbol, status)
			}
		}
	}
	fmt.Println()

	// Example 1: Modify Order Request
	// Replace "YOUR_ORDER_ID" with an actual pending order ID
	orderID := "YOUR_ORDER_ID"

	modifyReq := restgen.ModifyorderJSONRequestBody{
		OrderId:           ptr(orderID),
		OrderType:         ptr(restgen.OrderModifyRequestOrderTypeLIMIT),
		Quantity:          ptr(int32(10)),                              // New quantity
		Price:             ptr(float32(3450.00)),                       // New price
		DisclosedQuantity: ptr(int32(0)),
		Validity:          ptr(restgen.OrderModifyRequestValidityDAY),
	}

	fmt.Println("Modify Order Request:")
	fmt.Printf("  Order ID:        %s\n", *modifyReq.OrderId)
	fmt.Printf("  New Order Type:  %s\n", *modifyReq.OrderType)
	fmt.Printf("  New Quantity:    %d\n", *modifyReq.Quantity)
	fmt.Printf("  New Price:       %.2f\n", *modifyReq.Price)
	fmt.Println()

	// UNCOMMENT BELOW TO ACTUALLY MODIFY AN ORDER
	// WARNING: This will modify a REAL order!
	/*
		fmt.Println("Modifying order...")
		modifyResp, err := client.ModifyOrder(ctx, orderID, modifyReq)
		if err != nil {
			log.Printf("Error modifying order: %v", err)
		} else {
			fmt.Printf("Order modified! Status code: %d\n", modifyResp.StatusCode())
			if modifyResp.JSON200 != nil {
				if modifyResp.JSON200.OrderId != nil {
					fmt.Printf("Order ID: %s\n", *modifyResp.JSON200.OrderId)
				}
				if modifyResp.JSON200.OrderStatus != nil {
					fmt.Printf("Status: %s\n", *modifyResp.JSON200.OrderStatus)
				}
			}
		}
	*/

	// Example 2: Cancel Order
	cancelOrderID := "YOUR_CANCEL_ORDER_ID"

	fmt.Println("Cancel Order Request:")
	fmt.Printf("  Order ID to cancel: %s\n", cancelOrderID)
	fmt.Println()

	// UNCOMMENT BELOW TO ACTUALLY CANCEL AN ORDER
	// WARNING: This will cancel a REAL order!
	/*
		fmt.Println("Canceling order...")
		cancelResp, err := client.CancelOrder(ctx, cancelOrderID)
		if err != nil {
			log.Printf("Error canceling order: %v", err)
		} else {
			fmt.Printf("Order canceled! Status code: %d\n", cancelResp.StatusCode())
			if cancelResp.JSON200 != nil {
				if cancelResp.JSON200.OrderId != nil {
					fmt.Printf("Order ID: %s\n", *cancelResp.JSON200.OrderId)
				}
				if cancelResp.JSON200.OrderStatus != nil {
					fmt.Printf("Status: %s\n", *cancelResp.JSON200.OrderStatus)
				}
			}
		}
	*/

	// Suppress unused variable warnings
	_ = modifyReq
	_ = cancelOrderID

	fmt.Println("Order modification/cancellation examples completed")
	fmt.Println("(Operations NOT executed - uncomment code and provide valid order IDs)")
}
