// Package transport provides functionality for transporting messages between components.
package transport

import (
	"fmt"
	"io"
	"net/http"
)

// NewSSEReader initiates a connection to a Server-Sent Events (SSE) endpoint
// and returns a reader for the event stream.
//
// Parameters:
//   - ip: The IP address of the SSE server.
//   - port: The port number of the SSE server.
//   - path: The path to the SSE endpoint on the server.
//
// Returns:
//   - An io.Reader connected to the SSE stream if the connection is successful.
//   - An error if the connection fails or the server returns a non-200 status code.
func NewSSEReader(ip string, port int, path string) (io.Reader, error) {
	// Construct the URL
	url := fmt.Sprintf("http://%s:%d%s", ip, port, path)

	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSE request: %w", err)
	}

	// Set the necessary header for SSE
	req.Header.Set("Accept", "text/event-stream")

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SSE request to %s: %w", url, err)
	}

	// Check if the response status is OK (200)
	if resp.StatusCode != http.StatusOK {
		// Make sure to close the body even if the status is wrong
		resp.Body.Close()
		return nil, fmt.Errorf("received non-200 status code (%d) from SSE endpoint %s", resp.StatusCode, url)
	}

	// Return the response body which is the stream reader
	return resp.Body, nil
}
