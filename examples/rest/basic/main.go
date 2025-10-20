package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/samarthkathal/dhan-go/rest/client"
)

func main() {
	// 1. Create HTTP client
	httpClient := &http.Client{}

	// 2. Create authentication middleware
	accessToken := "your-access-token"
	authMiddleware := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("access-token", accessToken)
		return nil
	}

	// 3. Create Dhan API client
	dhanClient, err := client.NewClientWithResponses(
		"https://api.dhan.co",
		client.WithHTTPClient(httpClient),
		client.WithRequestEditorFn(authMiddleware),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// 4. Get holdings
	fmt.Println("Fetching holdings...")
	holdings, err := dhanClient.GetholdingsWithResponse(ctx, nil)
	if err != nil {
		log.Fatalf("Error fetching holdings: %v", err)
	}

	if holdings.StatusCode() == 200 && holdings.JSON200 != nil {
		fmt.Printf("Holdings fetched successfully: %d holdings\n", len(*holdings.JSON200))
	} else {
		fmt.Printf("API returned status: %d\n", holdings.StatusCode())
	}

	// 5. Get positions
	fmt.Println("\nFetching positions...")
	positions, err := dhanClient.GetpositionsWithResponse(ctx, nil)
	if err != nil {
		log.Fatalf("Error fetching positions: %v", err)
	}

	if positions.StatusCode() == 200 && positions.JSON200 != nil {
		fmt.Printf("Positions fetched successfully: %d positions\n", len(*positions.JSON200))
	} else {
		fmt.Printf("API returned status: %d\n", positions.StatusCode())
	}

	// 6. Get fund limits
	fmt.Println("\nFetching fund limits...")
	funds, err := dhanClient.FundlimitWithResponse(ctx, nil)
	if err != nil {
		log.Fatalf("Error fetching funds: %v", err)
	}

	if funds.StatusCode() == 200 && funds.JSON200 != nil {
		fmt.Printf("Fund limits fetched successfully\n")
		if funds.JSON200.AvailabelBalance != nil {
			fmt.Printf("Available balance: %.2f\n", *funds.JSON200.AvailabelBalance)
		}
	} else {
		fmt.Printf("API returned status: %d\n", funds.StatusCode())
	}

	// 7. Example: Place an order (commented out for safety)
	/*
		fmt.Println("\nPlacing order...")
		orderReq := client.PlaceorderJSONRequestBody{
			SecurityId:      ptr("1333"),
			ExchangeSegment: client.OrderRequestExchangeSegmentNSEEQ,
			TransactionType: client.OrderRequestTransactionTypeBUY,
			Quantity:        ptr(int32(1)),
			OrderType:       ptr(client.OrderRequestOrderTypeMARKET),
			ProductType:     ptr(client.OrderRequestProductTypeINTRADAY),
			Price:           ptr(float32(0)),
		}

		order, err := dhanClient.PlaceorderWithResponse(ctx, nil, orderReq)
		if err != nil {
			log.Fatalf("Error placing order: %v", err)
		}

		if order.StatusCode() == 200 && order.JSON200 != nil {
			fmt.Printf("Order placed successfully: %s\n", *order.JSON200.Data.OrderId)
		} else {
			fmt.Printf("API returned status: %d\n", order.StatusCode())
		}
	*/

	fmt.Println("\nAll operations completed successfully!")
}

// Helper function to create pointers
func ptr[T any](v T) *T {
	return &v
}
