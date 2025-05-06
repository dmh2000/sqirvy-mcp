# cmd directory Makefile

.PHONY: all build test clean

BUILD := $(if $(value BUILD),$(BUILD),../build)

all: build test

build:
	$(MAKE) -C sqirvy-mcp build

test:
	$(MAKE) -C sqirvy-mcp test

clean:
	$(MAKE) -C sqirvy-mcp clean
	@rm -f bin/mcp-server
	@rm -f bin/mcp-server.log
