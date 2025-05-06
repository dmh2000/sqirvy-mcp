# cmd directory Makefile

.PHONY: all build test clean

BUILD_DIR := $(if $(value BUILD_DIR),$(BUILD_DIR),../build)
RELEASE_DIR := $(if $(value RELEASE_DIR),$(RELEASE_DIR),../release)

all: build test

build:
	$(MAKE) -C sqirvy-mcp build

test:
	$(MAKE) -C sqirvy-mcp test

clean:
	$(MAKE) -C sqirvy-mcp clean
	@rm -f bin/mcp-server
	@rm -f bin/mcp-server.log

release: build test
	@mkdir -p $(RELEASE_DIR)
	@$(MAKE) $(SILENT) -C sqirvy-mcp release