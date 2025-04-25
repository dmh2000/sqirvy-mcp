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

// SSEWriter handles sending Server-Sent Events to a single connected client.
// It implements the io.Writer interface and assumes it's the only SSE writer instance.
type SSEWriter struct {
	logger *utils.Logger
	// Channel to receive the single active connection's writer and flusher
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

func (s *SSEWriter) writerServer() {
	// Start the server in a goroutine
	go func() {
		defer close(s.shutdownComplete)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Printf(utils.LevelError, "SSE server ListenAndServe error: %v", err)
		}
	}()
}

// NewSSEWriter creates and starts an HTTP server listening for a single SSE connection
// on the specified address and path. It returns an SSEWriter instance
// which acts as an io.Writer to send messages to the connected client.
// Assumes this is the only SSE writer needed by the server.
// The server runs in a background goroutine. Call the returned shutdown function
// to gracefully stop the server.
func NewSSEWriter(addr string, path string, logger *utils.Logger) (*SSEWriter, func(context.Context) error, error) {
	if logger == nil {
		// Create a default logger if none provided, though providing one is recommended
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
		sseWriter.writerServer()
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
	s.logger.Printf(utils.LevelInfo, "SSE client connected: %s", r.RemoteAddr)

	w.Header().Set("Content-Type", "text/event-stream")

	// Send the writer and flusher to the SSEWriter instance
	// Use a select with a default to prevent blocking if the channel is full
	// (e.g., if a second client connects before the first Write)
	select {
	case s.connChan <- struct {
		writer  http.ResponseWriter
		flusher http.Flusher
	}{w, w.(http.Flusher)}:
		s.logger.Printf(utils.LevelDebug, "Sent writer/flusher to connChan for %s", r.RemoteAddr)
	default:
		// This case should ideally not happen if only one client connects.
		// If it does, it might indicate a previous client didn't disconnect cleanly
		// or multiple clients are trying to connect simultaneously.
		s.logger.Printf(utils.LevelWarning, "connChan is full or closed. Unexpected new connection from %s ignored.", r.RemoteAddr)
		// We don't send an error back, just ignore the connection attempt,
		// as the writer is already handling (or waiting for) the intended client.
		return
	}

	// Signal that a connection is ready (non-blocking)
	select {
	case <-s.connected: // Already closed, do nothing
	default:
		close(s.connected) // Close the channel to signal connection established
		s.logger.Printf(utils.LevelDebug, "Closed 'connected' channel.")
	}

	for {
		time.Sleep(1 * time.Second)
	}
	// // Keep the connection open until the client disconnects
	// ctx := r.Context()
	// done := false
	// for !done {
	// 	select {
	// 	case <-ctx.Done():

	// 	default:
	// 		time.Sleep(1 * time.Second)
	// 	}
	// }
	// // <-ctx.Done() // Wait for the client to close the connection or the server to shut down

	// s.logger.Printf(utils.LevelInfo, "SSE client disconnected: %s", r.RemoteAddr)

	// // Clear the writer and flusher when the client disconnects
	// s.mu.Lock()
	// // Only clear if this was the active connection
	// if s.writer == w {
	// 	s.writer = nil
	// 	s.flusher = nil
	// 	s.logger.Printf(utils.LevelDebug, "Cleared active writer/flusher for disconnected client %s", r.RemoteAddr)
	// 	// Note: This SSEWriter instance handles only one connection lifecycle.
	// 	// If the client disconnects and reconnects, a new SSEWriter instance
	// 	// would typically be needed, or this one would need more complex logic
	// 	// to reset its state (e.g., reopen `connected` and clear `connChan`).
	// 	// The current design assumes a single, persistent connection for the writer's lifetime.
	// }
	// s.mu.Unlock()
}

// Write sends data formatted as an SSE 'data' event to the single connected client.
// It blocks until the client connects if called beforehand.
// It is safe for concurrent use (relative to the handler goroutine).
func (s *SSEWriter) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	// Defer unlock until the end of the function

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
			s.logger.Printf(utils.LevelError, "Write timed out waiting for SSE client connection")
			// Re-acquire lock before returning
			s.mu.Lock()
			return 0, fmt.Errorf("timeout waiting for SSE client connection")
		}
		// Lock was re-acquired within the select case if connection received.
		// If we timed out, the lock was re-acquired just above.
	}

	// Check again: Did we successfully get a connection?
	if s.writer == nil || s.flusher == nil {
		// This could happen if shutdown occurred while waiting, closing connChan.
		s.logger.Printf(utils.LevelWarning, "Write failed: No active connection after wait") // Use LevelWarning
		// Ensure unlock happens before returning
		defer s.mu.Unlock()
		return 0, io.ErrClosedPipe
	}

	// Unlock is deferred, so we proceed under lock protection
	defer s.mu.Unlock()

	// Format the message according to SSE spec (data: <payload>\n\n)
	// Assume p is a single complete message payload (e.g., a JSON object)
	sseMsg := fmt.Sprintf("data: %s\n\n", string(p))
	s.logger.Printf(utils.LevelDebug, "Writing SSE message: %s", sseMsg)

	_, err = fmt.Fprint(s.writer, sseMsg) // Write under lock
	if err != nil {
		s.logger.Printf(utils.LevelError, "Error writing to SSE stream: %v", err)
		// Consider clearing the writer/flusher here if the write fails
		// s.writer = nil
		// s.flusher = nil
		return 0, err
	}

	// Flush the data to the client (under lock)
	s.flusher.Flush()

	// Return the length of the original payload, not the formatted SSE message
	return len(p), nil
	// mu.Unlock() is called here via defer
}

// WaitForConnection blocks until the single client connects or the context is cancelled.
// Returns an error if the context is cancelled before a connection is established.
func (s *SSEWriter) WaitForConnection(ctx context.Context) error {
	s.logger.Printf(utils.LevelDebug, "Waiting for SSE client connection...")
	select {
	case <-s.connected: // Wait for the connected channel to be closed by the handler
		s.logger.Printf(utils.LevelInfo, "First SSE client connection established.")
		return nil
	case <-ctx.Done():
		s.logger.Printf(utils.LevelWarning, "Context cancelled while waiting for connection: %v", ctx.Err()) // Use LevelWarning
		return ctx.Err()
	}
}
