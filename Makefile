# Top-level Makefile for sqirvy-mcp project

.PHONY: all build test clean

all: build test

build:
	@echo "Building sqirvy-mcp project..."
	@$(MAKE) -C pkg build
	@$(MAKE) -C cmd build

test:
	@echo "Testing sqirvy-mcp project..."
	@$(MAKE) -C pkg test
	@$(MAKE) -C cmd test

clean:
	@echo "Cleaning sqirvy-mcp project..."
	@$(MAKE) -C pkg clean
	@$(MAKE) -C cmd clean
