package transport

import (
	"fmt"
	"io"
	"net/http"
	utils "sqirvy-mcp/pkg/utils"
	"sync"
)

type SSEparams struct {
	addr     string
	port     int
	endpoint string
	logger   *utils.Logger
}

func SseMakeParams(
	addr string,
	port int,
	endpoint string,
	logger *utils.Logger,
) SSEparams {
	return SSEparams{
		addr:     addr,
		port:     port,
		endpoint: endpoint,
		logger:   logger,
	}
}

// SSE sets up two HTTP servers for Server-Sent Events (SSE) communication.
// One server listens for GET requests (get_addr:get_port) to stream data *out* via SSE.
// The other server listens for POST requests (post_addr:post_port) to receive data *in*.
// It returns a receive-only channel (`get`) for incoming data (from POST requests)
// and a send-only channel (`post`) for outgoing data (to SSE clients).
func StartSSE(p SSEparams) (chan []byte, chan []byte) {

	// Using buffered channels might be preferable depending on the use case
	// postChan receives data from POST requests. It's assigned to the 'get' return value.
	postChan := make(chan []byte, 1)
	// getChan sends data to GET/SSE clients. It's assigned to the 'post' return value.
	getChan := make(chan []byte, 1)

	// --- GET Server (Server-Sent Events endpoint) ---
	mu := sync.Mutex{}
	connected := false
	go func() {
		postHandler := func(w http.ResponseWriter, r *http.Request) {
			p.logger.Printf(utils.LevelDebug, "POST handler: Received request: %s %s", r.Method, r.URL.Path)

			body, err := io.ReadAll(r.Body)
			if err != nil {
				p.logger.Printf(utils.LevelError, "POST handler: Error reading request body: %v", err)
				http.Error(w, "Error reading request body", http.StatusInternalServerError)
				return
			}
			defer r.Body.Close()
			p.logger.Printf(utils.LevelDebug, "POST handler: Received body: %s", string(body))

			// Send the received data to the post channel
			// Send the received data to the post channel (data received *from* POST)
			// Use a select with a default to prevent blocking if the channel is full or no receiver
			select {
			case postChan <- body: // Send to the channel assigned to the 'get' return variable
				p.logger.Println(utils.LevelDebug, "POST handler: Message sent to postChan successfully")
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, "Message received")
			default:
				// Handle case where channel is full or closed
				p.logger.Println(utils.LevelWarning, "POST handler: postChan is full or closed, message dropped.")
				http.Error(w, "Server busy, try again later", http.StatusServiceUnavailable)
			}
		}

		getHandler := func(w http.ResponseWriter, r *http.Request) {
			p.logger.Printf(utils.LevelDebug, "SSE handler: Received request: %s %s", r.Method, r.URL.Path)

			// validate the origin

			// Set headers for SSE
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK) // send an HTTP 200

			// Flush the headers to the client
			flusher, ok := w.(http.Flusher)
			if !ok {
				p.logger.Println(utils.LevelError, "SSE handler: Streaming unsupported!")
				http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
				return
			}
			flusher.Flush()

			p.logger.Println(utils.LevelDebug, "SSE handler: New client connected")

			// send initialization response even if no POST received

			msg := []byte(`{"jsonrpc": "2.0", "method": "notifications/initialized"}`)
			p.logger.Printf(utils.LevelDebug, "SSE handler: Sending message: %s", string(msg))
			w.Write(msg)
			flusher.Flush()

			// Listen for messages on the get channel (data to be sent to the client
			for msg := range getChan {
				p.logger.Printf(utils.LevelDebug, "SSE handler: Sending message: %s", string(msg))
				w.Write(msg)
				flusher.Flush() // Flush the data to the client
			}
			p.logger.Println(utils.LevelInfo, "SSE handler: getChan closed, client connection closing.")
		}

		sseHandler := func(w http.ResponseWriter, r *http.Request) {

			// only one client at a time
			mu.Lock()
			defer mu.Unlock()
			if !connected {
				connected = true
			} else {
				p.logger.Println(utils.LevelError, "SSE handler: Client already connected")
				http.Error(w, "Client already connected", http.StatusMethodNotAllowed)
				return
			}

			if r.Method == "GET" {
				getHandler(w, r)
			} else if r.Method == "POST" {
				postHandler(w, r)
			} else {
				p.logger.Printf(utils.LevelError, "SSE handler: Invalid request method: %s", r.Method)
			}
		}

		muxGet := http.NewServeMux()
		muxGet.HandleFunc(p.endpoint, sseHandler) // Handle requests to the specified endpoint

		listenAddr := fmt.Sprintf("%s:%d", p.addr, p.port) // Listen on address and port only
		p.logger.Printf(utils.LevelInfo, "Starting SSE GET server on %s\n", listenAddr)

		if err := http.ListenAndServe(listenAddr, muxGet); err != nil {
			// Use the logger's Fatalf which includes os.Exit(1)
			p.logger.Fatalf(utils.LevelError, "SSE GET server error: %v\n", err)
		}
	}()

	p.logger.Println(utils.LevelInfo, "SSE transport started successfully")
	return getChan, postChan
}
