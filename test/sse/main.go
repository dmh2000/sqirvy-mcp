package main

import (
	"fmt"
	transport "sqirvy-mcp/pkg/transport"
	utils "sqirvy-mcp/pkg/utils"
)

func main() {

	var logger utils.Logger
	sseParams := transport.SseMakeParams("localhost", 3001, "/sse", "localhost", 3002, "/messages", &logger)

	getChan, postChan := transport.StartSSE(sseParams)

	go func() {
		for msg := range postChan {
			fmt.Printf("Received message: %s\n", msg)
		}
	}()

	// initialized message
	msg := `
	{
		"jsonrpc": "2.0",
		"method": "notifications/initialized",
	}
	`
	getChan <- []byte(msg)
	fmt.Printf("Sent message: %s\n", msg)

	for {
		msg := `
		{
			"jsonrpc": "2.0",
			"method": "notifications/alert",
			"params": {
				"alert": "test"
			}
		}
		`
		getChan <- []byte(msg)
		fmt.Printf("Sent message: %s\n", msg)
	}
}
