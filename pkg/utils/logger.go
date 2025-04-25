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
	LevelDebug   = "DEBUG"   // Most verbose level - logs DEBUG, INFO, WARNING, and ERROR messages
	LevelInfo    = "INFO"    // Logs INFO, WARNING, and ERROR messages
	LevelWarning = "WARNING" // Logs WARNING and ERROR messages
	LevelError   = "ERROR"   // Least verbose level - logs only ERROR messages
)

// Define numeric values for log levels (lower value = higher priority/more verbose)
var logLevelValues = map[string]int{
	LevelDebug:   1,
	LevelInfo:    2,
	LevelWarning: 3,
	LevelError:   4,
}

// Logger wraps the standard Go logger to provide level-based logging.
type Logger struct {
	stdLogger *log.Logger
	level     string // Store level as a string ("INFO" or "DEBUG")
}

// New creates a new Logger instance.
// It takes an output writer, prefix string, standard log flags, and the minimum level string ("DEBUG", "INFO", "WARNING", or "ERROR") to output.
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

// SetLevel changes the minimum logging level for the logger using a string ("DEBUG", "INFO", "WARNING", or "ERROR").
// Defaults to "INFO" if an invalid level string is provided.
func (l *Logger) SetLevel(level string) {
	normalizedLevel := strings.ToUpper(level)
	// Validate the level - only accept defined levels
	if _, ok := logLevelValues[normalizedLevel]; !ok {
		normalizedLevel = LevelInfo // Default to INFO if invalid
	}
	l.level = normalizedLevel
}

// shouldLog checks if a message with the given level string should be logged based on the logger's current level.
// Logging is hierarchical: DEBUG logs everything, INFO logs INFO/WARNING/ERROR, WARNING logs WARNING/ERROR, ERROR logs only ERROR.
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

	// Log the message if its numeric level value is greater than or equal to the logger's numeric level value.
	// Example:
	// Logger Level | Message Level | Numeric Check | Logged?
	// -------------|---------------|---------------|--------
	// DEBUG (1)    | DEBUG (1)     | 1 >= 1        | Yes
	// DEBUG (1)    | INFO (2)      | 2 >= 1        | Yes
	// DEBUG (1)    | WARNING (3)   | 3 >= 1        | Yes
	// DEBUG (1)    | ERROR (4)     | 4 >= 1        | Yes
	// INFO (2)     | DEBUG (1)     | 1 >= 2        | No
	// INFO (2)     | INFO (2)      | 2 >= 2        | Yes
	// INFO (2)     | WARNING (3)   | 3 >= 2        | Yes
	// INFO (2)     | ERROR (4)     | 4 >= 2        | Yes
	// WARNING (3)  | DEBUG (1)     | 1 >= 3        | No
	// WARNING (3)  | INFO (2)      | 2 >= 3        | No
	// WARNING (3)  | WARNING (3)   | 3 >= 3        | Yes
	// WARNING (3)  | ERROR (4)     | 4 >= 3        | Yes
	// ERROR (4)    | DEBUG (1)     | 1 >= 4        | No
	// ERROR (4)    | INFO (2)      | 2 >= 4        | No
	// ERROR (4)    | WARNING (3)   | 3 >= 4        | No
	// ERROR (4)    | ERROR (4)     | 4 >= 4        | Yes
	return messageLevelValue >= loggerLevelValue
}

// Printf logs a formatted string if the message level is appropriate based on the logger's level.
// The first argument is the level string ("DEBUG", "INFO", "WARNING", or "ERROR").
// See shouldLog for details on which levels are logged.
func (l *Logger) Printf(level string, format string, v ...interface{}) {
	if l.shouldLog(level) {
		// Call Output with depth 2 to capture the caller's file/line correctly
		l.stdLogger.Output(2, fmt.Sprintf(format, v...))
	}
}

// Println logs a line if the message level is appropriate based on the logger's level.
// The first argument is the level string ("DEBUG", "INFO", "WARNING", or "ERROR").
// See shouldLog for details on which levels are logged.
func (l *Logger) Println(level string, v ...interface{}) {
	if l.shouldLog(level) {
		// Call Output with depth 2 to capture the caller's file/line correctly
		l.stdLogger.Output(2, fmt.Sprintln(v...))
	}
}

// Fatalf logs a formatted string and then calls os.Exit(1), regardless of the configured log level.
// The first argument is the level string ("DEBUG", "INFO", "WARNING", or "ERROR"), but it's mainly for consistency.
// Fatal messages are always output.
func (l *Logger) Fatalf(level string, format string, v ...interface{}) {
	// Fatal messages are always logged, regardless of level setting.
	l.stdLogger.Output(2, fmt.Sprintf(format, v...)) // Use Output with depth 2 to capture the caller's file/line
	os.Exit(1)
}

// Fatalln logs a line and then calls os.Exit(1), regardless of the configured log level.
// The first argument is the level string ("DEBUG", "INFO", "WARNING", or "ERROR"), but it's mainly for consistency.
// Fatal messages are always output.
func (l *Logger) Fatalln(level string, v ...interface{}) {
	// Fatal messages are always logged, regardless of level setting.
	l.stdLogger.Output(2, fmt.Sprintln(v...)) // Use Output with depth 2 to capture the caller's file/line
	os.Exit(1)
}

// StandardLogger returns the underlying standard log.Logger instance.
// This can be useful if direct access to the standard logger is needed.
func (l *Logger) StandardLogger() *log.Logger {
	return l.stdLogger
}
