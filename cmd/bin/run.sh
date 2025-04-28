#!/bin/bash
pushd .. && make build && popd
# npx @modelcontextprotocol/inspector ./mcp-server -log ./mcp-server.log ./bin/mcp-server -server-log-level DEBUG -server-ping-target 127.0.0.1
npx @modelcontextprotocol/inspector ./mcp-server -config ./.mcp-server 

