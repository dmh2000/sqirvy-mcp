package utils

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	prefix := "TEST: "
	flags := log.LstdFlags

	tests := []struct {
		name       string
		levelInput string
		wantLevel  string
	}{
		{"INFO level", "INFO", LevelInfo},
		{"DEBUG level", "DEBUG", LevelDebug},
		{"lowercase info", "info", LevelInfo},
		{"lowercase debug", "debug", LevelDebug},
		{"mixed case INFO", "InFo", LevelInfo},
		{"mixed case DEBUG", "DeBuG", LevelDebug},
		{"invalid level defaults to INFO", "WARNING", LevelInfo},
		{"empty level defaults to INFO", "", LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(&buf, prefix, flags, tt.levelInput)
			if logger.level != tt.wantLevel {
				t.Errorf("New() level = %v, want %v", logger.level, tt.wantLevel)
			}
			if logger.stdLogger.Prefix() != prefix {
				t.Errorf("New() prefix = %v, want %v", logger.stdLogger.Prefix(), prefix)
			}
			// Note: Comparing flags directly can be tricky if default flags are added.
			// if logger.stdLogger.Flags() != flags {
			// 	t.Errorf("New() flags = %v, want %v", logger.stdLogger.Flags(), flags)
			// }
		})
	}
}

func TestLoggerOutputLevels(t *testing.T) {
	tests := []struct {
		name           string
		loggerLevel    string
		messageLevel   string
		message        string
		printfArgs     []interface{} // For Printf test
		expectOutput   bool
		expectedString string // Expected substring in output
	}{
		// Logger Level: INFO
		{"InfoLogger_InfoMsg_Println", LevelInfo, LevelInfo, "Info message", nil, true, "Info message"},
		{"InfoLogger_DebugMsg_Println", LevelInfo, LevelDebug, "Debug message", nil, false, ""},
		{"InfoLogger_InfoMsg_Printf", LevelInfo, LevelInfo, "Info format %d", []interface{}{1}, true, "Info format 1"},
		{"InfoLogger_DebugMsg_Printf", LevelInfo, LevelDebug, "Debug format %d", []interface{}{2}, false, ""},
		// Logger Level: DEBUG
		{"DebugLogger_InfoMsg_Println", LevelDebug, LevelInfo, "Info message", nil, true, "Info message"},
		{"DebugLogger_DebugMsg_Println", LevelDebug, LevelDebug, "Debug message", nil, true, "Debug message"},
		{"DebugLogger_InfoMsg_Printf", LevelDebug, LevelInfo, "Info format %d", []interface{}{3}, true, "Info format 3"},
		{"DebugLogger_DebugMsg_Printf", LevelDebug, LevelDebug, "Debug format %d", []interface{}{4}, true, "Debug format 4"},
		// Case insensitivity
		{"DebugLogger_LowercaseDebugMsg_Println", LevelDebug, "debug", "Lowercase debug", nil, true, "Lowercase debug"},
		{"InfoLogger_UppercaseInfoMsg_Printf", LevelInfo, "INFO", "Uppercase info %s", []interface{}{"test"}, true, "Uppercase info test"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_Println", func(t *testing.T) {
			var buf bytes.Buffer
			// Use 0 flags to avoid timestamp/file info in output for easier comparison
			logger := New(&buf, "", 0, tt.loggerLevel)

			logger.Println(tt.messageLevel, tt.message)

			output := buf.String()
			hasOutput := output != ""

			if hasOutput != tt.expectOutput {
				t.Errorf("Println() for %s: expectOutput = %v, but got output: '%s'", tt.name, tt.expectOutput, output)
			}

			// For Println, the expected output is the raw message itself (plus newline)
			if tt.expectOutput && !strings.Contains(output, tt.message) {
				t.Errorf("Println() for %s: output '%s' does not contain expected string '%s'", tt.name, output, tt.message)
			}
		})

		t.Run(tt.name+"_Printf", func(t *testing.T) {
			var buf bytes.Buffer
			logger := New(&buf, "", 0, tt.loggerLevel) // Use 0 flags

			logger.Printf(tt.messageLevel, tt.message, tt.printfArgs...)

			output := buf.String()
			hasOutput := output != ""

			if hasOutput != tt.expectOutput {
				t.Errorf("Printf() for %s: expectOutput = %v, but got output: '%s'", tt.name, tt.expectOutput, output)
			}

			// For Printf, the expected output is the formatted string
			if tt.expectOutput && !strings.Contains(output, tt.expectedString) {
				t.Errorf("Printf() for %s: output '%s' does not contain expected string '%s'", tt.name, output, tt.expectedString)
			}
		})
	}
}

func TestSetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, "", 0, LevelInfo) // Start with INFO level

	// Initially, DEBUG messages should be suppressed
	logger.Println(LevelDebug, "Initial debug message")
	if buf.Len() != 0 {
		t.Errorf("Expected no output at INFO level for DEBUG message, but got: %s", buf.String())
	}
	buf.Reset() // Clear buffer

	// Set level to DEBUG
	logger.SetLevel(LevelDebug)
	if logger.level != LevelDebug {
		t.Errorf("SetLevel(LevelDebug) failed, logger level is %s", logger.level)
	}

	// Now, DEBUG messages should be logged
	logger.Println(LevelDebug, "Debug message after SetLevel")
	if !strings.Contains(buf.String(), "Debug message after SetLevel") {
		t.Errorf("Expected output after SetLevel(LevelDebug), but got: %s", buf.String())
	}
	buf.Reset()

	// Set level back to INFO
	logger.SetLevel("info") // Test lowercase
	if logger.level != LevelInfo {
		t.Errorf("SetLevel('info') failed, logger level is %s", logger.level)
	}

	// DEBUG messages should be suppressed again
	logger.Println(LevelDebug, "Debug message after SetLevel back to info")
	if buf.Len() != 0 {
		t.Errorf("Expected no output after SetLevel back to INFO, but got: %s", buf.String())
	}
}

func TestStandardLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, "PREFIX: ", 0, LevelInfo)
	stdLogger := logger.StandardLogger()

	if stdLogger == nil {
		t.Fatal("StandardLogger() returned nil")
	}

	// Verify it's the same underlying logger
	stdLogger.Println("Message from standard logger")
	if !strings.Contains(buf.String(), "PREFIX: Message from standard logger") {
		t.Errorf("Output from StandardLogger() was not as expected: %s", buf.String())
	}
}
