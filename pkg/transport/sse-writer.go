package transport

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	utils "sqirvy-mcp/pkg/utils"
)

// SSEWriter handles sending Server-Sent Events to a connected client.
// It implements the io.Writer interface.
type SSEWriter struct {
	logger *utils.Logger
	// Channel to receive the active connection's writer and flusher
	connChan chan struct {
		writer  http.ResponseWriter
		flusher http.Flusher
	}
	// The active connection's writer and flusher
	writer  http.ResponseWriter
	flusher http.Flusher
	// Mutex to protect access to writer and flusher
	mu sync.Mutex
	// Channel to signal when a connection is established
	connected chan struct{}
	// Channel to signal server shutdown is complete
	shutdownComplete chan struct{}
	// Store the server instance for shutdown
	server *http.Server
}

// NewSSEWriter creates and starts an HTTP server listening for SSE connections
// on the specified address and path. It returns an SSEWriter instance
// which acts as an io.Writer to send messages to the *first* connected client.
// The server runs in a background goroutine. Call the returned shutdown function
// to gracefully stop the server.
func NewSSEWriter(addr string, path string, logger *utils.Logger) (*SSEWriter, func(context.Context) error, error) {
	if logger == nil {
		// Create a default logger if none provided
		logger = utils.New(io.Discard, "SSEWriter: ", 0, utils.LevelInfo)
	}

	sseWriter := &SSEWriter{
		logger: logger,
		connChan: make(chan struct { // Buffered channel to avoid blocking handler if Write isn't called yet
			writer  http.ResponseWriter
			flusher http.Flusher
		}, 1),
		connected:        make(chan struct{}),
		shutdownComplete: make(chan struct{}),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(path, sseWriter.handleSSEConnection)

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	sseWriter.server = srv // Store server for shutdown

	// Start the server in a goroutine
	go func() {
		defer close(sseWriter.shutdownComplete)
		logger.Printf(utils.LevelInfo, "Starting SSE server on %s%s", addr, path)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Printf(utils.LevelError, "SSE server ListenAndServe error: %v", err)
		}
		logger.Printf(utils.LevelInfo, "SSE server on %s stopped", addr)
	}()

	// Return the writer and a shutdown function
	shutdownFunc := func(ctx context.Context) error {
		logger.Printf(utils.LevelInfo, "Shutting down SSE server on %s", addr)
		err := srv.Shutdown(ctx)
		if err == nil {
			// Wait for the server goroutine to finish
			<-sseWriter.shutdownComplete
			logger.Printf(utils.LevelInfo, "SSE server on %s shutdown complete", addr)
		} else {
			logger.Printf(utils.LevelError, "SSE server shutdown error: %v", err)
		}
		return err
	}

	return sseWriter, shutdownFunc, nil
}

// handleSSEConnection is the HTTP handler for incoming SSE connections.
func (s *SSEWriter) handleSSEConnection(w http.ResponseWriter, r *http.Request) {
	// Ensure the client accepts SSE
	if r.Header.Get("Accept") != "text/event-stream" {
		http.Error(w, "Unsupported media type. Expected 'text/event-stream'", http.StatusUnsupportedMediaType)
		s.logger.Printf(utils.LevelInfo, "Rejected connection from %s: incorrect Accept header '%s'", r.RemoteAddr, r.Header.Get("Accept"))
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Allow CORS for testing/flexibility

	// Get the Flusher interface
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		s.logger.Printf(utils.LevelError, "Could not get http.Flusher for connection from %s", r.RemoteAddr)
		return
	}

	s.logger.Printf(utils.LevelInfo, "SSE client connected: %s", r.RemoteAddr)

	// Send the writer and flusher to the SSEWriter instance
	// Use a select with a default to prevent blocking if the channel is full
	// (e.g., if a second client connects before the first Write)
	select {
	case s.connChan <- struct {
		writer  http.ResponseWriter
		flusher http.Flusher
	}{w, flusher}:
		s.logger.Printf(utils.LevelDebug, "Sent writer/flusher to connChan for %s", r.RemoteAddr)
	default:
		s.logger.Printf(utils.LevelWarning, "connChan is full or closed. Ignoring new connection from %s", r.RemoteAddr) // Use LevelWarning
		// Optionally close the connection immediately or send an error
		http.Error(w, "Server busy, try again later", http.StatusServiceUnavailable)
		return
	}

	// Signal that a connection is ready (non-blocking)
	select {
	case <-s.connected: // Already closed, do nothing
	default:
		close(s.connected) // Close the channel to signal connection established
		s.logger.Printf(utils.LevelDebug, "Closed 'connected' channel.")
	}

	// Keep the connection open until the client disconnects
	time.Sleep(30 * time.Second)
	ctx := r.Context()
	<-ctx.Done()

	s.logger.Printf(utils.LevelInfo, "SSE client disconnected: %s", r.RemoteAddr)

	// Clear the writer and flusher when the client disconnects
	s.mu.Lock()
	// Only clear if this was the active connection
	if s.writer == w {
		s.writer = nil
		s.flusher = nil
		s.logger.Printf(utils.LevelDebug, "Cleared active writer/flusher for disconnected client %s", r.RemoteAddr)
		// Reset connected channel for potential new connections?
		// For now, this writer supports only the *first* connection lifecycle.
		// To support reconnects, we'd need to reopen s.connected here.
	}
	s.mu.Unlock()
}

// Write sends data formatted as an SSE 'data' event to the connected client.
// It blocks until the first client connects.
// It is safe for concurrent use.
func (s *SSEWriter) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if we have an active connection, wait if not
	if s.writer == nil || s.flusher == nil {
		s.mu.Unlock() // Unlock while waiting
		s.logger.Printf(utils.LevelDebug, "Write waiting for connection...")

		// Wait for either a connection or the channel to close (shutdown)
		select {
		case connInfo, ok := <-s.connChan:
			if !ok {
				s.logger.Printf(utils.LevelWarning, "Write failed: connChan closed during wait") // Use LevelWarning
				return 0, io.ErrClosedPipe                                                       // Or a more specific error
			}
			s.logger.Printf(utils.LevelDebug, "Received connection from connChan.")
			s.mu.Lock() // Re-lock before modifying shared state
			s.writer = connInfo.writer
			s.flusher = connInfo.flusher
		case <-time.After(30 * time.Second): // Add a timeout for waiting?
			s.logger.Printf(utils.LevelError, "Write timed out waiting for connection")
			return 0, fmt.Errorf("timeout waiting for SSE client connection")
		}
		// Lock was re-acquired within the select block if connection received
	}

	// Check again after potentially waiting
	if s.writer == nil || s.flusher == nil {
		// This might happen if shutdown occurred while waiting
		s.logger.Printf(utils.LevelWarning, "Write failed: No active connection after wait") // Use LevelWarning
		return 0, io.ErrClosedPipe
	}

	// Format the message according to SSE spec (data: <payload>\n\n)
	// We assume p is a single complete message payload (e.g., a JSON object)
	sseMsg := fmt.Sprintf("data: %s\n\n", string(p))
	s.logger.Printf(utils.LevelDebug, "Writing SSE message: %s", sseMsg)

	_, err = fmt.Fprint(s.writer, sseMsg)
	if err != nil {
		s.logger.Printf(utils.LevelError, "Error writing to SSE stream: %v", err)
		// Consider clearing the writer/flusher here if the write fails
		// s.writer = nil
		// s.flusher = nil
		return 0, err
	}

	// Flush the data to the client
	s.flusher.Flush()

	// Return the length of the original payload, not the formatted SSE message
	return len(p), nil
}

// WaitForConnection blocks until the first client connects or the context is cancelled.
// Returns an error if the context is cancelled before a connection is established.
func (s *SSEWriter) WaitForConnection(ctx context.Context) error {
	s.logger.Printf(utils.LevelDebug, "Waiting for first SSE client connection...")
	select {
	case <-s.connected:
		s.logger.Printf(utils.LevelInfo, "First SSE client connection established.")
		return nil
	case <-ctx.Done():
		s.logger.Printf(utils.LevelWarning, "Context cancelled while waiting for connection: %v", ctx.Err()) // Use LevelWarning
		return ctx.Err()
	}
}
