package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	mcp "sqirvy-mcp/pkg/mcp"
	utils "sqirvy-mcp/pkg/utils"
)

// No need for configuration file constants here, they are defined in config.go

func main() {
	// --- Command Line Flags ---
	configPath := flag.String("config", "", "Path to the configuration file")
	logFilePath := flag.String("log", "./sqirvy-mcp.log", "Path to the log file (overrides config file)")
	logLevel := flag.String("log-level", "INFO", "Log level: DEBUG,INFO,WARNING,ERROR (overrides config file)")
	projectRoot := flag.String("project-root", ".", "Root path for file resources (overrides config file)")
	// Ping target flag removed as it's now provided by the client
	flag.Parse()

	// --- Load Configuration ---
	// Create a temporary logger for configuration loading
	tempLogger := utils.New(os.Stderr, "", log.LstdFlags, utils.LevelDebug)
	config, err := LoadConfig(*configPath, tempLogger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		// Exit on validation errors
		// Ping target validation removed as it's now provided by the client
		// Continue with default config for other errors, flags will override as needed
		tempLogger.Printf("DEBUG", "Continuing with default configuration")
	}

	// --- Override Configuration with Command Line Flags ---
	if *logFilePath != "" {
		config.Log.Output = *logFilePath
	}
	if *logLevel != "" {
		config.Log.Level = *logLevel
	}
	if *projectRoot != "" {
		config.Project.RootPath = *projectRoot
	}
	// Ping target flag handling removed as it's now provided by the client

	// Validate the final configuration (after applying command-line flags)
	if err := ValidateConfig(config, tempLogger); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal configuration error: %v\n", err)
		os.Exit(1)
	}

	// --- Logger Setup ---
	// Ensure the directory for the log file exists
	logDir := filepath.Dir(config.Log.Output)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating log directory %s: %v\n", logDir, err)
		os.Exit(1)
	}

	logFile, err := os.OpenFile(config.Log.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening log file %s: %v\n", config.Log.Output, err)
		os.Exit(1)
	}
	defer logFile.Close()

	// Initialize the custom logger with configured level
	logger := utils.New(logFile, "", log.LstdFlags|log.Lshortfile, config.Log.Level)
	logger.Println("DEBUG", "--------------------------------------------------") // Log separator
	logger.Println("DEBUG", "MCP Server starting...")                             // Startup message
	logger.Printf("DEBUG", "Logging to file: %s", config.Log.Output)
	logger.Printf("DEBUG", "Log level: %s", config.Log.Level)
	logger.Printf("DEBUG", "Project root: %s", config.Project.RootPath)
	// Ping target logging removed as it's now provided by the client

	// --- Server Initialization ---
	// Use standard input and output
	stdin := os.Stdin
	stdout := os.Stdout

	// Create and run the server with configuration
	server := NewServer(stdin, stdout, logger, config)
	err = server.Run()

	// --- Shutdown ---
	if err != nil {
		// Use Fatalf which always logs and exits
		logger.Fatalf("DEBUG", "Server exited with error: %v", err)
		// fmt.Fprintf(os.Stderr, "Server exited with error: %v\n", err) // Fatalf logs and exits
		// logger.Println("DEBUG", "--------------------------------------------------") // Not reached after Fatalf
		// os.Exit(1) // Not needed, Fatalf exits
	}

	logger.Println("DEBUG", "Server exited normally.")
	logger.Println("DEBUG", "--------------------------------------------------")
}

// Helper function to create a standard MethodNotFound error response
func createMethodNotFoundResponse(id mcp.RequestID, method string, logger *utils.Logger) ([]byte, error) {
	rpcErr := mcp.NewRPCError(mcp.ErrorCodeMethodNotFound, fmt.Sprintf("Method '%s' not found", method), nil)
	responseBytes, err := mcp.MarshalErrorResponse(id, rpcErr)
	if err != nil {
		logger.Printf("DEBUG", "Error marshalling MethodNotFound error response for ID %v: %v", id, err)
		// Return a generic internal error if marshalling fails
		genericErr := mcp.NewRPCError(mcp.ErrorCodeInternalError, "Failed to marshal error response", nil)
		// We might not be able to marshal this either, but try
		responseBytes, _ = mcp.MarshalErrorResponse(id, genericErr)
		return responseBytes, fmt.Errorf("failed to marshal MethodNotFound error response: %w", err)
	}
	return responseBytes, nil
}
