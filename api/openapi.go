// Package api contains the API's embedded OpenAPI contract.
package api

import _ "embed"

// OpenAPI is the complete OpenAPI document served by the application.
//
//go:embed openapi.yaml
var OpenAPI []byte
