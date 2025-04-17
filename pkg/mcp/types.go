package mcp

import (
	"encoding/json"
)

// MethodPing is the method name for the ping request.
const MethodPing = "ping"

// JSONRPCVersion is the fixed JSON-RPC version string.
const JSONRPCVersion = "2.0"

// RequestID represents the ID field in a JSON-RPC request/response, which can be a string or number.
type RequestID interface{}

// RPCRequest defines the structure for a JSON-RPC request.
type RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      RequestID   `json:"id"`
}

// RPCResponse defines the structure for a JSON-RPC response.
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	ID      RequestID       `json:"id"`
}

// Role defines the sender or recipient of messages and data.
type Role string

const (
	RoleAssistant Role = "assistant"
	RoleUser      Role = "user"
)

// Annotations provide optional metadata for client interpretation.
type Annotations struct {
	// Audience describes the intended customer (e.g., "user", "assistant").
	Audience []Role `json:"audience,omitempty"`
	// Priority indicates importance (1=most important, 0=least important).
	Priority *float64 `json:"priority,omitempty"` // Use pointer for optional 0 value
}
