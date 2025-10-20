// Package dhan provides tools for code generation
package dhan

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest -generate types,skip-prune,client -package client -response-type-suffix=Result -o rest/client/client.go openapi.json
