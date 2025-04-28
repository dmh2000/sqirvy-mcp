package mcp

import (
	"encoding/json"
	"fmt" // Keep fmt for error formatting in functions
)

// Method names for prompt operations.
const (
	MethodListPrompts = "prompts/list"
	MethodGetPrompt   = "prompts/get"
)

// PromptArgument describes an argument that a prompt template can accept.
type PromptArgument struct {
	// Description is a human-readable description of the argument.
	Description string `json:"description,omitempty"`
	// Name is the name of the argument.
	Name string `json:"name"`
	// Required indicates whether this argument must be provided.
	Required bool `json:"required,omitempty"` // Defaults to false if omitted
}

// Prompt represents a prompt or prompt template offered by the server.
type Prompt struct {
	// Arguments is a list of arguments the prompt template accepts.
	Arguments []PromptArgument `json:"arguments,omitempty"`
	// Description is an optional description of what the prompt provides.
	Description string `json:"description,omitempty"`
	// Name is the unique name of the prompt or prompt template.
	Name string `json:"name"`
}

// TextContent represents text content within a prompt message.
// Note: Duplicated from resources.go for clarity, consider consolidating.
type TextContent struct {
	Annotations *Annotations `json:"annotations,omitempty"`
	Text        string       `json:"text"`
	Type        string       `json:"type"` // Should be "text"
}

// ImageContent represents image content within a prompt message.
// Note: Duplicated from resources.go for clarity, consider consolidating.
type ImageContent struct {
	Annotations *Annotations `json:"annotations,omitempty"`
	Data        string       `json:"data"` // base64 encoded
	MimeType    string       `json:"mimeType"`
	Type        string       `json:"type"` // Should be "image"
}

// PromptMessage describes a message returned as part of a prompt.
// It's similar to SamplingMessage but supports embedded resources.
type PromptMessage struct {
	// Content holds the message data (TextContent, ImageContent, or EmbeddedResource).
	// Needs to be unmarshaled into the specific type based on the "type" field
	// after initial unmarshaling into json.RawMessage.
	Content json.RawMessage `json:"content"`
	// Role indicates the sender of the message (user or assistant).
	Role Role `json:"role"`
}

// ListPromptsParams defines the parameters for a "prompts/list" request.
type ListPromptsParams struct {
	// Cursor is an opaque token for pagination.
	Cursor string `json:"cursor,omitempty"`
}

// ListPromptsResult defines the result structure for a "prompts/list" response.
type ListPromptsResult struct {
	// Meta contains reserved protocol metadata.
	Meta map[string]interface{} `json:"_meta,omitempty"`
	// NextCursor is an opaque token for the next page of results.
	NextCursor string `json:"nextCursor,omitempty"`
	// Prompts is the list of prompts found.
	Prompts []Prompt `json:"prompts"`
}

// GetPromptParams defines the parameters for a "prompts/get" request.
type GetPromptParams struct {
	// Arguments to use for templating the prompt.
	Arguments map[string]string `json:"arguments,omitempty"`
	// Name is the name of the prompt or prompt template to retrieve.
	Name string `json:"name"`
}

// GetPromptResult defines the result structure for a "prompts/get" response.
// Note: The schema defines this as GetPromptResponse, using Result here for consistency.
type GetPromptResult struct {
	// Meta contains reserved protocol metadata.
	Meta map[string]interface{} `json:"_meta,omitempty"`
	// Description is an optional description for the prompt.
	Description string `json:"description,omitempty"`
	// Messages is the sequence of messages constituting the prompt.
	Messages []PromptMessage `json:"messages"`
}

// MarshalListPromptsRequest creates a JSON-RPC request for the prompts/list method.
// The id can be a string or an integer. If params is nil, default empty params will be used.
func MarshalListPromptsRequest(id RequestID, params *ListPromptsParams) ([]byte, error) {
	// Use default empty params if nil is provided
	var p interface{}
	if params != nil {
		p = params
	} else {
		p = struct{}{} // Empty object for params if none specified
	}

	req := RPCRequest{
		JSONRPC: JSONRPCVersion,
		Method:  MethodListPrompts,
		Params:  p,
		ID:      id,
	}
	return json.Marshal(req)
}

// UnmarshalListPromptsResponse parses a JSON-RPC response for a prompts/list request.
// It expects the standard JSON-RPC response format with the result nested in the "result" field.
// It returns the result, the response ID, any RPC error, and a general parsing error.
func UnmarshalListPromptsResponse(data []byte) (*ListPromptsResult, RequestID, *RPCError, error) {
	var resp RPCResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to unmarshal RPC response: %w", err)
	}

	// Check for JSON-RPC level error
	if resp.Error != nil {
		return nil, resp.ID, resp.Error, nil // Return RPC error, no result expected
	}

	// Check if the result field is present (it's optional in the RPCResponse struct)
	if len(resp.Result) == 0 || string(resp.Result) == "null" {
		// For ListPrompts, we expect a result object.
		return nil, resp.ID, nil, fmt.Errorf("received response with missing or null result field for method %s", MethodListPrompts)
	}

	// Unmarshal the actual result from the Result field
	var result ListPromptsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, resp.ID, nil, fmt.Errorf("failed to unmarshal ListPromptsResult from response result: %w", err)
	}

	return &result, resp.ID, nil, nil
}

// MarshalGetPromptRequest creates a JSON-RPC request for the prompts/get method.
// The id can be a string or an integer.
func MarshalGetPromptRequest(id RequestID, params GetPromptParams) ([]byte, error) {
	req := RPCRequest{
		JSONRPC: JSONRPCVersion,
		Method:  MethodGetPrompt,
		Params:  params,
		ID:      id,
	}
	return json.Marshal(req)
}

// UnmarshalGetPromptResponse parses a JSON-RPC response for a prompts/get request.
// It expects the standard JSON-RPC response format with the result nested in the "result" field.
// It returns the result, the response ID, any RPC error, and a general parsing error.
// Note: The Content field within each PromptMessage in the result's Messages array
// will contain json.RawMessage elements that need further unmarshaling by the caller.
func UnmarshalGetPromptResponse(data []byte) (*GetPromptResult, RequestID, *RPCError, error) {
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
		return nil, resp.ID, nil, fmt.Errorf("received response with missing or null result field for method %s", MethodGetPrompt)
	}

	// Unmarshal the actual result from the Result field
	var result GetPromptResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, resp.ID, nil, fmt.Errorf("failed to unmarshal GetPromptResult from response result: %w", err)
	}

	// The caller needs to process result.Messages[...].Content further
	return &result, resp.ID, nil, nil
}

// Note: Standard json.Marshal and json.Unmarshal can be used for the other defined types.
// For PromptMessage.Content, further processing is needed after unmarshaling
// to determine the concrete type (TextContent, ImageContent, or EmbeddedResource).
