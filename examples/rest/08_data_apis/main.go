// Package main demonstrates Data API usage with the Dhan Go SDK.
//
// This example shows:
// - Historical Data (daily OHLC)
// - Intraday Data (minute OHLC)
// - Market Quote (LTP, OHLC, full quote)
// - Option Chain
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

func main() {
	// Get access token from environment
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("DHAN_ACCESS_TOKEN environment variable not set")
	}

	ctx := context.Background()

	// Create REST client
	client, err := rest.NewClient(
		"https://api.dhan.co/v2",
		accessToken,
		nil, // Use default HTTP client
	)
	if err != nil {
		log.Fatalf("Failed to create REST client: %v", err)
	}

	fmt.Println("REST client created successfully")
	fmt.Println()

	// Example 1: Get LTP (Last Traded Price)
	fmt.Println("=== Market Quote: LTP ===")
	ltpReq := rest.MarketQuoteRequest{
		"NSE_EQ": {11536}, // TCS
	}
	ltpResp, err := client.GetLTP(ctx, ltpReq)
	if err != nil {
		log.Printf("Error fetching LTP: %v", err)
	} else {
		fmt.Printf("LTP Response: %+v\n", ltpResp)
	}
	fmt.Println()

	// Example 2: Get OHLC
	fmt.Println("=== Market Quote: OHLC ===")
	ohlcReq := rest.MarketQuoteRequest{
		"NSE_EQ": {11536}, // TCS
	}
	ohlcResp, err := client.GetOHLC(ctx, ohlcReq)
	if err != nil {
		log.Printf("Error fetching OHLC: %v", err)
	} else {
		fmt.Printf("OHLC Response: %+v\n", ohlcResp)
	}
	fmt.Println()

	// Example 3: Get Full Quote (includes market depth)
	fmt.Println("=== Market Quote: Full Quote ===")
	quoteReq := rest.MarketQuoteRequest{
		"NSE_EQ": {11536}, // TCS
	}
	quoteResp, err := client.GetQuote(ctx, quoteReq)
	if err != nil {
		log.Printf("Error fetching quote: %v", err)
	} else {
		fmt.Printf("Quote Response: %+v\n", quoteResp)
	}
	fmt.Println()

	// Example 4: Get Option Chain
	fmt.Println("=== Option Chain ===")
	// NIFTY 50 Index (IDX_I segment, security ID 13)
	optionChain, err := client.GetOptionChain(ctx, 13, "IDX_I", "2025-01-30")
	if err != nil {
		log.Printf("Error fetching option chain: %v", err)
	} else {
		fmt.Printf("Option Chain Last Price: %.2f\n", optionChain.Data.LastPrice)
		fmt.Printf("Number of strikes: %d\n", len(optionChain.Data.OC))
	}
	fmt.Println()

	// Example 5: Get Expiry List
	fmt.Println("=== Expiry List ===")
	expiryList, err := client.GetExpiryList(ctx, 13, "IDX_I")
	if err != nil {
		log.Printf("Error fetching expiry list: %v", err)
	} else {
		fmt.Printf("Available expiries: %d\n", len(expiryList.Data))
		for i, exp := range expiryList.Data {
			if i >= 5 {
				fmt.Printf("  ... and %d more\n", len(expiryList.Data)-5)
				break
			}
			fmt.Printf("  - %s\n", exp)
		}
	}
	fmt.Println()

	// Example 6: Historical Data (Daily OHLC)
	fmt.Println("=== Historical Data (Daily) ===")
	exchangeSegment := restgen.HistoricalChartsRequestExchangeSegmentNSEEQ
	instrument := restgen.HistoricalChartsRequestInstrumentEQUITY
	securityID := "11536" // TCS
	oi := false

	historicalReq := restgen.HistoricalchartsJSONRequestBody{
		SecurityId:      &securityID,
		ExchangeSegment: &exchangeSegment,
		Instrument:      &instrument,
		Oi:              &oi,
	}
	historicalResp, err := client.GetHistoricalData(ctx, historicalReq)
	if err != nil {
		log.Printf("Error fetching historical data: %v", err)
	} else if historicalResp.JSON200 != nil {
		data := historicalResp.JSON200
		if data.Close != nil && len(*data.Close) > 0 {
			fmt.Printf("Historical data points: %d\n", len(*data.Close))
			fmt.Printf("Latest close: %.2f\n", (*data.Close)[len(*data.Close)-1])
		}
	}
	fmt.Println()

	// Example 7: Intraday Data (Minute OHLC)
	fmt.Println("=== Intraday Data (Minute) ===")
	intradayExchange := restgen.IntradayChartsRequestExchangeSegmentNSEEQ
	intradayInstrument := restgen.IntradayChartsRequestInstrumentEQUITY
	interval := restgen.IntradayChartsRequestIntervalN5 // 5-minute candles

	intradayReq := restgen.IntradaychartsJSONRequestBody{
		SecurityId:      &securityID,
		ExchangeSegment: &intradayExchange,
		Instrument:      &intradayInstrument,
		Interval:        &interval,
		Oi:              &oi,
	}
	intradayResp, err := client.GetIntradayData(ctx, intradayReq)
	if err != nil {
		log.Printf("Error fetching intraday data: %v", err)
	} else if intradayResp.JSON200 != nil {
		data := intradayResp.JSON200
		if data.Close != nil && len(*data.Close) > 0 {
			fmt.Printf("Intraday data points: %d\n", len(*data.Close))
			fmt.Printf("Latest close: %.2f\n", (*data.Close)[len(*data.Close)-1])
		}
	}
	fmt.Println()

	fmt.Println("Data API examples completed")
}
