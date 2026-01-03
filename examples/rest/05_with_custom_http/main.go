// Package main demonstrates REST client with custom HTTP configuration.
//
// This example shows:
// - Creating a custom HTTP transport with connection pooling
// - Configuring timeouts for different use cases
// - Using the custom HTTP client with the REST client
//
// Configuration options:
// - MaxIdleConns: Maximum idle connections across all hosts
// - MaxIdleConnsPerHost: Maximum idle connections per host
// - MaxConnsPerHost: Maximum total connections per host
// - IdleConnTimeout: How long idle connections stay open
// - TLSHandshakeTimeout: TLS handshake timeout
// - ResponseHeaderTimeout: Time to wait for response headers
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
	"net/http"
	"os"
	"time"

	"github.com/samarthkathal/dhan-go/rest"
)

func main() {
	accessToken := os.Getenv("DHAN_ACCESS_TOKEN")
	if accessToken == "" {
		log.Fatal("DHAN_ACCESS_TOKEN environment variable not set")
	}

	ctx := context.Background()

	// Create custom HTTP transport for connection pooling
	transport := &http.Transport{
		// Connection pool settings
		MaxIdleConns:        100,              // Max idle connections across all hosts
		MaxIdleConnsPerHost: 10,               // Max idle connections per host
		MaxConnsPerHost:     10,               // Max total connections per host (0 = unlimited)
		IdleConnTimeout:     90 * time.Second, // Keep idle connections for 90s

		// TLS settings
		TLSHandshakeTimeout: 10 * time.Second,

		// Response settings
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	// Create HTTP client with custom transport
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second, // Overall request timeout
	}

	fmt.Println("Custom HTTP Configuration:")
	fmt.Printf("  MaxIdleConns:          %d\n", transport.MaxIdleConns)
	fmt.Printf("  MaxIdleConnsPerHost:   %d\n", transport.MaxIdleConnsPerHost)
	fmt.Printf("  MaxConnsPerHost:       %d\n", transport.MaxConnsPerHost)
	fmt.Printf("  IdleConnTimeout:       %v\n", transport.IdleConnTimeout)
	fmt.Printf("  TLSHandshakeTimeout:   %v\n", transport.TLSHandshakeTimeout)
	fmt.Printf("  ResponseHeaderTimeout: %v\n", transport.ResponseHeaderTimeout)
	fmt.Printf("  Overall Timeout:       %v\n", httpClient.Timeout)
	fmt.Println()

	// Create REST client with custom HTTP client
	client, err := rest.NewClient(
		"https://api.dhan.co",
		accessToken,
		httpClient,
	)
	if err != nil {
		log.Fatalf("Failed to create REST client: %v", err)
	}

	fmt.Println("REST client created with custom HTTP configuration")
	fmt.Println()

	// Benchmark: Make multiple requests to test connection pooling
	fmt.Println("Testing connection pooling with multiple requests...")
	iterations := 5
	var totalDuration time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()

		_, err := client.GetHoldings(ctx)
		if err != nil {
			log.Printf("Request %d error: %v", i+1, err)
			continue
		}

		duration := time.Since(start)
		totalDuration += duration
		fmt.Printf("  Request %d: %v\n", i+1, duration)
	}

	avgLatency := totalDuration / time.Duration(iterations)
	fmt.Printf("\nAverage latency: %v\n", avgLatency)
	fmt.Println()

	// Low-latency configuration example
	fmt.Println("Low-Latency Configuration (for HFT/Market Making):")
	lowLatencyTransport := &http.Transport{
		MaxIdleConns:          50,
		MaxIdleConnsPerHost:   5,
		MaxConnsPerHost:       5,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	}
	fmt.Printf("  MaxIdleConns:          %d (reduced)\n", lowLatencyTransport.MaxIdleConns)
	fmt.Printf("  MaxConnsPerHost:       %d (focused)\n", lowLatencyTransport.MaxConnsPerHost)
	fmt.Printf("  TLSHandshakeTimeout:   %v (shorter)\n", lowLatencyTransport.TLSHandshakeTimeout)
	fmt.Printf("  ResponseHeaderTimeout: %v (shorter)\n", lowLatencyTransport.ResponseHeaderTimeout)
	fmt.Println()

	// High-throughput configuration example
	fmt.Println("High-Throughput Configuration (for batch operations):")
	highThroughputTransport := &http.Transport{
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   20,
		MaxConnsPerHost:       20,
		IdleConnTimeout:       120 * time.Second,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
	fmt.Printf("  MaxIdleConns:          %d (increased)\n", highThroughputTransport.MaxIdleConns)
	fmt.Printf("  MaxConnsPerHost:       %d (increased)\n", highThroughputTransport.MaxConnsPerHost)
	fmt.Printf("  IdleConnTimeout:       %v (longer)\n", highThroughputTransport.IdleConnTimeout)
	fmt.Println()

	fmt.Println("Configuration Guidelines:")
	fmt.Println("  - Low-latency: Fewer connections, shorter timeouts")
	fmt.Println("  - High-throughput: More connections, longer timeouts")
	fmt.Println("  - Balance based on your use case and network conditions")
}
