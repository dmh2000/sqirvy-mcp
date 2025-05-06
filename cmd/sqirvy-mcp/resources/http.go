package resources

import (
	"fmt"
	"io"
	"net/http"
	"time"

	utils "sqirvy-mcp/pkg/utils"
)

// ReadHTTPResource fetches data from the specified HTTP URL and returns
// the raw bytes, MIME type, and any error encountered.
func ReadHTTPResource(uri string, logger *utils.Logger) ([]byte, string, error) {
	logger.Printf("DEBUG", "Fetching HTTP resource: %s", uri)

	// Create an HTTP client with reasonable timeouts
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create a new request
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, "", fmt.Errorf("error creating HTTP request: %w", err)
	}

	// Set a user agent to identify the client
	req.Header.Set("User-Agent", "Sqirvy-MCP/1.0")

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("error fetching HTTP resource: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("HTTP request failed with status code: %d", resp.StatusCode)
	}

	// Read the response body
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("error reading HTTP response: %w", err)
	}

	// Get the content type from the response headers
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		// Default to application/octet-stream if no Content-Type is provided
		mimeType = "application/octet-stream"
	}

	logger.Printf("DEBUG", "Successfully fetched HTTP resource (%d bytes, type: %s)", len(content), mimeType)
	return content, mimeType, nil
}
