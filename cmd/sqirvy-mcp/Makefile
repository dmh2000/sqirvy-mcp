.PHONY: build test clean release

BUILD_DIR := $(if $(value BUILD_DIR),$(BUILD_DIR),../../build)
RELEASE_DIR := $(if $(value RELEASE_DIR),$(RELEASE_DIR),../../release)

build:
	echo "Building mcp-server"
	staticcheck ./...
	go build  -o $(BUILD_DIR)/sqirvy-mcp .

test:
	@echo "no tests"

clean:
	@rm -f $(BUILD_DIR)/sqirvy-mcp.log

release: 
	echo "Building release binaries"
	mkdir -p $(RELEASE_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(RELEASE_DIR)/sqirvy-mcp-linux-amd64 .
	GOOS=windows GOARCH=amd64 go build -o $(RELEASE_DIR)/sqirvy-mcp-windows-amd64.exe .
	GOOS=darwin GOARCH=amd64 go build -o $(RELEASE_DIR)/sqirvy-mcp-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o $(RELEASE_DIR)/sqirvy-mcp-darwin-arm64 .
