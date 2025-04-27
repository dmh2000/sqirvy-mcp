package main

import (
	"fmt"
	"log"
	"os"
	transport "sqirvy-mcp/pkg/transport"
	utils "sqirvy-mcp/pkg/utils"
	"time"
)

func main() {

	f, err := os.OpenFile("sse.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Failed to open log file:", err)
	}
	logger := utils.New(f, "", log.LstdFlags|log.Lshortfile, utils.LevelDebug)

	logger.Printf(utils.LevelInfo, "sqirvy-mcp started")

	sseParams := transport.SseMakeParams("localhost",
		9000,
		"/sse",
		logger)

	_, postChan := transport.StartSSE(sseParams)

	go func() {
		for msg := range postChan {
			fmt.Printf("Received message: %s\n", msg)
		}
	}()

	for {
		// msg := `
		// {
		// 	"jsonrpc": "2.0",
		// 	"method": "notifications/alert",
		// 	"params": {
		// 		"alert": "test"
		// 	}
		// }
		// `
		// getChan <- []byte(msg)
		// fmt.Printf("Sent message: %s\n", msg)
		time.Sleep(60 * time.Second)
	}
}
