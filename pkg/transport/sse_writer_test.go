package transport

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	utils "sqirvy-mcp/pkg/utils"
	"strings"
	"sync"
	"testing"
	"time"
)

// Helper to find a free port
func getFreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func TestNewSSEWriter_StartAndShutdown(t *testing.T) {
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	path := "/sse"
	logger := utils.New(io.Discard, "TestSSE: ", 0, utils.LevelDebug) // Use io.Discard or bytes.Buffer for logs

	writer, shutdown, err := NewSSEWriter(addr, path, logger)
	if err != nil {
		t.Fatalf("NewSSEWriter failed: %v", err)
	}
	if writer == nil {
		t.Fatal("NewSSEWriter returned nil writer")
	}
	if shutdown == nil {
		t.Fatal("NewSSEWriter returned nil shutdown function")
	}

	// Give the server a moment to start
	time.Sleep(50 * time.Millisecond)

	// Test shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := shutdown(ctx); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Try connecting after shutdown (should fail)
	_, err = http.Get(fmt.Sprintf("http://%s%s", addr, path))
	if err == nil {
		t.Errorf("Expected connection error after shutdown, but got nil")
	}
}

func TestSSEWriter_ClientConnectAndWrite(t *testing.T) {
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	path := "/sse"
	logger := utils.New(io.Discard, "TestSSE: ", 0, utils.LevelDebug)

	sseWriter, shutdown, err := NewSSEWriter(addr, path, logger)
	if err != nil {
		t.Fatalf("NewSSEWriter failed: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		shutdown(ctx)
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	var receivedData string
	var clientErr error

	// Client goroutine
	go func() {
		defer wg.Done()
		client := &http.Client{Timeout: 5 * time.Second}
		req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s%s", addr, path), nil)
		req.Header.Set("Accept", "text/event-stream")

		resp, err := client.Do(req)
		if err != nil {
			clientErr = fmt.Errorf("client connection failed: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			clientErr = fmt.Errorf("expected status OK, got %s", resp.Status)
			return
		}
		if resp.Header.Get("Content-Type") != "text/event-stream" {
			clientErr = fmt.Errorf("expected content type text/event-stream, got %s", resp.Header.Get("Content-Type"))
			return
		}

		// Read the first event
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data:") {
				receivedData = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				break // Only read the first message for this test
			}
		}
		if err := scanner.Err(); err != nil {
			clientErr = fmt.Errorf("scanner error: %w", err)
		}
	}()

	// Wait for the client to connect using WaitForConnection
	ctxConnect, cancelConnect := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelConnect()
	if err := sseWriter.WaitForConnection(ctxConnect); err != nil {
		t.Fatalf("WaitForConnection failed: %v", err)
	}

	// Send data from the server side
	testMessage := `{"id": 1, "method": "test"}`
	n, err := sseWriter.Write([]byte(testMessage))
	if err != nil {
		t.Fatalf("sseWriter.Write failed: %v", err)
	}
	if n != len(testMessage) {
		t.Errorf("sseWriter.Write returned wrong length: got %d, want %d", n, len(testMessage))
	}

	// Wait for the client goroutine to finish
	wg.Wait()

	if clientErr != nil {
		t.Fatalf("Client encountered error: %v", clientErr)
	}

	if receivedData != testMessage {
		t.Errorf("Client received wrong data: got '%s', want '%s'", receivedData, testMessage)
	}
}

func TestSSEWriter_WriteBeforeConnect(t *testing.T) {
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	path := "/sse"
	logger := utils.New(io.Discard, "TestSSE: ", 0, utils.LevelDebug)

	sseWriter, shutdown, err := NewSSEWriter(addr, path, logger)
	if err != nil {
		t.Fatalf("NewSSEWriter failed: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		shutdown(ctx)
	}()

	var wg sync.WaitGroup
	wg.Add(2) // One for client, one for writer

	var receivedData string
	var clientErr error
	var writeErr error
	var writeN int

	testMessage := `{"value": "data sent before connect"}`

	// Writer goroutine - tries to write immediately
	go func() {
		defer wg.Done()
		// This write should block until the client connects
		writeN, writeErr = sseWriter.Write([]byte(testMessage))
	}()

	// Give the writer goroutine a chance to block
	time.Sleep(50 * time.Millisecond)

	// Client goroutine - connects after a delay
	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond) // Connect slightly later

		client := &http.Client{Timeout: 3 * time.Second}
		req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s%s", addr, path), nil)
		req.Header.Set("Accept", "text/event-stream")

		resp, err := client.Do(req)
		if err != nil {
			clientErr = fmt.Errorf("client connection failed: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			clientErr = fmt.Errorf("expected status OK, got %s", resp.Status)
			return
		}

		// Read the first event
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data:") {
				receivedData = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				break
			}
		}
		if err := scanner.Err(); err != nil {
			clientErr = fmt.Errorf("scanner error: %w", err)
		}
	}()

	// Wait for both goroutines
	wg.Wait()

	// Check results
	if writeErr != nil {
		t.Errorf("sseWriter.Write failed unexpectedly: %v", writeErr)
	}
	if writeN != len(testMessage) {
		t.Errorf("sseWriter.Write returned wrong length: got %d, want %d", writeN, len(testMessage))
	}
	if clientErr != nil {
		t.Fatalf("Client encountered error: %v", clientErr)
	}
	if receivedData != testMessage {
		t.Errorf("Client received wrong data: got '%s', want '%s'", receivedData, testMessage)
	}
}

func TestSSEWriter_ClientDisconnect(t *testing.T) {
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	path := "/sse"
	logger := utils.New(io.Discard, "TestSSE: ", 0, utils.LevelDebug)

	sseWriter, shutdown, err := NewSSEWriter(addr, path, logger)
	if err != nil {
		t.Fatalf("NewSSEWriter failed: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		shutdown(ctx)
	}()

	// Client connects and disconnects immediately
	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://%s%s", addr, path), nil)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Client connection failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK, got %s", resp.Status)
	}

	// Wait for the connection to be fully established on the server side
	// before closing it, ensuring the handler is ready.
	ctxConnect, cancelConnect := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelConnect()
	if err := sseWriter.WaitForConnection(ctxConnect); err != nil {
		t.Fatalf("WaitForConnection failed before disconnect: %v", err)
	}

	// Close the response body to simulate disconnect
	resp.Body.Close()

	// Give server time to process disconnect
	time.Sleep(100 * time.Millisecond)

	// Try writing after disconnect, should fail
	_, err = sseWriter.Write([]byte(`{"data": "after disconnect"}`))
	if err == nil {
		t.Errorf("Expected error when writing after client disconnect, but got nil")
	} else {
		// Check if the error indicates a closed pipe or similar network issue
		t.Logf("Got expected error after disconnect: %v", err)
		// Note: The exact error might vary (e.g., io.ErrClosedPipe, syscall.EPIPE, net.ErrClosed)
		// A more robust check might look for specific error types or substrings.
		// For now, just checking for non-nil error is sufficient.
	}

	// Internal state check (optional, requires exposing state or using logs)
	sseWriter.mu.Lock()
	if sseWriter.writer != nil || sseWriter.flusher != nil {
		t.Error("Internal writer/flusher state not cleared after client disconnect")
	}
	sseWriter.mu.Unlock()
}

func TestSSEWriter_WaitForConnection_Timeout(t *testing.T) {
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	path := "/test-wait-timeout"
	logger := utils.New(io.Discard, "TestSSE: ", 0, utils.LevelDebug)

	sseWriter, shutdown, err := NewSSEWriter(addr, path, logger)
	if err != nil {
		t.Fatalf("NewSSEWriter failed: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		shutdown(ctx)
	}()

	// Wait for connection with a short timeout, expecting it to fail
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = sseWriter.WaitForConnection(ctx)
	if err == nil {
		t.Errorf("Expected WaitForConnection to time out, but it succeeded")
	} else if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}
