package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sqirvy-mcp/pkg/transport"
	"sqirvy-mcp/pkg/utils"
)

func main() {
	// 1. Define address and path
	addr := "127.0.0.1:8080" // Example address: Listen on localhost port 8080
	path := "/events"        // Example path: Clients connect to /events

	// 2. Create logger
	// Log to standard error with timestamps and source file info
	// Set log level to Debug for verbose output during development/testing
	logger := utils.New(os.Stderr, "SSE-TEST: ", log.LstdFlags|log.Lshortfile, utils.LevelDebug)
	logger.Println(utils.LevelInfo, "Starting SSE test server...")

	// 3. Create SSEWriter
	// This starts the HTTP server in the background
	sseWriter, shutdown, err := transport.NewSSEWriter(addr, path, logger)
	if err != nil {
		logger.Fatalf(utils.LevelError, "Failed to create SSE writer: %v", err)
	}
	logger.Printf(utils.LevelInfo, "SSE server listening on http://%s%s", addr, path)
	logger.Println(utils.LevelInfo, "Connect using: curl -N http://"+addr+path)

	// 4. Defer shutdown to ensure server is stopped cleanly on exit
	defer func() {
		logger.Println(utils.LevelInfo, "Initiating server shutdown...")
		// Create a context with a timeout for the shutdown process
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(shutdownCtx); err != nil {
			logger.Printf(utils.LevelError, "Server shutdown error: %v", err)
		} else {
			logger.Println(utils.LevelInfo, "Server shutdown complete.")
		}
	}()

	// 5. Wait for a client connection or shutdown signal
	logger.Println(utils.LevelInfo, "Waiting for client connection...")
	// Create a context that can be cancelled by OS signals
	connectCtx, connectCancel := context.WithCancel(context.Background())
	defer connectCancel() // Ensure cancellation happens

	// Channel to listen for interrupt or terminate signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Goroutine to handle signals: cancel the connection wait if a signal is received
	go func() {
		sig := <-sigChan
		logger.Printf(utils.LevelInfo, "Received signal: %v. Cancelling connection wait.", sig)
		connectCancel()
	}()

	// Block until a client connects or the context is cancelled (by signal)
	if err := sseWriter.WaitForConnection(connectCtx); err != nil {
		// If the error is due to cancellation (signal received), log and exit gracefully.
		if err == context.Canceled {
			logger.Println(utils.LevelInfo, "Wait for connection cancelled by signal. Exiting.")
			return // Exit main, defer will handle shutdown
		}
		// For other errors during connection wait, log fatally.
		logger.Fatalf(utils.LevelError, "Error waiting for connection: %v. Exiting.", err)
		return // Exit main, defer handles shutdown
	}
	logger.Println(utils.LevelInfo, "Client connected!")

	// 6. Send data periodically now that a client is connected
	ticker := time.NewTicker(2 * time.Second) // Send a message every 2 seconds
	defer ticker.Stop()                       // Clean up the ticker

	messageCount := 0
	keepRunning := true
	for keepRunning {
		select {
		case <-ticker.C: // Triggered every 2 seconds
			messageCount++
			// Create a simple JSON message
			message := fmt.Sprintf(`{"count": %d, "timestamp": "%s"}`, messageCount, time.Now().Format(time.RFC3339))
			logger.Printf(utils.LevelDebug, "Sending message: %s", message)
			// Write the message to the connected SSE client
			_, writeErr := sseWriter.Write([]byte(message))
			if writeErr != nil {
				// Log the error and assume the client disconnected. Stop sending.
				// In a real application, you might want more robust error handling,
				// like checking the specific error type (e.g., io.ErrClosedPipe).
				logger.Printf(utils.LevelError, "Error writing message: %v. Assuming client disconnected.", writeErr)
				keepRunning = false // Exit the loop
			}
		case sig := <-sigChan: // Check signal channel again inside the loop
			logger.Printf(utils.LevelInfo, "Received signal: %v during send loop. Exiting.", sig)
			keepRunning = false // Exit the loop
			// No need to call connectCancel here as we are already past that stage
		}
	}

	logger.Println(utils.LevelInfo, "Exiting send loop.")
	// Shutdown is handled by the deferred function call
}
