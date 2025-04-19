# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build/Test Commands
- Run all tests: `go test ./...`
- Run package tests: `go test github.com/dmh2000/sqirvy-mcp/pkg/transport`
- Run single test: `go test github.com/dmh2000/sqirvy-mcp/pkg/transport -run TestReadMessages`
- Verbose output: Add `-v` flag to any test command
- Test coverage: `go test -cover ./...`
- All build rules should run 'staticcheck' if there are go files in a directory

## Code Style Guidelines
- Format: Use standard Go formatting with `gofmt`
- Imports: Standard library first, followed by external deps, alphabetically ordered
- Types: Use interfaces for testability, meaningful custom types for domain objects
- Naming: CamelCase for exported, camelCase for unexported, ALL_CAPS for constants
- Error handling: Explicit checking, custom error vars at package level with `Err` prefix
- Documentation: GoDoc comments for all exported functions, types, and packages
- Tests: Table-driven tests, descriptive naming, thorough edge case coverage
- Concurrency: Proper mutex usage, clear blocking/non-blocking behavior documentation
