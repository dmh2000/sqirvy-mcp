package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/dmh2000/sqirvy-mcp/pkg/transport" // Adjust import path if necessary
)

const (
	// Server details for the test server we will start
	serverPort = 3001   // Example port for the server to listen on
	serverPath = "/sse" // Example path for the server endpoint
)

func main() {
	log.Printf("Starting SSE server on port %d, path %s\n", serverPort, serverPath)

	// Use NewSSEServer to start the server
	// Note: The returned reader reads from an internal channel,
	// but the current server implementation doesn't send data to it.
	reader, stopFunc, err := transport.NewSSEServer(serverPort, serverPath)
	if err != nil {
		log.Fatalf("Failed to start SSE server: %v\n", err)
	}
	defer func() {
		log.Println("Stopping server...")
		if err := stopFunc(); err != nil {
			log.Printf("Error stopping server: %v\n", err)
		}
	}()

	log.Println("Server started successfully. Attempting to read from the associated reader...")
	log.Println("Note: The current server implementation doesn't actively send data through this reader.")

	// Attempt to read from the reader in a goroutine with a timeout,
	// as it might block indefinitely or return EOF immediately.
	readDone := make(chan struct{})
	var receivedData []byte
	var readErr error

	go func() {
		defer close(readDone)
		// This read will likely return EOF immediately because the channel
		// passed to SSEReader is closed when the server stops, and nothing
		// ever sends data to it before that. Or it might block if the channel
		// wasn't closed properly in some scenario (though stopFunc closes it).
		receivedData, readErr = io.ReadAll(reader)
		if readErr != nil && readErr != io.EOF {
			// Log error if it's not EOF
			log.Printf("Error reading from SSE reader: %v\n", readErr)
		} else if readErr == io.EOF {
			log.Println("Reader returned EOF.")
		}
	}()

	// Wait for the read goroutine to finish or timeout
	select {
	case <-readDone:
		log.Println("Reading finished.")
	case <-time.After(2 * time.Second): // Reduced timeout
		log.Println("Timeout waiting for read operation.")
		// Attempting to stop the server via defer might unblock the reader if it was stuck
	}

	// Print any received data (likely none)
	if len(receivedData) > 0 {
		fmt.Println("Received data:")
		os.Stdout.Write(receivedData)
	} else {
		log.Println("No data received from the reader.")
	}

	log.Println("Test finished.")
}
