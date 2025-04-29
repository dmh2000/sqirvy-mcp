package mcp

import (
	"encoding/json"
	"fmt"
	utils "sqirvy-mcp/pkg/utils"
)

// Method names for tool operations.
const (
	MethodListTools = "tools/list"
	MethodCallTool  = "tools/call"
)

// ToolInputSchema defines the expected parameters for a tool, represented as a JSON Schema object.
// Using map[string]interface{} for flexibility, but could be a more specific struct if the schema structure is fixed.
type ToolInputSchema map[string]interface{}

// Tool defines a tool the client can call.
type Tool struct {
	// Description is a human-readable description of the tool.
	Description string `json:"description,omitempty"`
	// InputSchema is a JSON Schema object defining the expected parameters.
	InputSchema ToolInputSchema `json:"inputSchema"`
	// Name is the name of the tool.
	Name string `json:"name"`
}

// ListToolsParams defines the parameters for a "tools/list" request.
type ListToolsParams struct {
	// Cursor is an opaque token for pagination.
	Cursor string `json:"cursor,omitempty"`
}

// ListToolsResult defines the result structure for a "tools/list" response.
type ListToolsResult struct {
	// Meta contains reserved protocol metadata.
	Meta map[string]interface{} `json:"_meta,omitempty"`
	// NextCursor is an opaque token for the next page of results.
	NextCursor string `json:"nextCursor,omitempty"`
	// Tools is the list of tools found.
	Tools []Tool `json:"tools"`
}

// CallToolParams defines the parameters for a "tools/call" request.
type CallToolParams struct {
	// Arguments are the parameters to pass to the tool.
	// Using map[string]interface{} for flexibility as argument types can vary.
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	// Name is the name of the tool to call.
	Name string `json:"name"`
}

// EmbeddedResource represents resource contents embedded in a message.
// Note: Duplicated from prompts.go, consider consolidating.
type EmbeddedResource struct {
	Annotations *Annotations    `json:"annotations,omitempty"`
	Resource    json.RawMessage `json:"resource"` // Can be TextResourceContents or BlobResourceContents
	Type        string          `json:"type"`     // Should be "resource"
}

// CallToolResult defines the result structure for a "tools/call" response.
type CallToolResult struct {
	// Meta contains reserved protocol metadata.
	Meta map[string]interface{} `json:"_meta,omitempty"`
	// Content holds the tool's output data (TextContent, ImageContent, or EmbeddedResource).
	// Each element needs to be unmarshaled into the specific type based on the "type" field
	// after initial unmarshaling into json.RawMessage.
	Content []json.RawMessage `json:"content"`
	// IsError indicates if the tool call resulted in an error. Defaults to false.
	IsError bool `json:"isError,omitempty"`
}

// ============================================
// Client-Side Functions
// ============================================

// MarshalListToolsRequest creates a JSON-RPC request for the tools/list method.
// Intended for use by the client.
// The id can be a string or an integer. If params is nil, default empty params will be used.
func MarshalListToolsRequest(id RequestID, params *ListToolsParams) ([]byte, error) {
	// Use default empty params if nil is provided
	var p interface{}
	if params != nil {
		p = params
	} else {
		p = struct{}{} // Empty object for params if none specified
	}

	req := RPCRequest{
		JSONRPC: JSONRPCVersion,
		Method:  MethodListTools,
		Params:  p,
		ID:      id,
	}
	return json.Marshal(req)
}

// UnmarshalListToolsResult parses a JSON-RPC response for a tools/list request.
// Intended for use by the client.
// It expects the standard JSON-RPC response format with the result nested in the "result" field.
// It returns the result by value, the response ID, any RPC error, and a general parsing error.
func UnmarshalListToolsResult(data []byte) (ListToolsResult, RequestID, *RPCError, error) {
	var resp RPCResponse
	var zeroResult ListToolsResult // Zero value to return on error
	if err := json.Unmarshal(data, &resp); err != nil {
		return zeroResult, nil, nil, fmt.Errorf("failed to unmarshal RPC response: %w", err)
	}

	// Check for JSON-RPC level error
	if resp.Error != nil {
		return zeroResult, resp.ID, resp.Error, nil // Return RPC error, no result expected
	}

	// Check if the result field is present
	if len(resp.Result) == 0 || string(resp.Result) == "null" {
		// For ListTools, we expect a result object.
		return zeroResult, resp.ID, nil, fmt.Errorf("received response with missing or null result field for method %s", MethodListTools)
	}

	// Unmarshal the actual result from the Result field
	var result ListToolsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return zeroResult, resp.ID, nil, fmt.Errorf("failed to unmarshal ListToolsResult from response result: %w", err)
	}

	return result, resp.ID, nil, nil
}

// MarshalCallToolRequest creates a JSON-RPC request for the tools/call method.
// Intended for use by the client.
// The id can be a string or an integer.
func MarshalCallToolRequest(id RequestID, params CallToolParams) ([]byte, error) {
	req := RPCRequest{
		JSONRPC: JSONRPCVersion,
		Method:  MethodCallTool,
		Params:  params,
		ID:      id,
	}
	return json.Marshal(req)
}

// UnmarshalCallToolResponse parses a JSON-RPC response for a tools/call request.
// Intended for use by the client.
// It expects the standard JSON-RPC response format with the result nested in the "result" field.
// It returns the result by value, the response ID, any RPC error, and a general parsing error.
// Note: The Content field within the result will contain json.RawMessage elements
// that need further unmarshaling into TextContent, ImageContent, or EmbeddedResource by the caller.
func UnmarshalCallToolResponse(data []byte) (CallToolResult, RequestID, *RPCError, error) {
	var resp RPCResponse
	var zeroResult CallToolResult // Zero value to return on error
	if err := json.Unmarshal(data, &resp); err != nil {
		return zeroResult, nil, nil, fmt.Errorf("failed to unmarshal RPC response: %w", err)
	}

	// Check for JSON-RPC level error
	if resp.Error != nil {
		return zeroResult, resp.ID, resp.Error, nil // Return RPC error, no result expected
	}

	// Check if the result field is present
	if len(resp.Result) == 0 || string(resp.Result) == "null" {
		return zeroResult, resp.ID, nil, fmt.Errorf("received response with missing or null result field for method %s", MethodCallTool)
	}

	// Unmarshal the actual result from the Result field
	var result CallToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return zeroResult, resp.ID, nil, fmt.Errorf("failed to unmarshal CallToolResult from response result: %w", err)
	}

	// The caller needs to process result.Content further
	return result, resp.ID, nil, nil
}

// ============================================
// Server-Side Functions
// ============================================

// UnmarshalListToolsRequest parses the parameters from a JSON-RPC request for the tools/list method.
// Intended for use by the server.
// It unmarshals the entire request and specifically parses the `params` field into ListToolsParams.
// It returns the parsed parameters by value, the request ID, any RPC error encountered during parsing, and a general parsing error.
func UnmarshalListToolsRequest(payload []byte, logger *utils.Logger) (ListToolsParams, RequestID, *RPCError, error) {
	var zeroParams ListToolsParams
	if logger == nil {
		return zeroParams, nil, nil, fmt.Errorf("logger cannot be nil")
	}

	var req RPCRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		err = fmt.Errorf("failed to unmarshal base list tools request: %w", err)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeParseError, err.Error(), nil)
		return zeroParams, nil, rpcErr, err
	}

	// Params are optional for tools/list (cursor)
	var params ListToolsParams
	if req.Params != nil {
		rawParams, ok := req.Params.(json.RawMessage)
		if !ok {
			err := fmt.Errorf("invalid type for params field: expected JSON object or null, got %T", req.Params)
			logger.Println("ERROR", err.Error())
			rpcErr := NewRPCError(ErrorCodeInvalidRequest, "Invalid params field type", err.Error())
			return zeroParams, req.ID, rpcErr, err
		}

		// Only unmarshal if params is not null and not empty
		if len(rawParams) > 0 && string(rawParams) != "null" {
			if err := json.Unmarshal(rawParams, &params); err != nil {
				err = fmt.Errorf("failed to unmarshal ListToolsParams from request params: %w", err)
				logger.Println("ERROR", err.Error())
				rpcErr := NewRPCError(ErrorCodeInvalidParams, "Invalid parameters for tools/list", err.Error())
				return zeroParams, req.ID, rpcErr, err
			}
		}
	}
	// If req.Params was nil or null, params remains the zero value, which is valid.

	// No specific validation needed for ListToolsParams fields (cursor is optional)
	return params, req.ID, nil, nil
}

// MarshalListToolsResult creates a JSON-RPC response containing the result of a tools/list request.
// Intended for use by the server.
// It wraps the provided ListToolsResult and marshals it into a standard RPCResponse.
func MarshalListToolsResult(id RequestID, result ListToolsResult, logger *utils.Logger) ([]byte, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	return MarshalResponse(id, result, logger)
}

// UnmarshalCallToolRequest parses the parameters from a JSON-RPC request for the tools/call method.
// Intended for use by the server.
// It unmarshals the entire request and specifically parses the `params` field into CallToolParams.
// It returns the parsed parameters by value, the request ID, any RPC error encountered during parsing, and a general parsing error.
func UnmarshalCallToolRequest(payload []byte, logger *utils.Logger) (CallToolParams, RequestID, *RPCError, error) {
	var zeroParams CallToolParams
	if logger == nil {
		return zeroParams, nil, nil, fmt.Errorf("logger cannot be nil")
	}

	var req RPCRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		err = fmt.Errorf("failed to unmarshal base call tool request: %w", err)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeParseError, err.Error(), nil)
		return zeroParams, nil, rpcErr, err
	}

	// Now, unmarshal the Params field specifically into CallToolParams
	var params CallToolParams

	// Handle cases where params might be missing or explicitly null in the JSON
	rawParams, ok := req.Params.(json.RawMessage)
	if !ok && req.Params != nil {
		err := fmt.Errorf("invalid type for params field: expected JSON object, got %T", req.Params)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeInvalidRequest, "Invalid params field type", err.Error())
		return zeroParams, req.ID, rpcErr, err
	}

	// For CallTool, the 'params' object itself is required and must contain 'name'.
	if len(rawParams) == 0 || string(rawParams) == "null" {
		err := fmt.Errorf("missing required params field for method %s", MethodCallTool)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeInvalidParams, "Missing required parameters object", nil)
		return zeroParams, req.ID, rpcErr, err
	}

	// Attempt to unmarshal the params
	if err := json.Unmarshal(rawParams, &params); err != nil {
		err = fmt.Errorf("failed to unmarshal CallToolParams from request params: %w", err)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeInvalidParams, "Invalid parameters for tools/call", err.Error())
		return zeroParams, req.ID, rpcErr, err
	}

	// Validate required fields within params (e.g., Name must not be empty)
	if params.Name == "" {
		err := fmt.Errorf("missing required 'name' field in params for method %s", MethodCallTool)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeInvalidParams, "Missing required 'name' parameter", nil)
		return zeroParams, req.ID, rpcErr, err
	}
	// Arguments are optional, no validation needed unless specific constraints exist.

	// Successfully parsed and validated params
	return params, req.ID, nil, nil
}

// MarshalCallToolResult creates a JSON-RPC response containing the result of a tools/call request.
// Intended for use by the server.
// It wraps the provided CallToolResult and marshals it into a standard RPCResponse.
func MarshalCallToolResult(id RequestID, result CallToolResult, logger *utils.Logger) ([]byte, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	return MarshalResponse(id, result, logger)
}
