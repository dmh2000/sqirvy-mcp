package mcp

import (
	"encoding/json"
	"fmt"
)

// Standard JSON-RPC 2.0 Error codes
// See: https://www.jsonrpc.org/specification#error_object
const (
	// ErrorCodeParseError indicates invalid JSON was received by the server.
	// An error occurred on the server while parsing the JSON text.
	ErrorCodeParseError int = -32700
	// ErrorCodeInvalidRequest indicates the JSON sent is not a valid Request object.
	ErrorCodeInvalidRequest int = -32600
	// ErrorCodeMethodNotFound indicates the method does not exist / is not available.
	ErrorCodeMethodNotFound int = -32601
	// ErrorCodeInvalidParams indicates invalid method parameter(s).
	ErrorCodeInvalidParams int = -32602
	// ErrorCodeInternalError indicates an internal JSON-RPC error.
	ErrorCodeInternalError int = -32603
	// -32000 to -32099 are reserved for implementation-defined server-errors.
)

// RPCError defines the structure for a JSON-RPC error object, according to the spec.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"` // Optional field for additional error info
}

// Error implements the standard Go error interface for RPCError.
func (e *RPCError) Error() string {
	return fmt.Sprintf("RPC error %d: %s", e.Code, e.Message)
}

// NewRPCError creates a new RPCError instance.
func NewRPCError(code int, message string, data interface{}) *RPCError {
	return &RPCError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// MarshalErrorResponse creates a JSON-RPC error response.
// The id should match the id of the request that caused the error.
// If the request ID cannot be determined (e.g., due to parse error), id should be nil.
func MarshalErrorResponse(id RequestID, rpcErr *RPCError) ([]byte, error) {
	resp := RPCResponse{
		JSONRPC: JSONRPCVersion,
		Error:   rpcErr,
		ID:      id, // Can be nil if request ID is unknown
	}
	return json.Marshal(resp)
}

// UnmarshalErrorResponse attempts to parse a JSON-RPC error response.
// It returns the RPCError details and the response ID if successful.
// Returns nil error if parsing is successful, even if the response isn't an error response.
// Check if the returned *RPCError is non-nil to confirm it's an error response.
func UnmarshalErrorResponse(data []byte) (*RPCError, RequestID, error) {
	var resp RPCResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		// If we can't even unmarshal the basic response structure, return a parse error.
		// We might not know the ID in this case.
		parseErr := NewRPCError(ErrorCodeParseError, fmt.Sprintf("Failed to parse JSON response: %v", err), nil)
		return parseErr, nil, fmt.Errorf("failed to unmarshal RPC response structure: %w", err)
	}

	// Return the error details (which might be nil if it wasn't an error response)
	// and the ID from the parsed response.
	return resp.Error, resp.ID, nil
}
