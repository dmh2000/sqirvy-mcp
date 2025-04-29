# MCP Package (`pkg/mcp`)

This package provides Go language types and functions for implementing the Model Context Protocol (MCP). It defines the data structures for MCP requests, responses, notifications, and errors, along with helper functions for marshaling these structures to JSON and unmarshaling JSON back into Go types.

## Functionality

This package provides Go types and functions for MCP message creation (marshaling) and parsing (unmarshaling), facilitating communication between MCP clients and servers. It defines structs for all standard MCP messages and offers helper functions tailored for both client and server implementations.

### Client

Helper functions designed for use within an MCP client application.

#### Initialize

*   `MarshalInitializeRequest(id RequestID, params InitializeParams) ([]byte, error)`: Creates the JSON payload for an `initialize` request.
*   `UnmarshalInitializeResult(data []byte) (*InitializeResult, RequestID, *RPCError, error)`: Parses the JSON payload of an `initialize` response, returning the result, ID, and any potential RPC or parsing errors.

#### Prompts

*   `MarshalListPromptsRequest(id RequestID, params *ListPromptsParams) ([]byte, error)`: Creates the JSON payload for a `prompts/list` request.
*   `UnmarshalListPromptsResult(data []byte) (ListPromptsResult, RequestID, *RPCError, error)`: Parses the JSON payload of a `prompts/list` response.
*   `MarshalGetPromptRequest(id RequestID, params GetPromptParams) ([]byte, error)`: Creates the JSON payload for a `prompts/get` request.
*   `UnmarshalGetPromptResult(data []byte) (GetPromptResult, RequestID, *RPCError, error)`: Parses the JSON payload of a `prompts/get` response. Note: The `Content` field within `PromptMessage` requires further unmarshaling by the caller.

#### Resources

*   `MarshalListResourcesRequest(id RequestID, params *ListResourcesParams) ([]byte, error)`: Creates the JSON payload for a `resources/list` request.
*   `UnmarshalListResourcesResult(data []byte) (*ListResourcesResult, RequestID, *RPCError, error)`: Parses the JSON payload of a `resources/list` response.
*   `MarshalReadResourcesRequest(id RequestID, params ReadResourceParams) ([]byte, error)`: Creates the JSON payload for a `resources/read` request.
*   `UnmarshalReadResourcesResult(data []byte) (*ReadResourceResult, RequestID, *RPCError, error)`: Parses the JSON payload of a `resources/read` response. Note: The `Contents` field requires further unmarshaling by the caller.
*   `UnmarshalListResourcesTemplatesResult(data []byte) (*ListResourcesTemplatesResult, RequestID, *RPCError, error)`: Parses the JSON payload of a `resources/templates/list` response.

#### Tools

*   `MarshalListToolsRequest(id RequestID, params *ListToolsParams) ([]byte, error)`: Creates the JSON payload for a `tools/list` request.
*   `UnmarshalListToolsResult(data []byte) (ListToolsResult, RequestID, *RPCError, error)`: Parses the JSON payload of a `tools/list` response.
*   `MarshalCallToolRequest(id RequestID, params CallToolParams) ([]byte, error)`: Creates the JSON payload for a `tools/call` request.
*   `UnmarshalCallToolResponse(data []byte) (CallToolResult, RequestID, *RPCError, error)`: Parses the JSON payload of a `tools/call` response. Note: The `Content` field requires further unmarshaling by the caller.

### Server

Helper functions designed for use within an MCP server application. These often require a `*utils.Logger` instance for error reporting.

#### Initialize

*   `UnmarshalInitializeRequest(payload []byte, logger *utils.Logger) (*InitializeParams, RequestID, *RPCError, error)`: Parses the JSON payload of an incoming `initialize` request, returning the parameters, ID, and any potential RPC or parsing errors.
*   `MarshalInitializeResult(id RequestID, result InitializeResult, logger *utils.Logger) ([]byte, error)`: Creates the JSON payload for a successful `initialize` response.
*   `NewInitializeResult(...) InitializeResult`: Helper to construct a standard `InitializeResult` struct.

#### Prompts

*   `UnmarshalListPromptsRequest(payload []byte, logger *utils.Logger) (ListPromptsParams, RequestID, *RPCError, error)`: Parses the JSON payload of an incoming `prompts/list` request.
*   `MarshalListPromptsResult(id RequestID, result ListPromptsResult, logger *utils.Logger) ([]byte, error)`: Creates the JSON payload for a successful `prompts/list` response.
*   `NewListPromptsResult(...) ListPromptsResult`: Helper to construct a `ListPromptsResult` struct.
*   `UnmarshalGetPromptRequest(payload []byte, logger *utils.Logger) (GetPromptParams, RequestID, *RPCError, error)`: Parses the JSON payload of an incoming `prompts/get` request.
*   `MarshalGetPromptResult(id RequestID, result GetPromptResult, logger *utils.Logger) ([]byte, error)`: Creates the JSON payload for a successful `prompts/get` response.
*   `NewGetPromptResult(...) GetPromptResult`: Helper to construct a `GetPromptResult` struct.

#### Resources

*   `UnmarshalListResourcesRequest(id RequestID, payload []byte, logger *utils.Logger) (*RPCRequest, RequestID, *RPCError, error)`: Parses the base JSON payload of an incoming `resources/list` request (further parsing of params might be needed).
*   `MarshalListResourcesResult(id RequestID, resourcesList []Resource, cursor string, logger *utils.Logger) ([]byte, error)`: Creates the JSON payload for a successful `resources/list` response.
*   `UnmarshalReadResourceRequest(payload []byte, logger *utils.Logger) (*ReadResourceParams, RequestID, *RPCError, error)`: Parses the JSON payload of an incoming `resources/read` request.
*   `MarshalReadResourceResult(id RequestID, result ReadResourceResult, logger *utils.Logger) ([]byte, error)`: Creates the JSON payload for a successful `resources/read` response.
*   `NewReadResourcesResult(uri string, mimetype string, contents []byte) (ReadResourceResult, error)`: Helper to construct a `ReadResourceResult` from raw content, handling text/blob distinction and encoding.
*   `MarshalListResourcesTemplatesResult(id RequestID, params *ListResourcesTemplatesParams) ([]byte, error)`: Creates the JSON payload for a `resources/templates/list` *request* (Note: Likely intended to marshal a *result*, but currently marshals request params).

#### Tools

*   `UnmarshalListToolsRequest(payload []byte, logger *utils.Logger) (ListToolsParams, RequestID, *RPCError, error)`: Parses the JSON payload of an incoming `tools/list` request.
*   `MarshalListToolsResult(id RequestID, result ListToolsResult, logger *utils.Logger) ([]byte, error)`: Creates the JSON payload for a successful `tools/list` response.
*   `UnmarshalCallToolRequest(payload []byte, logger *utils.Logger) (CallToolParams, RequestID, *RPCError, error)`: Parses the JSON payload of an incoming `tools/call` request.
*   `MarshalCallToolResult(id RequestID, result CallToolResult, logger *utils.Logger) ([]byte, error)`: Creates the JSON payload for a successful `tools/call` response.

### Common

*   **Type Definitions:** Defines Go structs corresponding to the various MCP message types and data structures specified in the [MCP schema](schema.json) (e.g., `RPCRequest`, `RPCResponse`, `Resource`, `Prompt`, `Tool`, `TextContent`, etc.).
*   **Error Handling:** Defines standard MCP error codes (e.g., `ErrorCodeParseError`, `ErrorCodeMethodNotFound`) and provides functions (`NewRPCError`, `MarshalErrorResponse`, `UnmarshalErrorResponse`) for creating and handling JSON-RPC error responses.
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
