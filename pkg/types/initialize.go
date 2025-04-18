package mcp

import (
	"encoding/json"
	"fmt"
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

// UnmarshalInitializeResponse parses a JSON-RPC response for an initialize request.
// It expects the standard JSON-RPC response format with the result nested in the "result" field.
// It returns the result, the response ID, any RPC error, and a general parsing error.
func UnmarshalInitializeResponse(data []byte) (*InitializeResult, RequestID, *RPCError, error) {
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
