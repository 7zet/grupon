// Package api owns repository-level Go generators for the API service.
package api

//go:generate go tool oapi-codegen -config internal/http/gen/oapi-codegen.yaml ../../packages/api-spec/openapi.yaml
//go:generate go tool sqlc generate -f ../../sqlc.yaml
