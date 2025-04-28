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
		{"WARNING level", "WARNING", LevelWarning},
		{"ERROR level", "ERROR", LevelError},
		{"lowercase info", "info", LevelInfo},
		{"lowercase debug", "debug", LevelDebug},
		{"lowercase warning", "warning", LevelWarning},
		{"lowercase error", "error", LevelError},
		{"mixed case INFO", "InFo", LevelInfo},
		{"mixed case DEBUG", "DeBuG", LevelDebug},
		{"mixed case WARNING", "WaRnInG", LevelWarning},
		{"mixed case ERROR", "ErRoR", LevelError},
		{"invalid level defaults to INFO", "INVALID", LevelInfo},
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
		{"InfoLogger_WarningMsg_Println", LevelInfo, LevelWarning, "Warning message", nil, true, "Warning message"},
		{"InfoLogger_ErrorMsg_Println", LevelInfo, LevelError, "Error message", nil, true, "Error message"},
		{"InfoLogger_InfoMsg_Printf", LevelInfo, LevelInfo, "Info format %d", []interface{}{1}, true, "Info format 1"},
		{"InfoLogger_DebugMsg_Printf", LevelInfo, LevelDebug, "Debug format %d", []interface{}{2}, false, ""},
		{"InfoLogger_WarningMsg_Printf", LevelInfo, LevelWarning, "Warning format %d", []interface{}{3}, true, "Warning format 3"},
		{"InfoLogger_ErrorMsg_Printf", LevelInfo, LevelError, "Error format %d", []interface{}{4}, true, "Error format 4"},

		// Logger Level: DEBUG
		{"DebugLogger_InfoMsg_Println", LevelDebug, LevelInfo, "Info message", nil, true, "Info message"},
		{"DebugLogger_DebugMsg_Println", LevelDebug, LevelDebug, "Debug message", nil, true, "Debug message"},
		{"DebugLogger_WarningMsg_Println", LevelDebug, LevelWarning, "Warning message", nil, true, "Warning message"},
		{"DebugLogger_ErrorMsg_Println", LevelDebug, LevelError, "Error message", nil, true, "Error message"},
		{"DebugLogger_InfoMsg_Printf", LevelDebug, LevelInfo, "Info format %d", []interface{}{5}, true, "Info format 5"},
		{"DebugLogger_DebugMsg_Printf", LevelDebug, LevelDebug, "Debug format %d", []interface{}{6}, true, "Debug format 6"},
		{"DebugLogger_WarningMsg_Printf", LevelDebug, LevelWarning, "Warning format %d", []interface{}{7}, true, "Warning format 7"},
		{"DebugLogger_ErrorMsg_Printf", LevelDebug, LevelError, "Error format %d", []interface{}{8}, true, "Error format 8"},

		// Logger Level: WARNING
		{"WarningLogger_InfoMsg_Println", LevelWarning, LevelInfo, "Info message", nil, false, ""},
		{"WarningLogger_DebugMsg_Println", LevelWarning, LevelDebug, "Debug message", nil, false, ""},
		{"WarningLogger_WarningMsg_Println", LevelWarning, LevelWarning, "Warning message", nil, true, "Warning message"},
		{"WarningLogger_ErrorMsg_Println", LevelWarning, LevelError, "Error message", nil, true, "Error message"},
		{"WarningLogger_InfoMsg_Printf", LevelWarning, LevelInfo, "Info format %d", []interface{}{9}, false, ""},
		{"WarningLogger_DebugMsg_Printf", LevelWarning, LevelDebug, "Debug format %d", []interface{}{10}, false, ""},
		{"WarningLogger_WarningMsg_Printf", LevelWarning, LevelWarning, "Warning format %d", []interface{}{11}, true, "Warning format 11"},
		{"WarningLogger_ErrorMsg_Printf", LevelWarning, LevelError, "Error format %d", []interface{}{12}, true, "Error format 12"},

		// Logger Level: ERROR
		{"ErrorLogger_InfoMsg_Println", LevelError, LevelInfo, "Info message", nil, false, ""},
		{"ErrorLogger_DebugMsg_Println", LevelError, LevelDebug, "Debug message", nil, false, ""},
		{"ErrorLogger_WarningMsg_Println", LevelError, LevelWarning, "Warning message", nil, false, ""},
		{"ErrorLogger_ErrorMsg_Println", LevelError, LevelError, "Error message", nil, true, "Error message"},
		{"ErrorLogger_InfoMsg_Printf", LevelError, LevelInfo, "Info format %d", []interface{}{13}, false, ""},
		{"ErrorLogger_DebugMsg_Printf", LevelError, LevelDebug, "Debug format %d", []interface{}{14}, false, ""},
		{"ErrorLogger_WarningMsg_Printf", LevelError, LevelWarning, "Warning format %d", []interface{}{15}, false, ""},
		{"ErrorLogger_ErrorMsg_Printf", LevelError, LevelError, "Error format %d", []interface{}{16}, true, "Error format 16"},

		// Case insensitivity
		{"DebugLogger_LowercaseDebugMsg_Println", LevelDebug, "debug", "Lowercase debug", nil, true, "Lowercase debug"},
		{"InfoLogger_UppercaseInfoMsg_Printf", LevelInfo, "INFO", "Uppercase info %s", []interface{}{"test"}, true, "Uppercase info test"},
		{"WarningLogger_MixedCaseWarningMsg_Println", LevelWarning, "WaRnInG", "Mixed case warning", nil, true, "Mixed case warning"},
		{"ErrorLogger_LowercaseErrorMsg_Printf", LevelError, "error", "Lowercase error %s", []interface{}{"test"}, true, "Lowercase error test"},
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

	// Test INFO level logs INFO, WARNING, and ERROR messages
	logger.Println(LevelInfo, "Info message at INFO level")
	if !strings.Contains(buf.String(), "Info message at INFO level") {
		t.Errorf("Expected INFO message to be logged at INFO level, but got: %s", buf.String())
	}
	buf.Reset()

	logger.Println(LevelWarning, "Warning message at INFO level")
	if !strings.Contains(buf.String(), "Warning message at INFO level") {
		t.Errorf("Expected WARNING message to be logged at INFO level, but got: %s", buf.String())
	}
	buf.Reset()

	logger.Println(LevelError, "Error message at INFO level")
	if !strings.Contains(buf.String(), "Error message at INFO level") {
		t.Errorf("Expected ERROR message to be logged at INFO level, but got: %s", buf.String())
	}
	buf.Reset()

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

	// Set level to WARNING
	logger.SetLevel(LevelWarning)
	if logger.level != LevelWarning {
		t.Errorf("SetLevel(LevelWarning) failed, logger level is %s", logger.level)
	}

	// DEBUG and INFO messages should be suppressed at WARNING level
	logger.Println(LevelDebug, "Debug message at WARNING level")
	if buf.Len() != 0 {
		t.Errorf("Expected no output for DEBUG message at WARNING level, but got: %s", buf.String())
	}
	buf.Reset()

	logger.Println(LevelInfo, "Info message at WARNING level")
	if buf.Len() != 0 {
		t.Errorf("Expected no output for INFO message at WARNING level, but got: %s", buf.String())
	}
	buf.Reset()

	// WARNING messages should be logged at WARNING level
	logger.Println(LevelWarning, "Warning message at WARNING level")
	if !strings.Contains(buf.String(), "Warning message at WARNING level") {
		t.Errorf("Expected output for WARNING message at WARNING level, but got: %s", buf.String())
	}
	buf.Reset()

	// ERROR messages should be logged at WARNING level
	logger.Println(LevelError, "Error message at WARNING level")
	if !strings.Contains(buf.String(), "Error message at WARNING level") {
		t.Errorf("Expected output for ERROR message at WARNING level, but got: %s", buf.String())
	}
	buf.Reset()

	// Set level to ERROR
	logger.SetLevel(LevelError)
	if logger.level != LevelError {
		t.Errorf("SetLevel(LevelError) failed, logger level is %s", logger.level)
	}

	// DEBUG, INFO, and WARNING messages should be suppressed at ERROR level
	logger.Println(LevelDebug, "Debug message at ERROR level")
	if buf.Len() != 0 {
		t.Errorf("Expected no output for DEBUG message at ERROR level, but got: %s", buf.String())
	}
	buf.Reset()

	logger.Println(LevelInfo, "Info message at ERROR level")
	if buf.Len() != 0 {
		t.Errorf("Expected no output for INFO message at ERROR level, but got: %s", buf.String())
	}
	buf.Reset()

	logger.Println(LevelWarning, "Warning message at ERROR level")
	if buf.Len() != 0 {
		t.Errorf("Expected no output for WARNING message at ERROR level, but got: %s", buf.String())
	}
	buf.Reset()

	// ERROR messages should be logged at ERROR level
	logger.Println(LevelError, "Error message at ERROR level")
	if !strings.Contains(buf.String(), "Error message at ERROR level") {
		t.Errorf("Expected output for ERROR message at ERROR level, but got: %s", buf.String())
	}
	buf.Reset()

	// Test case insensitivity with SetLevel
	logger.SetLevel("info") // Test lowercase
	if logger.level != LevelInfo {
		t.Errorf("SetLevel('info') failed, logger level is %s", logger.level)
	}

	// DEBUG messages should be suppressed again
	logger.Println(LevelDebug, "Debug message after SetLevel back to info")
	if buf.Len() != 0 {
		t.Errorf("Expected no output after SetLevel back to INFO, but got: %s", buf.String())
	}

	// Test mixed case
	logger.SetLevel("WaRnInG")
	if logger.level != LevelWarning {
		t.Errorf("SetLevel('WaRnInG') failed, logger level is %s", logger.level)
	}

	// Test invalid level defaults to INFO
	logger.SetLevel("INVALID_LEVEL")
	if logger.level != LevelInfo {
		t.Errorf("SetLevel with invalid level did not default to INFO, got: %s", logger.level)
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
