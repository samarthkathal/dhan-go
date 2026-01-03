// Package dhan provides tools for code generation
package dhan

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest -generate types,skip-prune,client -package restgen -response-type-suffix=Result -o internal/restgen/client.go openapi.json
