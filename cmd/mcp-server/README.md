# MCP Server Example

This directory contains the implementation of a simple server that communicates using the Model Context Protocol (MCP). It demonstrates how to handle basic MCP requests and responses over standard input/output.

## Functionality

The `mcp-server` program acts as an MCP server, listening for JSON-RPC messages on standard input and sending responses on standard output. It implements handlers for several core MCP methods:

*   `initialize`: Handles the initial handshake with the client, negotiating capabilities.
*   `ping`: Responds to ping requests.
*   `tools/list`: Lists available tools (currently includes a `ping` tool).
*   `tools/call`: Executes a specific tool (currently supports the `ping` tool).
*   `prompts/list`: Lists available prompt templates (currently includes a `query` prompt).
*   `prompts/get`: Retrieves the content of a specific prompt template.
*   `resources/list`: Lists available resources (currently includes an example file resource).
*   `resources/templates/list`: Lists available resource templates (currently includes a `random_data` template).
*   `resources/read`: Reads the content of a specified resource URI (supports `file://` and `data://random_data`).

The server uses a configuration file and command-line flags to set logging behavior, project root path for file resources, and other settings.

## Building and Running

The server can be built and run using the provided Makefiles and shell scripts in the parent `cmd` directory.

1.  **Build:** Navigate to the `cmd` directory and run `make build`. This will build the `mcp-server` executable and place it in the `cmd/bin` directory.
    ```bash
    # From the repository root
    make -C cmd build
    ```

2.  **Run:** Navigate to the `cmd/bin` directory. The `run.sh` script provides an example of how to start the server, typically by piping its standard I/O to an MCP client like the `mcp-inspector`.
    ```bash
    # From the repository root
    cd cmd/bin
    ./run.sh
    ```
    The `run.sh` script first builds the server (ensuring you have the latest version) and then runs it with a sample configuration file (`.mcp-server`).

## Configuration

The server's behavior can be configured using a YAML file (`.mcp-server` by default) and command-line flags. Command-line flags override settings in the configuration file.

Configuration file search paths (in order of priority):
1.  Path specified by the `--config` flag.
2.  `.mcp-server` in the current working directory.
3.  `$HOME/.config/mcp-server/.mcp-server`.

Available configuration options and command-line flags:

*   **Log Level:**
    *   Config: `log.level` (e.g., `DEBUG`, `INFO`)
    *   Flag: `--log-level`
*   **Log Output:**
    *   Config: `log.output` (path to log file)
    *   Flag: `--log`
*   **Project Root Path:**
    *   Config: `project.rootPath` (base directory for `file://` resources)
    *   Flag: `--project-root`

An example configuration file (`cmd/bin/.mcp-server`) is provided.

## Logging

The server logs messages to the console (during config loading) and to a file specified by the configuration or `--log` flag. The log level (`DEBUG` or `INFO`) controls the verbosity. The `cmd/bin/log.sh` script can be used to tail the log file while the server is running.

```bash
# From the repository root
cd cmd/bin
./log.sh
```
