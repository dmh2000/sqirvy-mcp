package transport

import (
	"fmt"
	"io"
	"log"
	"net/http"
	utils "sqirvy-mcp/pkg/utils"
)

type SSEparams struct {
	get_addr      string
	get_port      int
	get_endpoint  string
	post_addr     string
	post_port     int
	post_endpoint string
	logger        *utils.Logger
}

func SseMakeParams(get_addr string,
	get_port int,
	get_endpoint string,
	post_addr string, post_port int,
	post_endpoint string,
	logger *utils.Logger,
) SSEparams {
	return SSEparams{
		get_addr:      get_addr,
		get_port:      get_port,
		get_endpoint:  get_endpoint,
		post_addr:     post_addr,
		post_port:     post_port,
		post_endpoint: post_endpoint,
		logger:        logger,
	}
}

// SSE sets up two HTTP servers for Server-Sent Events (SSE) communication.
// One server listens for GET requests (get_addr:get_port) to stream data *out* via SSE.
// The other server listens for POST requests (post_addr:post_port) to receive data *in*.
// It returns a receive-only channel (`get`) for incoming data (from POST requests)
// and a send-only channel (`post`) for outgoing data (to SSE clients).
func StartSSE(params SSEparams) (get chan []byte, post chan []byte) {

	// Using buffered channels might be preferable depending on the use case
	// postChan receives data from POST requests. It's assigned to the 'get' return value.
	postChan := make(chan []byte, 1)
	// getChan sends data to GET/SSE clients. It's assigned to the 'post' return value.
	getChan := make(chan []byte, 1)
	get = postChan // Assign postChan (data *from* POST) to the receive-only return channel 'get'
	post = getChan // Assign getChan (data *to* GET/SSE) to the send-only return channel 'post'

	// --- GET Server (Server-Sent Events endpoint) ---
	go func() {
		sseHandler := func(w http.ResponseWriter, r *http.Request) {
			// Set headers for SSE
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			w.Header().Set("Access-Control-Allow-Origin", "*") // Allow CORS if needed

			params.logger.Println(utils.LevelDebug, "SSE handler: New client connected")
			flusher, ok := w.(http.Flusher)
			if !ok {
				params.logger.Println(utils.LevelError, "SSE handler: Streaming unsupported!")
				http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
				return
			}

			// Send an initial message to the client
			params.logger.Println(utils.LevelDebug, "SSE handler: Sending initial endpoint message")
			fmt.Fprintf(w, "event: endpoint\n")
			fmt.Fprintf(w, "data: /messages?session_id=1234\n\n")
			flusher.Flush() // Flush the data to the client

			// Listen for messages on the get channel (data to be sent out) and send them to the client
			for msg := range getChan { // Read from the channel assigned to the 'post' return variable
				params.logger.Printf(utils.LevelDebug, "SSE handler: Sending message: %s", string(msg))
				_, err := fmt.Fprintf(w, "data: %s\n\n", string(msg)) // SSE format: "data: <message>\n\n"
				if err != nil {
					params.logger.Printf(utils.LevelError, "SSE handler: Error writing message to client: %v", err)
					break // Stop trying to send if writing fails
				}
				flusher.Flush() // Flush the data to the client
			}
			params.logger.Println(utils.LevelInfo, "SSE handler: getChan closed, client connection closing.")
		}

		muxGet := http.NewServeMux()
		muxGet.HandleFunc(params.get_endpoint, sseHandler) // Handle requests to the specified endpoint

		listenAddrGet := fmt.Sprintf("%s:%d", params.get_addr, params.get_port) // Listen on address and port only
		params.logger.Printf(utils.LevelInfo, "Starting SSE GET server on %s, endpoint %s\n", listenAddrGet, params.get_endpoint)
		if err := http.ListenAndServe(listenAddrGet, muxGet); err != nil {
			// Use the logger's Fatalf which includes os.Exit(1)
			params.logger.Fatalf(utils.LevelError, "SSE GET server error: %v\n", err)
		}
	}()

	// --- POST Server (Receives data to be sent via SSE) ---
	go func() {
		postHandler := func(w http.ResponseWriter, r *http.Request) {
			params.logger.Printf(utils.LevelDebug, "POST handler: Received request: %s %s", r.Method, r.URL.Path)
			if r.Method != http.MethodPost {
				params.logger.Printf(utils.LevelWarning, "POST handler: Method not allowed: %s", r.Method)
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				params.logger.Printf(utils.LevelError, "POST handler: Error reading request body: %v", err)
				http.Error(w, "Error reading request body", http.StatusInternalServerError)
				return
			}
			defer r.Body.Close()
			params.logger.Printf(utils.LevelDebug, "POST handler: Received body: %s", string(body))

			// Send the received data to the post channel
			// Send the received data to the post channel (data received *from* POST)
			// Use a select with a default to prevent blocking if the channel is full or no receiver
			select {
			case postChan <- body: // Send to the channel assigned to the 'get' return variable
				params.logger.Println(utils.LevelDebug, "POST handler: Message sent to postChan successfully")
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, "Message received")
			default:
				// Handle case where channel is full or closed
				params.logger.Println(utils.LevelWarning, "POST handler: postChan is full or closed, message dropped.")
				http.Error(w, "Server busy, try again later", http.StatusServiceUnavailable)
			}
		}

		muxPost := http.NewServeMux()
		muxPost.HandleFunc(params.post_endpoint, postHandler) // Handle requests to the specified endpoint

		listenAddrPost := fmt.Sprintf("%s:%d", params.post_addr, params.post_port) // Listen on address and port only
		params.logger.Printf(utils.LevelInfo, "Starting SSE POST server on %s, endpoint %s\n", listenAddrPost, params.post_endpoint)
		if err := http.ListenAndServe(listenAddrPost, muxPost); err != nil {
			// Use the logger's Fatalf which includes os.Exit(1)
			params.logger.Fatalf(utils.LevelError, "SSE POST server error: %v\n", err)
		}
	}()

	params.logger.Println(utils.LevelInfo, "SSE transport started successfully")
	return get, post
}
