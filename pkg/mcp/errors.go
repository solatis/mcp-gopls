package mcp

import (
	"errors"
)

// Erreurs communes du package MCP
var (
	ErrLSPClientNotInitialized = errors.New("LSP client not initialized")
	ErrInvalidRequest          = errors.New("invalid request format")
	ErrUnsupportedFeature      = errors.New("unsupported feature")
)
