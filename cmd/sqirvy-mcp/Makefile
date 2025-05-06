.PHONY: build test clean

BUILD := $(if $(value BUILD),$(BUILD),../../build)

build:
	echo "Building mcp-server"
	staticcheck ./...
	go build  -o $(BUILD)/sqirvy-mcp .

test:
	@echo "no tests"

clean:
	@rm -f $(BUILD)/sqirvy-mcp.log

