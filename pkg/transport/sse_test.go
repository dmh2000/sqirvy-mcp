package transport

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Helper to find an available port
func findAvailablePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", ":0") // :0 means assign a random available port
	if err != nil {
		t.Fatalf("Failed to find an available port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

// waitForServerReady tries to connect to the server until it succeeds or times out
func waitForServerReady(t *testing.T, port int, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// TestNewSSEServer tests the basic functionality of the SSE server
func TestNewSSEServer(t *testing.T) {
	port := findAvailablePort(t)
	path := "/events"
	
	// Create a new SSE server
	reader, stopFunc, err := NewSSEServer(port, path)
	if err != nil {
		t.Fatalf("Failed to create SSE server: %v", err)
	}
	defer func() {
		if err := stopFunc(); err != nil {
			t.Logf("Error stopping server: %v", err)
		}
	}()
	
	// Wait for the server to start
	if !waitForServerReady(t, port, 2*time.Second) {
		t.Fatalf("Server did not start within timeout period")
	}
	
	// Verify the reader is not nil
	if reader == nil {
		t.Fatal("Reader should not be nil")
	}
	
	// Test that the server responds to requests
	client := &http.Client{Timeout: 1 * time.Second}
	
	// Test with correct Accept header
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d%s", port, path), nil)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to connect to SSE server: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Errorf("Expected Content-Type text/event-stream, got %s", resp.Header.Get("Content-Type"))
	}
	
	// Test with incorrect Accept header
	req, _ = http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d%s", port, path), nil)
	req.Header.Set("Accept", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to connect to SSE server: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusNotAcceptable {
		t.Errorf("Expected status 406, got %d", resp.StatusCode)
	}
	
	// Test with incorrect method
	resp, err = client.Post(fmt.Sprintf("http://127.0.0.1:%d%s", port, path), "text/plain", strings.NewReader("test"))
	if err != nil {
		t.Fatalf("Failed to connect to SSE server: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", resp.StatusCode)
	}
}

// TestSSEServerStop tests that the server can be stopped properly
func TestSSEServerStop(t *testing.T) {
	port := findAvailablePort(t)
	path := "/events"
	
	// Create a new SSE server
	_, stopFunc, err := NewSSEServer(port, path)
	if err != nil {
		t.Fatalf("Failed to create SSE server: %v", err)
	}
	
	// Wait for the server to start
	if !waitForServerReady(t, port, 2*time.Second) {
		t.Fatalf("Server did not start within timeout period")
	}
	
	// Stop the server
	if err := stopFunc(); err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}
	
	// Allow time for the server to stop
	time.Sleep(100 * time.Millisecond)
	
	// Verify the server is no longer accepting connections
	client := &http.Client{Timeout: 500 * time.Millisecond}
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d%s", port, path), nil)
	req.Header.Set("Accept", "text/event-stream")
	_, err = client.Do(req)
	if err == nil {
		t.Fatal("Expected connection to fail after server stop, but it succeeded")
	}
}

// TestSSEServerMultipleClients tests that the server can handle multiple clients
func TestSSEServerMultipleClients(t *testing.T) {
	port := findAvailablePort(t)
	path := "/events"
	
	// Create a new SSE server
	_, stopFunc, err := NewSSEServer(port, path)
	if err != nil {
		t.Fatalf("Failed to create SSE server: %v", err)
	}
	defer stopFunc()
	
	// Wait for the server to start
	if !waitForServerReady(t, port, 2*time.Second) {
		t.Fatalf("Server did not start within timeout period")
	}
	
	// Connect multiple clients
	client := &http.Client{Timeout: 1 * time.Second}
	
	// Connect first client
	req1, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d%s", port, path), nil)
	req1.Header.Set("Accept", "text/event-stream")
	resp1, err := client.Do(req1)
	if err != nil {
		t.Fatalf("Failed to connect first client: %v", err)
	}
	defer resp1.Body.Close()
	
	if resp1.StatusCode != http.StatusOK {
		t.Errorf("First client: Expected status 200, got %d", resp1.StatusCode)
	}
	
	// Connect second client
	req2, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d%s", port, path), nil)
	req2.Header.Set("Accept", "text/event-stream")
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("Failed to connect second client: %v", err)
	}
	defer resp2.Body.Close()
	
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Second client: Expected status 200, got %d", resp2.StatusCode)
	}
}
