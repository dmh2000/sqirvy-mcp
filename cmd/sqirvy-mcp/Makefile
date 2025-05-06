.PHONY: build test clean

BUILD_DIR := $(if $(value BUILD_DIR),$(BUILD_DIR),../../build)

build:
	echo "Building mcp-server"
	staticcheck ./...
	go build  -o $(BUILD_DIR)/sqirvy-mcp .

test:
	@echo "no tests"

clean:
	@rm -f $(BUILD_DIR)/sqirvy-mcp.log

