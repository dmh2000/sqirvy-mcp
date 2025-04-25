// Package transport provides functionality for transporting messages between components.
package transport

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// SSEServer represents a server that handles Server-Sent Events
type SSEServer struct {
	server   *http.Server
	stopChan chan struct{}
	stopOnce sync.Once
}

// SSEWriter wraps an http.ResponseWriter and http.Flusher to implement io.Writer
type SSEWriter struct {
	w http.ResponseWriter
	f http.Flusher
}

// Write implements io.Writer by writing data and flushing
func (sw *SSEWriter) Write(p []byte) (int, error) {
	n, err := sw.w.Write(p)
	if err == nil {
		sw.f.Flush()
	}
	return n, err
}

// SSEReader implements io.Reader to read from an SSE stream
type SSEReader struct {
	dataChan <-chan []byte
	buffer   []byte
}

// Read implements io.Reader by reading from the data channel
func (sr *SSEReader) Read(p []byte) (int, error) {
	// If we have data in the buffer, use it first
	if len(sr.buffer) > 0 {
		n := copy(p, sr.buffer)
		sr.buffer = sr.buffer[n:]
		return n, nil
	}

	// Wait for data from the channel - simple receive operation
	data, ok := <-sr.dataChan
	if !ok {
		return 0, io.EOF
	}
	
	// If the provided buffer is too small, store the remainder
	if len(data) > len(p) {
		n := copy(p, data)
		sr.buffer = data[n:]
		return n, nil
	}
	
	return copy(p, data), nil
}

// Global map to track server instances by port
var (
	servers     = make(map[int]*SSEServer)
	serversLock sync.Mutex
)

// NewSSEServer creates a new SSE server that listens for GET requests
// and forwards data to a reader. It returns an io.Reader that can be used
// to read the data sent by the server.
func NewSSEServer(port int, path string) (io.Reader, func() error, error) {
	// Create a channel to receive data from the server
	dataChan := make(chan []byte, 10)
	
	// Create a mux to handle requests
	mux := http.NewServeMux()
	
	// Create a server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	
	// Create a stop channel
	stopChan := make(chan struct{})
	
	// Create an SSE server
	sseServer := &SSEServer{
		server:   server,
		stopChan: stopChan,
	}
	
	// Register the server in the global map
	serversLock.Lock()
	servers[port] = sseServer
	serversLock.Unlock()
	
	// Handle GET requests for SSE
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		// Only handle GET requests
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}
		
		// Set headers for SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()
		
		log.Printf("SSE client connected on GET %s", path)
		
		// Keep the connection open until the client disconnects
		<-r.Context().Done()
		log.Printf("SSE client disconnected on GET %s", path)
	})
	
	// Start the server in a goroutine
	go func() {
		log.Printf("Starting SSE server on port %d", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("SSE server error: %v", err)
		}
		log.Printf("SSE server on port %d stopped", port)
		close(stopChan)
		
		// Remove from global map when server stops
		serversLock.Lock()
		delete(servers, port)
		serversLock.Unlock()
	}()
	
	// Create a reader
	reader := &SSEReader{
		dataChan: dataChan,
	}
	
	// Create a stop function
	stopFunc := func() error {
		var err error
		sseServer.stopOnce.Do(func() {
			log.Printf("Stopping SSE server on port %d", port)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err = server.Shutdown(ctx)
			close(dataChan)
		})
		return err
	}
	
	return reader, stopFunc, nil
}

// NewSSEWriter creates a new SSE writer for the given port and path
func NewSSEWriter(port int, path string) (<-chan io.Writer, func() error, error) {
	// Create a channel for writers
	writerChan := make(chan io.Writer, 5)
	
	// Create a mux to handle requests
	mux := http.NewServeMux()
	
	// Create a server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	
	// Create a stop channel
	stopChan := make(chan struct{})
	
	// Create an SSE server
	sseServer := &SSEServer{
		server:   server,
		stopChan: stopChan,
	}
	
	// Register the server in the global map
	serversLock.Lock()
	if _, exists := servers[port]; exists {
		serversLock.Unlock()
		return nil, nil, fmt.Errorf("path %s on port %d is already configured for writing", path, port)
	}
	servers[port] = sseServer
	serversLock.Unlock()
	
	// Handle GET requests for SSE
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		// Only handle GET requests
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}
		
		// Set headers for SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()
		
		// Create an SSE writer
		writer := &SSEWriter{
			w: w,
			f: flusher,
		}
		
		// Send the writer to the application
		writerChan <- writer
		
		log.Printf("SSE client connected on GET %s, writer provided.", path)
		
		// Keep the connection open until the client disconnects
		<-r.Context().Done()
		log.Printf("SSE client disconnected on GET %s.", path)
	})
	
	// Start the server in a goroutine
	go func() {
		log.Printf("Starting SSE server on port %d", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("SSE server error: %v", err)
		}
		log.Printf("SSE server on port %d stopped", port)
		close(stopChan)
		
		// Remove from global map when server stops
		serversLock.Lock()
		delete(servers, port)
		serversLock.Unlock()
	}()
	
	// Create a stop function
	stopFunc := func() error {
		var err error
		sseServer.stopOnce.Do(func() {
			log.Printf("Stopping SSE server on port %d", port)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err = server.Shutdown(ctx)
			close(writerChan)
		})
		return err
	}
	
	return writerChan, stopFunc, nil
}

// NewSSEReader creates a new SSE reader for the given port and path
func NewSSEReader(port int, path string) (<-chan io.ReadCloser, func() error, error) {
	// Create a channel for readers
	readerChan := make(chan io.ReadCloser, 5)
	
	// Create a mux to handle requests
	mux := http.NewServeMux()
	
	// Create a server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	
	// Create a stop channel
	stopChan := make(chan struct{})
	
	// Create an SSE server
	sseServer := &SSEServer{
		server:   server,
		stopChan: stopChan,
	}
	
	// Register the server in the global map
	serversLock.Lock()
	if _, exists := servers[port]; exists {
		serversLock.Unlock()
		return nil, nil, fmt.Errorf("path %s on port %d is already configured for reading", path, port)
	}
	servers[port] = sseServer
	serversLock.Unlock()
	
	// Handle POST requests
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		// Only handle POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		// Send the request body to the application
		readerChan <- r.Body
		
		log.Printf("Received POST on %s, body provided to reader channel.", path)
		
		// Respond with 202 Accepted
		w.WriteHeader(http.StatusAccepted)
	})
	
	// Start the server in a goroutine
	go func() {
		log.Printf("Starting SSE server on port %d", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("SSE server error: %v", err)
		}
		log.Printf("SSE server on port %d stopped", port)
		close(stopChan)
		
		// Remove from global map when server stops
		serversLock.Lock()
		delete(servers, port)
		serversLock.Unlock()
	}()
	
	// Create a stop function
	stopFunc := func() error {
		var err error
		sseServer.stopOnce.Do(func() {
			log.Printf("Stopping SSE server on port %d", port)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err = server.Shutdown(ctx)
			close(readerChan)
		})
		return err
	}
	
	return readerChan, stopFunc, nil
}

// StopServer stops the server on the given port
func StopServer(port int) error {
	serversLock.Lock()
	sseServer, exists := servers[port]
	serversLock.Unlock()
	
	if !exists {
		return fmt.Errorf("no server running on port %d", port)
	}
	
	var err error
	sseServer.stopOnce.Do(func() {
		log.Printf("Stopping SSE server on port %d", port)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err = sseServer.server.Shutdown(ctx)
	})
	
	return err
}
