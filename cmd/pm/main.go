package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/commands"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/mcp"
	"github.com/mkrowiarz/mcp-symfony-stack/internal/tui"
)

func main() {
	// Check for help before flag parsing
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			// Check if it's a command help request
			if len(os.Args) > 2 {
				// pm <command> --help
				break // Let command handlers deal with it
			}
			printHelp()
			return
		}
	}

	mcpFlag := flag.Bool("mcp", false, "Run as MCP server (stdio transport)")
	flag.Parse()

	if *mcpFlag {
		if err := mcp.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	args := flag.Args()
	if len(args) > 0 {
		switch args[0] {
		case "init":
			handleInit(args[1:])
			return
		case "checkout":
			handleCheckout(args[1:])
			return
		case "switch":
			handleSwitch(args[1:])
			return
		case "help", "--help", "-h":
			printHelp()
			return
		}
	}

	if err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	help := `pm - Project Manager for Docker Compose-based development

Usage:
  pm                    Run interactive TUI (default)
  pm --mcp              Run as MCP server for Claude Code
  pm <command> [flags]  Run specific command

Commands:
  init                  Generate config for current project
  checkout <branch>     Switch git branch and database
  switch                Switch database for current branch
  help                  Show this help message

Init Flags:
  --write, -w           Write config to .haive/config.json
  --namespace, -n       Wrap config in "pm" namespace

Checkout Flags:
  --create, -c          Create new branch
  --clone-from=<db>     Clone data from specified database

Switch Flags:
  --clone-from=<db>     Clone data from specified database

Examples:
  pm init --write                    # Create config file
  pm init --namespace --write        # Create namespaced config
  pm checkout feature/x --create     # Create branch with new db
  pm checkout main                   # Switch to main branch+db
  pm switch                          # Switch db for current branch

Config file locations (checked in order):
  1. .claude/project.json (recommended)
  2. .haive/config.json
  3. .haive.json

For more information: https://github.com/mkrowiarz/mcp-symfony-stack
`
	fmt.Println(help)
}

func wrapInNamespace(config string) string {
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(config), &cfg); err != nil {
		return config
	}

	wrapper := map[string]interface{}{
		"pm": cfg,
	}

	data, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return config
	}

	return string(data)
}

func handleInit(args []string) {
	writeFlag := false
	namespaceFlag := false
	for _, arg := range args {
		if arg == "--write" || arg == "-w" {
			writeFlag = true
		}
		if arg == "--namespace" || arg == "-n" {
			namespaceFlag = true
		}
	}

	result, err := commands.Init(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	config := result.SuggestedConfig
	if namespaceFlag {
		config = wrapInNamespace(config)
	}

	if writeFlag {
		configDir := ".haive"
		configPath := filepath.Join(configDir, "config.json")

		if _, err := os.Stat(configPath); err == nil {
			fmt.Fprintf(os.Stderr, "Config file already exists: %s\n", configPath)
			fmt.Fprintf(os.Stderr, "Remove it first or use 'pm init' to preview.\n")
			os.Exit(1)
		}

		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Created: %s\n", configPath)
	} else {
		fmt.Println(config)
	}
}

func handleCheckout(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: pm checkout <branch> [--create] [--clone-from=<db>]\n")
		os.Exit(1)
	}

	branch := args[0]
	createFlag := false
	cloneFrom := ""

	for _, arg := range args[1:] {
		if arg == "--create" || arg == "-c" {
			createFlag = true
		}
		if strings.HasPrefix(arg, "--clone-from=") {
			cloneFrom = strings.TrimPrefix(arg, "--clone-from=")
		}
	}

	result, err := commands.Checkout(".", branch, createFlag, cloneFrom)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Switched to branch: %s\n", result.Branch)
	fmt.Printf("✓ Using database: %s\n", result.Database)
	if result.Created {
		fmt.Printf("✓ Created new database\n")
	}
	if result.Cloned {
		fmt.Printf("✓ Cloned data from source database\n")
	}
}

func handleSwitch(args []string) {
	cloneFrom := ""
	for _, arg := range args {
		if strings.HasPrefix(arg, "--clone-from=") {
			cloneFrom = strings.TrimPrefix(arg, "--clone-from=")
		}
	}

	result, err := commands.Switch(".", cloneFrom)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Current branch: %s\n", result.Branch)
	fmt.Printf("✓ Using database: %s\n", result.Database)
	if result.Created {
		fmt.Printf("✓ Created new database\n")
	}
	if result.Cloned {
		fmt.Printf("✓ Cloned data from source database\n")
	}
}
