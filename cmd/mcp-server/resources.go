package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	// prompts "sqirvy/cmd/mcp-server/prompts"
	resources "sqirvy-mcp/cmd/mcp-server/resources"
	mcp "sqirvy-mcp/pkg/mcp"
)

// Define the example file resource as a package-level variable
var exampleFileResource mcp.Resource = mcp.Resource{
	Name:        "example.txt", // A user-friendly name
	URI:         "file:///documents/example.txt",
	Description: "An example text file.",
	MimeType:    "text/plain", // Assuming text/plain
	// Size could be determined by os.Stat if needed
}

// handleReadResource handles the "resources/read" request.
// It parses the request, determines the resource type (e.g., file, data),
// calls the appropriate reader function, and formats the response.
func (s *Server) handleReadResource(id mcp.RequestID, payload []byte) ([]byte, error) {
	s.logger.Printf("DEBUG", "Handle  : resources/read request (ID: %v)", id)

	var req mcp.RPCRequest
	var params mcp.ReadResourceParams
	if err := json.Unmarshal(payload, &req); err != nil {
		err = fmt.Errorf("failed to unmarshal base read resource request: %w", err)
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeParseError, err.Error(), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}

	// Explanation: While req.Params could be accessed directly as map[string]interface{},
	// re-marshalling and then unmarshalling into the specific params struct provides:
	// 1. Consistency with other handlers (e.g., initialize).
	// 2. Implicit validation against the expected struct (ReadResourceParams).
	// 3. Better maintainability if the params struct evolves.
	// 4. Type safety in subsequent code using the 'params' variable.

	// Marshal the params interface{} back to bytes
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		err = fmt.Errorf("failed to re-marshal read resource params: %w", err)
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInvalidParams, err.Error(), nil) // InvalidParams as structure was likely wrong
		return s.marshalErrorResponse(id, rpcErr)
	}

	// Now unmarshal the bytes into the specific params struct
	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		err = fmt.Errorf("failed to unmarshal specific read resource params: %w", err)
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInvalidParams, err.Error(), nil) // InvalidParams as content was wrong
		return s.marshalErrorResponse(id, rpcErr)
	}

	// Parse the URI
	parsedURI, err := url.Parse(params.URI)
	if err != nil {
		err = fmt.Errorf("failed to parse resource URI '%s': %w", params.URI, err)
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInvalidParams, err.Error(), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}

	// --- Route based on URI scheme/path ---
	var resourceContentBytes []byte
	var resourceMimeType string
	var resourceErr error

	switch parsedURI.Scheme {
	case "data":
		if parsedURI.Host == "random_data" {
			// Delegate to the specific handler in templates.go (which uses resources.RandomData)
			// Note: handleRandomDataResource already marshals the full response.
			return s.handleRandomDataResource(id, params, parsedURI)
		}
		resourceErr = fmt.Errorf("unsupported data URI host: %s", parsedURI.Host)

	case "file":
		// Delegate to the file reader in resources/read.go
		resourceContentBytes, resourceMimeType, resourceErr = resources.ReadFileResource(params.URI, s.logger)

	default:
		// Scheme not supported
		resourceErr = fmt.Errorf("resource URI scheme '%s' not supported", parsedURI.Scheme)
	}

	// --- Handle errors from resource reading ---
	if resourceErr != nil {
		s.logger.Printf("DEBUG", "Error reading resource URI '%s': %v", params.URI, resourceErr)
		// Determine appropriate RPC error code based on the error type
		// TODO: Refine error mapping (e.g., distinguish not found, permission denied)
		rpcErrCode := mcp.ErrorCodeInternalError // Default to internal error
		if strings.Contains(resourceErr.Error(), "not found") {
			// Use a specific code if available, e.g., a custom server error code or InvalidParams
			rpcErrCode = mcp.ErrorCodeInvalidParams // Or a custom -320xx code
		} else if strings.Contains(resourceErr.Error(), "permission denied") {
			rpcErrCode = mcp.ErrorCodeInternalError // Or a custom -320xx code
		} else if strings.Contains(resourceErr.Error(), "unsupported") || strings.Contains(resourceErr.Error(), "invalid") {
			rpcErrCode = mcp.ErrorCodeInvalidParams
		}
		rpcErr := mcp.NewRPCError(rpcErrCode, resourceErr.Error(), map[string]string{"uri": params.URI})
		return s.marshalErrorResponse(id, rpcErr)
	}

	result, err := mcp.NewReadResourcesResult(params.URI, resourceMimeType, resourceContentBytes)
	if err != nil {
		err = fmt.Errorf("failed to create read resource result for %s: %w", params.URI, err)
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInternalError, err.Error(), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}

	return s.marshalResponse(id, result)
}

// --- Prepare successful response ---
// Create the appropriate content structure (Text or Blob)
// For now, assume text based on our simple ReadFileResource
// TODO: Add logic to create BlobResourceContents if mimeType indicates binary
// var resourceContents interface{}
// if strings.HasPrefix(resourceMimeType, "text/") || resourceMimeType == "application/json" { // Basic check for text
// 	resourceContents = mcp.TextResourceContents{
// 		URI:      params.URI,
// 		MimeType: resourceMimeType,
// 		Text:     string(resourceContentBytes),
// 	}
// } else {
// 	resourceContents = mcp.BlobResourceContents{
// 		URI:      params.URI,
// 		MimeType: resourceMimeType,
// 		Blob:     base64.StdEncoding.EncodeToString(resourceContentBytes),
// 	}
// }

// Marshal the specific content structure (TextResourceContents)
// contentBytes, err := json.Marshal(resourceContents)
// if err != nil {
// 	err = fmt.Errorf("failed to marshal resource contents for %s: %w", params.URI, err)
// 	s.logger.Println("DEBUG", err.Error())
// 	rpcErr := mcp.NewRPCError(mcp.ErrorCodeInternalError, err.Error(), nil)
// 	return s.marshalErrorResponse(id, rpcErr)
// }

// Create the final result structure containing the marshalled content
// result := mcp.ReadResourceResult{
// 	Contents: []json.RawMessage{json.RawMessage(content)},
// }
