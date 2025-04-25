// Package transport provides functionality for transporting messages between components.
package transport

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// postWriter implements io.WriteCloser. It buffers writes and sends the
// accumulated data as an HTTP POST request upon closing.
type postWriter struct {
	url    string
	buffer *bytes.Buffer
	closed bool
}

// Write appends data to the internal buffer.
func (pw *postWriter) Write(p []byte) (n int, err error) {
	if pw.closed {
		return 0, io.ErrClosedPipe // Use io.ErrClosedPipe to indicate the writer is closed
	}
	// Delegate writing to the buffer
	return pw.buffer.Write(p)
}

// Close sends the buffered data via HTTP POST to the configured URL.
func (pw *postWriter) Close() error {
	if pw.closed {
		return io.ErrClosedPipe // Already closed
	}
	pw.closed = true // Mark as closed immediately

	// Create the POST request with the buffered data
	req, err := http.NewRequest("POST", pw.url, pw.buffer) // bytes.Buffer implements io.Reader
	if err != nil {
		return fmt.Errorf("failed to create SSE POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	client := &http.Client{} // Consider using a shared client if needed
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute SSE POST request to %s: %w", pw.url, err)
	}
	defer resp.Body.Close() // Ensure the response body is always closed

	// Check for successful status codes (2xx range)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Attempt to read the body for more context, but ignore errors here
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("received non-2xx status code (%d) from SSE POST endpoint %s: %s", resp.StatusCode, pw.url, string(bodyBytes))
	}

	// Success
	return nil
}

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

// NewSSEWriter creates a factory function that produces io.WriteClosers.
// Each WriteCloser, when closed, sends the data written to it via HTTP POST
// to the specified endpoint with Content-Type "application/json".
//
// Parameters:
//   - ip: The IP address of the target server.
//   - port: The port number of the target server.
//   - path: The path on the target server.
//
// Returns:
//   - A function `func() (io.WriteCloser, error)` that creates a new writer instance.
//   - An error during factory creation is not expected with this implementation, so it's nil.
func NewSSEWriter(ip string, port int, path string) (func() (io.WriteCloser, error), error) {
	// Construct the URL once for the closure
	url := fmt.Sprintf("http://%s:%d%s", ip, port, path)

	// Return the factory function (closure)
	return func() (io.WriteCloser, error) {
		// Each call to this function creates a new postWriter instance
		// with its own buffer.
		return &postWriter{
			url:    url,
			buffer: bytes.NewBuffer([]byte{}), // Initialize with an empty buffer
			closed: false,
		}, nil // No error expected during writer creation itself
	}, nil // No error during the creation of the factory function itself
}
