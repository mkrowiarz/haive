package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
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
				// haive <command> --help
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
		case "worktree", "wt":
			handleWorktree(args[1:])
			return
		case "serve":
			handleServe(args[1:])
			return
		case "mcp":
			handleMCP(args[1:])
			return
		case "tui":
			if err := tui.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
				os.Exit(1)
			}
			return
		case "help", "--help", "-h":
			printHelp()
			return
		}
	}

	printHelp()
}

func printHelp() {
	// ANSI color codes
	var (
		reset   = "\033[0m"
		bold    = "\033[1m"
		dim     = "\033[2m"
		cyan    = "\033[36m"
		green   = "\033[32m"
		yellow  = "\033[33m"
		magenta = "\033[35m"

	)

	fmt.Println()
	fmt.Println(cyan + "haive" + reset + " - Development Environment Manager for Docker Compose-based development")
	fmt.Println()
	fmt.Println(bold + "Usage:" + reset)
	fmt.Println("  " + green + "haive <command> [flags]" + reset + "  Run specific command")
	fmt.Println("  " + green + "haive --mcp" + reset + "              Run as MCP server for Claude Code")
	fmt.Println()
	fmt.Println(bold + "Commands:" + reset)
	fmt.Println("  " + yellow + "tui" + reset + "                   Run interactive TUI")
	fmt.Println("  " + yellow + "init" + reset + "                  Generate config for current project")
	fmt.Println("  " + yellow + "checkout <branch>" + reset + "     Switch git branch and database")
	fmt.Println("  " + yellow + "switch" + reset + "                Switch database for current branch")
	fmt.Println("  " + yellow + "worktree <cmd>" + reset + "        Manage git worktrees")
	fmt.Println("  " + yellow + "serve <cmd>" + reset + "           Start/stop worktree containers")
	fmt.Println("  " + yellow + "mcp <cmd>" + reset + "             Manage MCP server configuration")
	fmt.Println("  " + yellow + "help" + reset + "                  Show this help message")
	fmt.Println()
	fmt.Println(bold + "Init Flags:" + reset)
	fmt.Println("  " + magenta + "--write, -w" + reset + "           Write config to .haive.toml")
	fmt.Println("  " + magenta + "--ai, -a" + reset + "              Show AI configuration instructions")
	fmt.Println()
	fmt.Println(bold + "Checkout Flags:" + reset)
	fmt.Println("  " + magenta + "--create, -c" + reset + "          Create new branch")
	fmt.Println("  " + magenta + "--clone-from=<db>" + reset + "     Clone data from specified database")
	fmt.Println()
	fmt.Println(bold + "Switch Flags:" + reset)
	fmt.Println("  " + magenta + "--clone-from=<db>" + reset + "     Clone data from specified database")
	fmt.Println()
	fmt.Println(bold + "Worktree Commands:" + reset)
	fmt.Println("  " + yellow + "list" + reset + "                  List all worktrees")
	fmt.Println("  " + yellow + "create <branch>" + reset + "       Create new worktree")
	fmt.Println("  " + yellow + "remove <branch>" + reset + "       Remove worktree")
	fmt.Println()
	fmt.Println(bold + "Worktree Flags:" + reset)
	fmt.Println("  " + magenta + "--new-branch, -n" + reset + "      Create new branch (with create)")
	fmt.Println()
	fmt.Println(bold + "Serve Commands:" + reset)
	fmt.Println("  " + yellow + "serve" + reset + "                  Start containers for current worktree")
	fmt.Println("  " + yellow + "serve stop" + reset + "             Stop containers for current worktree")
	fmt.Println()
	fmt.Println(bold + "MCP Commands:" + reset)
	fmt.Println("  " + yellow + "mcp install <tool>" + reset + "     Install MCP config for AI tool (claude/kimi/codex)")
	fmt.Println()
	fmt.Println(bold + "Examples:" + reset)
	fmt.Println("  " + green + "haive init --write" + reset + "                 # Create config file")
	
	fmt.Println("  " + green + "haive checkout feature/x --create" + reset + "  # Create branch with new db")
	fmt.Println("  " + green + "haive checkout main" + reset + "                # Switch to main branch+db")
	fmt.Println("  " + green + "haive switch" + reset + "                       # Switch db for current branch")
	fmt.Println("  " + green + "haive worktree list" + reset + "                # List worktrees")
	fmt.Println("  " + green + "haive worktree create feature/x" + reset + "    # Create worktree")
	fmt.Println("  " + green + "haive worktree remove feature/x" + reset + "    # Remove worktree")
	fmt.Println("  " + green + "haive serve" + reset + "                         # Start worktree app")
	fmt.Println("  " + green + "haive serve stop" + reset + "                      # Stop worktree app")
	fmt.Println()
	fmt.Println(green + "Config file:" + reset + " .haive.toml")
	fmt.Println()
	fmt.Println(dim + "For more information:" + reset + " https://github.com/mkrowiarz/mcp-symfony-stack")
	fmt.Println()
}


func handleInit(args []string) {
	writeFlag := false
	aiFlag := false
	for _, arg := range args {
		if arg == "--write" || arg == "-w" {
			writeFlag = true
		}
		if arg == "--ai" || arg == "-a" {
			aiFlag = true
		}
	}

	if aiFlag {
		printAIInstructions()
		return
	}

	result, err := commands.Init(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	config := result.SuggestedConfig

	if writeFlag {
		configPath := ".haive.toml"

		if _, err := os.Stat(configPath); err == nil {
			fmt.Fprintf(os.Stderr, "Config file already exists: %s\n", configPath)
			fmt.Fprintf(os.Stderr, "Remove it first or use 'haive init' to preview.\n")
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

func printAIInstructions() {
	fmt.Println(`# Haive AI Configuration Instructions

You are helping configure haive - a development environment manager for Docker Compose projects.

## Detection (Analyze the project)

1. **Docker Compose Files**: Find all docker-compose*.yml, compose*.yaml files
2. **Database Service**: Look for services named: database, db, mysql, postgres, mariadb
3. **Project Type**: Check for framework indicators:
   - composer.json → PHP/Symfony
   - package.json → Node.js
   - go.mod → Go
   - Cargo.toml → Rust
4. **Worktree Support**: Check if .worktrees/ directory exists or is in .gitignore

## Configuration Sections

` + "```toml" + `
# REQUIRED: Docker compose files
[docker]
compose_files = ["docker-compose.yml"]  # List all compose files in merge order

# OPTIONAL: Database operations
[database]
service = "database"              # Docker service name for DB container
dsn = "${DATABASE_URL}"           # Connection string, supports env vars
allowed = ["myapp", "myapp_*"]    # Allowed DB names (glob patterns)
dumps_path = "var/dumps"          # Where to store DB dumps (default: var/dumps)

# OPTIONAL: Worktree support
[worktree]
base_path = ".worktrees"          # Where worktrees are created
db_per_worktree = true            # Auto-create DB per worktree

[worktree.copy]                   # Files to copy to new worktrees
include = [".env.local", "docker/**/*.yaml"]
exclude = ["vendor/", "node_modules/", ".git/"]

[worktree.env]                    # Per-worktree env configuration
file = ".env.local"
var_name = "DATABASE_URL"

# OPTIONAL: Container management per worktree
[serve]
compose_files = ["docker/dev/compose.app.yaml"]

[serve.worktree]                  # Worktree-specific compose overrides
compose_files = ["docker/dev/compose.worktree.yaml"]
` + "```" + `

## Common Patterns

**Simple project with database:**
` + "```toml" + `
[docker]
compose_files = ["docker-compose.yml"]

[database]
service = "db"
dsn = "${DATABASE_URL}"
allowed = ["myapp", "myapp_*"]
` + "```" + `

**Modular compose with worktrees:**
` + "```toml" + `
[docker]
compose_files = ["compose.yaml", "docker/dev/compose.database.yaml"]

[database]
service = "database"
dsn = "mysql://root:root@database:3306/${DB_NAME}"
allowed = ["myapp", "myapp_*"]

[worktree]
base_path = ".worktrees"
db_per_worktree = true

[worktree.copy]
include = [".env.local", "docker/**/*.yaml"]

[worktree.env]
file = ".env.local"
var_name = "DATABASE_URL"

[serve]
compose_files = ["compose.yaml", "docker/dev/compose.database.yaml", "docker/dev/compose.app.yaml"]

[serve.worktree]
compose_files = ["docker/dev/compose.app.yaml"]
` + "```" + `

## Steps to Configure

1. Run 'haive init' to see auto-detected config
2. Check which docker-compose files exist and their purpose
3. Identify if database service exists and its name
4. Check if worktree support is desired (multiple parallel branches)
5. Create .haive.toml with appropriate sections
6. Run 'haive init' again to validate (should show your config)
7. Test: 'haive worktree list', 'haive checkout <branch>'

## Validation

After creating config, verify:
- 'haive init' shows expected compose files
- 'cat .haive.toml' has valid TOML syntax
- Database service name matches docker-compose service
- Env var in DSN exists in .env or .env.local

## Full Reference

See: https://github.com/mkrowiarz/haive/blob/main/AGENTS.md
`)
}

func handleCheckout(args []string) {
	// Handle help flags
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			printCheckoutHelp()
			return
		}
	}

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: haive checkout <branch> [--create] [--clone-from=<db>]\n")
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
	// Handle help flags
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			printSwitchHelp()
			return
		}
	}

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

func gitBranchExists(branch string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

func handleWorktree(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: haive worktree <list|create|remove> [options]\n")
		os.Exit(1)
	}

	// Handle help flags
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		printWorktreeHelp()
		return
	}

	switch args[0] {
	case "list", "ls":
		worktrees, err := commands.List(".")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if len(worktrees) == 0 {
			fmt.Println("No worktrees found")
			return
		}

		fmt.Println("Worktrees:")
		for _, wt := range worktrees {
			prefix := "  "
			if wt.IsMain {
				prefix = "* "
			}
			fmt.Printf("%s%s  (%s)\n", prefix, wt.Branch, wt.Path)
		}

	case "create", "add":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: haive worktree create <branch> [--new-branch]\n")
			os.Exit(1)
		}

		branch := args[1]
		newBranch := false
		for _, arg := range args[2:] {
			if arg == "--new-branch" || arg == "-n" {
				newBranch = true
			}
		}

		// Auto-detect if branch exists
		if !newBranch {
			if !gitBranchExists(branch) {
				fmt.Printf("Branch '%s' doesn't exist. Creating new branch...\n", branch)
				newBranch = true
			}
		}

		result, err := commands.Create(".", branch, newBranch)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ Created worktree: %s\n", result.Branch)
		fmt.Printf("✓ Path: %s\n", result.Path)

	case "remove", "rm", "delete":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: haive worktree remove <branch>\n")
			os.Exit(1)
		}

		branch := args[1]
		result, err := commands.Remove(".", branch)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ Removed worktree: %s\n", branch)
		fmt.Printf("✓ Path: %s\n", result.Path)

	default:
		fmt.Fprintf(os.Stderr, "Unknown worktree command: %s\n", args[0])
		fmt.Fprintf(os.Stderr, "Usage: haive worktree <list|create|remove> [options]\n")
		os.Exit(1)
	}
}

func printWorktreeHelp() {
	var (
		reset   = "\033[0m"
		bold    = "\033[1m"
		cyan    = "\033[36m"
		green   = "\033[32m"
		yellow  = "\033[33m"
		magenta = "\033[35m"
	)

	fmt.Println()
	fmt.Println(cyan + "haive worktree" + reset + " - Manage git worktrees")
	fmt.Println()
	fmt.Println(bold + "Usage:" + reset)
	fmt.Println("  " + green + "haive worktree <command> [options]" + reset)
	fmt.Println()
	fmt.Println(bold + "Commands:" + reset)
	fmt.Println("  " + yellow + "list, ls" + reset + "              List all worktrees")
	fmt.Println("  " + yellow + "create, add <branch>" + reset + "  Create new worktree")
	fmt.Println("  " + yellow + "remove, rm <branch>" + reset + "   Remove worktree")
	fmt.Println()
	fmt.Println(bold + "Flags:" + reset)
	fmt.Println("  " + magenta + "--new-branch, -n" + reset + "      Create new branch (with create)")
	fmt.Println()
	fmt.Println(bold + "Examples:" + reset)
	fmt.Println("  " + green + "haive worktree list" + reset + "                  # List all worktrees")
	fmt.Println("  " + green + "haive worktree create feature/x" + reset + "      # Create worktree from existing branch")
	fmt.Println("  " + green + "haive worktree create feature/x -n" + reset + "   # Create worktree with new branch")
	fmt.Println("  " + green + "haive worktree remove feature/x" + reset + "      # Remove worktree")
	fmt.Println()
}

func handleServe(args []string) {
	// Handle help flags
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			printServeHelp()
			return
		}
	}

	// ANSI color codes
	var (
		reset = "\033[0m"
		red   = "\033[31m"
		green = "\033[32m"
		cyan  = "\033[36m"
	)

	// Check if it's a stop command
	if len(args) > 0 && args[0] == "stop" {
		if err := commands.Stop("."); err != nil {
			fmt.Println()
			fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
			fmt.Println()
			os.Exit(1)
		}

		fmt.Printf("%s✓%s Stopped worktree app container\n", green, reset)
		return
	}

	// Start the worktree app container
	result, err := commands.Serve(".")
	if err != nil {
		fmt.Println()
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		fmt.Println()
		os.Exit(1)
	}

	fmt.Printf("%s✓%s Worktree: %s\n", green, reset, result.Branch)
	fmt.Printf("%s✓%s Started containers\n", green, reset)
	fmt.Printf("%s✓%s URL: %s%s%s\n", green, reset, cyan, result.URL, reset)
}

func printCheckoutHelp() {
	var (
		reset   = "\033[0m"
		bold    = "\033[1m"
		cyan    = "\033[36m"
		green   = "\033[32m"
		magenta = "\033[35m"
	)

	fmt.Println()
	fmt.Println(cyan + "haive checkout" + reset + " - Switch git branch and database")
	fmt.Println()
	fmt.Println(bold + "Usage:" + reset)
	fmt.Println("  " + green + "haive checkout <branch> [flags]" + reset)
	fmt.Println()
	fmt.Println(bold + "Flags:" + reset)
	fmt.Println("  " + magenta + "--create, -c" + reset + "          Create new branch")
	fmt.Println("  " + magenta + "--clone-from=<db>" + reset + "     Clone data from specified database")
	fmt.Println()
	fmt.Println(bold + "Examples:" + reset)
	fmt.Println("  " + green + "haive checkout main" + reset + "                    # Switch to main")
	fmt.Println("  " + green + "haive checkout feature/x --create" + reset + "      # Create branch with new db")
	fmt.Println("  " + green + "haive checkout feature/x -c" + reset + "            # Shorthand for --create")
	fmt.Println("  " + green + "haive checkout feature/x --clone-from=main" + reset + " # Clone from main db")
	fmt.Println()
}

func printSwitchHelp() {
	var (
		reset   = "\033[0m"
		bold    = "\033[1m"
		cyan    = "\033[36m"
		green   = "\033[32m"
		magenta = "\033[35m"
	)

	fmt.Println()
	fmt.Println(cyan + "haive switch" + reset + " - Switch database for current branch")
	fmt.Println()
	fmt.Println(bold + "Usage:" + reset)
	fmt.Println("  " + green + "haive switch [flags]" + reset)
	fmt.Println()
	fmt.Println(bold + "Flags:" + reset)
	fmt.Println("  " + magenta + "--clone-from=<db>" + reset + "     Clone data from specified database")
	fmt.Println()
	fmt.Println(bold + "Examples:" + reset)
	fmt.Println("  " + green + "haive switch" + reset + "                             # Switch to branch database")
	fmt.Println("  " + green + "haive switch --clone-from=main" + reset + "           # Clone and switch to main db")
	fmt.Println()
}

func printServeHelp() {
	var (
		reset = "\033[0m"
		bold  = "\033[1m"
		cyan  = "\033[36m"
		green = "\033[32m"
		dim   = "\033[2m"
	)

	fmt.Println()
	fmt.Println(cyan + "haive serve" + reset + " - Manage containers for current worktree")
	fmt.Println()
	fmt.Println(bold + "Usage:" + reset)
	fmt.Println("  " + green + "haive serve" + reset + "             Start containers")
	fmt.Println("  " + green + "haive serve stop" + reset + "        Stop containers")
	fmt.Println()
	fmt.Println(bold + "Configuration:" + reset)
	fmt.Println("  Add a [serve] section to your .haive.toml:")
	fmt.Println()
	fmt.Println(dim + "  [serve]" + reset)
	fmt.Println(dim + "  compose_files = [\"docker-compose.yml\", \"compose.override.yml\"]" + reset)
	fmt.Println()
	fmt.Println(bold + "Examples:" + reset)
	fmt.Println("  " + green + "haive serve" + reset + "             # Start containers")
	fmt.Println("  " + green + "haive serve stop" + reset + "        # Stop containers")
	fmt.Println()
}

func handleMCP(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: haive mcp install <claude|kimi|codex>\n")
		os.Exit(1)
	}

	switch args[0] {
	case "install", "add":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: haive mcp install <claude|kimi|codex>\n")
			os.Exit(1)
		}
		installMCP(args[1])
	case "remove", "rm":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: haive mcp remove <claude|kimi|codex>\n")
			os.Exit(1)
		}
		removeMCP(args[1])
	case "help", "--help", "-h":
		printMCPHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown mcp command: %s\n", args[0])
		fmt.Fprintf(os.Stderr, "Usage: haive mcp install <claude|kimi|codex>\n")
		os.Exit(1)
	}
}

func installMCP(tool string) {
	var (
		reset  = "\033[0m"
		green  = "\033[32m"
		red    = "\033[31m"
		cyan   = "\033[36m"
		yellow = "\033[33m"
	)

	var err error
	switch tool {
	case "claude", "claude-code":
		err = installClaudeMCP()
	case "kimi", "kimi-cli":
		err = installKimiMCP()
	case "codex":
		err = installCodexMCP()
	default:
		fmt.Fprintf(os.Stderr, "Unknown tool: %s\n", tool)
		fmt.Fprintf(os.Stderr, "Supported tools: claude, kimi, codex\n")
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}

	fmt.Printf("%s✓%s MCP config installed for %s%s%s\n", green, reset, cyan, tool, reset)
	fmt.Println()
	fmt.Println("Restart your AI assistant or start a new session to use haive tools.")
	fmt.Println()
	fmt.Println(yellow + "Available tools:" + reset)
	fmt.Println("  • project_info       - Get project information")
	fmt.Println("  • list_worktrees     - List all worktrees")
	fmt.Println("  • create_worktree    - Create a new worktree")
	fmt.Println("  • remove_worktree    - Remove a worktree")
	fmt.Println("  • list_databases     - List all databases")
	fmt.Println("  • clone_database     - Clone a database")
	fmt.Println("  • dump_database      - Create a database dump")
	fmt.Println("  • drop_database      - Drop a database")
	fmt.Println("  • import_database    - Import a database from dump")
}

func removeMCP(tool string) {
	var (
		reset = "\033[0m"
		green = "\033[32m"
		red   = "\033[31m"
	)

	var err error
	switch tool {
	case "claude", "claude-code":
		err = removeClaudeMCP()
	case "kimi", "kimi-cli":
		err = removeKimiMCP()
	case "codex":
		err = removeCodexMCP()
	default:
		fmt.Fprintf(os.Stderr, "Unknown tool: %s\n", tool)
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		os.Exit(1)
	}

	fmt.Printf("%s✓%s MCP config removed for %s\n", green, reset, tool)
}

func installClaudeMCP() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "settings.json")
	
	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"haive": map[string]interface{}{
				"command": "haive",
				"args":    []string{"--mcp"},
			},
		},
	}

	// Try to read existing config
	if data, err := os.ReadFile(configPath); err == nil {
		var existing map[string]interface{}
		if err := json.Unmarshal(data, &existing); err == nil {
			// Merge with existing
			if mcpServers, ok := existing["mcpServers"].(map[string]interface{}); ok {
				mcpServers["haive"] = config["mcpServers"].(map[string]interface{})["haive"]
				config["mcpServers"] = mcpServers
			} else {
				existing["mcpServers"] = config["mcpServers"]
			}
			config = existing
		}
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func removeClaudeMCP() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".claude", "settings.json")
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	if mcpServers, ok := config["mcpServers"].(map[string]interface{}); ok {
		delete(mcpServers, "haive")
		if len(mcpServers) == 0 {
			delete(config, "mcpServers")
		}
	}

	data, err = json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func installKimiMCP() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(homeDir, ".config", "kimi")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "mcp.json")
	
	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"haive": map[string]interface{}{
				"command": "haive",
				"args":    []string{"--mcp"},
			},
		},
	}

	// Try to read existing config
	if data, err := os.ReadFile(configPath); err == nil {
		var existing map[string]interface{}
		if err := json.Unmarshal(data, &existing); err == nil {
			if mcpServers, ok := existing["mcpServers"].(map[string]interface{}); ok {
				mcpServers["haive"] = config["mcpServers"].(map[string]interface{})["haive"]
				config["mcpServers"] = mcpServers
			} else {
				existing["mcpServers"] = config["mcpServers"]
			}
			config = existing
		}
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func removeKimiMCP() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".config", "kimi", "mcp.json")
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	if mcpServers, ok := config["mcpServers"].(map[string]interface{}); ok {
		delete(mcpServers, "haive")
		if len(mcpServers) == 0 {
			delete(config, "mcpServers")
		}
	}

	data, err = json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func installCodexMCP() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(homeDir, ".codex")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.json")
	
	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"haive": map[string]interface{}{
				"command": "haive",
				"args":    []string{"--mcp"},
			},
		},
	}

	// Try to read existing config
	if data, err := os.ReadFile(configPath); err == nil {
		var existing map[string]interface{}
		if err := json.Unmarshal(data, &existing); err == nil {
			if mcpServers, ok := existing["mcpServers"].(map[string]interface{}); ok {
				mcpServers["haive"] = config["mcpServers"].(map[string]interface{})["haive"]
				config["mcpServers"] = mcpServers
			} else {
				existing["mcpServers"] = config["mcpServers"]
			}
			config = existing
		}
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func removeCodexMCP() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(homeDir, ".codex", "config.json")
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	if mcpServers, ok := config["mcpServers"].(map[string]interface{}); ok {
		delete(mcpServers, "haive")
		if len(mcpServers) == 0 {
			delete(config, "mcpServers")
		}
	}

	data, err = json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func printMCPHelp() {
	var (
		reset  = "\033[0m"
		bold   = "\033[1m"
		cyan   = "\033[36m"
		green  = "\033[32m"
		yellow = "\033[33m"
	)

	fmt.Println()
	fmt.Println(cyan + "haive mcp" + reset + " - Manage MCP server configuration for AI assistants")
	fmt.Println()
	fmt.Println(bold + "Usage:" + reset)
	fmt.Println("  " + green + "haive mcp install <tool>" + reset + "   Install MCP config")
	fmt.Println("  " + green + "haive mcp remove <tool>" + reset + "    Remove MCP config")
	fmt.Println()
	fmt.Println(bold + "Supported tools:" + reset)
	fmt.Println("  " + yellow + "claude" + reset + "                Claude Code")
	fmt.Println("  " + yellow + "kimi" + reset + "                  Kimi CLI")
	fmt.Println("  " + yellow + "codex" + reset + "                 OpenAI Codex CLI")
	fmt.Println()
	fmt.Println(bold + "Examples:" + reset)
	fmt.Println("  " + green + "haive mcp install claude" + reset + "   # Install for Claude Code")
	fmt.Println("  " + green + "haive mcp install kimi" + reset + "     # Install for Kimi CLI")
	fmt.Println("  " + green + "haive mcp remove claude" + reset + "    # Remove from Claude Code")
	fmt.Println()
	fmt.Println(bold + "What this does:" + reset)
	fmt.Println("  Installs haive as an MCP server so your AI assistant can:")
	fmt.Println("  • List and manage worktrees")
	fmt.Println("  • Clone, dump, and import databases")
	fmt.Println("  • Get project information")
	fmt.Println()
}
