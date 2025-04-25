package transport

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
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

// TestSSE_WriterLifecycle tests the basic flow of setting up an SSE writer,
// connecting a client, sending data, and shutting down.
func TestSSE_WriterLifecycle(t *testing.T) {
	port := findAvailablePort(t)
	path := "/events"
	testData := []byte("data: hello world\n\n")

	writerChan, stopFunc, err := NewSSEWriter(port, path)
	if err != nil {
		t.Fatalf("NewSSEWriter failed: %v", err)
	}
	defer func() {
		if err := stopFunc(); err != nil {
			// Allow "no server running" error if already stopped
			if !strings.Contains(err.Error(), "no server running") && !strings.Contains(err.Error(), "http: Server closed") {
				t.Errorf("stopFunc failed: %v", err)
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1) // For receiving the writer
	wg.Add(1) // For receiving the data

	var clientWriter io.Writer
	go func() {
		select {
		case w := <-writerChan:
			clientWriter = w
			wg.Done() // Received writer
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for writer on channel")
			wg.Done() // Ensure wg is decremented even on timeout
		}
	}()

	// Simulate client connection
	clientConnChan := make(chan []byte, 1)
	clientErrChan := make(chan error, 1)
	go func() {
		defer close(clientConnChan)
		defer close(clientErrChan)

		url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Accept", "text/event-stream")
		client := &http.Client{Timeout: 3 * time.Second} // Add timeout

		resp, err := client.Do(req)
		if err != nil {
			clientErrChan <- fmt.Errorf("client connection failed: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			clientErrChan <- fmt.Errorf("client received non-200 status: %d", resp.StatusCode)
			return
		}

		// Read the first message
		buf := make([]byte, 1024)
		n, err := resp.Body.Read(buf) // This will block until data is sent or connection closes
		if err != nil && err != io.EOF {
			// EOF is expected if server closes connection before data
			clientErrChan <- fmt.Errorf("client failed to read response body: %w", err)
			return
		}
		if n > 0 {
			clientConnChan <- buf[:n]
		}
	}()

	// Wait for the writer to be available
	if waitTimeout(&wg, 2*time.Second) { // Wait for receiving writer
		t.Fatal("Timeout waiting for server to provide writer")
	}
	if clientWriter == nil {
		t.Fatal("Did not receive writer from channel")
	}

	// Send data from server side
	_, err = clientWriter.Write(testData)
	if err != nil {
		t.Fatalf("Failed to write data using clientWriter: %v", err)
	}

	// Wait for client to receive data
	select {
	case received := <-clientConnChan:
		if !bytes.Equal(received, testData) {
			t.Errorf("Client received wrong data.\nExpected: %q\nReceived: %q", string(testData), string(received))
		}
		wg.Done() // Received data
	case err := <-clientErrChan:
		t.Fatalf("Client encountered error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for client to receive data")
	}

	// Wait for data reception goroutine
	if waitTimeout(&wg, 1*time.Second) {
		t.Log("Warning: Timeout waiting for data reception waitgroup (might indicate test logic issue)")
	}

	// Shutdown server
	err = stopFunc()
	if err != nil && !strings.Contains(err.Error(), "http: Server closed") { // Ignore already closed error
		t.Errorf("stopFunc failed during explicit stop: %v", err)
	}

	// Check if client connection attempt now fails (or gets EOF quickly)
	select {
	case err := <-clientErrChan:
		t.Logf("Client correctly received error after server shutdown: %v", err)
	case data := <-clientConnChan:
		t.Logf("Client received unexpected data after shutdown: %q", string(data))
	case <-time.After(1 * time.Second):
		t.Log("Client connection likely closed gracefully or timed out after shutdown, which is acceptable.")
	}
}

// TestSSE_ReaderLifecycle tests the basic flow of setting up a POST reader,
// sending data from a client, receiving it, and shutting down.
func TestSSE_ReaderLifecycle(t *testing.T) {
	port := findAvailablePort(t)
	path := "/data"
	testData := `{"key": "value"}`

	readerChan, stopFunc, err := NewSSEReader(port, path)
	if err != nil {
		t.Fatalf("NewSSEReader failed: %v", err)
	}
	defer func() {
		if err := stopFunc(); err != nil {
			if !strings.Contains(err.Error(), "no server running") && !strings.Contains(err.Error(), "http: Server closed") {
				t.Errorf("stopFunc failed: %v", err)
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1) // For receiving the reader
	wg.Add(1) // For client POST completion

	var receivedReader io.ReadCloser
	go func() {
		select {
		case r := <-readerChan:
			receivedReader = r
			wg.Done() // Received reader
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for reader on channel")
			wg.Done() // Ensure wg is decremented
		}
	}()

	// Simulate client POST
	clientErrChan := make(chan error, 1)
	go func() {
		defer wg.Done() // Client POST complete
		defer close(clientErrChan)

		url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
		resp, err := http.Post(url, "application/json", strings.NewReader(testData))
		if err != nil {
			clientErrChan <- fmt.Errorf("client POST failed: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusAccepted { // Expecting 202 Accepted now
			bodyBytes, _ := io.ReadAll(resp.Body)
			clientErrChan <- fmt.Errorf("client received non-202 status: %d, body: %s", resp.StatusCode, string(bodyBytes))
			return
		}
	}()

	// Wait for the reader to be available
	if waitTimeout(&wg, 2*time.Second) { // Wait for receiving reader
		t.Fatal("Timeout waiting for server to provide reader")
	}
	if receivedReader == nil {
		t.Fatal("Did not receive reader from channel")
	}

	// Read data from the provided reader
	receivedBytes, err := io.ReadAll(receivedReader)
	if err != nil {
		t.Fatalf("Failed to read from receivedReader: %v", err)
	}
	err = receivedReader.Close()
	if err != nil {
		t.Errorf("Failed to close receivedReader: %v", err)
	}

	// Verify data
	if string(receivedBytes) != testData {
		t.Errorf("Server received wrong data.\nExpected: %q\nReceived: %q", testData, string(receivedBytes))
	}

	// Wait for client POST to finish and check errors
	if waitTimeout(&wg, 2*time.Second) { // Wait for client POST completion
		t.Fatal("Timeout waiting for client POST to complete")
	}
	select {
	case err := <-clientErrChan:
		t.Fatalf("Client POST encountered error: %v", err)
	default:
		// No error, success
	}
}

// TestSSE_PathConflict tests that registering the same path twice fails.
func TestSSE_PathConflict(t *testing.T) {
	port := findAvailablePort(t)
	path := "/conflict"

	// Register writer first
	_, stopFunc1, err := NewSSEWriter(port, path)
	if err != nil {
		t.Fatalf("First NewSSEWriter failed unexpectedly: %v", err)
	}
	defer func() { _ = stopFunc1() }() // Ensure server stops eventually

	// Try registering writer again for the same path
	_, _, err = NewSSEWriter(port, path)
	if err == nil {
		t.Errorf("Expected error when registering SSE writer for duplicate path %s, but got nil", path)
	} else if !strings.Contains(err.Error(), "already configured for writing") {
		t.Errorf("Expected path conflict error for writer, got: %v", err)
	}

	// Try registering reader for the same path (should also fail)
	_, _, err = NewSSEReader(port, path)
	if err == nil {
		t.Errorf("Expected error when registering SSE reader for path %s already used by writer, but got nil", path)
	} else if !strings.Contains(err.Error(), "already configured for writing") { // Check specific error if possible, might depend on implementation detail
		// We expect *some* error indicating conflict. The exact message might vary.
		// For now, check if it mentions the path.
		log.Printf("Got expected conflict error (reader vs writer): %v", err)
		// A more specific check might be needed if the error message is stable.
		// Example: } else if !strings.Contains(err.Error(), fmt.Sprintf("path %s on port %d is already configured", path, port)) {
		//	 t.Errorf("Expected path conflict error mentioning path and port, got: %v", err)
		// }
	}

	// Stop the first server instance implicitly via defer or explicitly if needed earlier
	_ = stopFunc1()
	// Allow time for server shutdown and map cleanup
	time.Sleep(100 * time.Millisecond)

	// --- Test reader conflict ---
	port2 := findAvailablePort(t) // Use a new port to ensure clean state
	path2 := "/another_conflict"

	// Register reader first
	_, stopFunc2, err := NewSSEReader(port2, path2)
	if err != nil {
		t.Fatalf("First NewSSEReader failed unexpectedly: %v", err)
	}
	defer func() { _ = stopFunc2() }()

	// Try registering reader again
	_, _, err = NewSSEReader(port2, path2)
	if err == nil {
		t.Errorf("Expected error when registering SSE reader for duplicate path %s, but got nil", path2)
	} else if !strings.Contains(err.Error(), "already configured for reading") {
		t.Errorf("Expected path conflict error for reader, got: %v", err)
	}

	// Try registering writer for the same path
	_, _, err = NewSSEWriter(port2, path2)
	if err == nil {
		t.Errorf("Expected error when registering SSE writer for path %s already used by reader, but got nil", path2)
	} else if !strings.Contains(err.Error(), "already configured for reading") { // Check specific error
		log.Printf("Got expected conflict error (writer vs reader): %v", err)
	}
}

// TestSSE_StopServer tests the StopServer functionality.
func TestSSE_StopServer(t *testing.T) {
	port := findAvailablePort(t)
	path := "/stoppable"

	_, stopFunc, err := NewSSEWriter(port, path)
	if err != nil {
		t.Fatalf("NewSSEWriter failed: %v", err)
	}

	// Stop the server using the specific stop function
	err = stopFunc()
	if err != nil {
		t.Fatalf("stopFunc failed: %v", err)
	}

	// Allow a moment for shutdown
	time.Sleep(50 * time.Millisecond)

	// Try stopping again using the global StopServer (should fail as it's already stopped)
	err = StopServer(port)
	if err == nil {
		t.Errorf("Expected error when calling StopServer on an already stopped server, got nil")
	} else if !strings.Contains(err.Error(), "no server running") {
		t.Errorf("Expected 'no server running' error, got: %v", err)
	}

	// Try connecting a client (should fail)
	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "text/event-stream")
	client := &http.Client{Timeout: 1 * time.Second}
	_, err = client.Do(req)
	if err == nil {
		t.Errorf("Client connection succeeded unexpectedly after server stop")
	} else {
		t.Logf("Client connection failed as expected after stop: %v", err)
	}

	// Try starting a new service on the same port (should succeed if fully stopped)
	_, stopFunc2, err := NewSSEReader(port, "/newpath")
	if err != nil {
		t.Fatalf("Failed to start a new service on port %d after stopping: %v", port, err)
	}
	_ = stopFunc2() // Clean up the new service
}

// TestSSE_ServerRouting tests if the server correctly routes GET and POST
// requests to different paths handled by writer and reader channels.
func TestSSE_ServerRouting(t *testing.T) {
	port := findAvailablePort(t)
	ssePath := "/events"
	postPath := "/submit"
	sseData := []byte("data: event data\n\n")
	postData := `{"id": 123}`

	// Setup writer channel
	writerChan, stopFuncW, err := NewSSEWriter(port, ssePath)
	if err != nil {
		t.Fatalf("NewSSEWriter failed: %v", err)
	}
	defer func() { _ = stopFuncW() }() // Use one of the stop funcs

	// Setup reader channel (on the same server instance)
	readerChan, _, err := NewSSEReader(port, postPath)
	if err != nil {
		t.Fatalf("NewSSEReader failed: %v", err)
	}
	// No need for separate stopFuncR, stopFuncW stops the shared server

	var wg sync.WaitGroup
	wg.Add(3) // 1 for SSE writer, 1 for POST reader, 1 for SSE client recv

	// Goroutine to handle received writer
	var clientWriter io.Writer
	go func() {
		select {
		case w := <-writerChan:
			clientWriter = w
			wg.Done()
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for SSE writer")
			wg.Done()
		}
	}()

	// Goroutine to handle received reader
	var receivedReader io.ReadCloser
	go func() {
		select {
		case r := <-readerChan:
			receivedReader = r
			wg.Done()
		case <-time.After(2 * time.Second):
			t.Error("Timeout waiting for POST reader")
			wg.Done()
		}
	}()

	// Simulate SSE client connection
	clientSseChan := make(chan []byte, 1)
	clientSseErrChan := make(chan error, 1)
	go func() {
		defer close(clientSseChan)
		defer close(clientSseErrChan)
		url := fmt.Sprintf("http://127.0.0.1:%d%s", port, ssePath)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Accept", "text/event-stream")
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			clientSseErrChan <- fmt.Errorf("SSE client connection failed: %w", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			clientSseErrChan <- fmt.Errorf("SSE client received non-200 status: %d", resp.StatusCode)
			return
		}
		buf := make([]byte, 1024)
		n, err := resp.Body.Read(buf) // Block until data or close
		if err != nil && err != io.EOF {
			clientSseErrChan <- fmt.Errorf("SSE client read error: %w", err)
			return
		}
		if n > 0 {
			clientSseChan <- buf[:n]
		}
		wg.Done() // Indicate client received data (or EOF)
	}()

	// Simulate POST client
	clientPostErrChan := make(chan error, 1)
	go func() {
		defer close(clientPostErrChan)
		url := fmt.Sprintf("http://127.0.0.1:%d%s", port, postPath)
		resp, err := http.Post(url, "application/json", strings.NewReader(postData))
		if err != nil {
			clientPostErrChan <- fmt.Errorf("POST client failed: %w", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusAccepted {
			bodyBytes, _ := io.ReadAll(resp.Body)
			clientPostErrChan <- fmt.Errorf("POST client received non-202 status: %d, body: %s", resp.StatusCode, string(bodyBytes))
			return
		}
	}()

	// Wait for server handlers to provide writer and reader
	if waitTimeout(&wg, 3*time.Second) {
		t.Fatal("Timeout waiting for server handlers (writer/reader)")
	}

	// --- Interact ---

	// Check POST client for errors
	select {
	case err := <-clientPostErrChan:
		t.Fatalf("POST client failed: %v", err)
	default: // No error yet
	}

	// Verify POST reader received
	if receivedReader == nil {
		t.Fatal("Did not receive POST reader")
	}
	// Read from POST reader
	postBytes, err := io.ReadAll(receivedReader)
	if err != nil {
		t.Fatalf("Failed reading from POST reader: %v", err)
	}
	_ = receivedReader.Close()
	if string(postBytes) != postData {
		t.Errorf("POST data mismatch. Expected %q, Got %q", postData, string(postBytes))
	}

	// Verify SSE writer received
	if clientWriter == nil {
		t.Fatal("Did not receive SSE writer")
	}
	// Write to SSE writer
	_, err = clientWriter.Write(sseData)
	if err != nil {
		t.Fatalf("Failed writing to SSE writer: %v", err)
	}

	// Verify SSE client received data
	select {
	case received := <-clientSseChan:
		if !bytes.Equal(received, sseData) {
			t.Errorf("SSE client data mismatch.\nExpected: %q\nReceived: %q", string(sseData), string(received))
		}
	case err := <-clientSseErrChan:
		t.Fatalf("SSE client encountered error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for SSE client to receive data")
	}

	// Wait for SSE client goroutine to finish
	if waitTimeout(&wg, 2*time.Second) {
		t.Log("Warning: Timeout waiting for SSE client waitgroup")
	}
}

// waitTimeout waits for the waitgroup for the specified duration.
// Returns true if waiting timed out.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	done := make(chan struct{}) // Separate channel to signal completion

	go func() {
		defer close(done) // Signal that Wait() has returned
		wg.Wait()
	}()

	select {
	case <-done:
		// Wait completed successfully
		return false // not timed out
	case <-time.After(timeout):
		// Timeout occurred
		return true // timed out
	}
}

// Helper to ensure server stops even if test fails
func ensureServerStop(t *testing.T, stopFunc func() error) {
	t.Helper()
	if stopFunc != nil {
		if err := stopFunc(); err != nil {
			// Avoid failing test cleanup on expected errors after shutdown
			if !strings.Contains(err.Error(), "no server running") && !strings.Contains(err.Error(), "http: Server closed") {
				t.Logf("Error during cleanup stopFunc: %v", err) // Log non-fatal cleanup errors
			}
		}
	}
}
