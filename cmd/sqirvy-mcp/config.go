package main

import (
	"fmt"
	"os"
	"path/filepath"

	utils "sqirvy-mcp/pkg/utils"

	"gopkg.in/yaml.v3"
)

// Config holds the configuration for the MCP server
type Config struct {
	// Logging configuration
	Log struct {
		Level  string `yaml:"level"`  // Log level (DEBUG, INFO)
		Output string `yaml:"output"` // Path to log file
	} `yaml:"log"`

	// Project configuration
	Project struct {
		RootPath string `yaml:"rootPath"` // Root path for file resources
	} `yaml:"project"`

	// Tools configuration
	Tools struct {
		// Note: Ping target has been removed as it's now provided by the client
	} `yaml:"tools"`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	config := &Config{}

	// Default logging configuration
	config.Log.Level = utils.LevelDebug
	config.Log.Output = "mcp-server.log"

	// Default project configuration
	// Try to use current working directory as default project root
	cwd, err := os.Getwd()
	if err == nil {
		config.Project.RootPath = cwd
	} else {
		// Fallback to a reasonable default if we can't get the current directory
		config.Project.RootPath = "."
	}

	// Default tools configuration is empty now

	return config
}

// Configuration file constants
const (
	defaultConfigFileName = ".mcp-server"
	configDirName         = "mcp-server"
)

// ValidateConfig validates the configuration values
// Returns an error if any validation fails
func ValidateConfig(config *Config, logger *utils.Logger) error {
	// Ping target validation has been removed as it's now provided by the client

	// Add more validations here as needed

	return nil
}

// LoadConfig loads the configuration from a YAML file based on the following priority:
// 1. If configPath is provided, use that file
// 2. Look for the config file in the current working directory
// 3. Look for the config file in $HOME/.config/mcp-server/
// If no configuration file is found, it returns the default configuration
func LoadConfig(configPath string, logger *utils.Logger) (*Config, error) {
	// Start with default configuration
	config := DefaultConfig()

	// List of paths to try, in order of priority
	pathsToTry := []string{}

	// 1. If config path is provided, use that file
	if configPath != "" {
		pathsToTry = append(pathsToTry, configPath)
	} else {
		// 2. Try current working directory
		cwd, err := os.Getwd()
		if err == nil {
			pathsToTry = append(pathsToTry, filepath.Join(cwd, defaultConfigFileName))
		}

		// 3. Try $HOME/.config/mcp-server/
		homeDir, err := os.UserHomeDir()
		if err == nil {
			pathsToTry = append(pathsToTry, filepath.Join(homeDir, ".config", configDirName, defaultConfigFileName))
		}
	}

	// Try each path in order
	var lastErr error
	for _, path := range pathsToTry {
		// Check if the file exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if logger != nil {
				logger.Printf("DEBUG", "Configuration file %s not found, trying next location", path)
			}
			continue
		}

		// Try to load the configuration
		data, err := os.ReadFile(path)
		if err != nil {
			lastErr = fmt.Errorf("error reading configuration file %s: %w", path, err)
			if logger != nil {
				logger.Printf("DEBUG", "Error reading configuration file: %v", lastErr)
			}
			continue
		}

		// Parse the YAML
		if err := yaml.Unmarshal(data, config); err != nil {
			lastErr = fmt.Errorf("error parsing configuration file %s: %w", path, err)
			if logger != nil {
				logger.Printf("DEBUG", "Error parsing configuration file: %v", lastErr)
			}
			continue
		}

		// Validate the configuration
		if err := ValidateConfig(config, logger); err != nil {
			lastErr = fmt.Errorf("error validating configuration from %s: %w", path, err)
			if logger != nil {
				logger.Printf("DEBUG", "Error validating configuration: %v", lastErr)
			}
			return nil, lastErr
		}

		// Successfully loaded and validated the configuration
		if logger != nil {
			logger.Printf("DEBUG", "Loaded and validated configuration from %s", path)
		}
		return config, nil
	}

	// If we got here, we couldn't load any configuration file
	if len(pathsToTry) > 0 && lastErr != nil {
		// Return the last error if we had one
		return config, lastErr
	}

	// No config file found, but that's not an error - we'll use defaults
	if logger != nil {
		logger.Printf("DEBUG", "No configuration file found, using defaults")
	}

	// Validate the default configuration
	if err := ValidateConfig(config, logger); err != nil {
		if logger != nil {
			logger.Printf("DEBUG", "Error validating default configuration: %v", err)
		}
		return nil, err
	}

	return config, nil
}

// SaveConfig saves the configuration to a YAML file
func SaveConfig(config *Config, configPath string) error {
	// Create the directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating configuration directory: %w", err)
	}

	// Marshal the configuration to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("error marshalling configuration: %w", err)
	}

	// Write the configuration file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("error writing configuration file: %w", err)
	}

	return nil
}
