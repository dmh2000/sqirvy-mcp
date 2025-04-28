# cmd directory Makefile

.PHONY: all build test clean

all: build test

build:
	$(MAKE) -C mcp-server build

test:
	$(MAKE) -C mcp-server test

clean:
	$(MAKE) -C mcp-server clean
	@rm -f bin/mcp-server
	@rm -f bin/mcp-server.log
