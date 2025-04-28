package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	// Added for crypto/rand.Int
	resources "sqirvy-mcp/cmd/mcp-server/resources"
	mcp "sqirvy-mcp/pkg/mcp"
	// Import the custom logger
)

// Define the random_data template
var RandomDataTemplate mcp.ResourcesTemplates = mcp.ResourcesTemplates{
	Name:        "random_data",
	URITemplate: "data://random_data?length={length}", // RFC 6570 template
	Description: "Returns a string of random ASCII characters. Use URI like 'data://random_data?length=N' in resources/read, where N is the desired length.",
	MimeType:    "text/plain",
}

// handleRandomDataResource processes a read request specifically for the data://random_data URI.
// It extracts the length, generates data, and marshals the response or error.
func (s *Server) handleRandomDataResource(id mcp.RequestID, params mcp.ReadResourceParams, parsedURI *url.URL) ([]byte, error) {
	s.logger.Printf("DEBUG", "Processing random_data resource for URI: %s", params.URI)

	// Get the length parameter
	lengthStr := parsedURI.Query().Get("length")
	if lengthStr == "" {
		err := fmt.Errorf("missing 'length' query parameter in URI: %s", params.URI)
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInvalidParams, err.Error(), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}

	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		err = fmt.Errorf("invalid 'length' query parameter '%s': %w", lengthStr, err)
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInvalidParams, err.Error(), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}

	// Generate random data using the function from resources.go
	randomString, err := resources.RandomData(length)
	if err != nil {
		// RandomData already logs details, just wrap the error for the RPC response
		err = fmt.Errorf("failed to generate random data for URI %s: %w", params.URI, err)
		s.logger.Println("DEBUG", err.Error())
		// Check if the error was due to invalid length (positive, max)
		// Use errors.Is for specific error types if RandomData returns them, otherwise check message
		if strings.Contains(err.Error(), "length must be positive") || strings.Contains(err.Error(), "exceeds maximum allowed length") {
			rpcErr := mcp.NewRPCError(mcp.ErrorCodeInvalidParams, err.Error(), nil)
			return s.marshalErrorResponse(id, rpcErr)
		}
		// Otherwise, treat as internal error
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInternalError, err.Error(), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}

	// Prepare the result content
	content := mcp.TextResourceContents{
		URI:      params.URI,
		MimeType: "text/plain",
		Text:     randomString,
	}
	contentBytes, err := json.Marshal(content)
	if err != nil {
		err = fmt.Errorf("failed to marshal TextResourceContents for %s: %w", params.URI, err)
		s.logger.Println("DEBUG", err.Error())
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInternalError, err.Error(), nil)
		return s.marshalErrorResponse(id, rpcErr)
	}

	result := mcp.ReadResourceResult{
		Contents: []json.RawMessage{json.RawMessage(contentBytes)},
	}
	return s.marshalResponse(id, result)
}
