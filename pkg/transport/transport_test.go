package transport

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/dmh2000/sqirvy-mcp/pkg/utils"
)

func TestReadMessages(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedMsgs  []string
		expectError   bool
		expectedError error
	}{
		{
			name:          "Valid JSON messages",
			input:         "{\"key\":\"value1\"}\n{\"key\":\"value2\"}\n",
			expectedMsgs:  []string{`{"key":"value1"}`, `{"key":"value2"}`},
			expectError:   true, // EOF is expected when the reader is closed
			expectedError: ErrReaderClosed,
		},
		{
			name:          "Empty messages are skipped",
			input:         "{\"key\":\"value1\"}\n\n{\"key\":\"value2\"}\n",
			expectedMsgs:  []string{`{"key":"value1"}`, `{"key":"value2"}`},
			expectError:   true,
			expectedError: ErrReaderClosed,
		},
		{
			name:          "Invalid JSON messages are skipped",
			input:         "{\"key\":\"value1\"}\n{invalid json}\n{\"key\":\"value2\"}\n",
			expectedMsgs:  []string{`{"key":"value1"}`, `{"key":"value2"}`},
			expectError:   true,
			expectedError: ErrReaderClosed,
		},
		{
			name:          "Whitespace is trimmed",
			input:         "  {\"key\":\"value1\"}  \n{\"key\":\"value2\"}\n",
			expectedMsgs:  []string{`{"key":"value1"}`, `{"key":"value2"}`},
			expectError:   true,
			expectedError: ErrReaderClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a reader with the test input
			reader := strings.NewReader(tt.input)

			// Create a buffered channel to receive messages
			msgChan := make(chan []byte, 10)

			// Create a logger that writes to a buffer
			var logBuf bytes.Buffer
			logger := utils.New(&logBuf, "", log.LstdFlags, utils.LevelDebug)

			transport := NewTransport(reader, nil, msgChan, logger)

			// Run the ReadMessages function in a goroutine
			errChan := make(chan error, 1)
			go func() {
				errChan <- transport.ReadMessages()
			}()

			// Collect received messages
			var receivedMsgs []string
			msgTimeout := time.After(1 * time.Second)
			errTimeout := time.After(2 * time.Second)

			// First collect all messages that are immediately available
		msgLoop:
			for {
				select {
				case msg := <-msgChan:
					receivedMsgs = append(receivedMsgs, string(msg))
				case <-msgTimeout:
					// If we timeout waiting for messages, break the loop
					break msgLoop
				}
			}

			// Now wait for the error
			var testErr error
			select {
			case testErr = <-errChan:
				// Got the error
			case <-errTimeout:
				t.Errorf("Timeout waiting for ReadMessages to return")
			}

			// Check if the error is as expected
			if tt.expectError {
				if testErr != tt.expectedError {
					t.Errorf("Expected error %v, got %v", tt.expectedError, testErr)
				}
			} else if testErr != nil {
				t.Errorf("Unexpected error: %v", testErr)
			}

			// Close the channel to clean up
			close(msgChan)

			// Check if we received all expected messages
			if len(receivedMsgs) != len(tt.expectedMsgs) {
				t.Errorf("Expected %d messages, got %d", len(tt.expectedMsgs), len(receivedMsgs))
			}

			// Check if the messages match
			for i, expected := range tt.expectedMsgs {
				if i < len(receivedMsgs) {
					if receivedMsgs[i] != expected {
						t.Errorf("Message %d: expected %q, got %q", i, expected, receivedMsgs[i])
					}
				}
			}

			// Check logs for expected content
			logOutput := logBuf.String()
			t.Logf("Log output: %s", logOutput)

			// Check for specific log messages based on the test case
			if strings.Contains(tt.input, "invalid json") && !strings.Contains(logOutput, "Invalid JSON") {
				t.Errorf("Expected log to contain 'Invalid JSON' for invalid JSON input")
			}

			if strings.Contains(tt.input, "\n\n") && !strings.Contains(logOutput, "empty message") {
				t.Errorf("Expected log to contain 'empty message' for empty line input")
			}
		})
	}
}

func TestReadMessagesChannelClosed(t *testing.T) {
	// Create a reader with a valid JSON message
	reader := strings.NewReader("{\"key\":\"value\"}\n")
	var writer strings.Builder

	// Create a channel
	msgChan := make(chan []byte, 1)

	// Create a logger
	var logBuf bytes.Buffer
	logger := utils.New(&logBuf, "", log.LstdFlags, utils.LevelDebug)

	// Create a transport
	transport := NewTransport(reader, &writer, msgChan, logger)

	// Close the channel before starting ReadMessages
	close(msgChan)

	// Call ReadMessages directly - it should immediately detect the closed channel
	err := transport.ReadMessages()

	// Check if we got the expected error
	if err != ErrChannelClosed {
		t.Errorf("Expected error %v, got %v", ErrChannelClosed, err)
	}
}

func TestReadMessagesChannelFull(t *testing.T) {

	// Create a channel with capacity 1 to test full channel behavior
	msgChan := make(chan []byte, 1)

	// Create a logger that writes to a buffer
	var logBuf bytes.Buffer
	logger := utils.New(&logBuf, "", log.LstdFlags, utils.LevelDebug)

	// Create a transport
	reader := strings.NewReader("{\"key\":\"value1\"}\n{\"key\":\"value2\"}\n{\"key\":\"value3\"}\n")
	writer := strings.Builder{}
	transport := NewTransport(reader, &writer, msgChan, logger)

	// Run the ReadMessages function in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- transport.ReadMessages()
	}()

	// Wait a moment to ensure the goroutine starts processing
	time.Sleep(100 * time.Millisecond)

	// Don't read from the channel to make it fill up
	// Wait for the function to complete
	select {
	case err := <-errChan:
		// We expect the reader closed error
		if err != ErrReaderClosed {
			t.Errorf("Expected error %v, got %v", ErrReaderClosed, err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for ReadMessages to return")
	}

	// Check logs for channel full message
	logOutput := logBuf.String()
	t.Logf("Log output: %s", logOutput)

	// We should have at least one message about channel being full
	if !strings.Contains(logOutput, "Channel is full") {
		t.Error("Expected log to contain 'Channel is full' message")
	}

	// Clean up
	close(msgChan)
}

func TestReadMessagesReaderError(t *testing.T) {
	// Create a reader that will return an error
	errReader := ErrorReader{err: io.ErrUnexpectedEOF}
	errWriter := strings.Builder{}

	// Create a channel
	msgChan := make(chan []byte, 1)

	// Create a logger
	var logBuf bytes.Buffer
	logger := utils.New(&logBuf, "", log.LstdFlags, utils.LevelDebug)

	// Create a transport
	transport := NewTransport(&errReader, &errWriter, msgChan, logger)

	// Call ReadMessages and expect the reader error
	err := transport.ReadMessages()

	// Check if we got the expected error
	if err != io.ErrUnexpectedEOF {
		t.Errorf("Expected error %v, got %v", io.ErrUnexpectedEOF, err)
	}
}

// ErrorReader is a mock reader that always returns an error
type ErrorReader struct {
	err error
}

func (r *ErrorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}

// ErrorWriter is a mock writer that can be configured to return an error
type ErrorWriter struct {
	err     error
	written []byte
}

func (w *ErrorWriter) Write(p []byte) (n int, err error) {
	if w.err != nil {
		return 0, w.err
	}
	w.written = append(w.written, p...)
	return len(p), nil
}

// TestSendMessage tests the basic functionality of the SendMessage function
func TestSendMessage(t *testing.T) {
	// Create a logger
	var logBuf bytes.Buffer
	logger := utils.New(&logBuf, "", log.LstdFlags, utils.LevelDebug)

	msg := `{"key": "value"}`
	// Test payload
	payload := []byte(msg) // JSON message

	// Call SendMessage
	writer := strings.Builder{}
	reader := strings.NewReader(msg)
	transport := NewTransport(reader, &writer, nil, logger)

	transport.SendMessage(payload)

	// Wait a short time for the goroutine to complete
	time.Sleep(100 * time.Millisecond)

	// Check the output
	expected := string(payload) + "\n" // Expect the payload with a newline
	if writer.String() != expected {
		t.Errorf("Expected output %q, got %q", expected, writer.String())
	}

	// Check that no errors were logged
	logOutput := logBuf.String()
	if strings.Contains(logOutput, "Error writing message") {
		t.Errorf("Unexpected error in log: %s", logOutput)
	}
}

// TestSendMessageError tests error handling in the SendMessage function
func TestSendMessageError(t *testing.T) {
	// Create a logger
	var logBuf bytes.Buffer
	logger := utils.New(&logBuf, "", log.LstdFlags, utils.LevelDebug)

	// Test payload
	payload := []byte(`{"key":"value"}`) // JSON message

	// Create an error writer that will return an error
	reader := ErrorReader{err: io.ErrUnexpectedEOF}
	writer := ErrorWriter{err: errors.New("write error")}
	transport := NewTransport(&reader, &writer, nil, logger)

	// Call SendMessage
	transport.SendMessage(payload)

	// Wait a short time for the goroutine to complete
	time.Sleep(100 * time.Millisecond)

	// Check that the error was logged
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "Error writing message") || !strings.Contains(logOutput, "write error") {
		t.Errorf("Expected error log, got: %s", logOutput)
	}
}

// TestSendMessageConcurrent tests that the mutex properly synchronizes access to the writer
func TestSendMessageConcurrent(t *testing.T) {
	// Create a logger
	var logBuf bytes.Buffer
	logger := utils.New(&logBuf, "", log.LstdFlags, utils.LevelDebug)

	// Call SendMessage

	// Number of messages to send
	numMessages := 10

	writer := strings.Builder{}
	reader := strings.NewReader("")
	transport := NewTransport(reader, &writer, nil, logger)

	// Send multiple messages concurrently
	for i := range numMessages {
		payload := fmt.Appendf(nil, `{"index":%d}`, i)
		transport.SendMessage(payload)
	}

	// Wait for all goroutines to complete
	time.Sleep(500 * time.Millisecond)

	// Check the output - we should have numMessages lines
	output := writer.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != numMessages {
		t.Errorf("Expected %d lines, got %d", numMessages, len(lines))
	}

	// Check that each line is a valid JSON message
	for i, line := range lines {
		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			t.Errorf("Line %d is not valid JSON: %s, error: %v", i, line, err)
		}
	}

	// Check that no errors were logged
	logOutput := logBuf.String()
	if strings.Contains(logOutput, "Error writing message") {
		t.Errorf("Unexpected error in log: %s", logOutput)
	}
}
