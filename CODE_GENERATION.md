# Code Generation SOP

This document describes the Standard Operating Procedure (SOP) for generating and updating the Dhan Go SDK client code from the OpenAPI specification.

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Quick Regeneration](#quick-regeneration)
4. [Updating OpenAPI Spec](#updating-openapi-spec)
5. [Verification Checklist](#verification-checklist)
6. [Troubleshooting](#troubleshooting)
7. [Understanding the Generation](#understanding-the-generation)

---

## Overview

The Dhan Go SDK uses [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) to automatically generate type-safe client code from the official Dhan v2 OpenAPI 3.0.1 specification.

**What gets generated:**
- All type definitions (request/response models)
- Client struct with all API methods
- Type-safe request builders
- Response wrapper types

**File**: `client/generated.go` (~8,941 lines)

**Source**: `openapi.json` (Dhan v2 OpenAPI spec)

---

## Prerequisites

Ensure you have:

1. **Go 1.21 or higher**
   ```bash
   go version
   ```

2. **oapi-codegen** (automatically installed via `go generate`)
   - No manual installation needed
   - Specified in `tools.go`

3. **OpenAPI specification**
   - File: `openapi.json`
   - Must be valid OpenAPI 3.0.x format

---

## Quick Regeneration

### Step 1: Run Generation

```bash
# From project root
go generate ./...
```

This command:
1. Reads `tools.go`
2. Downloads oapi-codegen v2 (if not cached)
3. Reads `openapi.json`
4. Generates `client/generated.go`

**Expected output:**
```
# github.com/samarthkathal/dhan-go
```

No output means success!

### Step 2: Verify Compilation

```bash
# Build all packages
go build ./...
```

Should complete without errors.

### Step 3: Run Examples

```bash
# Test basic example compiles
cd examples/01_basic
go build .

cd ../02_with_middleware
go build .

cd ../03_graceful_shutdown
go build .

cd ../04_all_features
go build .
```

All examples should compile successfully.

---

## Updating OpenAPI Spec

### When to Update

- Dhan releases new API endpoints
- Existing endpoints change (new fields, different types)
- Bug fixes in the official spec

### How to Update

#### Option 1: Download from Dhan

1. Visit Dhan API documentation
2. Download latest OpenAPI spec
3. Replace `openapi.json`
4. Run generation

```bash
# Backup current spec
cp openapi.json openapi.json.backup

# Download new spec (example URL)
curl -o openapi.json https://api.dhan.co/v2/openapi.json

# Generate
go generate ./...

# Verify
go build ./...
```

#### Option 2: Manual Edit

If you need to fix issues in the spec:

1. Edit `openapi.json`
2. Validate JSON syntax
   ```bash
   python3 -m json.tool openapi.json > /dev/null
   ```
3. Regenerate
   ```bash
   go generate ./...
   ```

### Validation

After updating the spec, verify:

```bash
# 1. JSON is valid
python3 -m json.tool openapi.json > /dev/null

# 2. Generation succeeds
go generate ./...

# 3. Code compiles
go build ./...

# 4. Examples compile
cd examples/01_basic && go build .
cd ../02_with_middleware && go build .
cd ../03_graceful_shutdown && go build .
cd ../04_all_features && go build .
```

---

## Verification Checklist

After regenerating client code, complete this checklist:

### 1. Generation Succeeded

```bash
go generate ./...
# Should complete without errors
```

### 2. File Generated

```bash
ls -lh client/generated.go
# Should show file exists (~500KB+)
```

### 3. Code Compiles

```bash
go build ./...
# Should complete without errors
```

### 4. Package Imports

```bash
go list -m all | grep oapi-codegen
# Should show oapi-codegen dependencies
```

### 5. Examples Compile

```bash
# All examples should compile
for dir in examples/*/; do
    echo "Building $dir"
    (cd "$dir" && go build .) || echo "FAILED: $dir"
done
```

### 6. Run Tests (if any)

```bash
go test ./...
```

### 7. Check Generated Methods

```bash
# Count generated methods (should be ~31 endpoints)
grep -c "WithResponse(ctx context.Context" client/generated.go
```

### 8. Spot Check

Open `client/generated.go` and verify:
- Package declaration: `package client`
- Imports look correct
- No obvious syntax errors
- Methods exist (e.g., `GetpositionsWithResponse`)

---

## Troubleshooting

### Issue: "command not found: oapi-codegen"

**Cause**: Go can't find oapi-codegen

**Solution**:
```bash
# Clean Go cache
go clean -modcache

# Try again
go generate ./...
```

### Issue: "undefined: OpenAPI types"

**Cause**: Missing import in generated code

**Solution**: Check if `openapi.json` is valid:
```bash
python3 -m json.tool openapi.json > /dev/null
```

If invalid, fix JSON syntax and regenerate.

### Issue: "duplicate type name"

**Cause**: OpenAPI spec has naming conflicts

**Solution**: The generation command uses `-response-type-suffix=Result` to avoid this. If still occurring:

1. Check `tools.go` has the flag:
   ```go
   -response-type-suffix=Result
   ```

2. If needed, add more specific suffix:
   ```go
   -response-type-suffix=Response
   ```

### Issue: Examples don't compile after regeneration

**Cause**: API signatures changed

**Solution**:
1. Check what changed:
   ```bash
   git diff client/generated.go
   ```

2. Update examples to match new signatures

3. Common changes:
   - New required fields in requests
   - Field type changes (string → int)
   - New enum values

### Issue: "too many API methods"

**Cause**: Dhan added new endpoints

**Solution**: This is expected!
- New endpoints are available automatically
- Update examples if you want to showcase them
- Update `USAGE_GUIDE.md` with new methods

---

## Understanding the Generation

### Generation Command

Located in `tools.go`:

```go
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest -generate types,skip-prune,client -package client -response-type-suffix=Result -o client/generated.go openapi.json
```

**Breakdown:**

- `go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest`
  - Download and run oapi-codegen v2 (latest)

- `-generate types,skip-prune,client`
  - Generate type definitions
  - Don't prune unused types
  - Generate client code

- `-package client`
  - Generated code goes in `client` package

- `-response-type-suffix=Result`
  - Append "Result" to response wrapper types
  - Prevents naming conflicts (e.g., `GetpositionsResult` vs `Position` model)

- `-o client/generated.go`
  - Output file

- `openapi.json`
  - Input OpenAPI specification

### What Gets Generated

#### 1. Type Definitions

All request/response models from OpenAPI `components.schemas`:

```go
type OrderRequest struct {
    SecurityId      *string                       `json:"securityId,omitempty"`
    ExchangeSegment OrderRequestExchangeSegment   `json:"exchangeSegment"`
    TransactionType OrderRequestTransactionType   `json:"transactionType"`
    // ...
}
```

#### 2. Client Struct

```go
type ClientWithResponses struct {
    // ...
}
```

#### 3. API Methods

For each endpoint in the spec:

```go
func (c *ClientWithResponses) GetpositionsWithResponse(
    ctx context.Context,
    params *GetpositionsParams,
    reqEditors ...RequestEditorFn,
) (*GetpositionsResult, error) {
    // ...
}
```

#### 4. Response Wrappers

```go
type GetpositionsResult struct {
    Body         []byte
    HTTPResponse *http.Response
    JSON200      *PositionResponse
    // ...
}

func (r GetpositionsResult) StatusCode() int {
    // ...
}
```

#### 5. Constructor

```go
func NewClientWithResponses(
    server string,
    opts ...ClientOption,
) (*ClientWithResponses, error) {
    // ...
}
```

#### 6. Options

```go
func WithHTTPClient(doer HttpRequestDoer) ClientOption {
    // ...
}

func WithRequestEditorFn(fn RequestEditorFn) ClientOption {
    // ...
}

func WithBaseURL(baseURL string) ClientOption {
    // ...
}
```

---

## Development Workflow

### Making Changes

1. **Don't edit `client/generated.go`** - It gets overwritten!
2. **Do edit**:
   - `utils/*.go` - Custom utilities
   - `examples/*/main.go` - Examples
   - Documentation files

### Before Committing

```bash
# 1. Ensure generated code is up to date
go generate ./...

# 2. Format code
go fmt ./...

# 3. Verify build
go build ./...

# 4. Run tests
go test ./...

# 5. Verify examples
for dir in examples/*/; do
    (cd "$dir" && go build .)
done
```

### Git Workflow

```bash
# Check what changed
git status

# If generated code changed:
git diff client/generated.go

# Add files
git add .

# Commit
git commit -m "Regenerate client from updated OpenAPI spec"
```

---

## Maintenance Schedule

### Regular Checks

**Weekly**:
- Check Dhan API documentation for updates
- Review Dhan's release notes

**Monthly**:
- Re-download OpenAPI spec
- Regenerate client
- Test all examples
- Update documentation if needed

**When Issues Reported**:
- Check if regeneration fixes the issue
- Compare spec versions
- Update examples if API changed

---

## Reference

### Useful Commands

```bash
# Generate client
go generate ./...

# Check generated file
wc -l client/generated.go

# List all API methods
grep "WithResponse(ctx context.Context" client/generated.go

# Find specific method
grep -i "placeorder" client/generated.go

# Validate OpenAPI spec
python3 -m json.tool openapi.json > /dev/null

# Clean and rebuild
go clean -cache
go build ./...
```

### Files Involved

- `tools.go` - Generation directive
- `openapi.json` - OpenAPI specification (input)
- `client/generated.go` - Generated client (output)
- `go.mod` - Dependencies (oapi-codegen)

### External Links

- [oapi-codegen Documentation](https://github.com/oapi-codegen/oapi-codegen)
- [Dhan API Documentation](https://dhanhq.co/docs/v2/)
- [OpenAPI Specification](https://swagger.io/specification/)

---

## Quick Reference Card

```
┌────────────────────────────────────────────────────────┐
│  QUICK REFERENCE: CODE GENERATION                      │
├────────────────────────────────────────────────────────┤
│                                                        │
│  Generate:     go generate ./...                       │
│  Build:        go build ./...                          │
│  Test:         go test ./...                           │
│                                                        │
│  Input:        openapi.json                            │
│  Output:       client/generated.go                     │
│  Config:       tools.go                                │
│                                                        │
│  DON'T EDIT:   client/generated.go                     │
│  DO EDIT:      utils/*.go, examples/*, docs/*.md       │
│                                                        │
└────────────────────────────────────────────────────────┘
```

---

**For questions, see [USAGE_GUIDE.md](USAGE_GUIDE.md) or create an issue on GitHub.**
