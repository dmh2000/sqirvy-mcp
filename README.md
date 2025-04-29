# Sqirvy MCP - Model Context Protocol Implementation

This repository contains a Go implementation of the Model Context Protocol (MCP), designed to facilitate communication between AI models/clients and backend servers/tools. It provides both a core library for the protocol itself and an example server implementation.

## Overview

The Model Context Protocol (MCP) defines a standard way for AI clients (like large language models or specialized agents) to interact with servers that provide context, tools, or other resources. This project aims to provide robust and easy-to-use Go packages for building MCP-compliant clients and servers.

## Project Structure

The project is organized into the following main directories:

*   **`cmd/`**: Contains executable applications.
    *   **`cmd/mcp-server/`**: An example MCP server implementation demonstrating how to use the `pkg/mcp` and `pkg/transport` packages. It handles standard MCP requests like `initialize`, `ping`, `tools/list`, `resources/read`, etc., over standard I/O. See [cmd/mcp-server/README.md](cmd/mcp-server/README.md) for details.
*   **`pkg/`**: Contains reusable library packages.
    *   **`pkg/mcp/`**: The core package implementing the MCP specification. It defines Go types for all MCP messages (requests, responses, notifications, errors) and provides functions for marshaling and unmarshaling these messages to/from JSON. See [pkg/mcp/README.md](pkg/mcp/README.md) for details.
    It includes type definitions for all definitions in the official [MCP schema specification](https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/schema/2025-03-26/schema.json). That file is included in pkg/mcp/schema.json.
    
    *   **`pkg/transport/`**: Provides an abstraction layer for sending and receiving MCP messages over different communication channels, primarily focusing on standard I/O (`io.Reader`/`io.Writer`). See [pkg/transport/README.md](pkg/transport/README.md) for details.
    *   **`pkg/utils/`**: Contains general utility functions used across the project, currently focused on providing a flexible, level-based logger. See [pkg/utils/README.md](pkg/utils/README.md) for details.

## Getting Started

1.  **Build the Server:**
    ```bash
    make -C cmd build
    ```
2.  **Run the Server:**
    Navigate to the `cmd/bin` directory and use the provided script:
    ```bash
    cd cmd/bin
    ./run.sh
    ```
    This script runs the `mcp-server` executable, which will listen for MCP messages on standard input and send responses to standard output. You can interact with it using an MCP client or tool like `mcp-inspector`.

3.  **Run Tests:**
    To run tests for all packages:
    ```bash
    make test
    ```
    Or for a specific package:
    ```bash
    go test ./pkg/mcp/...
    go test ./pkg/transport/...
    go test ./pkg/utils/...
    ```


