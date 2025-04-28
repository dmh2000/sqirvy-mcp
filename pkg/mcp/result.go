package mcp

import (
	"encoding/json"
	"fmt"
	utils "sqirvy-mcp/pkg/utils"
)

const protocolVersion = "2024-11-05"
const serverName = "sqirvy-mcp"
const serverVersion = "0.1.0"

// sendResponse marshals a successful result into a full RPCResponse and sends it.
// Returns the marshalled bytes and any error during marshalling.
// It does *not* send the bytes itself.
func MarshalResponse(id RequestID, result interface{}, logger *utils.Logger) ([]byte, error) {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		err = fmt.Errorf("failed to marshal result for response ID %v: %w", id, err)
		logger.Println("DEBUG", err.Error())
		// Return bytes for an internal error instead
		rpcErr := NewRPCError(ErrorCodeInternalError, "Failed to marshal response result", nil)
		errorBytes, marshalErr := MarshalErrorResponse(id, rpcErr)
		// If we can't even marshal the error, return the original error and nil bytes
		if marshalErr != nil {
			logger.Printf("DEBUG", "CRITICAL: Failed to marshal error response for result marshalling failure: %v", marshalErr)
			return nil, err // Return the original marshalling error
		}
		return errorBytes, err // Return the marshalled error bytes and the original error
	}

	resp := RPCResponse{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  resultBytes,
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		// This is highly unlikely if result marshalling worked, but handle defensively
		err = fmt.Errorf("failed to marshal final response object for ID %v: %w", id, err)
		logger.Println("DEBUG", err.Error())
		// Return bytes for an internal error instead
		rpcErr := NewRPCError(ErrorCodeInternalError, "Failed to marshal final response object", nil)
		errorBytes, marshalErr := MarshalErrorResponse(id, rpcErr)
		if marshalErr != nil {
			logger.Printf("DEBUG", "CRITICAL: Failed to marshal error response for final response marshalling failure: %v", marshalErr)
			return nil, err // Return the original marshalling error
		}
		return errorBytes, err // Return the marshalled error bytes and the original error
	}

	return respBytes, nil // Return marshalled success response bytes and nil error
}
