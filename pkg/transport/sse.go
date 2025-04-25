// Package transport provides functionality for transporting messages between components.
package transport

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

// sseServerInstance manages a single HTTP server listening on a specific port.
// It handles routing requests based on path and method (GET for SSE writes, POST for reads).
type sseServerInstance struct {
	server         *http.Server
	mux            *http.ServeMux
	writerChannels map[string]chan io.Writer    // path -> chan for SSE writers (GET)
	readerChannels map[string]chan io.ReadCloser // path -> chan for POST readers
	mu             sync.RWMutex                 // Protects the channel maps
	stopOnce       sync.Once
	stopped        chan struct{} // Closed when server is stopped
}

// sseResponseWriter wraps http.ResponseWriter and http.Flusher to implement io.Writer
// for sending SSE messages.
type sseResponseWriter struct {
	w http.ResponseWriter
	f http.Flusher
}

// Write writes data to the underlying http.ResponseWriter and flushes it.
func (srw *sseResponseWriter) Write(p []byte) (int, error) {
	n, err := srw.w.Write(p)
	if err == nil {
		// Flush the data to the client immediately
		srw.f.Flush()
	}
	return n, err
}

var (
	servers     = make(map[int]*sseServerInstance)
	serversLock sync.Mutex
)

// getServerInstance retrieves or creates a shared sseServerInstance for a given port.
// It starts the HTTP server in a goroutine if it's newly created.
func getServerInstance(port int) (*sseServerInstance, error) {
	serversLock.Lock()
	defer serversLock.Unlock()

	instance, exists := servers[port]
	if exists {
		// Check if the server is still running (or was stopped)
		select {
		case <-instance.stopped:
			// Server was stopped, need to create a new one
			exists = false
		default:
			// Server is running or starting
		}
	}

	if !exists {
		mux := http.NewServeMux()
		addr := fmt.Sprintf(":%d", port)
		instance = &sseServerInstance{
			mux:            mux,
			writerChannels: make(map[string]chan io.Writer),
			readerChannels: make(map[string]chan io.ReadCloser),
			stopped:        make(chan struct{}),
		}
		instance.server = &http.Server{
			Addr:    addr,
			Handler: instance, // The instance itself will handle routing
		}

		// Start the server in a goroutine
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("failed to listen on port %d: %w", port, err)
		}

		go func() {
			log.Printf("Starting SSE/POST server on %s", addr)
			err := instance.server.Serve(listener)
			if err != nil && err != http.ErrServerClosed {
				log.Printf("HTTP server error on port %d: %v", port, err)
			}
			log.Printf("Stopped SSE/POST server on %s", addr)
			close(instance.stopped) // Signal that the server has stopped

			// Clean up the entry in the global map
			serversLock.Lock()
			delete(servers, port)
			serversLock.Unlock()
		}()
		servers[port] = instance
	}

	return instance, nil
}

// StopServer gracefully shuts down the server listening on the specified port.
func StopServer(port int) error {
	serversLock.Lock()
	instance, exists := servers[port]
	// Don't remove from map here, let the server goroutine do it on exit
	serversLock.Unlock()

	if !exists {
		return fmt.Errorf("no server running on port %d", port)
	}

	var err error
	instance.stopOnce.Do(func() {
		log.Printf("Attempting graceful shutdown of server on port %d...", port)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = instance.server.Shutdown(ctx)
		// The server goroutine will close instance.stopped and remove from map
	})

	if err != nil {
		return fmt.Errorf("error shutting down server on port %d: %w", port, err)
	}
	log.Printf("Server on port %d shutdown initiated.", port)
	return nil
}

// ServeHTTP implements http.Handler for sseServerInstance.
// It routes requests to the appropriate handler based on path and method.
func (s *sseServerInstance) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock() // Lock for reading channel maps

	path := r.URL.Path
	method := r.Method

	// Handle GET requests (SSE connection initiation)
	if method == http.MethodGet {
		writerChan, chanExists := s.writerChannels[path]
		s.mu.RUnlock() // Unlock before potentially blocking operations

		if !chanExists {
			http.NotFound(w, r)
			log.Printf("No SSE writer channel found for GET %s", path)
			return
		}

		// Check Accept header
		if accept := r.Header.Get("Accept"); accept != "text/event-stream" {
			http.Error(w, "Accept header must be text/event-stream", http.StatusNotAcceptable)
			return
		}

		// Check for http.Flusher support
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		// Set headers for SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		flusher.Flush() // Ensure headers are sent

		// Create the writer for this specific client connection
		clientWriter := &sseResponseWriter{w: w, f: flusher}

		// Send the writer to the application, handle potential closed channel
		select {
		case writerChan <- clientWriter:
			log.Printf("SSE client connected on GET %s, writer provided.", path)
			// Keep the connection open until the client disconnects
			<-r.Context().Done()
			log.Printf("SSE client disconnected on GET %s.", path)
		case <-s.stopped:
			log.Printf("Server stopping, cannot provide writer for GET %s.", path)
			// Server is stopping, don't block, just return
		default:
			// If the channel buffer is full (if any), or receiver is slow.
			// This indicates an issue in the consuming application.
			// We might log this, but the connection is established.
			// For simplicity, we assume the channel is ready.
			// A more robust implementation might use a timeout or non-blocking send.
			log.Printf("Warning: SSE writer channel for GET %s might be blocked or full.", path)
			// Still wait for disconnect
			<-r.Context().Done()
			log.Printf("SSE client disconnected on GET %s (channel potentially blocked).", path)

		}
		return
	}

	// Handle POST requests (Client sending data to server)
	if method == http.MethodPost {
		readerChan, chanExists := s.readerChannels[path]
		s.mu.RUnlock() // Unlock before potentially blocking operations

		if !chanExists {
			http.NotFound(w, r)
			log.Printf("No reader channel found for POST %s", path)
			return
		}

		// Optional: Check Content-Type, e.g., application/json
		// if ct := r.Header.Get("Content-Type"); ct != "application/json" {
		// 	http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		// 	return
		// }

		// Send the request body to the application channel
		// The application is responsible for reading and closing the body.
		select {
		case readerChan <- r.Body:
			log.Printf("Received POST on %s, body provided to reader channel.", path)
			// Respond immediately assuming async processing by the application
			w.WriteHeader(http.StatusAccepted) // 202 Accepted
		case <-s.stopped:
			log.Printf("Server stopping, cannot provide reader for POST %s.", path)
			http.Error(w, "Server shutting down", http.StatusServiceUnavailable)
		default:
			// Channel likely blocked or full.
			log.Printf("Warning: Reader channel for POST %s might be blocked or full.", path)
			http.Error(w, "Server busy, try again later", http.StatusServiceUnavailable)
		}
		return
	}

	// Method not allowed for other HTTP methods
	s.mu.RUnlock() // Ensure unlock in this path too
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
}

// NewSSEReader sets up a listener for POST requests on the given port and path.
// When a POST request is received, its body (as an io.ReadCloser) is sent
// through the returned channel. The caller is responsible for reading and closing the body.
// It returns the channel, a function to stop the underlying server, and any setup error.
//
func NewSSEReader(port int, path string) (<-chan io.ReadCloser, func() error, error) {
	instance, err := getServerInstance(port)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get/create server instance for port %d: %w", port, err)
	}

	instance.mu.Lock()
	defer instance.mu.Unlock()

	// Check if a reader channel already exists for this path
	if _, exists := instance.readerChannels[path]; exists {
		// This path is already configured for reading POSTs.
		// Return the existing channel or handle as an error?
		// For now, let's return an error to avoid ambiguity.
		// Alternatively, could return the existing channel if idempotent setup is desired.
		return nil, nil, fmt.Errorf("path %s on port %d is already configured for reading", path, port)
	}

	// Create and store the channel for this path
	readerChan := make(chan io.ReadCloser, 1) // Buffer 1 to allow handler to send before receiver is ready
	instance.readerChannels[path] = readerChan

	// The handler is part of the instance's ServeHTTP method now.

	stopFunc := func() error {
		return StopServer(port)
	}

	log.Printf("Configured POST reader for path %s on port %d", path, port)
	return readerChan, stopFunc, nil
}

// NewSSEWriter sets up a listener for GET requests on the given port and path
// with the 'Accept: text/event-stream' header. When a client connects,
// an io.Writer representing the SSE connection to that client is sent
// through the returned channel. The caller uses this writer to send SSE events.
// It returns the channel, a function to stop the underlying server, and any setup error.
//
func NewSSEWriter(port int, path string) (<-chan io.Writer, func() error, error) {
	instance, err := getServerInstance(port)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get/create server instance for port %d: %w", port, err)
	}

	instance.mu.Lock()
	defer instance.mu.Unlock()

	// Check if a writer channel already exists for this path
	if _, exists := instance.writerChannels[path]; exists {
		return nil, nil, fmt.Errorf("path %s on port %d is already configured for writing", path, port)
	}

	// Create and store the channel for this path
	writerChan := make(chan io.Writer, 1) // Buffer 1? Consider implications. Allows handler to send before app reads.
	instance.writerChannels[path] = writerChan

	// The handler is part of the instance's ServeHTTP method now.

	stopFunc := func() error {
		return StopServer(port)
	}

	log.Printf("Configured SSE writer for path %s on port %d", path, port)
	return writerChan, stopFunc, nil
}
