package transport

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSSEPostWriter(t *testing.T) {
	// Create a test server
	var receivedBody []byte
	var receivedContentType string
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		
		// Read body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		
		// Store received data for verification
		receivedBody = body
		receivedContentType = r.Header.Get("Content-Type")
		
		// Return success
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	
	// Parse server URL to get host and port
	serverURL := server.URL
	parts := strings.Split(strings.TrimPrefix(serverURL, "http://"), ":")
	host := parts[0]
	port := 0
	if len(parts) > 1 {
		// Extract port number from URL
		portStr := parts[1]
		_, err := fmt.Sscanf(portStr, "%d", &port)
		if err != nil {
			t.Fatalf("Failed to parse port from URL %s: %v", serverURL, err)
		}
	}
	
	// Test data
	testData := []byte(`{"message":"Hello, SSE!"}`)
	
	// Create SSE writer
	writer := NewSSEWriter(host, port, "/")
	
	// Write data
	n, err := writer.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}
	
	// Check number of bytes written
	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}
	
	// Check received data
	if string(receivedBody) != string(testData) {
		t.Errorf("Expected server to receive %q, got %q", string(testData), string(receivedBody))
	}
	
	// Check content type
	if receivedContentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", receivedContentType)
	}
	
	// Test with custom headers
	headers := map[string]string{
		"Content-Type": "text/plain",
		"X-Custom":     "test-value",
	}
	
	// Create writer with custom headers
	writerWithHeaders := NewSSEWriterWithHeaders(host, port, "/", headers)
	
	// Write data
	_, err = writerWithHeaders.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write data with custom headers: %v", err)
	}
	
	// Check content type
	if receivedContentType != "text/plain" {
		t.Errorf("Expected Content-Type text/plain, got %s", receivedContentType)
	}
}
