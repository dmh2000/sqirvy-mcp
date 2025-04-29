# Transport Package (`pkg/transport`)

This package provides an abstraction layer for sending and receiving messages, typically JSON-RPC messages used by the Model Context Protocol (MCP), over standard I/O streams or other `io.Reader`/`io.Writer` interfaces.

## Functionality

*   **Transport Interface:** Defines a `Transport` interface with methods for reading messages (`ReadMessages`) and sending messages (`SendMessage`).
*   **Implementation (`TransportImpl`):** Provides a concrete implementation of the `Transport` interface.
    *   Reads messages line by line from an `io.Reader` using a `bufio.Scanner`.
    *   Sends received messages (as byte slices) to a provided channel (`msgChan`) for processing by higher-level application logic (e.g., an MCP server).
    *   Writes outgoing messages (byte slices) to an `io.Writer`, appending a newline character as required by the protocol.
    *   Uses a mutex (`sync.Mutex`) to ensure thread-safe writes.
    *   Integrates with the `pkg/utils/logger` for logging transport activities and errors.
*   **Standard I/O Helpers:** Includes `NewStdioReader()` and `NewStdioWriter()` functions to easily create readers and writers connected to the process's standard input and standard output.
*   **Testing:** Contains unit tests (`transport_test.go`) to verify the reading and writing logic, including handling of empty messages and potential I/O errors.

## Usage

1.  **Import:** Import the package:
    ```go
    import "sqirvy-mcp/pkg/transport"
    import "sqirvy-mcp/pkg/utils" // For logger
    ```
2.  **Create:** Instantiate `TransportImpl` using `NewTransport`, providing an `io.Reader`, `io.Writer`, a channel for incoming messages, and a logger. Often, `NewStdioReader()` and `NewStdioWriter()` are used for standard I/O.
    ```go
    msgChan := make(chan []byte)
    logger := utils.New(...) // Initialize your logger
    reader := transport.NewStdioReader()
    writer := transport.NewStdioWriter()
    tp := transport.NewTransport(reader, writer, msgChan, logger)
    ```
3.  **Read:** Start reading messages in a separate goroutine:
    ```go
    go func() {
        err := tp.ReadMessages()
        if err != nil {
            logger.Printf("ERROR", "Transport read error: %v", err)
        }
        close(msgChan) // Signal end of messages
    }()
    ```
4.  **Process:** Receive messages from `msgChan` in your main application loop.
5.  **Send:** Use `tp.SendMessage(payload)` to send outgoing messages.
