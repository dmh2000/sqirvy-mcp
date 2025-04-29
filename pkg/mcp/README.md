# MCP Package (`pkg/mcp`)

This package provides Go language types and functions for implementing the Model Context Protocol (MCP). It defines the data structures for MCP requests, responses, notifications, and errors, along with helper functions for marshaling these structures to JSON and unmarshaling JSON back into Go types.

## Functionality

*   **Type Definitions:** Defines Go structs corresponding to the various MCP message types and data structures specified in the [MCP schema](schema.json). This includes:
    *   Base JSON-RPC structures (`RPCRequest`, `RPCResponse`, `RPCError`).
    *   Core MCP messages (`Initialize`, `Ping`).
    *   Resource management messages (`ListResources`, `ReadResource`, `ListResourcesTemplates`).
    *   Prompt management messages (`ListPrompts`, `GetPrompt`).
    *   Tool management messages (`ListTools`, `CallTool`).
    *   Common data types (`Resource`, `Prompt`, `Tool`, `TextContent`, `BlobResourceContents`, etc.).
*   **Marshaling/Unmarshaling:** Provides functions to serialize Go structs into valid MCP JSON messages (requests and responses) and deserialize incoming JSON messages back into the corresponding Go structs.
    *   **Client-Side Helpers:** Functions like `MarshalInitializeRequest`, `UnmarshalInitializeResult`, `MarshalListToolsRequest`, `UnmarshalListToolsResult`, etc., are designed for use by an MCP client implementation. They handle creating request JSON and parsing response JSON.
    *   **Server-Side Helpers:** Functions like `UnmarshalInitializeRequest`, `MarshalInitializeResult`, `UnmarshalListToolsRequest`, `MarshalListToolsResult`, etc., are designed for use by an MCP server implementation. They handle parsing incoming request JSON and creating response JSON. These often include validation logic and require a logger instance.
*   **Error Handling:** Defines standard MCP error codes and provides functions (`NewRPCError`, `MarshalErrorResponse`, `UnmarshalErrorResponse`) for creating and handling JSON-RPC error responses.
*   **Testing:** Includes comprehensive unit tests (`*_test.go`) for marshaling and unmarshaling functions to ensure correctness and compliance with the expected JSON format.

## Usage

To use this package in your Go application:

1.  **Import:** Import the package into your Go files:
    ```go
    import "sqirvy-mcp/pkg/mcp"
    import "sqirvy-mcp/pkg/utils" // Often needed for server-side functions requiring a logger
    ```
2.  **Instantiate Types:** Create instances of the defined structs (e.g., `mcp.InitializeParams`, `mcp.ListToolsResult`).
3.  **Marshal/Unmarshal:**
    *   **Client:** Use `Marshal...Request` functions to create JSON requests and `Unmarshal...Result`/`Unmarshal...Response` functions to parse JSON responses from the server.
    *   **Server:** Use `Unmarshal...Request` functions to parse incoming JSON requests from the client and `Marshal...Result`/`Marshal...Response` or `MarshalErrorResponse` functions to create JSON responses. Remember to pass a `*utils.Logger` to server-side functions where required.

Refer to the specific `*.go` files (e.g., `initialize.go`, `resources.go`, `tools.go`, `prompts.go`, `error.go`) and their corresponding test files (`*_test.go`) for detailed examples of how each function and type is used.
