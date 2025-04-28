package main

import (
	"encoding/json"
	"fmt"

	// prompts "sqirvy/cmd/mcp-server/prompts"
	prompts "sqirvy-mcp/cmd/mcp-server/prompts"
	mcp "sqirvy-mcp/pkg/mcp"
)

const (
	QueryPromptName = "query"
)

// handleQueryPrompt handles the "prompts/get" request for the sqirvy_query prompt
// It returns the prompt messages as defined in the sqirvyPrompt function
func (s *Server) handleQueryPrompt(id mcp.RequestID, params mcp.GetPromptParams) ([]byte, error) {
	s.logger.Printf("DEBUG", "Handle  : prompts/get request for '%s' (ID: %v)", params.Name, id)

	// Create a text content message with the prompt
	content := mcp.TextContent{
		Type: "text",
		Text: prompts.QueryPrompt(params.Name, params.Arguments),
	}

	// Marshal the content into json.RawMessage
	contentBytes, err := json.Marshal(content)
	if err != nil {
		err = fmt.Errorf("failed to marshal sqirvy_query prompt content: %w", err)
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInternalError, err.Error(), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}

	// Create the prompt message with the system role
	message := mcp.PromptMessage{
		Role:    mcp.RoleAssistant,
		Content: json.RawMessage(contentBytes),
	}

	// Create the result with the message
	result := mcp.GetPromptResult{
		Description: "A prompt for querying information using the Sqirvy system",
		Messages:    []mcp.PromptMessage{message},
	}

	// Marshal the successful response
	return s.marshalResponse(id, result)
}
