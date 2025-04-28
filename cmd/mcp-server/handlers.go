package main

import (
	"encoding/json"
	"fmt"

	mcp "sqirvy-mcp/pkg/mcp"
)

// --- Initialization Handler ---

// handleInitializeRequest handles the "initialize" request.
// It validates the request, performs capability negotiation (currently basic),
// and returns the marshalled InitializeResult response bytes or marshalled error response bytes.
func (s *Server) handleInitializeRequest(id mcp.RequestID, payload []byte) ([]byte, error) {
	var req mcp.RPCRequest // Use the base request type first
	if err := json.Unmarshal(payload, &req); err != nil {
		err = fmt.Errorf("failed to unmarshal base initialize request structure: %w", err)
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeParseError, err.Error(), nil)
		// Marshal and return the error response bytes
		errorBytes, marshalErr := s.marshalErrorResponse(id, rpcErr)
		if marshalErr != nil {
			return nil, marshalErr // Return marshalling error if that fails too
		}
		return errorBytes, err // Return marshalled error and the original error
	}

	// Check if Params field is present and is a valid JSON object/array
	if req.Params == nil {
		err := fmt.Errorf("initialize request missing 'params' field")
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInvalidRequest, err.Error(), nil)
		errorBytes, marshalErr := s.marshalErrorResponse(id, rpcErr)
		if marshalErr != nil {
			return nil, marshalErr
		}
		return errorBytes, err
	}

	// Ensure req.Params is json.RawMessage before unmarshalling into specific type
	paramsRaw, ok := req.Params.(json.RawMessage)
	if !ok {
		// This might happen if params is not an object/array in the JSON
		// Attempt to remarshal and then treat as RawMessage if needed, or handle error
		tempParamsBytes, err := json.Marshal(req.Params)
		if err != nil {
			err = fmt.Errorf("initialize request 'params' field is not a valid JSON object/array (marshal check failed): %w", err)
			s.logger.Println("DEBUG", err.Error())
			rpcErr := mcp.NewRPCError(mcp.ErrorCodeInvalidParams, err.Error(), nil)
			errorBytes, marshalErr := s.marshalErrorResponse(id, rpcErr)
			if marshalErr != nil {
				return nil, marshalErr
			}
			return errorBytes, err
		}
		paramsRaw = json.RawMessage(tempParamsBytes)
	}

	// Now unmarshal params specifically into InitializeParams
	var params mcp.InitializeParams
	if err := json.Unmarshal(paramsRaw, &params); err != nil {
		err = fmt.Errorf("failed to unmarshal initialize params object: %w", err)
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInvalidParams, err.Error(), nil)
		errorBytes, marshalErr := s.marshalErrorResponse(id, rpcErr)
		if marshalErr != nil {
			return nil, marshalErr
		}
		return errorBytes, err
	}

	// --- Capability Negotiation (Basic Example) ---
	if params.ProtocolVersion == "" {
		err := fmt.Errorf("client initialize request missing protocolVersion")
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInvalidParams, err.Error(), nil)
		errorBytes, marshalErr := s.marshalErrorResponse(id, rpcErr)
		if marshalErr != nil {
			return nil, marshalErr
		}
		return errorBytes, err
	}
	// Basic check: Log if client version differs, but proceed using our version.
	if params.ProtocolVersion != s.serverVersion {
		s.logger.Printf("DEBUG", "Client requested protocol version '%s', server using '%s'", params.ProtocolVersion, s.serverVersion)
	}
	// TODO: Add more robust version negotiation if needed.
	// TODO: Inspect params.Capabilities and potentially enable/disable server features.

	// // --- Prepare Response ---
	result := mcp.NewInitializeResult(
		&mcp.ServerCapabilitiesPrompts{ListChanged: false},
		&mcp.ServerCapabilitiesResources{ListChanged: false, Subscribe: false},
		&mcp.ServerCapabilitiesTools{ListChanged: false},
	)

	responseBytes, err := mcp.MarshalInitializeResult(id, result, s.logger)
	if err != nil {
		return responseBytes, err // Return the error bytes and the original marshalling error
	}

	return responseBytes, nil // Return success response bytes and nil error
}

// --- Handlers for other methods ---
// These handlers now return the marshalled response/error bytes and any error encountered during marshalling.
// They no longer call sendResponse/sendErrorResponse directly.

func (s *Server) handleListTools(id mcp.RequestID) ([]byte, error) {
	s.logger.Printf("DEBUG", "Handle  : tools/list request (ID: %v)", id)

	// Define the ping tool
	pingTool := mcp.Tool{
		Name:        pingToolName, // Use constant from ping.go
		Description: "Pings the network address once.",
		InputSchema: mcp.ToolInputSchema{
			"type": "object",
			"properties": map[string]interface{}{
				"address": map[string]interface{}{
					"type":        "string",
					"description": "The IP address or hostname to ping",
				},
			},
			"required": []string{"address"},
		},
	}

	// TODO: Add other tools here if needed.
	tools := []mcp.Tool{pingTool}

	result := mcp.ListToolsResult{
		Tools: tools,
		// NextCursor: "", // Omit if no pagination needed yet
	}
	// Marshal the success response
	return s.marshalResponse(id, result)
}

// handleCallTool parses the tool call request and routes to the specific tool handler.
// Note: This function is now primarily responsible for parsing and routing.
// The actual tool logic is delegated (e.g., to handlePingTool).
func (s *Server) handleCallTool(id mcp.RequestID, payload []byte) ([]byte, error) {
	s.logger.Printf("DEBUG", "Handle  : tools/call request (ID: %v)", id)

	var req mcp.RPCRequest
	var params mcp.CallToolParams

	// Unmarshal the base request to access params
	if err := json.Unmarshal(payload, &req); err != nil {
		err = fmt.Errorf("failed to unmarshal base tool call request: %w", err)
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeParseError, err.Error(), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}

	// Marshal params back to bytes
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		err = fmt.Errorf("failed to re-marshal tool call params: %w", err)
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInvalidParams, err.Error(), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}

	// Unmarshal into the specific CallToolParams struct
	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		err = fmt.Errorf("failed to unmarshal specific tool call params: %w", err)
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInvalidParams, err.Error(), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}

	// Route based on the tool name
	switch params.Name {
	case pingToolName:
		// Delegate to the specific handler in ping.go
		return s.handlePingTool(id, params)
	// Add cases for other tools here
	// case "another_tool":
	//     return s.handleAnotherTool(id, params)
	default:
		s.logger.Printf("DEBUG", "Received call for unknown tool '%s' (ID: %v)", params.Name, id)
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeMethodNotFound, fmt.Sprintf("Tool '%s' not found", params.Name), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}
}

func (s *Server) handleListPrompts(id mcp.RequestID) ([]byte, error) {
	s.logger.Printf("DEBUG", "Handle  : prompts/list request (ID: %v)", id)

	// Define the query prompt
	sqirvyQueryPrompt := mcp.Prompt{
		Name:        QueryPromptName,
		Description: "A prompt for querying information using the Sqirvy system",
		Arguments: []mcp.PromptArgument{
			{Name: "A", Description: "The user's query", Required: false},
			{Name: "B", Description: "The user's query", Required: false},
			{Name: "C", Description: "The user's query", Required: false},
		},
	}

	p := []mcp.Prompt{sqirvyQueryPrompt}
	r := mcp.NewListPromptResult(p)
	return s.marshalResponse(id, r)
}

func (s *Server) handleGetPrompt(id mcp.RequestID, payload []byte) ([]byte, error) {
	s.logger.Printf("DEBUG", "Handle  : prompts/get request (ID: %v)", id)

	var req mcp.RPCRequest
	var params mcp.GetPromptParams

	// Unmarshal the base request to access params
	if err := json.Unmarshal(payload, &req); err != nil {
		err = fmt.Errorf("failed to unmarshal base get prompt request: %w", err)
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeParseError, err.Error(), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}

	// Marshal params back to bytes
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		err = fmt.Errorf("failed to re-marshal get prompt params: %w", err)
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInvalidParams, err.Error(), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}

	// Unmarshal into the specific GetPromptParams struct
	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		err = fmt.Errorf("failed to unmarshal specific get prompt params: %w", err)
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInvalidParams, err.Error(), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}

	// Route based on the prompt name
	switch params.Name {
	case QueryPromptName:
		// Delegate to the specific handler in sqirvy_query.go
		return s.handleQueryPrompt(id, params)
	default:
		s.logger.Printf("DEBUG", "Received get request for unknown prompt '%s' (ID: %v)", params.Name, id)
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeMethodNotFound, fmt.Sprintf("Prompt '%s' not found", params.Name), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}
}

func (s *Server) handleListResources(id mcp.RequestID) ([]byte, error) {
	s.logger.Printf("DEBUG", "Handle  : resources/list request (ID: %v)", id)

	resourcesList := []mcp.Resource{exampleFileResource} // Use the package-level variable
	result, err := mcp.MarshalListResourcesResult(id, resourcesList, "", s.logger)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// handleListResourcesTemplatess handles the "resources/templates/list" request.
func (s *Server) handleListResourcesTemplatess(id mcp.RequestID) ([]byte, error) {
	s.logger.Printf("DEBUG", "Handle  : resources/templates/list request (ID: %v)", id)

	// TODO: Add other resource templates here if needed
	templates := []mcp.ResourcesTemplates{RandomDataTemplate}

	result := mcp.ListResourcesTemplatessResult{
		ResourcesTemplatess: templates,
		// NextCursor: "", // Implement pagination if needed
	}
	return s.marshalResponse(id, result)
}
