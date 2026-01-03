// Package main demonstrates order placement with the Dhan Go SDK.
//
// This example shows:
// - Placing market and limit orders
// - Using correct request types and enums
// - Using pointer helpers for optional fields
//
// WARNING: This example contains code that can place REAL orders.
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

	"github.com/samarthkathal/dhan-go/internal/restgen"
	"github.com/samarthkathal/dhan-go/rest"
)

// Helper functions for creating pointers
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

	// Example 1: Market Order (BUY)
	// This creates the request but does NOT execute it
	marketOrderReq := restgen.PlaceorderJSONRequestBody{
		SecurityId:      ptr("1333"),                                   // TCS security ID
		ExchangeSegment: restgen.OrderRequestExchangeSegmentNSEEQ,      // NSE Equity
		TransactionType: restgen.OrderRequestTransactionTypeBUY,        // BUY
		Quantity:        ptr(int32(1)),                                 // 1 share
		OrderType:       ptr(restgen.OrderRequestOrderTypeMARKET),      // Market order
		ProductType:     ptr(restgen.OrderRequestProductTypeCNC),       // Cash and Carry (delivery)
		Price:           ptr(float32(0)),                               // Market order, price ignored
		Validity:        ptr(restgen.OrderRequestValidityDAY),          // Day validity
	}

	fmt.Println("Market Order Request (BUY TCS):")
	fmt.Printf("  Security ID:     %s\n", *marketOrderReq.SecurityId)
	fmt.Printf("  Exchange:        %s\n", marketOrderReq.ExchangeSegment)
	fmt.Printf("  Transaction:     %s\n", marketOrderReq.TransactionType)
	fmt.Printf("  Quantity:        %d\n", *marketOrderReq.Quantity)
	fmt.Printf("  Order Type:      %s\n", *marketOrderReq.OrderType)
	fmt.Printf("  Product Type:    %s\n", *marketOrderReq.ProductType)
	fmt.Println()

	// Example 2: Limit Order (BUY)
	limitOrderReq := restgen.PlaceorderJSONRequestBody{
		SecurityId:      ptr("1594"),                                   // Infosys security ID
		ExchangeSegment: restgen.OrderRequestExchangeSegmentNSEEQ,      // NSE Equity
		TransactionType: restgen.OrderRequestTransactionTypeBUY,        // BUY
		Quantity:        ptr(int32(5)),                                 // 5 shares
		OrderType:       ptr(restgen.OrderRequestOrderTypeLIMIT),       // Limit order
		ProductType:     ptr(restgen.OrderRequestProductTypeINTRADAY),  // Intraday
		Price:           ptr(float32(1500.00)),                         // Limit price
		Validity:        ptr(restgen.OrderRequestValidityDAY),          // Day validity
	}

	fmt.Println("Limit Order Request (BUY Infosys):")
	fmt.Printf("  Security ID:     %s\n", *limitOrderReq.SecurityId)
	fmt.Printf("  Exchange:        %s\n", limitOrderReq.ExchangeSegment)
	fmt.Printf("  Transaction:     %s\n", limitOrderReq.TransactionType)
	fmt.Printf("  Quantity:        %d\n", *limitOrderReq.Quantity)
	fmt.Printf("  Order Type:      %s\n", *limitOrderReq.OrderType)
	fmt.Printf("  Product Type:    %s\n", *limitOrderReq.ProductType)
	fmt.Printf("  Price:           %.2f\n", *limitOrderReq.Price)
	fmt.Println()

	// Example 3: Stop Loss Order (SELL)
	stopLossOrderReq := restgen.PlaceorderJSONRequestBody{
		SecurityId:      ptr("1333"),                                      // TCS security ID
		ExchangeSegment: restgen.OrderRequestExchangeSegmentNSEEQ,         // NSE Equity
		TransactionType: restgen.OrderRequestTransactionTypeSELL,          // SELL
		Quantity:        ptr(int32(1)),                                    // 1 share
		OrderType:       ptr(restgen.OrderRequestOrderTypeSTOPLOSS),       // Stop Loss
		ProductType:     ptr(restgen.OrderRequestProductTypeCNC),          // Delivery
		Price:           ptr(float32(3400.00)),                            // Target price
		TriggerPrice:    ptr(float32(3350.00)),                            // Trigger price
		Validity:        ptr(restgen.OrderRequestValidityDAY),             // Day validity
	}

	fmt.Println("Stop Loss Order Request (SELL TCS):")
	fmt.Printf("  Security ID:     %s\n", *stopLossOrderReq.SecurityId)
	fmt.Printf("  Exchange:        %s\n", stopLossOrderReq.ExchangeSegment)
	fmt.Printf("  Transaction:     %s\n", stopLossOrderReq.TransactionType)
	fmt.Printf("  Quantity:        %d\n", *stopLossOrderReq.Quantity)
	fmt.Printf("  Order Type:      %s\n", *stopLossOrderReq.OrderType)
	fmt.Printf("  Price:           %.2f\n", *stopLossOrderReq.Price)
	fmt.Printf("  Trigger Price:   %.2f\n", *stopLossOrderReq.TriggerPrice)
	fmt.Println()

	// UNCOMMENT BELOW TO ACTUALLY PLACE AN ORDER
	// WARNING: This will place a REAL order!
	/*
		fmt.Println("Placing market order...")
		orderResp, err := client.PlaceOrder(ctx, marketOrderReq)
		if err != nil {
			log.Printf("Error placing order: %v", err)
		} else if orderResp.JSON200 != nil {
			fmt.Printf("Order placed successfully!\n")
			if orderResp.JSON200.OrderId != nil {
				fmt.Printf("Order ID: %s\n", *orderResp.JSON200.OrderId)
			}
			if orderResp.JSON200.OrderStatus != nil {
				fmt.Printf("Status: %s\n", *orderResp.JSON200.OrderStatus)
			}
		}
	*/

	// Suppress unused variable warnings
	_ = ctx
	_ = client
	_ = marketOrderReq
	_ = limitOrderReq
	_ = stopLossOrderReq

	fmt.Println("Order examples completed (orders NOT placed - uncomment code to place)")
}
