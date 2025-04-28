.PHONY: build clean

build:
	staticcheck ./...
	go build  -o ../bin/mcp-server .

clean:
	@rm -f mcp-server.log

