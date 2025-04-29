package mcp

import (
	"encoding/json"
	"fmt" // Keep fmt for error formatting in functions
	utils "sqirvy-mcp/pkg/utils"
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

// ============================================
// Client-Side Functions
// ============================================

// MarshalListPromptsRequest creates a JSON-RPC request for the prompts/list method.
// Intended for use by the client.
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

// UnmarshalListPromptsResult parses a JSON-RPC response for a prompts/list request.
// Intended for use by the client.
// It expects the standard JSON-RPC response format with the result nested in the "result" field.
// It returns the result by value, the response ID, any RPC error, and a general parsing error.
func UnmarshalListPromptsResult(data []byte) (ListPromptsResult, RequestID, *RPCError, error) {
	var resp RPCResponse
	var zeroResult ListPromptsResult
	if err := json.Unmarshal(data, &resp); err != nil {
		return zeroResult, nil, nil, fmt.Errorf("failed to unmarshal RPC response: %w", err)
	}

	// Check for JSON-RPC level error
	if resp.Error != nil {
		return zeroResult, resp.ID, resp.Error, nil // Return RPC error, no result expected
	}

	// Check if the result field is present (it's optional in the RPCResponse struct)
	if len(resp.Result) == 0 || string(resp.Result) == "null" {
		// For ListPrompts, we expect a result object.
		return zeroResult, resp.ID, nil, fmt.Errorf("received response with missing or null result field for method %s", MethodListPrompts)
	}

	// Unmarshal the actual result from the Result field
	var result ListPromptsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return zeroResult, resp.ID, nil, fmt.Errorf("failed to unmarshal ListPromptsResult from response result: %w", err)
	}

	return result, resp.ID, nil, nil
}

// MarshalGetPromptRequest creates a JSON-RPC request for the prompts/get method.
// Intended for use by the client.
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

// UnmarshalGetPromptResult parses a JSON-RPC response for a prompts/get request.
// Intended for use by the client.
// It expects the standard JSON-RPC response format with the result nested in the "result" field.
// It returns the result, the response ID, any RPC error, and a general parsing error.
// Note: The Content field within each PromptMessage in the result's Messages array
// will contain json.RawMessage elements that need further unmarshaling by the caller.
func UnmarshalGetPromptResult(data []byte) (GetPromptResult, RequestID, *RPCError, error) {
	var resp RPCResponse
	var zeroResult GetPromptResult
	if err := json.Unmarshal(data, &resp); err != nil {
		return zeroResult, nil, nil, fmt.Errorf("failed to unmarshal RPC response: %w", err)
	}

	// Check for JSON-RPC level error
	if resp.Error != nil {
		return zeroResult, resp.ID, resp.Error, nil // Return RPC error, no result expected
	}

	// Check if the result field is present
	if len(resp.Result) == 0 || string(resp.Result) == "null" {
		return zeroResult, resp.ID, nil, fmt.Errorf("received response with missing or null result field for method %s", MethodGetPrompt)
	}

	// Unmarshal the actual result from the Result field
	var result GetPromptResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return zeroResult, resp.ID, nil, fmt.Errorf("failed to unmarshal GetPromptResult from response result: %w", err)
	}

	// The caller needs to process result.Messages[...].Content further
	return result, resp.ID, nil, nil
}

// ============================================
// Server-Side Request Unmarshaling
// ============================================

// UnmarshalListPromptsRequest parses the parameters from a JSON-RPC request for the prompts/list method.
// Intended for use by the server.
// It unmarshals the entire request and specifically parses the `params` field into ListPromptsParams.
// It returns the parsed parameters by value, the request ID, any RPC error encountered during parsing, and a general parsing error.
func UnmarshalListPromptsRequest(payload []byte, logger *utils.Logger) (ListPromptsParams, RequestID, *RPCError, error) {
	var zeroParams ListPromptsParams
	if logger == nil {
		return zeroParams, nil, nil, fmt.Errorf("logger cannot be nil")
	}

	var req RPCRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		err = fmt.Errorf("failed to unmarshal base list prompts request: %w", err)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeParseError, err.Error(), nil)
		return zeroParams, nil, rpcErr, err
	}

	// Params are optional for prompts/list (cursor)
	var params ListPromptsParams
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
				err = fmt.Errorf("failed to unmarshal ListPromptsParams from request params: %w", err)
				logger.Println("ERROR", err.Error())
				rpcErr := NewRPCError(ErrorCodeInvalidParams, "Invalid parameters for prompts/list", err.Error())
				return zeroParams, req.ID, rpcErr, err
			}
		}
	}
	// If req.Params was nil or null, params remains the zero value, which is valid.

	// No specific validation needed for ListPromptsParams fields (cursor is optional)
	return params, req.ID, nil, nil
}

// UnmarshalGetPromptRequest parses the parameters from a JSON-RPC request for the prompts/get method.
// Intended for use by the server.
// It unmarshals the entire request and specifically parses the `params` field into GetPromptParams.
// It returns the parsed parameters by value, the request ID, any RPC error encountered during parsing, and a general parsing error.
func UnmarshalGetPromptRequest(payload []byte, logger *utils.Logger) (GetPromptParams, RequestID, *RPCError, error) {
	var zeroParams GetPromptParams
	if logger == nil {
		return zeroParams, nil, nil, fmt.Errorf("logger cannot be nil")
	}

	var req RPCRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		err = fmt.Errorf("failed to unmarshal base get prompt request: %w", err)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeParseError, err.Error(), nil)
		return zeroParams, nil, rpcErr, err
	}

	// Now, unmarshal the Params field specifically into GetPromptParams
	var params GetPromptParams

	// Handle cases where params might be missing or explicitly null in the JSON
	rawParams, ok := req.Params.(json.RawMessage)
	if !ok && req.Params != nil {
		err := fmt.Errorf("invalid type for params field: expected JSON object, got %T", req.Params)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeInvalidRequest, "Invalid params field type", err.Error())
		return zeroParams, req.ID, rpcErr, err
	}

	// For GetPrompt, the 'params' object itself is required and must contain 'name'.
	if len(rawParams) == 0 || string(rawParams) == "null" {
		err := fmt.Errorf("missing required params field for method %s", MethodGetPrompt)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeInvalidParams, "Missing required parameters object", nil)
		return zeroParams, req.ID, rpcErr, err
	}

	// Attempt to unmarshal the params
	if err := json.Unmarshal(rawParams, &params); err != nil {
		err = fmt.Errorf("failed to unmarshal GetPromptParams from request params: %w", err)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeInvalidParams, "Invalid parameters for prompts/get", err.Error())
		return zeroParams, req.ID, rpcErr, err
	}

	// Validate required fields within params (e.g., Name must not be empty)
	if params.Name == "" {
		err := fmt.Errorf("missing required 'name' field in params for method %s", MethodGetPrompt)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeInvalidParams, "Missing required 'name' parameter", nil)
		return zeroParams, req.ID, rpcErr, err
	}
	// Arguments are optional, no validation needed unless specific constraints exist.

	// Successfully parsed and validated params
	return params, req.ID, nil, nil
}

// ============================================
// Server-Side Response Marshaling / Creation
// ============================================

// MarshalGetPromptResult marshals a successful GetPromptResult into a full RPCResponse.
// Intended for use by the server.
func MarshalGetPromptResult(id RequestID, result GetPromptResult, logger *utils.Logger) ([]byte, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	return MarshalResponse(id, result, logger)
}

// NewGetPromptResult creates a new GetPromptResult structure.
// Intended for use by the server.
func NewGetPromptResult(messages []PromptMessage) GetPromptResult {
	return GetPromptResult{
		Messages: messages,
	}
}

// MarshalListPromptsResult marshals a successful ListPromptsResult into a full RPCResponse.
// Intended for use by the server.
func MarshalListPromptsResult(id RequestID, result ListPromptsResult, logger *utils.Logger) ([]byte, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	return MarshalResponse(id, result, logger)
}

// NewListPromptsResult creates a new ListPromptsResult structure.
// Intended for use by the server.
func NewListPromptsResult(prompts []Prompt) ListPromptsResult {
	return ListPromptsResult{
		Prompts: prompts,
	}
}
