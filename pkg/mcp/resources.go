// Package mcp implements the Model Communication Protocol (MCP) types and
// marshaling/unmarshaling logic. This file specifically defines the structures
// and functions related to MCP resource management, including listing resources,
// listing resource templates, and reading resource content.
package mcp

import (
	"encoding/base64"
	"encoding/json"
	"fmt" // Keep fmt for error formatting in functions
	utils "sqirvy-mcp/pkg/utils"
	"strings"
)

// Method names for resource operations.
const (
	MethodListResources           = "resources/list"
	MethodReadResource            = "resources/read"
	MethodListResourcesTemplatess = "resources/templates/list" // Added for resource templates
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

// ResourcesTemplates describes a template for resources available on the server.
type ResourcesTemplates struct {
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

// ListResourcesTemplatessParams defines the parameters for a "resources/templates/list" request.
type ListResourcesTemplatessParams struct {
	// Cursor is an opaque token for pagination.
	Cursor string `json:"cursor,omitempty"`
}

// ListResourcesTemplatessResult defines the result structure for a "resources/templates/list" response.
type ListResourcesTemplatessResult struct {
	// Meta contains reserved protocol metadata.
	Meta map[string]interface{} `json:"_meta,omitempty"`
	// NextCursor is an opaque token for the next page of results.
	NextCursor string `json:"nextCursor,omitempty"`
	// ResourcesTemplatess is the list of resource templates found.
	ResourcesTemplatess []ResourcesTemplates `json:"resourceTemplates"`
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

// ============================================
// List Resources
// ============================================

// MarshalListResourcesRequest creates a JSON-RPC request for the resources/list method.
// Intended for use by the client.
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

// UnmarshalListResourcesResult parses a JSON-RPC response for a resources/list request.
// Intended for use by the client.
// It expects the standard JSON-RPC response format with the result nested in the "result" field.
// It returns the result, the response ID, any RPC error, and a general parsing error.
func UnmarshalListResourcesResult(data []byte) (*ListResourcesResult, RequestID, *RPCError, error) {
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

// MarshalListResourcesResult creates a JSON-RPC response containing the result of a resources/list request.
// Intended for use by the server.
// It wraps the provided list of resources and cursor into a ListResourcesResult and marshals it into a standard RPCResponse.
func MarshalListResourcesResult(id RequestID, resourcesList []Resource, cursor string, logger *utils.Logger) ([]byte, error) {
	result := ListResourcesResult{
		Resources:  resourcesList,
		NextCursor: cursor,
	}
	return MarshalResponse(id, result, logger)
}

// UnmarshalListResourcesRequest parses the parameters from a JSON-RPC request for the resources/list method.
// Intended for use by the server.
// Note: This function currently only unmarshals the base RPCRequest structure.
// Further unmarshaling of the `Params` field into `ListResourcesParams` would be needed
// if the server needs to access specific parameters like the cursor.
// It returns the base request object, the request ID, any RPC error encountered during base parsing, and a general parsing error.
func UnmarshalListResourcesRequest(id RequestID, payload []byte, logger *utils.Logger) (*RPCRequest, RequestID, *RPCError, error) {
	var req RPCRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		err = fmt.Errorf("failed to unmarshal base list resources request: %w", err)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeParseError, err.Error(), nil)
		return nil, id, rpcErr, err
	}
	// TODO: Consider unmarshaling req.Params into ListResourcesParams if needed by the server handler.
	// var params ListResourcesParams
	// if err := json.Unmarshal(req.Params.(json.RawMessage), &params); err != nil { ... }

	return &req, id, nil, nil
}

// ============================================
// RESOURCES TEMPLATES
// ============================================

// UnmarshalListResourcesTemplatessResult parses a JSON-RPC response for a resources/templates/list request.
// Intended for use by the client.
// It expects the standard JSON-RPC response format with the result nested in the "result" field.
// It returns the result, the response ID, any RPC error, and a general parsing error.
func UnmarshalListResourcesTemplatessResult(data []byte) (*ListResourcesTemplatessResult, RequestID, *RPCError, error) {
	var resp RPCResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to unmarshal RPC response: %w", err)
	}

	if resp.Error != nil {
		return nil, resp.ID, resp.Error, nil
	}

	if len(resp.Result) == 0 || string(resp.Result) == "null" {
		return nil, resp.ID, nil, fmt.Errorf("received response with missing or null result field for method %s", MethodListResourcesTemplatess)
	}

	var result ListResourcesTemplatessResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, resp.ID, nil, fmt.Errorf("failed to unmarshal ListResourcesTemplatessResult from response result: %w", err)
	}

	return &result, resp.ID, nil, nil
}

// MarshalReadResourcesTemplateRequest creates a JSON-RPC request for the resources/read method
// (used here potentially for reading a resource described by a template, although the method name is generic).
// Intended for use by the client.
// The id can be a string or an integer.
func MarshalReadResourcesTemplateRequest(id RequestID, params ReadResourceParams) ([]byte, error) {
	req := RPCRequest{
		JSONRPC: JSONRPCVersion,
		Method:  MethodReadResource, // Note: Uses the standard read method
		Params:  params,
		ID:      id,
	}
	return json.Marshal(req)
}

// MarshalListResourcesTemplatessResult creates a JSON-RPC request for the resources/templates/list method.
// Intended for use by the server to marshal the *request* parameters (likely a mistake in original code, should marshal a *result*).
// If this is intended to marshal the *result* from the server, it should accept ListResourcesTemplatessResult.
// If it's intended to marshal the *request* from the client, the function name is misleading and duplicates client logic.
// Assuming it's meant to marshal the *request* from the client side based on its structure:
func MarshalListResourcesTemplatessResult(id RequestID, params *ListResourcesTemplatessParams) ([]byte, error) {
	// TODO: Review function purpose. Name suggests marshaling a result (server), but implementation marshals a request (client).
	// If server-side result marshaling: func MarshalListResourcesTemplatessResult(id RequestID, result ListResourcesTemplatessResult, logger *utils.Logger) ([]byte, error)
	// If client-side request marshaling: func MarshalListResourcesTemplatessRequest(id RequestID, params *ListResourcesTemplatessParams) ([]byte, error)

	var p interface{}
	if params != nil {
		p = params
	} else {
		p = struct{}{} // Empty object for params if none specified
	}

	req := RPCRequest{
		JSONRPC: JSONRPCVersion,
		Method:  MethodListResourcesTemplatess,
		Params:  p,
		ID:      id,
	}
	return json.Marshal(req)
}

// UnmarshalReadResourcesTemplateRequest parses the parameters from a JSON-RPC request potentially related to resource templates.
// Intended for use by the server.
// Note: This function currently only unmarshals the base RPCRequest structure.
// Further unmarshaling of the `Params` field into specific template-related parameters might be needed.
// It returns the base request object, the request ID, any RPC error encountered during base parsing, and a general parsing error.
func UnmarshalReadResourcesTemplateRequest(id RequestID, payload []byte, logger *utils.Logger) (*RPCRequest, RequestID, *RPCError, error) {
	var req RPCRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		// Assuming this is intended to parse a request that might use ReadResourceParams for a template URI
		err = fmt.Errorf("failed to unmarshal base read resource (template) request: %w", err)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeParseError, err.Error(), nil)
		return nil, id, rpcErr, err
	}
	// TODO: Consider unmarshaling req.Params into ReadResourceParams if needed by the server handler.
	// var params ReadResourceParams
	// if err := json.Unmarshal(req.Params.(json.RawMessage), &params); err != nil { ... }

	return &req, id, nil, nil
}

// ============================================
// READ RESOURCES
// ============================================

// MarshalReadResourcesRequest creates a JSON-RPC request for the resources/read method.
// Intended for use by the client.
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

// UnmarshalReadResourceRequest parses the parameters from a JSON-RPC request for the resources/read method.
// Intended for use by the server.
// Note: This function currently only unmarshals the base RPCRequest structure.
// Further unmarshaling of the `Params` field into `ReadResourceParams` is required
// for the server to know which resource URI to read.
// It returns the base request object, the request ID, any RPC error encountered during base parsing, and a general parsing error.
func UnmarshalReadResourceRequest(id RequestID, payload []byte, logger *utils.Logger) (*RPCRequest, RequestID, *RPCError, error) {
	var req RPCRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		err = fmt.Errorf("failed to unmarshal base read resource request: %w", err)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeParseError, err.Error(), nil)
		return nil, id, rpcErr, err
	}
	// IMPORTANT: Server handler MUST unmarshal req.Params into ReadResourceParams
	// var params ReadResourceParams
	// if err := json.Unmarshal(req.Params.(json.RawMessage), &params); err != nil {
	//     // Handle error: Invalid Params
	//     rpcErr := NewRPCError(ErrorCodeInvalidParams, "Failed to parse ReadResourceParams", err.Error())
	//     // ... return error response ...
	// }
	// // Use params.URI

	return &req, id, nil, nil
}

// MarshalReadResourceResult creates a JSON-RPC response containing the result of a resources/read request.
// Intended for use by the server.
// It wraps the provided ReadResourceResult into a standard RPCResponse.
func MarshalReadResourceResult(id RequestID, result ReadResourceResult, logger *utils.Logger) ([]byte, error) {
	return MarshalResponse(id, result, logger)
}

// UnmarshalReadResourcesResult parses a JSON-RPC response for a resources/read request.
// Intended for use by the client.
// It expects the standard JSON-RPC response format with the result nested in the "result" field.
// Note: The Contents field within the result will contain json.RawMessage elements
// that need further unmarshaling into TextResourceContents or BlobResourceContents by the caller.
// It returns the result, the response ID, any RPC error, and a general parsing error.
func UnmarshalReadResourcesResult(data []byte) (*ReadResourceResult, RequestID, *RPCError, error) {
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

// NewReadResourcesResult creates a ReadResourceResult containing a single content item (either text or blob)
// based on the provided MIME type and raw byte contents.
// Intended for use by the server when constructing a response.
// It automatically handles base64 encoding for non-text types.
func NewReadResourcesResult(uri string, mimetype string, contents []byte) (ReadResourceResult, error) {
	var result ReadResourceResult
	var content json.RawMessage
	var err error

	// Determine if content is text or blob based on MIME type
	if strings.HasPrefix(mimetype, "text/") || mimetype == "application/json" || mimetype == "" { // Treat empty MIME as text for safety
		text := TextResourceContents{
			URI:      uri,
			MimeType: mimetype,
			Text:     string(contents),
		}
		content, err = json.Marshal(text)
		if err != nil {
			return result, fmt.Errorf("failed to marshal text resource contents: %w", err)
		}
	} else { // Treat as blob otherwise
		blob := BlobResourceContents{
			URI:      uri,
			MimeType: mimetype,
			Blob:     base64.StdEncoding.EncodeToString(contents),
		}
		content, err = json.Marshal(blob)
		if err != nil {
			return result, fmt.Errorf("failed to marshal blob resource contents: %w", err)
		}
	}

	// Wrap the marshaled content in json.RawMessage and add to the result
	result.Contents = []json.RawMessage{json.RawMessage(content)}
	return result, nil
}
