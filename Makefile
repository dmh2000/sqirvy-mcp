# Top-level Makefile for sqirvy-mcp project

.PHONY: all build test clean

SILENT=-s

all: build test

build:
	@echo "Building sqirvy-mcp project..."
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
