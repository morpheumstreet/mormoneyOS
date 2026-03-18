// Package tools provides MCP tool adapters that bridge to the existing Tool Registry.
// Each tool file (marketplace/search.go, marketplace/install.go, etc.) is a thin
// wrapper that calls the corresponding use case — DRY, SRP, no duplication.
package tools
