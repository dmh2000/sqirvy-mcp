# Top-level Makefile for sqirvy-mcp project

.PHONY: all build test clean

SILENT=-s
BUILD_DIR=build
RELEASE_DIR=release

all: build test

build:
	@echo "Building sqirvy-mcp project..."
	@mkdir -p $(BUILD_DIR)
	@touch build/sqirvy-mcp.log
	@$(MAKE) $(SILENT) -C pkg build
	@$(MAKE) $(SILENT) -C cmd build

test:
	@echo "Testing sqirvy-mcp project..."
	@$(MAKE) $(SILENT) -C pkg test
	@$(MAKE) $(SILENT) -C cmd test

clean:
	@echo "Cleaning sqirvy-mcp project..."
	@$(MAKE) $(SILENT) -C pkg clean
	@$(MAKE) $(SILENT) -C cmd clean
	@-rm -rf $(BUILD_DIR)

release: build test
	@echo "Building release binaries"
	@mkdir -p $(RELEASE_DIR)
	@$(MAKE) $(SILENT) -C cmd release
