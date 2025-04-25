// Package transport provides functionality for transporting messages between components.
// It includes utilities for reading, validating, and sending messages over channels.
package transport

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"sync"

	utils "sqirvy-mcp/pkg/utils"
)

// Common errors
var (
	ErrChannelClosed = errors.New("channel is closed")
	ErrReaderClosed  = errors.New("reader is closed")
)

type Transport interface {
	ReadMessages() error
	SendMessage(payload []byte) error
}

type TransportImpl struct {
	reader  io.Reader
	writer  io.Writer
	msgChan chan<- []byte
	logger  *utils.Logger
	mu      sync.Mutex
}

// NewTransport creates a new Transport instance from the provided reader, writer, message channel, and logger.
// The returned Transport instance will read messages from the reader, validate them as JSON, and send them to the channel.
// The Transport instance will also write messages from the channel to the writer.
// The logger is used to log any errors encountered when reading, validating, or sending messages.
// The mutex is used to synchronize access to the writer.
func NewTransport(reader io.Reader, writer io.Writer, msgChan chan<- []byte, logger *utils.Logger) Transport {
	return &TransportImpl{
		reader:  reader,
		writer:  writer,
		msgChan: msgChan,
		logger:  logger,
		mu:      sync.Mutex{},
	}
}

// ReadMessages reads messages from a reader and sends them to a channel.
// A message is a stream of bytes delimited by a newline character.
// The function will continue reading until the reader is closed or an error occurs.
// If the channel is closed, the function will return an error.
// If the reader is closed, the function will return an error.
// The function will log and skip empty messages and invalid JSON messages.
// If the channel is full, the message will be logged and discarded.
// Valid JSON messages will be sent to the channel if there is space available.

func (t *TransportImpl) ReadMessages() error {
	scanner := bufio.NewScanner(t.reader)

	for scanner.Scan() {
		// Get the message and trim whitespace
		msg := strings.TrimSpace(scanner.Text())

		// Skip empty messages
		if msg == "" {
			t.logger.Println(utils.LevelInfo, "Received empty message, skipping")
			continue
		}

		// Validate JSON
		var js json.RawMessage
		if err := json.Unmarshal([]byte(msg), &js); err != nil {
			t.logger.Printf(utils.LevelInfo, "Invalid JSON message received: %s, error: %v", msg, err)
			continue
		}

		// Try to send the message to the channel
		msgBytes := []byte(msg)

		// Use a defer/recover to handle potential panic from sending to a closed channel
		var sendErr error
		func() {
			defer func() {
				if r := recover(); r != nil {
					// The channel is closed if we panic on send
					sendErr = ErrChannelClosed
				}
			}()

			// Use a blocking send - will wait if channel is full
			t.msgChan <- msgBytes
		}()

		// Check if the channel is closed
		if sendErr != nil {
			return sendErr
		}
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return err
	}

	// If we get here, the reader is closed
	return ErrReaderClosed
}

// SendMessage asynchronously sends a message to the provided writer.
// It appends a newline character to the payload and writes it to the writer.
// The function returns immediately while the actual sending happens in a goroutine.
// The mutex ensures that only one goroutine can write to the writer at a time.
// Any errors that occur during writing are logged but not returned to the caller.
//
// Parameters:
//   - payload: The message bytes to send
//   - writer: The io.Writer to write the message to
//   - mu: A mutex to synchronize access to the writer (must be the same mutex used for all writes to this writer)
//   - logger: A logger to record any errors
func (t *TransportImpl) SendMessage(payload []byte) error {
	// Launch a goroutine to handle the actual sending
	var rerr error
	func(p []byte) {
		t.mu.Lock()
		defer t.mu.Unlock()

		// Append a newline to the payload
		messageWithNewline := append(p, '\n')

		// Write the payload to the writer
		_, err := t.writer.Write(messageWithNewline)
		if err != nil {
			t.logger.Printf(utils.LevelInfo, "Error writing message: %v", err)
			rerr = err
		}
	}(payload) // Pass payload as argument to avoid closure issues
	return rerr
}

// NewStdioReader returns an io.Reader that reads from os.Stdin
func NewStdioReader() io.Reader {
	reader := os.Stdin
	return reader
}

// NewStdioWriter returns an io.Writer that writes to os.Stdout
func NewStdioWriter() io.Writer {
	writer := os.Stdout
	return writer
}
