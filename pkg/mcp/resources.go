package mcp

import (
	"encoding/json"
	"fmt" // Keep fmt for error formatting in functions
)

// Method names for resource operations.
const (
	MethodListResources         = "resources/list"
	MethodReadResource          = "resources/read"
	MethodListResourceTemplates = "resources/templates/list" // Added for resource templates
)

// Resource represents a known resource the server can read.
type Resource struct {
	Annotations *Annotations `json:"annotations,omitempty"`
	// Description is a human-readable description of the resource.
	Description string `json:"description,omitempty"`
	// MimeType is the MIME type of the resource, if known.
	MimeType string `json:"mimeType,omitempty"`
	// Name is a human-readable name for the resource.
	Name string `json:"name"`
	// Size is the raw size in bytes, if known.
	Size *int `json:"size,omitempty"` // Use pointer for optional 0 value
	// URI is the unique identifier for the resource.
	URI string `json:"uri"`
}

// ResourceTemplate describes a template for resources available on the server.
type ResourceTemplate struct {
	Annotations *Annotations `json:"annotations,omitempty"`
	// Description is a human-readable description of the template.
	Description string `json:"description,omitempty"`
	// MimeType is the MIME type for resources matching this template, if uniform.
	MimeType string `json:"mimeType,omitempty"`
	// Name is a human-readable name for the type of resource this template refers to.
	Name string `json:"name"`
	// URITemplate is an RFC 6570 URI template.
	URITemplate string `json:"uriTemplate"`
}

// ListResourcesParams defines the parameters for a "resources/list" request.
type ListResourcesParams struct {
	// Cursor is an opaque token for pagination.
	Cursor string `json:"cursor,omitempty"`
}

// ListResourcesResult defines the result structure for a "resources/list" response.
type ListResourcesResult struct {
	// Meta contains reserved protocol metadata.
	Meta map[string]interface{} `json:"_meta,omitempty"`
	// NextCursor is an opaque token for the next page of results.
	NextCursor string `json:"nextCursor,omitempty"`
	// Resources is the list of resources found.
	Resources []Resource `json:"resources"`
}

// ListResourceTemplatesParams defines the parameters for a "resources/templates/list" request.
type ListResourceTemplatesParams struct {
	// Cursor is an opaque token for pagination.
	Cursor string `json:"cursor,omitempty"`
}

// ListResourceTemplatesResult defines the result structure for a "resources/templates/list" response.
type ListResourceTemplatesResult struct {
	// Meta contains reserved protocol metadata.
	Meta map[string]interface{} `json:"_meta,omitempty"`
	// NextCursor is an opaque token for the next page of results.
	NextCursor string `json:"nextCursor,omitempty"`
	// ResourceTemplates is the list of resource templates found.
	ResourceTemplates []ResourceTemplate `json:"resourceTemplates"`
}

// ReadResourceParams defines the parameters for a "resources/read" request.
type ReadResourceParams struct {
	// URI is the identifier of the resource to read.
	URI string `json:"uri"`
}

// TextResourceContents represents the text content of a resource.
type TextResourceContents struct {
	// MimeType is the MIME type of the resource, if known.
	MimeType string `json:"mimeType,omitempty"`
	// Text is the content of the resource.
	Text string `json:"text"`
	// URI is the identifier of the resource.
	URI string `json:"uri"`
}

// BlobResourceContents represents the binary content of a resource.
type BlobResourceContents struct {
	// Blob is the base64-encoded binary data.
	Blob string `json:"blob"`
	// MimeType is the MIME type of the resource, if known.
	MimeType string `json:"mimeType,omitempty"`
	// URI is the identifier of the resource.
	URI string `json:"uri"`
}

// ReadResourceResult defines the result structure for a "resources/read" response.
type ReadResourceResult struct {
	// Meta contains reserved protocol metadata.
	Meta map[string]interface{} `json:"_meta,omitempty"`
	// Contents holds the resource data, which can be text or blob.
	// Each element needs to be unmarshaled into either TextResourceContents or BlobResourceContents.
	// Example: Check for the presence of "text" or "blob" field after unmarshaling into json.RawMessage.
	Contents []json.RawMessage `json:"contents"`
}

// MarshalListResourcesRequest creates a JSON-RPC request for the resources/list method.
// The id can be a string or an integer. If params is nil, default empty params will be used.
func MarshalListResourcesRequest(id RequestID, params *ListResourcesParams) ([]byte, error) {
	// Use default empty params if nil is provided
	var p interface{}
	if params != nil {
		p = params
	} else {
		p = struct{}{} // Empty object for params if none specified
	}

	req := RPCRequest{
		JSONRPC: JSONRPCVersion,
		Method:  MethodListResources,
		Params:  p,
		ID:      id,
	}
	return json.Marshal(req)
}

// UnmarshalListResourcesResponse parses a JSON-RPC response for a resources/list request.
// It expects the standard JSON-RPC response format with the result nested in the "result" field.
// It returns the result, the response ID, any RPC error, and a general parsing error.
func UnmarshalListResourcesResponse(data []byte) (*ListResourcesResult, RequestID, *RPCError, error) {
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
		// Handle cases where result might be legitimately null or empty if needed,
		// otherwise, it might indicate an issue if a result was expected.
		// For ListResources, we expect a result object.
		return nil, resp.ID, nil, fmt.Errorf("received response with missing or null result field for method %s", MethodListResources)
	}

	// Unmarshal the actual result from the Result field
	var result ListResourcesResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, resp.ID, nil, fmt.Errorf("failed to unmarshal ListResourcesResult from response result: %w", err)
	}

	return &result, resp.ID, nil, nil
}

// MarshalListResourceTemplatesRequest creates a JSON-RPC request for the resources/templates/list method.
// The id can be a string or an integer. If params is nil, default empty params will be used.
func MarshalListResourceTemplatesRequest(id RequestID, params *ListResourceTemplatesParams) ([]byte, error) {
	var p interface{}
	if params != nil {
		p = params
	} else {
		p = struct{}{} // Empty object for params if none specified
	}

	req := RPCRequest{
		JSONRPC: JSONRPCVersion,
		Method:  MethodListResourceTemplates,
		Params:  p,
		ID:      id,
	}
	return json.Marshal(req)
}

// UnmarshalListResourceTemplatesResponse parses a JSON-RPC response for a resources/templates/list request.
func UnmarshalListResourceTemplatesResponse(data []byte) (*ListResourceTemplatesResult, RequestID, *RPCError, error) {
	var resp RPCResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to unmarshal RPC response: %w", err)
	}

	if resp.Error != nil {
		return nil, resp.ID, resp.Error, nil
	}

	if len(resp.Result) == 0 || string(resp.Result) == "null" {
		return nil, resp.ID, nil, fmt.Errorf("received response with missing or null result field for method %s", MethodListResourceTemplates)
	}

	var result ListResourceTemplatesResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, resp.ID, nil, fmt.Errorf("failed to unmarshal ListResourceTemplatesResult from response result: %w", err)
	}

	return &result, resp.ID, nil, nil
}

// MarshalReadResourcesRequest creates a JSON-RPC request for the resources/read method.
// The id can be a string or an integer.
func MarshalReadResourcesRequest(id RequestID, params ReadResourceParams) ([]byte, error) {
	req := RPCRequest{
		JSONRPC: JSONRPCVersion,
		Method:  MethodReadResource,
		Params:  params,
		ID:      id,
	}
	return json.Marshal(req)
}

// UnmarshalReadResourcesResponse parses a JSON-RPC response for a resources/read request.
// It expects the standard JSON-RPC response format with the result nested in the "result" field.
// It returns the result, the response ID, any RPC error, and a general parsing error.
// Note: The Contents field within the result will contain json.RawMessage elements
// that need further unmarshaling into TextResourceContents or BlobResourceContents by the caller.
func UnmarshalReadResourcesResponse(data []byte) (*ReadResourceResult, RequestID, *RPCError, error) {
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
		return nil, resp.ID, nil, fmt.Errorf("received response with missing or null result field for method %s", MethodReadResource)
	}

	// Unmarshal the actual result from the Result field
	var result ReadResourceResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, resp.ID, nil, fmt.Errorf("failed to unmarshal ReadResourceResult from response result: %w", err)
	}

	// The caller needs to process result.Contents further
	return &result, resp.ID, nil, nil
}

// Note: Standard json.Marshal and json.Unmarshal can be used for the other defined types.
// For ReadResourceResult.Contents, further processing is needed after unmarshaling
// to determine the concrete type (TextResourceContents or BlobResourceContents) of each element.
