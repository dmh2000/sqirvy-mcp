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

			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
				return
			}

			// Send an initial message to the client
			fmt.Fprintf(w, "event: endpoint\n")
			fmt.Fprintf(w, "data: /messages?session_id=1234\n\n")
			flusher.Flush() // Flush the data to the client

			// Listen for messages on the get channel (data to be sent out) and send them to the client
			for msg := range getChan { // Read from the channel assigned to the 'post' return variable
				fmt.Fprintf(w, "data: %s\n\n", string(msg)) // SSE format: "data: <message>\n\n"
				flusher.Flush()                             // Flush the data to the client
			}
			log.Println("SSE handler: getChan closed, client connection closing.")
		}

		muxGet := http.NewServeMux()
		muxGet.HandleFunc("/", sseHandler) // Handle requests to the root path

		listenAddrGet := fmt.Sprintf("%s:%d/%s", params.get_addr, params.get_port, params.get_endpoint)
		log.Printf("Starting SSE GET server on %s\n", listenAddrGet)
		if err := http.ListenAndServe(listenAddrGet, muxGet); err != nil {
			log.Fatalf("SSE GET server error: %v\n", err) // Use Fatalf to exit if server fails
		}
	}()

	// --- POST Server (Receives data to be sent via SSE) ---
	go func() {
		postHandler := func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Error reading request body", http.StatusInternalServerError)
				log.Printf("Error reading POST body: %v\n", err)
				return
			}
			defer r.Body.Close()

			// Send the received data to the get channel
			// Send the received data to the post channel (data received *from* POST)
			// Use a select with a default to prevent blocking if the channel is full or no receiver
			select {
			case postChan <- body: // Send to the channel assigned to the 'get' return variable
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, "Message received")
			default:
				// Handle case where channel is full or closed (optional)
				http.Error(w, "Server busy, try again later", http.StatusServiceUnavailable)
				log.Println("POST handler: postChan is full or closed, message dropped.")
			}
		}

		muxPost := http.NewServeMux()
		muxPost.HandleFunc("/", postHandler) // Handle requests to the root path

		listenAddrPost := fmt.Sprintf("%s:%d/%s", params.post_addr, params.post_port, params.post_endpoint)
		log.Printf("Starting SSE POST server on %s\n", listenAddrPost)
		if err := http.ListenAndServe(listenAddrPost, muxPost); err != nil {
			log.Fatalf("SSE POST server error: %v\n", err) // Use Fatalf to exit if server fails
		}
	}()

	return get, post
}
