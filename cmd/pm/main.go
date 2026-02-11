package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/mcp"
)

func main() {
	mcpFlag := flag.Bool("mcp", false, "Run as MCP server")
	flag.Parse()

	if *mcpFlag {
		if err := mcp.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Println("phase 1: core library only")
}
