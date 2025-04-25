package transport

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestNewSSEReader_Success tests the successful creation of an SSE reader
// and reading data from a mock SSE server.
func TestNewSSEReader_Success(t *testing.T) {
	expectedData := "data: hello\n\ndata: world\n\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check method and headers
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if accept := r.Header.Get("Accept"); accept != "text/event-stream" {
			t.Errorf("Expected Accept header 'text/event-stream', got '%s'", accept)
		}

		// Send SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(expectedData)) // Ignore write error in test server
	}))
	defer server.Close()

	// Parse server URL to get IP, port, path
	parsedURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Failed to parse test server URL: %v", err)
	}
	hostParts := strings.Split(parsedURL.Host, ":")
	ip := hostParts[0]
	port, _ := strconv.Atoi(hostParts[1])
	path := parsedURL.Path

	// Call NewSSEReader
	reader, err := NewSSEReader(ip, port, path)
	if err != nil {
		t.Fatalf("NewSSEReader failed: %v", err)
	}
	defer func() {
		if closer, ok := reader.(io.Closer); ok {
			closer.Close()
		}
	}()

	// Read data from the reader
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read from SSE reader: %v", err)
	}

	// Verify the data
	if string(data) != expectedData {
		t.Errorf("Expected data %q, got %q", expectedData, string(data))
	}
}

// TestNewSSEReader_ServerError tests NewSSEReader when the server returns a non-200 status.
func TestNewSSEReader_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	parsedURL, _ := url.Parse(server.URL)
	hostParts := strings.Split(parsedURL.Host, ":")
	ip := hostParts[0]
	port, _ := strconv.Atoi(hostParts[1])
	path := parsedURL.Path

	_, err := NewSSEReader(ip, port, path)
	if err == nil {
		t.Fatal("Expected an error due to non-200 status, but got nil")
	}

	if !strings.Contains(err.Error(), "received non-200 status code (500)") {
		t.Errorf("Expected error message to contain 'received non-200 status code (500)', got: %v", err)
	}
}

// TestNewSSEReader_ConnectionError tests NewSSEReader when the server is unreachable.
func TestNewSSEReader_ConnectionError(t *testing.T) {
	// Use a port that is likely not in use
	ip := "127.0.0.1"
	port := 65530 // High port, unlikely to be used
	path := "/events"

	_, err := NewSSEReader(ip, port, path)
	if err == nil {
		t.Fatal("Expected a connection error, but got nil")
	}

	// Check if the error indicates a connection failure (specific error might vary)
	if !strings.Contains(err.Error(), "connection refused") && !strings.Contains(err.Error(), "failed to execute SSE request") {
		t.Errorf("Expected a connection error, got: %v", err)
	}
}

// TestNewSSEWriter_Success tests writing data using the writer returned by NewSSEWriter.
func TestNewSSEWriter_Success(t *testing.T) {
	expectedData := `{"message":"hello sse"}`
	var receivedData []byte
	var receivedContentType string
	var wg sync.WaitGroup
	wg.Add(1) // Expect one request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		// Check method and headers
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		receivedContentType = r.Header.Get("Content-Type")
		if receivedContentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", receivedContentType)
		}

		// Read body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		receivedData = body

		// Send success response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	parsedURL, _ := url.Parse(server.URL)
	hostParts := strings.Split(parsedURL.Host, ":")
	ip := hostParts[0]
	port, _ := strconv.Atoi(hostParts[1])
	path := parsedURL.Path

	// Get the writer factory
	writerFactory, err := NewSSEWriter(ip, port, path)
	if err != nil {
		t.Fatalf("NewSSEWriter failed: %v", err)
	}

	// Get a writer instance
	writer, err := writerFactory()
	if err != nil {
		t.Fatalf("Writer factory failed: %v", err)
	}

	// Write data
	_, err = writer.Write([]byte(expectedData))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Close the writer to trigger the POST
	err = writer.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Wait for the server handler to complete
	if waitTimeout(&wg, 1*time.Second) {
		t.Fatal("Timeout waiting for server handler")
	}

	// Verify received data
	if string(receivedData) != expectedData {
		t.Errorf("Expected server to receive data %q, got %q", expectedData, string(receivedData))
	}
}

// TestNewSSEWriter_ServerError tests NewSSEWriter when the server returns a non-2xx status.
func TestNewSSEWriter_ServerError(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		w.WriteHeader(http.StatusBadRequest) // 400 Bad Request
		_, _ = w.Write([]byte("Invalid request format"))
	}))
	defer server.Close()

	parsedURL, _ := url.Parse(server.URL)
	hostParts := strings.Split(parsedURL.Host, ":")
	ip := hostParts[0]
	port, _ := strconv.Atoi(hostParts[1])
	path := parsedURL.Path

	writerFactory, _ := NewSSEWriter(ip, port, path)
	writer, _ := writerFactory()

	_, _ = writer.Write([]byte(`{"data":1}`))
	err := writer.Close() // This triggers the POST

	if err == nil {
		t.Fatal("Expected an error due to non-2xx status, but got nil")
	}

	if !strings.Contains(err.Error(), "received non-2xx status code (400)") || !strings.Contains(err.Error(), "Invalid request format") {
		t.Errorf("Expected error message indicating 400 status and body, got: %v", err)
	}

	// Ensure handler was called
	if waitTimeout(&wg, 1*time.Second) {
		t.Fatal("Timeout waiting for server handler")
	}
}

// TestNewSSEWriter_ConnectionError tests NewSSEWriter when the server is unreachable.
func TestNewSSEWriter_ConnectionError(t *testing.T) {
	ip := "127.0.0.1"
	port := 65531 // Another high port
	path := "/post"

	writerFactory, _ := NewSSEWriter(ip, port, path)
	writer, _ := writerFactory()

	_, _ = writer.Write([]byte(`{"data":2}`))
	err := writer.Close() // This triggers the POST

	if err == nil {
		t.Fatal("Expected a connection error, but got nil")
	}

	if !strings.Contains(err.Error(), "connection refused") && !strings.Contains(err.Error(), "failed to execute SSE POST request") {
		t.Errorf("Expected a connection error, got: %v", err)
	}
}

// TestPostWriter_WriteAfterClose tests that writing to a closed postWriter returns an error.
func TestPostWriter_WriteAfterClose(t *testing.T) {
	pw := &postWriter{
		url:    "http://example.com",
		buffer: bytes.NewBuffer([]byte{}),
		closed: false,
	}

	// Close it (without actually sending)
	pw.closed = true

	_, err := pw.Write([]byte("test"))
	if err != io.ErrClosedPipe {
		t.Errorf("Expected error %v when writing to closed writer, got %v", io.ErrClosedPipe, err)
	}
}

// TestPostWriter_CloseAfterClose tests that closing an already closed postWriter returns an error.
func TestPostWriter_CloseAfterClose(t *testing.T) {
	pw := &postWriter{
		url:    "http://example.com",
		buffer: bytes.NewBuffer([]byte{}),
		closed: false,
	}

	// Close it (mark as closed)
	pw.closed = true

	err := pw.Close()
	if err != io.ErrClosedPipe {
		t.Errorf("Expected error %v when closing already closed writer, got %v", io.ErrClosedPipe, err)
	}
}

// waitTimeout waits for the waitgroup for the specified duration.
// Returns true if waiting timed out.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}
