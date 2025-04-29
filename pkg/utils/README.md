# Utils Package (`pkg/utils`)

This package provides general utility functions used across the `sqirvy-mcp` project. Currently, it primarily focuses on logging.

## Functionality

*   **Logger:** Provides a flexible `Logger` type that wraps the standard Go `log.Logger`.
    *   **Level-Based Logging:** Supports different logging levels (`DEBUG`, `INFO`, `WARNING`, `ERROR`). Messages are only output if their level is at or above the logger's configured level.
    *   **Standard Interface:** Offers familiar `Printf`, `Println`, `Fatalf`, `Fatalln` methods, requiring a level string as the first argument.
    *   **Configurable Output:** Allows specifying the output `io.Writer` (e.g., `os.Stderr`, a file).
    *   **Configurable Level:** The logging level can be set during creation or changed later using `SetLevel`. Invalid levels default to `INFO`.
    *   **Standard Logger Access:** Provides access to the underlying `*log.Logger` via `StandardLogger()`.
*   **Testing:** Includes unit tests (`logger_test.go`) to verify level filtering, output correctness, and level setting.

## Usage

1.  **Import:** Import the package:
    ```go
    import "sqirvy-mcp/pkg/utils"
    import "os"
    import "log" // For flags
    ```
2.  **Create:** Instantiate a logger using `New`:
    ```go
    // Log INFO and above to stderr
    logger := utils.New(os.Stderr, "MCP_SERVER: ", log.LstdFlags, utils.LevelInfo)

    // Log DEBUG and above to a file
    // file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    // if err != nil { ... }
    // fileLogger := utils.New(file, "MCP_SERVER: ", log.LstdFlags, utils.LevelDebug)
    ```
3.  **Log Messages:** Use the `Println` or `Printf` methods, specifying the level:
    ```go
    logger.Println(utils.LevelInfo, "Server started successfully.")
    logger.Printf(utils.LevelDebug, "Received message payload: %s", string(payload))
    ```
4.  **Change Level:** Adjust the logging level dynamically if needed:
    ```go
    logger.SetLevel(utils.LevelDebug) // Start logging DEBUG messages
    ```
