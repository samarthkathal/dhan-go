// Package main demonstrates all MarketFeed data types and callbacks.
//
// This example shows:
// - Ticker callback (LTP + Last Traded Time)
// - Quote callback (OHLC, volume, buy/sell quantities)
// - OI callback (Open Interest for derivatives)
// - PrevClose callback (Previous close price)
// - Full callback (Complete data with 5-level market depth)
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
)

func main() {
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("DHAN_ACCESS_TOKEN environment variable not set")
	}

	fmt.Println("MarketFeed All Data Types Example")
	fmt.Println()

	// Create client with ALL callback types
	client, err := marketfeed.NewClient(
		accessToken,

		// Ticker callback: LTP + Last Traded Time (16 bytes)
		marketfeed.WithTickerCallback(func(data *marketfeed.TickerData) {
			fmt.Printf("TICKER | ID: %d | LTP: %.2f | Time: %v\n",
				data.Header.SecurityID,
				data.LastTradedPrice,
				data.GetTradeTime().Format("15:04:05"))
		}),

		// Quote callback: Complete trade data (50 bytes)
		marketfeed.WithQuoteCallback(func(data *marketfeed.QuoteData) {
			fmt.Printf("QUOTE  | ID: %d | O: %.2f H: %.2f L: %.2f C: %.2f | Vol: %d\n",
				data.Header.SecurityID,
				data.DayOpen,
				data.DayHigh,
				data.DayLow,
				data.DayClose,
				data.Volume)
			fmt.Printf("       | LTP: %.2f | Avg: %.2f | Buy: %d | Sell: %d\n",
				data.LastTradedPrice,
				data.AverageTradedPrice,
				data.TotalBuyQuantity,
				data.TotalSellQuantity)
			fmt.Printf("       | Change: %.2f (%.2f%%)\n",
				data.GetDayChange(),
				data.GetDayChangePercent())
		}),

		// OI callback: Open Interest (12 bytes)
		marketfeed.WithOICallback(func(data *marketfeed.OIData) {
			fmt.Printf("OI     | ID: %d | Exchange: %s | OI: %d\n",
				data.Header.SecurityID,
				data.GetExchangeName(),
				data.OpenInterest)
		}),

		// PrevClose callback: Previous close reference (16 bytes)
		marketfeed.WithPrevCloseCallback(func(data *marketfeed.PrevCloseData) {
			fmt.Printf("PREV   | ID: %d | Exchange: %s | Close: %.2f | Prev OI: %d\n",
				data.Header.SecurityID,
				data.GetExchangeName(),
				data.PreviousClosePrice,
				data.PreviousOpenInterest)
		}),

		// Full callback: Complete data + market depth (162 bytes)
		marketfeed.WithFullCallback(func(data *marketfeed.FullData) {
			fmt.Printf("FULL   | ID: %d | LTP: %.2f | OI: %d\n",
				data.Header.SecurityID,
				data.LastTradedPrice,
				data.OpenInterest)

			// Market depth (5 levels)
			fmt.Println("       | Market Depth:")
			fmt.Println("       | BID                     ASK")
			for i := 0; i < 5; i++ {
				depth := data.Depth[i]
				fmt.Printf("       | %.2f x %5d    %.2f x %5d\n",
					depth.BidPrice, depth.BidQuantity,
					depth.AskPrice, depth.AskQuantity)
			}

			// Best bid/ask helpers
			bidPrice, bidQty := data.GetBestBid()
			askPrice, askQty := data.GetBestAsk()
			spread := data.GetSpread()
			fmt.Printf("       | Best Bid: %.2f x %d | Best Ask: %.2f x %d | Spread: %.2f\n",
				bidPrice, bidQty, askPrice, askQty, spread)
			fmt.Println()
		}),

		// Error callback
		marketfeed.WithErrorCallback(func(err error) {
			log.Printf("ERROR  | %v", err)
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	fmt.Println("MarketFeed client created with ALL callbacks")
	fmt.Println()
	fmt.Println("Data Types:")
	fmt.Println("  - Ticker: LTP updates (16 bytes)")
	fmt.Println("  - Quote: OHLC, volume (50 bytes)")
	fmt.Println("  - OI: Open interest (12 bytes)")
	fmt.Println("  - PrevClose: Reference prices (16 bytes)")
	fmt.Println("  - Full: Complete with depth (162 bytes)")
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

	// Subscribe to instruments
	instruments := []marketfeed.Instrument{
		{SecurityID: "1333", ExchangeSegment: marketfeed.ExchangeNSEEQ}, // TCS
	}

	fmt.Println("Subscribing to TCS (1333)...")
	if err := client.Subscribe(context.Background(), instruments); err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	fmt.Println("Subscribed")
	fmt.Println()
	fmt.Println("Receiving all data types... (Press Ctrl+C to stop)")
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
