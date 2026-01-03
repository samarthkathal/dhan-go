// Package main demonstrates basic REST API usage with the Dhan Go SDK.
//
// This example shows:
// - Creating a REST client with access token
// - Fetching holdings, positions, orders, and fund limits
// - Proper response handling
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

	"github.com/samarthkathal/dhan-go/rest"
)

func main() {
	// Get access token from environment
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("DHAN_ACCESS_TOKEN environment variable not set")
	}

	ctx := context.Background()

	// Create REST client
	client, err := rest.NewClient(
		"https://api.dhan.co",
		accessToken,
		nil, // Use default HTTP client
	)
	if err != nil {
		log.Fatalf("Failed to create REST client: %v", err)
	}

	fmt.Println("REST client created successfully")
	fmt.Println()

	// Example 1: Get Holdings
	fmt.Println("Fetching holdings...")
	holdings, err := client.GetHoldings(ctx)
	if err != nil {
		log.Printf("Error fetching holdings: %v", err)
	} else if holdings.JSON200 != nil {
		fmt.Printf("Holdings fetched: %d items\n", len(*holdings.JSON200))
		for i, h := range *holdings.JSON200 {
			if i >= 3 {
				fmt.Printf("  ... and %d more\n", len(*holdings.JSON200)-3)
				break
			}
			if h.TradingSymbol != nil {
				fmt.Printf("  - %s\n", *h.TradingSymbol)
			}
		}
	}
	fmt.Println()

	// Example 2: Get Positions
	fmt.Println("Fetching positions...")
	positions, err := client.GetPositions(ctx)
	if err != nil {
		log.Printf("Error fetching positions: %v", err)
	} else if positions.JSON200 != nil {
		fmt.Printf("Positions fetched: %d items\n", len(*positions.JSON200))
	}
	fmt.Println()

	// Example 3: Get Orders
	fmt.Println("Fetching orders...")
	orders, err := client.GetOrders(ctx)
	if err != nil {
		log.Printf("Error fetching orders: %v", err)
	} else if orders.JSON200 != nil {
		fmt.Printf("Orders fetched: %d items\n", len(*orders.JSON200))
	}
	fmt.Println()

	// Example 4: Get Fund Limits
	fmt.Println("Fetching fund limits...")
	fundLimits, err := client.GetFundLimits(ctx)
	if err != nil {
		log.Printf("Error fetching fund limits: %v", err)
	} else if fundLimits.JSON200 != nil {
		if fundLimits.JSON200.AvailabelBalance != nil {
			fmt.Printf("Available balance: %.2f\n", *fundLimits.JSON200.AvailabelBalance)
		}
		if fundLimits.JSON200.CollateralAmount != nil {
			fmt.Printf("Collateral amount: %.2f\n", *fundLimits.JSON200.CollateralAmount)
		}
	}
	fmt.Println()

	fmt.Println("Basic REST operations completed")
}
