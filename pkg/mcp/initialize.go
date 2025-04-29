package mcp

import (
	"encoding/json"
	"fmt"
	utils "sqirvy-mcp/pkg/utils"
)

// MethodInitialize is the method name for the initialize request.
const MethodInitialize = "initialize"

// Implementation describes the name and version of an MCP implementation (client or server).
type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientCapabilities defines the capabilities a client may support.
// Using map[string]interface{} for flexibility with experimental and future capabilities.
type ClientCapabilities struct {
	// Experimental holds non-standard capabilities.
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	// Roots indicates support for listing roots.
	Roots *struct {
		ListChanged bool `json:"listChanged,omitempty"`
	} `json:"roots,omitempty"`
	// Sampling indicates support for LLM sampling.
	Sampling map[string]interface{} `json:"sampling,omitempty"` // Use map for flexibility
}

// InitializeParams defines the parameters for an "initialize" request.
type InitializeParams struct {
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      Implementation     `json:"clientInfo"`
	ProtocolVersion string             `json:"protocolVersion"`
	// Add other optional fields from the spec like processId, rootUri, trace, workspaceFolders if needed.
}

// ServerCapabilities defines the capabilities a server may support.
// Using map[string]interface{} for flexibility.
type ServerCapabilities struct {
	// Experimental holds non-standard capabilities.
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	// Logging indicates support for sending log messages.
	Logging map[string]interface{} `json:"logging,omitempty"` // Use map for flexibility
	// Prompts indicates support for prompt templates.
	Prompts *ServerCapabilitiesPrompts `json:"prompts,omitempty"`
	// Resources indicates support for resources.
	Resources *ServerCapabilitiesResources `json:"resources,omitempty"`
	// Tools indicates support for tools.
	Tools *ServerCapabilitiesTools `json:"tools,omitempty"`
	// Add other capabilities like completion if needed.
}

// ServerCapabilitiesPrompts defines specific capabilities related to prompts.
type ServerCapabilitiesPrompts struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ServerCapabilitiesResources defines specific capabilities related to resources.
type ServerCapabilitiesResources struct {
	ListChanged bool `json:"listChanged,omitempty"`
	Subscribe   bool `json:"subscribe,omitempty"` // Added subscribe capability
}

// ServerCapabilitiesTools defines specific capabilities related to tools.
type ServerCapabilitiesTools struct {
	ListChanged bool `json:"listChanged,omitempty"`
	// Add other tool-related capabilities here if needed
}

// InitializeResult defines the result structure for an "initialize" response.
type InitializeResult struct {
	// Meta contains reserved protocol metadata.
	Meta            map[string]interface{} `json:"_meta,omitempty"`
	Capabilities    ServerCapabilities     `json:"capabilities"`
	Instructions    string                 `json:"instructions,omitempty"`
	ProtocolVersion string                 `json:"protocolVersion"`
	ServerInfo      Implementation         `json:"serverInfo"`
}

// MarshalInitializeRequest creates a JSON-RPC request for the initialize method.
// Intended for use by the client.
// The id can be a string or an integer.
func MarshalInitializeRequest(id RequestID, params InitializeParams) ([]byte, error) {
	req := RPCRequest{
		JSONRPC: JSONRPCVersion,
		Method:  MethodInitialize,
		Params:  params,
		ID:      id,
	}
	return json.Marshal(req)
}

// UnmarshalInitializeResult parses a JSON-RPC response for an initialize request.
// Intended for use by the client.
// It expects the standard JSON-RPC response format with the result nested in the "result" field.
// It returns the result, the response ID, any RPC error, and a general parsing error.
func UnmarshalInitializeResult(data []byte) (*InitializeResult, RequestID, *RPCError, error) {
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
		return nil, resp.ID, nil, fmt.Errorf("received response with missing or null result field for method %s", MethodInitialize)
	}

	// Unmarshal the actual result from the Result field
	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, resp.ID, nil, fmt.Errorf("failed to unmarshal InitializeResult from response result: %w", err)
	}

	return &result, resp.ID, nil, nil
}

// ---------------------------------------------------------
// Request Unmarshaling (Server-Side)
// ---------------------------------------------------------

// UnmarshalInitializeRequest parses the parameters from a JSON-RPC request for the initialize method.
// Intended for use by the server.
// It unmarshals the entire request and specifically parses the `params` field into InitializeParams.
// It returns the parsed parameters, the request ID, any RPC error encountered during parsing, and a general parsing error.
func UnmarshalInitializeRequest(payload []byte, logger *utils.Logger) (*InitializeParams, RequestID, *RPCError, error) {
	var req RPCRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		err = fmt.Errorf("failed to unmarshal base initialize request: %w", err)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeParseError, err.Error(), nil)
		// Return nil params, nil ID (as we couldn't parse it), the RPC error, and the Go error
		return nil, nil, rpcErr, err
	}

	// Now, unmarshal the Params field specifically into InitializeParams
	var params InitializeParams

	// Handle cases where params might be missing or explicitly null in the JSON
	rawParams, ok := req.Params.(json.RawMessage)
	if !ok && req.Params != nil {
		// This case means Params was not a JSON object/array/null, which is invalid for this method.
		err := fmt.Errorf("invalid type for params field: expected JSON object, got %T", req.Params)
		logger.Println("ERROR", err.Error())
		// Use InvalidRequest as the structure itself is wrong if params isn't marshalable
		rpcErr := NewRPCError(ErrorCodeInvalidRequest, "Invalid params field type", err.Error())
		return nil, req.ID, rpcErr, err
	}

	// For Initialize, the 'params' object itself is required.
	if len(rawParams) == 0 || string(rawParams) == "null" {
		err := fmt.Errorf("missing required params field for method %s", MethodInitialize)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeInvalidParams, "Missing required parameters object", nil)
		return nil, req.ID, rpcErr, err
	}

	// Attempt to unmarshal the params
	if err := json.Unmarshal(rawParams, &params); err != nil {
		err = fmt.Errorf("failed to unmarshal InitializeParams from request params: %w", err)
		logger.Println("ERROR", err.Error())
		// Use InvalidParams error code as the request structure was valid, but params content wasn't
		rpcErr := NewRPCError(ErrorCodeInvalidParams, "Invalid parameters for initialize", err.Error())
		return nil, req.ID, rpcErr, err
	}

	// Validate required fields within params
	if params.ProtocolVersion == "" {
		err := fmt.Errorf("missing required 'protocolVersion' field in params for method %s", MethodInitialize)
		logger.Println("ERROR", err.Error())
		rpcErr := NewRPCError(ErrorCodeInvalidParams, "Missing required 'protocolVersion' parameter", nil)
		return nil, req.ID, rpcErr, err
	}
	// ClientInfo and Capabilities are structs, so they will exist but might be empty.
	// Further validation could be added here if specific fields within them are required.
	// For example, checking ClientInfo.Name:
	// if params.ClientInfo.Name == "" { ... }

	// Successfully parsed and validated params
	return &params, req.ID, nil, nil
}

// ---------------------------------------------------------
// Response Marshaling (Server-Side)
// ---------------------------------------------------------

// MarshalInitializeResult marshals a successful InitializeResult into a full RPCResponse.
// Intended for use by the server.
// Returns the marshalled bytes and any error during marshalling.
// It does *not* send the bytes itself.
func MarshalInitializeResult(id RequestID, result InitializeResult, logger *utils.Logger) ([]byte, error) {
	return MarshalResponse(id, result, logger)
}

func NewInitializeResult(
	prompts *ServerCapabilitiesPrompts,
	resources *ServerCapabilitiesResources,
	tools *ServerCapabilitiesTools,
) InitializeResult {
	return InitializeResult{
		ProtocolVersion: protocolVersion,
		Capabilities: ServerCapabilities{
			Prompts:   prompts,
			Resources: resources,
			Tools:     tools,
		},
		ServerInfo: Implementation{
			Name:    serverName,
			Version: serverVersion,
		},
	}
}
