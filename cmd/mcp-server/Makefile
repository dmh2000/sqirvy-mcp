.PHONY: build test clean

build:
	staticcheck ./...
	go build  -o ../bin/mcp-server .

test:
	@echo "no tests"

clean:
	@rm -f mcp-server.log

