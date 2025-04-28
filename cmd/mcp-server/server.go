package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	// Use the absolute module path
	"bytes" // Added for peekMessageType
	resources "sqirvy-mcp/cmd/mcp-server/resources"
	mcp "sqirvy-mcp/pkg/mcp"
	utils "sqirvy-mcp/pkg/utils"
)

const (
	notificationInitialized = "initialized" // Standard notification method from client after initialize response
)

// peekMessageType attempts to unmarshal just enough to get the method/id/error.
// This is useful for logging before full unmarshalling and handling.
func peekMessageType(logger *utils.Logger, payload []byte) (method string, id mcp.RequestID, isNotification bool, isResponse bool, isError bool) {
	var base struct {
		Method  string          `json:"method"`
		ID      mcp.RequestID   `json:"id"`      // Can be string, number, or null/absent
		Error   json.RawMessage `json:"error"`   // Check if non-null
		Result  json.RawMessage `json:"result"`  // Check if non-null
		Params  json.RawMessage `json:"params"`  // Needed to differentiate req/notification
		JSONRPC string          `json:"jsonrpc"` // Check for presence
	}

	// Use a decoder to ignore unknown fields gracefully
	decoder := json.NewDecoder(bytes.NewReader(payload))

	if err := decoder.Decode(&base); err != nil {
		// Cannot determine type if basic unmarshal fails
		logger.Printf("DEBUG", "Failed to decode base JSON-RPC structure: %v", err)
		return "", nil, false, false, false
	}

	// Basic JSON-RPC validation
	if base.JSONRPC != "2.0" {
		logger.Printf("DEBUG", "Invalid JSON-RPC version: %s", base.JSONRPC)
		return "", nil, false, false, false // Not a valid JSON-RPC 2.0 message
	}

	id = base.ID // Store the ID (can be nil)
	method = base.Method

	// Determine message type based on fields present according to JSON-RPC 2.0 spec
	hasID := base.ID != nil
	hasMethod := base.Method != ""
	hasResult := len(base.Result) > 0 && string(base.Result) != "null"
	hasError := len(base.Error) > 0 && string(base.Error) != "null"

	isNotification = !hasID && hasMethod          // Notification: MUST NOT have id, MUST have method
	isResponse = hasID && (hasResult || hasError) // Response: MUST have id, MUST have result OR error (but not both)
	isError = hasID && hasError                   // Error Response: MUST have id, MUST have error

	// If it's not a notification or response, it should be a request
	// isRequest := hasID && hasMethod && !hasResult && !hasError
	return method, id, isNotification, isResponse, isError
}

// Server handles the MCP communication logic.
type Server struct {
	reader           *bufio.Reader
	writer           io.Writer     // Using io.Writer for flexibility, though likely os.Stdout
	logger           *utils.Logger // Use the custom logger type
	mu               sync.Mutex    // Protects writer access
	initialized      bool
	serverVersion    string
	serverInfo       mcp.Implementation
	incomingMessages chan []byte   // Channel for incoming message payloads
	shutdown         chan struct{} // Channel to signal shutdown
	config           *Config       // Server configuration
}

// NewServer creates a new MCP server instance.
func NewServer(reader io.Reader, writer io.Writer, logger *utils.Logger, config *Config) *Server {
	return &Server{
		reader:           bufio.NewReader(reader),
		writer:           writer,
		logger:           logger,
		initialized:      false,
		serverVersion:    "2024-11-05",          // Align with your spec/schema version
		incomingMessages: make(chan []byte, 10), // Buffered channel
		shutdown:         make(chan struct{}),
		config:           config,
		serverInfo: mcp.Implementation{
			Name:    "GoMCPExampleServer",
			Version: "0.1.0", // Example version
		},
	}
}

// Run starts the server's main loop.
func (s *Server) Run() error {
	s.initialized = false // Ensure server starts in non-initialized state

	// Initialize the project root path function
	resources.GetProjectRootPath = func() string {
		return s.config.Project.RootPath
	}

	// 1. Start background reader loop immediately
	go s.readLoop()

	// 3. Main processing loop
	for {
		// s.logger.Print("Waiting for incoming messages...")
		select {
		case payload := <-s.incomingMessages:
			// Process the received message
			s.processMessage(payload)
		case <-s.shutdown:
			s.logger.Println("DEBUG", "Shutdown signal received. Exiting processing loop.")
			return nil // Normal shutdown
		}
	}
}

// readLoop continuously reads messages from the transport and sends them to the incomingMessages channel.
// readLoop continuously reads messages (lines) from the server's reader (s.reader),
// sending valid JSON payloads to the incomingMessages channel.
// It exits when the reader encounters an error (like io.EOF).
func (s *Server) readLoop() {
	defer func() {
		s.logger.Println("DEBUG", "Exiting read loop.")
		close(s.shutdown) // Signal the main loop to shut down when reading stops
	}()

	// Use the server's buffered reader directly
	for {
		// s.logger.Println("Waiting for line from s.reader...")
		// Read until newline. Assumes one JSON message per line.
		payload, err := s.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				s.logger.Println("DEBUG", "EOF received from reader. Shutting down read loop.")
			} else {
				s.logger.Printf("DEBUG", "Error reading from reader: %v", err)
			}
			return // Exit loop on EOF or any other error
		}

		// Trim trailing newline characters for correct JSON parsing
		payload = bytes.TrimSpace(payload)
		if len(payload) == 0 {
			s.logger.Println("DEBUG", "Received empty line, skipping.")
			continue // Skip empty lines
		}

		// Basic validation: Check if it looks like JSON
		if !(bytes.HasPrefix(payload, []byte("{")) && bytes.HasSuffix(payload, []byte("}"))) {
			s.logger.Printf("DEBUG", "Received line does not look like JSON object, skipping: %s", string(payload))
			continue
		}

		// Send the raw payload (single line) to the processing loop
		// Use a select with a default to prevent blocking if the channel is full,
		// though the channel is buffered. Consider error handling if it fills up.
		select {
		case s.incomingMessages <- payload:
			// Successfully sent to channel
		default:
			s.logger.Println("DEBUG", "Warning: incomingMessages channel full. Discarding message.")
			// Or potentially block, log more severely, or increase buffer size.
		}
	}
}

// processMessage determines the type of message and routes it appropriately.
// It also handles the initial state transitions (waiting for initialize, waiting for initialized).
func (s *Server) processMessage(payload []byte) {
	method, id, isNotification, isResponse, isError := peekMessageType(s.logger, payload)
	s.logger.Printf("INFO", "R:%s", string(payload)) // INFO for received JSON
	// --- State Machine: Before Initialization ---
	if !s.initialized {
		// State 1: Waiting for "initialize" request
		if method == mcp.MethodInitialize && !isNotification && id != nil {
			// s.logger.Printf("Received 'initialize' request (ID: %v) while not initialized.", id)
			responseBytes, handleErr := s.handleInitializeRequest(id, payload)
			// Send response (success or error marshalled by handler)
			if handleErr != nil {
				s.logger.Printf("DEBUG", "Error during handling of 'initialize' request (ID: %v): %v", id, handleErr)
				os.Exit(1) // Exit if initialization fails critically
			}
			if responseBytes != nil {
				if sendErr := s.sendRawMessage(responseBytes); sendErr != nil {
					// Use Fatalf for critical send errors
					s.logger.Fatalf("DEBUG", "FATAL: Failed to send initialize response/error for request ID %v: %v", id, sendErr)
				} else {
					s.initialized = true // Set initialized state after sending response
				}
			}
			return
		}
	}

	// --- State Machine: Initialized ---
	// Handle messages received *after* initialization is complete.
	// s.logger.Printf("Server is initialized. Processing message (Method: %s, ID: %v)", method, id)

	if isNotification {
		// Handle 'initialized' notification received *after* already initialized (benign)
		if method == notificationInitialized || method == "notifications/initialized" {
			return
		}
		s.logger.Printf("DEBUG", "Received Notification (Method: %s). No response needed.", method)
		// Handle other specific notifications like $/cancel if needed
		return
	}

	if isResponse || isError {
		// Server shouldn't receive responses unless it sent requests (not implemented yet)
		s.logger.Printf("DEBUG", "Warning: Received unexpected Response/Error message (ID: %v, Method: %s, IsError: %t). Ignoring.", id, method, isError)
		return
	}

	// It's a Request (must have ID and method, not result/error)
	if id == nil || method == "" {
		s.logger.Printf("DEBUG", "Error: Received message that is not a valid Request, Notification, or Response. Payload: %s", string(payload))
		// Cannot send error response if ID is missing.
		return
	}

	// s.logger.Printf("Received Request (ID: %v, Method: %s)", id, method)

	var responseBytes []byte
	var handleErr error // Error returned by the handler function itself

	// Route to the appropriate handler
	switch method {
	case mcp.MethodInitialize:
		// Handle duplicate 'initialize' request after initialization
		s.logger.Printf("DEBUG", "Error: Received duplicate 'initialize' request (ID: %v) after initialization.", id)
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInvalidRequest, "Server already initialized", nil)
		responseBytes, handleErr = s.marshalErrorResponse(id, rpcErr) // Use helper

	case mcp.MethodListTools:
		responseBytes, handleErr = s.handleListTools(id)
	case mcp.MethodCallTool:
		// Pass the full payload to handleCallTool for parsing params
		responseBytes, handleErr = s.handleCallTool(id, payload)
	case mcp.MethodListPrompts:
		responseBytes, handleErr = s.handleListPrompts(id)
	case mcp.MethodGetPrompt:
		responseBytes, handleErr = s.handleGetPrompt(id, payload)
	case mcp.MethodListResources:
		responseBytes, handleErr = s.handleListResources(id)
	case mcp.MethodListResourceTemplates: // Added case for templates list
		responseBytes, handleErr = s.handleListResourceTemplates(id)
	case mcp.MethodReadResource: // Handle resources/read
		responseBytes, handleErr = s.handleReadResource(id, payload)
	case mcp.MethodPing: // Handle ping
		responseBytes, handleErr = s.handlePingRequest(id)
	// Add cases for other supported methods like logging/setLevel, etc.
	default:
		s.logger.Printf("DEBUG", "Received unsupported method '%s' for request ID %v", method, id)
		responseBytes, handleErr = createMethodNotFoundResponse(id, method, s.logger)
	}

	// --- Response Sending ---
	if handleErr != nil {
		// The handler failed internally (e.g., failed to marshal its *intended* response/error).
		s.logger.Printf("DEBUG", "Error during handling of request (ID: %v, Method: %s): %v", id, method, handleErr)
		// If responseBytes is not nil here, it means the handler *did* manage to marshal an error response despite the internal error.
		if responseBytes == nil {
			// If the handler couldn't even produce an error response, create a generic one.
			s.logger.Printf("DEBUG", "Handler failed without producing an error response. Creating generic InternalError.")
			rpcErr := mcp.NewRPCError(mcp.ErrorCodeInternalError, fmt.Sprintf("Internal server error processing method %s", method), nil)
			responseBytes, _ = mcp.MarshalErrorResponse(id, rpcErr) // Ignore marshal error here, send if possible
		}
	}

	// Send the response (either success or error marshalled by the handler or the generic error)
	if responseBytes != nil {
		if sendErr := s.sendRawMessage(responseBytes); sendErr != nil {
			// Use Fatalf for critical send errors
			s.logger.Fatalf("DEBUG", "FATAL: Failed to send response/error for request ID %v: %v", id, sendErr)
		}
	} else {
		// This case should ideally not happen if handlers always return marshalled bytes or an error
		s.logger.Printf("DEBUG", "Warning: No response bytes generated for request (ID: %v, Method: %s), handleErr was: %v", id, method, handleErr)
	}
}

// sendRawMessage sends pre-marshalled bytes asynchronously using a goroutine.
// It logs the payload and launches a goroutine to perform the write and flush.
// Errors during the write operation are logged within the goroutine.
// This function returns immediately (nil error).
func (s *Server) sendRawMessage(payload []byte) error {
	// Launch a goroutine to handle the actual sending
	go func(p []byte) {
		s.mu.Lock()
		defer s.mu.Unlock()

		if _, err := s.writer.Write(p); err != nil {
			s.logger.Printf("DEBUG", "Error in async sendRawMessage: failed to write message payload: %v", err)
			return // Exit goroutine on write error
		}

		// Add newline after the JSON payload
		if _, err := s.writer.Write([]byte("\n")); err != nil {
			s.logger.Printf("DEBUG", "Error in async sendRawMessage: failed to write newline: %v", err)
			// Continue to attempt flush even if newline fails
		}
	}(payload) // Pass payload as argument to avoid closure issues

	return nil // Return immediately
}

// sendResponse marshals a successful result into a full RPCResponse and sends it.
// Returns the marshalled bytes and any error during marshalling.
// It does *not* send the bytes itself.
func (s *Server) marshalResponse(id mcp.RequestID, result interface{}) ([]byte, error) {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		err = fmt.Errorf("failed to marshal result for response ID %v: %w", id, err)
		s.logger.Println("DEBUG", err.Error())
		// Return bytes for an internal error instead
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInternalError, "Failed to marshal response result", nil)
		errorBytes, marshalErr := mcp.MarshalErrorResponse(id, rpcErr)
		// If we can't even marshal the error, return the original error and nil bytes
		if marshalErr != nil {
			s.logger.Printf("DEBUG", "CRITICAL: Failed to marshal error response for result marshalling failure: %v", marshalErr)
			return nil, err // Return the original marshalling error
		}
		return errorBytes, err // Return the marshalled error bytes and the original error
	}

	resp := mcp.RPCResponse{
		JSONRPC: mcp.JSONRPCVersion,
		Result:  resultBytes,
		ID:      id,
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		// This is highly unlikely if result marshalling worked, but handle defensively
		err = fmt.Errorf("failed to marshal final response object for ID %v: %w", id, err)
		s.logger.Println("DEBUG", err.Error())
		// Return bytes for an internal error instead
		rpcErr := mcp.NewRPCError(mcp.ErrorCodeInternalError, "Failed to marshal final response object", nil)
		errorBytes, marshalErr := mcp.MarshalErrorResponse(id, rpcErr)
		if marshalErr != nil {
			s.logger.Printf("DEBUG", "CRITICAL: Failed to marshal error response for final response marshalling failure: %v", marshalErr)
			return nil, err // Return the original marshalling error
		}
		return errorBytes, err // Return the marshalled error bytes and the original error
	}
	// log the response string as type "INFO"
	s.logger.Printf("INFO", "S:%s", string(respBytes))

	return respBytes, nil // Return marshalled success response bytes and nil error
}

// marshalErrorResponse marshals an RPCError into a full RPCResponse.
// Returns the marshalled bytes and any error during marshalling.
// It does *not* send the bytes itself.
func (s *Server) marshalErrorResponse(id mcp.RequestID, rpcErr *mcp.RPCError) ([]byte, error) {
	responseBytes, err := mcp.MarshalErrorResponse(id, rpcErr)
	if err != nil {
		// Log the failure to marshal the *intended* error
		s.logger.Printf("DEBUG", "CRITICAL: Failed to marshal error response (Code: %d, Msg: %s) for ID %v: %v", rpcErr.Code, rpcErr.Message, id, err)

		// Try to marshal a more generic internal error response
		genericErr := mcp.NewRPCError(mcp.ErrorCodeInternalError, "Failed to marshal original error response", nil)
		responseBytes, err = mcp.MarshalErrorResponse(id, genericErr)
		if err != nil {
			// If we can't even marshal the generic error, log and return the error
			s.logger.Printf("DEBUG", "CRITICAL: Failed to marshal generic error response for ID %v: %v. No error response possible.", id, err)
			return nil, fmt.Errorf("failed to marshal even generic error response: %w", err)
		}
		// Return the generic error bytes, but also the original error that occurred
		return responseBytes, fmt.Errorf("failed to marshal original error response (Code: %d), sending generic error instead: %w", rpcErr.Code, err)
	}
	// Return the successfully marshalled error response bytes and nil error
	return responseBytes, nil
}
