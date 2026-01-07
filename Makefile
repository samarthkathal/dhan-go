.PHONY: all setup tools generate generate-openapi generate-easyjson build build-race \
        test test-v test-race test-cover bench bench-json bench-marketfeed bench-fulldepth \
        bench-cpu bench-mem vet fmt lint clean examples help

# Default target
all: generate build test

# === SETUP ===

# Install all development tools
setup: tools
	go mod download
	go mod tidy

# Install code generation and dev tools
tools:
	go install github.com/mailru/easyjson/easyjson@latest
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@echo "Tools installed successfully"

# === CODE GENERATION ===

# Run all code generators (OpenAPI + easyjson)
generate: generate-openapi generate-easyjson
	@echo "All code generation complete"

# Generate OpenAPI REST client from openapi.json
generate-openapi:
	go generate ./...
	@echo "OpenAPI client generated: internal/restgen/client.go"

# Generate easyjson marshalers
generate-easyjson:
	cd orderupdate && easyjson -all types.go
	cd rest && easyjson -all types.go
	@echo "easyjson marshalers generated"

# === BUILD ===

# Build all packages
build:
	go build ./...

# Build with race detector
build-race:
	go build -race ./...

# === TESTING ===

# Run all tests
test:
	go test ./...

# Run tests with verbose output
test-v:
	go test -v ./...

# Run tests with race detector
test-race:
	go test -race ./...

# Run tests with coverage
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# === BENCHMARKS ===

# Run all benchmarks
bench:
	go test -bench=. -benchmem ./benchmarks/

# Run JSON parsing benchmarks only
bench-json:
	go test -bench=Unmarshal -benchmem ./benchmarks/

# Run marketfeed benchmarks
bench-marketfeed:
	go test -bench=. -benchmem ./benchmarks/ -run=^$$ -bench='With.*Data'

# Run fulldepth benchmarks
bench-fulldepth:
	go test -bench=. -benchmem ./benchmarks/ -run=^$$ -bench='Depth'

# Run benchmarks with CPU profiling
bench-cpu:
	go test -bench=. -benchmem -cpuprofile=cpu.prof ./benchmarks/
	@echo "CPU profile: cpu.prof (use 'go tool pprof cpu.prof')"

# Run benchmarks with memory profiling
bench-mem:
	go test -bench=. -benchmem -memprofile=mem.prof ./benchmarks/
	@echo "Memory profile: mem.prof (use 'go tool pprof mem.prof')"

# === CODE QUALITY ===

# Run go vet
vet:
	go vet ./...

# Format code
fmt:
	go fmt ./...

# Run all linters (requires golangci-lint)
lint:
	@which golangci-lint > /dev/null || (echo "Install golangci-lint: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

# === CLEANUP ===

# Clean build artifacts
clean:
	rm -f coverage.out coverage.html
	rm -f cpu.prof mem.prof
	go clean ./...

# === EXAMPLES ===

# Build all examples
examples:
	go build ./examples/...

# === HELP ===

help:
	@echo "Dhan Go SDK - Available targets:"
	@echo ""
	@echo "  Setup & Tools:"
	@echo "    make setup             - Install dependencies and tools"
	@echo "    make tools             - Install dev tools (easyjson, oapi-codegen)"
	@echo ""
	@echo "  Code Generation:"
	@echo "    make generate          - Run all code generators"
	@echo "    make generate-openapi  - Generate OpenAPI REST client"
	@echo "    make generate-easyjson - Generate easyjson marshalers"
	@echo ""
	@echo "  Build:"
	@echo "    make build             - Build all packages"
	@echo "    make build-race        - Build with race detector"
	@echo ""
	@echo "  Testing:"
	@echo "    make test              - Run all tests"
	@echo "    make test-v            - Run tests (verbose)"
	@echo "    make test-race         - Run tests with race detector"
	@echo "    make test-cover        - Run tests with coverage report"
	@echo ""
	@echo "  Benchmarks:"
	@echo "    make bench             - Run all benchmarks"
	@echo "    make bench-json        - Run JSON parsing benchmarks"
	@echo "    make bench-marketfeed  - Run marketfeed benchmarks"
	@echo "    make bench-fulldepth   - Run fulldepth benchmarks"
	@echo "    make bench-cpu         - Run benchmarks with CPU profiling"
	@echo "    make bench-mem         - Run benchmarks with memory profiling"
	@echo ""
	@echo "  Code Quality:"
	@echo "    make fmt               - Format code"
	@echo "    make vet               - Run go vet"
	@echo "    make lint              - Run golangci-lint"
	@echo ""
	@echo "  Cleanup:"
	@echo "    make clean             - Remove build artifacts"
