package main

import (
	"encoding/json"
	"fmt"
	"time"

	tools "sqirvy-mcp/cmd/sqirvy-mcp/tools"
	mcp "sqirvy-mcp/pkg/mcp"
)

const (
	onlineTimeout  = 5 * time.Second // Timeout for the online command
	onlineToolName = "online"
)

// handleOnlineTool handles the "tools/call" request specifically for the "online" tool.
// It executes the online command and returns the result or an error.
func (s *Server) handleOnlineTool(id mcp.RequestID, params mcp.CallToolParams) ([]byte, error) {
	s.logger.Printf("DEBUG", "Handle  : tools/call request for '%s' (ID: %v)", params.Name, id)

	// Extract the address parameter
	addressParam, ok := params.Arguments["address"]
	if !ok {
		err := fmt.Errorf("missing required parameter 'address'")
		s.logger.Printf("DEBUG", "Error: %v", err)
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInvalidParams, err.Error(), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}

	// Convert the address parameter to string
	address, ok := addressParam.(string)
	if !ok {
		err := fmt.Errorf("'address' parameter must be a string")
		s.logger.Printf("DEBUG", "Error: %v", err)
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInvalidParams, err.Error(), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}

	// Validate the address (basic validation)
	if address == "" {
		err := fmt.Errorf("'address' parameter cannot be empty")
		s.logger.Printf("DEBUG", "Error: %v", err)
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInvalidParams, err.Error(), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}

	// Execute the online command with the provided address
	output, err := tools.OnlineHost(address, onlineTimeout)

	var result mcp.CallToolResult
	var content mcp.TextContent

	if err != nil {
		s.logger.Printf("DEBUG", "Error executing online to %s: %v", address, err)
		// Ping failed, return the error message in the content
		content = mcp.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Error pinging %s: %v", address, err),
		}
		result.IsError = true // Indicate it's a tool-level error
	} else {
		s.logger.Printf("DEBUG", "Ping to %s successful. Output:\n%s", address, output)
		content = mcp.TextContent{
			Type: "text",
			Text: output,
		}
		result.IsError = false
	}

	// Marshal the content into json.RawMessage
	contentBytes, marshalErr := json.Marshal(content)
	if marshalErr != nil {
		err = fmt.Errorf("failed to marshal online result content: %w", marshalErr)
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInternalError, err.Error(), nil)
		return s.marshalErrorResponse(id, rpcErr) // Return marshalled JSON-RPC error
	}

	result.Content = []json.RawMessage{json.RawMessage(contentBytes)}

	// Marshal the successful (or tool-error) CallToolResult response
	return s.marshalResponse(id, result)
}
