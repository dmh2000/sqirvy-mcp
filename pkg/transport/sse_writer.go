// Package transport provides functionality for transporting messages between components.
package transport

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// SSEPostWriter is a custom io.Writer that sends data as POST requests to an SSE endpoint
type SSEPostWriter struct {
	client  *http.Client
	url     string
	mu      sync.Mutex
	headers map[string]string
}

// Write implements the io.Writer interface by sending data as a POST request
func (w *SSEPostWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Create a new request with the data
	req, err := http.NewRequest(http.MethodPost, w.url, bytes.NewReader(p))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default Content-Type if not specified in headers
	if _, hasContentType := w.headers["Content-Type"]; !hasContentType {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set any custom headers
	for key, value := range w.headers {
		req.Header.Set(key, value)
	}

	// Send the request
	resp, err := w.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("server returned non-success status: %d", resp.StatusCode)
	}

	// Return the number of bytes written
	return len(p), nil
}

// NewSSEWriter creates a new SSEPostWriter that sends data to the specified endpoint
func NewSSEWriter(ip string, port int, path string) io.Writer {
	// Create the URL from the components
	url := fmt.Sprintf("http://%s:%d%s", ip, port, path)

	// Create a client with reasonable defaults
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	return &SSEPostWriter{
		client:  client,
		url:     url,
		headers: make(map[string]string),
	}
}

// NewSSEWriterWithHeaders creates a new SSEPostWriter with custom headers
func NewSSEWriterWithHeaders(ip string, port int, path string, headers map[string]string) io.Writer {
	writer := NewSSEWriter(ip, port, path).(*SSEPostWriter)
	writer.headers = headers
	return writer
}
