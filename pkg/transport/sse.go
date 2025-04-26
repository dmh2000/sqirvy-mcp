package transport

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

// SSE sets up two HTTP servers for Server-Sent Events (SSE) communication.
// One server listens for GET requests (get_addr:get_port) to stream data *out* via SSE.
// The other server listens for POST requests (post_addr:post_port) to receive data *in*.
// It returns a receive-only channel (`get`) for incoming data (from POST requests)
// and a send-only channel (`post`) for outgoing data (to SSE clients).
func SSE(get_addr string, get_port int, post_addr string, post_port int) (get <-chan []byte, post chan<- []byte) {

	// Using buffered channels might be preferable depending on the use case
	getChan := make(chan []byte, 1)      // Channel for receiving data from POST requests
	postChan := make(chan []byte, 1)     // Channel for sending data to GET/SSE clients
	get = getChan                        // Assign to the return variable (receive-only view)
	post = postChan                      // Assign to the return variable (send-only view)

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

			// Listen for messages on the post channel and send them to the client
			for msg := range postChan { // Read from the send-only channel's underlying chan
				fmt.Fprintf(w, "data: %s\n\n", string(msg)) // SSE format: "data: <message>\n\n"
				flusher.Flush()                             // Flush the data to the client
			}
			log.Println("SSE handler: postChan closed, client connection closing.")
		}

		muxGet := http.NewServeMux()
		muxGet.HandleFunc("/", sseHandler) // Handle requests to the root path

		listenAddrGet := fmt.Sprintf("%s:%d", get_addr, get_port)
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
			// Use a select with a default to prevent blocking if the channel is full or no receiver
			select {
			case getChan <- body: // Send to the receive-only channel's underlying chan
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, "Message received")
			default:
				// Handle case where channel is full or closed (optional)
				http.Error(w, "Server busy, try again later", http.StatusServiceUnavailable)
				log.Println("POST handler: getChan is full or closed, message dropped.")
			}
		}

		muxPost := http.NewServeMux()
		muxPost.HandleFunc("/", postHandler) // Handle requests to the root path

		listenAddrPost := fmt.Sprintf("%s:%d", post_addr, post_port)
		log.Printf("Starting SSE POST server on %s\n", listenAddrPost)
		if err := http.ListenAndServe(listenAddrPost, muxPost); err != nil {
			log.Fatalf("SSE POST server error: %v\n", err) // Use Fatalf to exit if server fails
		}
	}()

	return get, post
}
