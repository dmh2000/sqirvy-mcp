package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings" // Added for ToUpper
)

// Define valid log level strings
const (
	LevelDebug = "DEBUG" // More verbose level - logs both DEBUG and INFO messages
	LevelInfo  = "INFO"  // Less verbose level - logs only INFO messages
)

// Define numeric values for log levels (higher is more verbose)
var logLevelValues = map[string]int{
	LevelDebug: 1, // Lower value = higher priority, so DEBUG (1) includes INFO (2)
	LevelInfo:  2,
}

// Logger wraps the standard Go logger to provide level-based logging.
type Logger struct {
	stdLogger *log.Logger
	level     string // Store level as a string ("INFO" or "DEBUG")
}

// New creates a new Logger instance.
// It takes an output writer, prefix string, standard log flags, and the minimum level string ("INFO" or "DEBUG") to output.
// Defaults to "INFO" if an invalid level string is provided.
func New(out io.Writer, prefix string, flag int, level string) *Logger {
	normalizedLevel := strings.ToUpper(level)
	// Validate the level - only accept defined levels
	if _, ok := logLevelValues[normalizedLevel]; !ok {
		normalizedLevel = LevelInfo // Default to INFO if invalid
	}
	return &Logger{
		stdLogger: log.New(out, prefix, flag),
		level:     normalizedLevel,
	}
}

// SetLevel changes the minimum logging level for the logger using a string ("INFO" or "DEBUG").
// Defaults to "INFO" if an invalid level string is provided.
func (l *Logger) SetLevel(level string) {
	normalizedLevel := strings.ToUpper(level)
	// Validate the level - only accept defined levels
	if _, ok := logLevelValues[normalizedLevel]; !ok {
		normalizedLevel = LevelInfo // Default to INFO if invalid
	}
	l.level = normalizedLevel
}

// shouldLog checks if a message with the given level string should be logged.
// It implements hierarchical logging where setting a level means "log this level and above".
func (l *Logger) shouldLog(messageLevel string) bool {
	// Normalize case for comparison
	normalizedMessageLevel := strings.ToUpper(messageLevel)

	// Get numeric values for the logger level and message level
	loggerLevelValue, loggerOk := logLevelValues[l.level]
	messageLevelValue, messageOk := logLevelValues[normalizedMessageLevel]

	// If either level is invalid, use safe defaults
	if !loggerOk {
		loggerLevelValue = logLevelValues[LevelInfo] // Default to INFO
	}
	if !messageOk {
		return false // Don't log messages with invalid levels
	}

	// Log the message if its level is greater than or equal to the logger's level
	// (lower numeric value = higher priority, e.g., DEBUG=1 is higher priority than INFO=2)
	// This means if logger is set to DEBUG (1), it will log both DEBUG (1) and INFO (2)
	// If logger is set to INFO (2), it will only log INFO (2) messages
	return messageLevelValue >= loggerLevelValue
}

// Printf logs a formatted string if the message level is appropriate based on the logger's level.
// The first argument is the level string ("INFO" or "DEBUG").
// Messages will be logged if their level is equal to or lower priority than the logger's level.
// For example, if the logger is set to DEBUG level, it will log both DEBUG and INFO messages.
// If the logger is set to INFO level, it will only log INFO messages.
func (l *Logger) Printf(level string, format string, v ...interface{}) {
	if l.shouldLog(level) {
		// Call Output with depth 3 to capture the caller's file/line
		l.stdLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// Println logs a line if the message level is appropriate based on the logger's level.
// The first argument is the level string ("INFO" or "DEBUG").
// Messages will be logged if their level is equal to or lower priority than the logger's level.
// For example, if the logger is set to DEBUG level, it will log both DEBUG and INFO messages.
// If the logger is set to INFO level, it will only log INFO messages.
func (l *Logger) Println(level string, v ...interface{}) {
	if l.shouldLog(level) {
		// Call Output with depth 3 to capture the caller's file/line
		l.stdLogger.Output(2, fmt.Sprintln(v...))
	}
}

// Fatalf logs a formatted string and then calls os.Exit(1), regardless of the configured log level.
// The first argument is the level string ("INFO" or "DEBUG"), but it's mainly for consistency.
// Fatal messages are always output.
func (l *Logger) Fatalf(level string, format string, v ...interface{}) {
	// Fatal messages are always logged, regardless of level setting.
	l.stdLogger.Output(2, fmt.Sprintf(format, v...)) // Use Output with depth 3 to capture the caller's file/line
	os.Exit(1)
}

// Fatalln logs a line and then calls os.Exit(1), regardless of the configured log level.
// The first argument is the level string ("INFO" or "DEBUG"), but it's mainly for consistency.
// Fatal messages are always output.
func (l *Logger) Fatalln(level string, v ...interface{}) {
	// Fatal messages are always logged, regardless of level setting.
	l.stdLogger.Output(2, fmt.Sprintln(v...)) // Use Output with depth 3 to capture the caller's file/line
	os.Exit(1)
}

// StandardLogger returns the underlying standard log.Logger instance.
// This can be useful if direct access to the standard logger is needed.
func (l *Logger) StandardLogger() *log.Logger {
	return l.stdLogger
}
