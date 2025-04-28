package resources

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings" // Added for HasPrefix and TrimPrefix

	utils "sqirvy-mcp/pkg/utils" // Import the custom logger
)

// GetProjectRootPath returns the project root path from the server configuration.
// This is defined as a function to allow for configuration-based path setting.
var GetProjectRootPath func() string

// ReadFileResource reads the content of a file specified by a file:// URI.
// It returns the content as bytes, the determined MIME type, and any error.
func ReadFileResource(uri string, logger *utils.Logger) ([]byte, string, error) {
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return nil, "", fmt.Errorf("invalid URI format: %w", err)
	}

	if parsedURI.Scheme != "file" {
		return nil, "", fmt.Errorf("unsupported URI scheme: %s", parsedURI.Scheme)
	}

	// Convert file URI path to a system path.
	// Handle potential differences in path separators and encoding.
	// For file://hostname/path, Host is usually empty or localhost on Unix-like systems.
	// For file:///path, Path starts with /.
	filePath := parsedURI.Path
	if parsedURI.Host != "" && parsedURI.Host != "localhost" {
		// Handle UNC paths if necessary, though less common for typical file URIs
		// For simplicity, we'll assume standard file paths here.
		logger.Printf("DEBUG", "Warning: file URI host '%s' ignored, treating path as '%s'", parsedURI.Host, filePath)
	}

	// Use the configured project root path
	projectRoot := filepath.Clean(GetProjectRootPath())
	logger.Printf("DEBUG", "Using configured project root directory: %s", projectRoot)

	// Treat the URI path as relative to the project root.
	// Strip leading '/' from the URI path.
	relativePath := strings.TrimPrefix(parsedURI.Path, "/")

	// Join the project root with the relative path and clean it.
	filePath = filepath.Join(projectRoot, relativePath)
	filePath = filepath.Clean(filePath) // Clean the combined path

	// Security Check: Ensure the final path is still within the project root.
	// This helps prevent path traversal attacks (e.g., file:///../outside_project).
	if !strings.HasPrefix(filePath, projectRoot) {
		logger.Printf("DEBUG", "Security Alert: Attempt to access file outside project root. Requested URI: %s, Resolved Path: %s", uri, filePath)
		return nil, "", fmt.Errorf("permission denied: cannot access files outside project root")
	}

	logger.Printf("DEBUG", "Attempting to read file relative to project root: %s", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", fmt.Errorf("file not found: %s", filePath)
		}
		if os.IsPermission(err) {
			return nil, "", fmt.Errorf("permission denied reading file: %s", filePath)
		}
		return nil, "", fmt.Errorf("error opening file %s: %w", filePath, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, "", fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	// Basic MIME type detection (can be improved with libraries like net/http.DetectContentType)
	// For now, assume text/plain for simplicity.
	mimeType := "text/plain"
	// Example using http.DetectContentType (requires importing "net/http")
	// if len(content) > 0 {
	//     mimeType = http.DetectContentType(content)
	// }
	// logger.Printf("Detected MIME type for %s: %s", filePath, mimeType)

	return content, mimeType, nil
}
