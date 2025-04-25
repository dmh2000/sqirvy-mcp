package transport

import (
	"os"
	"testing"
)

// TestNewStdioReader verifies that NewStdioReader returns os.Stdin.
func TestNewStdioReader(t *testing.T) {
	reader := NewStdioReader()

	// Check if the returned reader is os.Stdin
	if reader != os.Stdin {
		t.Errorf("NewStdioReader() did not return os.Stdin")
	}

	// Optional: Check if it's assignable to the expected type (*os.File)
	if _, ok := reader.(*os.File); !ok {
		t.Errorf("NewStdioReader() returned type %T, expected *os.File", reader)
	}
}

// TestNewStdioWriter verifies that NewStdioWriter returns os.Stdout.
func TestNewStdioWriter(t *testing.T) {
	writer := NewStdioWriter()

	// Check if the returned writer is os.Stdout
	if writer != os.Stdout {
		t.Errorf("NewStdioWriter() did not return os.Stdout")
	}

	// Optional: Check if it's assignable to the expected type (*os.File)
	if _, ok := writer.(*os.File); !ok {
		t.Errorf("NewStdioWriter() returned type %T, expected *os.File", writer)
	}
}
