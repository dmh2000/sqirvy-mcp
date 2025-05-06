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
	MethodListResources          = "resources/list"
	MethodReadResource           = "resources/read"
	MethodListResourcesTemplates = "resources/templates/list" // Added for resource templates
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

// ListResourcesTemplatesParams defines the parameters for a "resources/templates/list" request.
type ListResourcesTemplatesParams struct {
	// Cursor is an opaque token for pagination.
	Cursor string `json:"cursor,omitempty"`
}

// ListResourcesTemplatesResult defines the result structure for a "resources/templates/list" response.
type ListResourcesTemplatesResult struct {
	// Meta contains reserved protocol metadata.
	Meta map[string]interface{} `json:"_meta,omitempty"`
	// NextCursor is an opaque token for the next page of results.
	NextCursor string `json:"nextCursor,omitempty"`
	// ResourcesTemplates is the list of resource templates found.
	ResourcesTemplates []ResourcesTemplates `json:"resourceTemplates"`
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

// MarshalListResourcesTemplatesRequest creates a JSON-RPC request for the resources/templates/list method.
// Intended for use by the client.
// The id can be a string or an integer. If params is nil, default empty params will be used.
func MarshalListResourcesTemplatesRequest(id RequestID, params *ListResourcesTemplatesParams) ([]byte, error) {
	// Use default empty params if nil is provided
	var p interface{}
	if params != nil {
		p = params
	} else {
		p = struct{}{} // Empty object for params if none specified
	}

	req := RPCRequest{
		JSONRPC: JSONRPCVersion,
		Method:  MethodListResourcesTemplates,
		Params:  p,
		ID:      id,
	}
	return json.Marshal(req)
}

// UnmarshalListResourcesTemplatesResult parses a JSON-RPC response for a resources/templates/list request.
// Intended for use by the client.
// It expects the standard JSON-RPC response format with the result nested in the "result" field.
// It returns the result, the response ID, any RPC error, and a general parsing error.
func UnmarshalListResourcesTemplatesResult(data []byte) (*ListResourcesTemplatesResult, RequestID, *RPCError, error) {
	var resp RPCResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to unmarshal RPC response: %w", err)
	}

	if resp.Error != nil {
		return nil, resp.ID, resp.Error, nil
	}

	if len(resp.Result) == 0 || string(resp.Result) == "null" {
		return nil, resp.ID, nil, fmt.Errorf("received response with missing or null result field for method %s", MethodListResourcesTemplates)
	}

	var result ListResourcesTemplatesResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, resp.ID, nil, fmt.Errorf("failed to unmarshal ListResourcesTemplatesResult from response result: %w", err)
	}

	return &result, resp.ID, nil, nil
}

// MarshalListResourcesTemplatesResult creates a JSON-RPC response containing the result of a resources/templates/list request.
// Intended for use by the server.
// It wraps the provided list of resource templates and cursor into a ListResourcesTemplatesResult and marshals it into a standard RPCResponse.
func MarshalListResourcesTemplatesResult(id RequestID, templatesListp []ResourcesTemplates, cursor string, logger *utils.Logger) ([]byte, error) {
	result := ListResourcesTemplatesResult{
		ResourcesTemplates: templatesListp,
		NextCursor:         cursor,
	}
	return MarshalResponse(id, result, logger)
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
// It unmarshals the entire request and specifically parses the `params` field into ReadResourceParams.
// It returns the parsed parameters, the request ID, any RPC error encountered during parsing, and a general parsing error.
func UnmarshalReadResourceRequest(payload []byte, logger *utils.Logger) (*ReadResourceParams, RequestID, *RPCError, error) {
	var req RPCRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		err = fmt.Errorf("failed to unmarshal base read resource request: %w", err)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeParseError, err.Error(), nil)
		// Return nil params, nil ID (as we couldn't parse it), the RPC error, and the Go error
		return nil, nil, rpcErr, err
	}

	// Now, unmarshal the Params field specifically into ReadResourceParams
	var params ReadResourceParams

	// Handle cases where params might be missing or explicitly null in the JSON
	rawParams, ok := req.Params.(json.RawMessage)
	if !ok && rawParams != nil {
		logger.Println(fmt.Sprintf("%v:%v", req.Params, rawParams))
		// This case means Params was not a JSON object/array/null, which is invalid for this method.
		err := fmt.Errorf("invalid type for params field: expected JSON object, got %T:%v", req.Params, req.Params)
		logger.Println("ERROR", err.Error())
		// Use InvalidRequest as the structure itself is wrong if params isn't marshalable
		rpcErr := NewRPCError(ErrorCodeInvalidRequest, "Invalid params field type", err.Error())
		return nil, req.ID, rpcErr, err
	}

	// For ReadResource, the 'params' object itself is required and must contain 'uri'.
	if len(rawParams) == 0 || string(rawParams) == "null" {
		err := fmt.Errorf("missing required params field for method %s", MethodReadResource)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeInvalidParams, "Missing required parameters object", nil)
		return nil, req.ID, rpcErr, err
	}

	// Attempt to unmarshal the params
	if err := json.Unmarshal(rawParams, &params); err != nil {
		err = fmt.Errorf("failed to unmarshal ReadResourceParams from request params: %w", err)
		logger.Println("ERROR", err.Error())
		// Use InvalidParams error code as the request structure was valid, but params content wasn't
		rpcErr := NewRPCError(ErrorCodeInvalidParams, "Invalid parameters for resources/read", err.Error())
		return nil, req.ID, rpcErr, err
	}

	// Validate required fields within params (e.g., URI must not be empty)
	if params.URI == "" {
		err := fmt.Errorf("missing required 'uri' field in params for method %s", MethodReadResource)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeInvalidParams, "Missing required 'uri' parameter", nil)
		return nil, req.ID, rpcErr, err
	}

	// Successfully parsed and validated params
	return &params, req.ID, nil, nil
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
