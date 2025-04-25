package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/dmh2000/sqirvy-mcp/pkg/transport" // Adjust import path if necessary
)

const (
	// Default connection details - change these as needed
	targetIP   = "127.0.0.1"
	targetPort = 8080 // Example port, replace with the actual client port
	targetPath = "/mcp" // Example path, replace with the actual client SSE path
)

func main() {
	log.Printf("Attempting to connect to SSE endpoint at http://%s:%d%s\n", targetIP, targetPort, targetPath)

	// Use NewSSEReader to establish the connection
	reader, err := transport.NewSSEReader(targetIP, targetPort, targetPath)
	if err != nil {
		log.Fatalf("Failed to connect to SSE endpoint: %v\n", err)
	}

	// Ensure the response body is closed when main exits
	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	} else {
		log.Println("Warning: Reader returned from NewSSEReader does not implement io.Closer")
	}

	log.Println("Successfully connected. Reading response stream...")

	// Read the entire response stream
	// For a true SSE client, you'd likely want to use a dedicated SSE parser
	// library (like github.com/r3labs/sse/v2) to handle events properly.
	// This example just reads the raw stream until EOF or error.
	responseData, err := io.ReadAll(reader)
	if err != nil {
		// Don't use Fatalf here if we want to print what we *did* receive
		log.Printf("Error reading from SSE stream: %v\n", err)
	}

	// Print the received data
	fmt.Println("Received data:")
	os.Stdout.Write(responseData) // Use Write to handle potential non-UTF8 data

	log.Println("Finished reading stream.")
}
