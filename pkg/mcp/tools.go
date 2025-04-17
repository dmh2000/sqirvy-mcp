package mcp

import (
	"encoding/json"
	"fmt" // Added for error formatting
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

// MarshalListToolsRequest creates a JSON-RPC request for the tools/list method.
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

// UnmarshalListToolsResponse parses a JSON-RPC response for a tools/list request.
// It expects the standard JSON-RPC response format with the result nested in the "result" field.
// It returns the result, the response ID, any RPC error, and a general parsing error.
func UnmarshalListToolsResponse(data []byte) (*ListToolsResult, RequestID, *RPCError, error) {
	var resp RPCResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to unmarshal RPC response: %w", err)
	}

	// Check for JSON-RPC level error
	if resp.Error != nil {
		return nil, resp.ID, resp.Error, nil // Return RPC error, no result expected
	}

	// Check if the result field is present
	if len(resp.Result) == 0 || string(resp.Result) == "null" {
		// For ListTools, we expect a result object.
		return nil, resp.ID, nil, fmt.Errorf("received response with missing or null result field for method %s", MethodListTools)
	}

	// Unmarshal the actual result from the Result field
	var result ListToolsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, resp.ID, nil, fmt.Errorf("failed to unmarshal ListToolsResult from response result: %w", err)
	}

	return &result, resp.ID, nil, nil
}

// MarshalCallToolRequest creates a JSON-RPC request for the tools/call method.
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
// It expects the standard JSON-RPC response format with the result nested in the "result" field.
// It returns the result, the response ID, any RPC error, and a general parsing error.
// Note: The Content field within the result will contain json.RawMessage elements
// that need further unmarshaling into TextContent, ImageContent, or EmbeddedResource by the caller.
func UnmarshalCallToolResponse(data []byte) (*CallToolResult, RequestID, *RPCError, error) {
	var resp RPCResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to unmarshal RPC response: %w", err)
	}

	// Check for JSON-RPC level error
	if resp.Error != nil {
		return nil, resp.ID, resp.Error, nil // Return RPC error, no result expected
	}

	// Check if the result field is present
	if len(resp.Result) == 0 || string(resp.Result) == "null" {
		return nil, resp.ID, nil, fmt.Errorf("received response with missing or null result field for method %s", MethodCallTool)
	}

	// Unmarshal the actual result from the Result field
	var result CallToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, resp.ID, nil, fmt.Errorf("failed to unmarshal CallToolResult from response result: %w", err)
	}

	// The caller needs to process result.Content further
	return &result, resp.ID, nil, nil
}

// Note: Standard json.Marshal and json.Unmarshal can be used for the other defined types.
// For CallToolResult.Content and EmbeddedResource.Resource, further processing is needed after unmarshaling
// to determine the concrete type.
