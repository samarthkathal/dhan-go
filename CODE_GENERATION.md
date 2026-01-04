# Code Generation

The Dhan Go SDK uses [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) to generate client code from the OpenAPI specification.

## Quick Reference

| Item | Path |
|------|------|
| Input | `openapi.json` |
| Output | `internal/restgen/client.go` |
| Config | `tools.go` |

## Regenerate Client

```bash
# Generate
go generate ./...

# Verify
go build ./...
```

## Generation Command

From `tools.go`:

```go
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest \
  -generate types,skip-prune,client \
  -package restgen \
  -response-type-suffix=Result \
  -o internal/restgen/client.go \
  openapi.json
```

## Update OpenAPI Spec

```bash
# Backup and download new spec
cp openapi.json openapi.json.backup
curl -o openapi.json https://api.dhan.co/v2/openapi.json

# Regenerate and verify
go generate ./...
go build ./...
```

## What's Generated

- **Types**: All request/response models from OpenAPI schemas
- **Client**: `ClientWithResponses` struct with all API methods
- **Response wrappers**: `*Result` types with status codes and parsed JSON

## Important Notes

- **Don't edit** `internal/restgen/client.go` - it gets overwritten
- **Do edit** `rest/*.go` (wrapper), `middleware/*.go`, `examples/*`
- Package name is `restgen`, not `client`
- Response types use `Result` suffix (e.g., `GetpositionsResult`)

## Troubleshooting

| Issue | Solution |
|-------|----------|
| oapi-codegen not found | `go clean -modcache && go generate ./...` |
| Invalid JSON | `python3 -m json.tool openapi.json > /dev/null` |
| Examples don't compile | Check `git diff internal/restgen/` for API changes |
