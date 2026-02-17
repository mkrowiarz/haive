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
	"github.com/mkrowiarz/mcp-symfony-stack/internal/core/types"
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
	// ANSI color codes
	var (
		reset   = "\033[0m"
		bold    = "\033[1m"
		dim     = "\033[2m"
		cyan    = "\033[36m"
		green   = "\033[32m"
		yellow  = "\033[33m"
		magenta = "\033[35m"
		gray    = "\033[90m"
	)

	fmt.Println()
	fmt.Println(cyan + "haive" + reset + " - Development Environment Manager for Docker Compose-based development")
	fmt.Println()
	fmt.Println(bold + "Usage:" + reset)
	fmt.Println("  " + green + "haive" + reset + "                   Run interactive TUI (default)")
	fmt.Println("  " + green + "haive --mcp" + reset + "           Run as MCP server for Claude Code")
	fmt.Println("  " + green + "haive <command> [flags]" + reset + "  Run specific command")
	fmt.Println()
	fmt.Println(bold + "Commands:" + reset)
	fmt.Println("  " + yellow + "init" + reset + "                  Generate config for current project")
	fmt.Println("  " + yellow + "checkout <branch>" + reset + "     Switch git branch and database")
	fmt.Println("  " + yellow + "switch" + reset + "                Switch database for current branch")
	fmt.Println("  " + yellow + "worktree <cmd>" + reset + "        Manage git worktrees")
	fmt.Println("  " + yellow + "help" + reset + "                  Show this help message")
	fmt.Println()
	fmt.Println(bold + "Init Flags:" + reset)
	fmt.Println("  " + magenta + "--write, -w" + reset + "           Write config to .haive/config.json")
	fmt.Println("  " + magenta + "--namespace, -n" + reset + "       Wrap config in \"haive\" namespace")
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
	fmt.Println("  " + yellow + "serve" + reset + "                  Start app container for current worktree")
	fmt.Println("  " + yellow + "serve stop" + reset + "             Stop app container for current worktree")
	fmt.Println()
	fmt.Println(bold + "Examples:" + reset)
	fmt.Println("  " + green + "haive init --write" + reset + "                 # Create config file")
	fmt.Println("  " + green + "haive init --namespace --write" + reset + "     # Create namespaced config")
	fmt.Println("  " + green + "haive checkout feature/x --create" + reset + "  # Create branch with new db")
	fmt.Println("  " + green + "haive checkout main" + reset + "                # Switch to main branch+db")
	fmt.Println("  " + green + "haive switch" + reset + "                       # Switch db for current branch")
	fmt.Println("  " + green + "haive worktree list" + reset + "                # List worktrees")
	fmt.Println("  " + green + "haive worktree create feature/x" + reset + "    # Create worktree")
	fmt.Println("  " + green + "haive worktree remove feature/x" + reset + "    # Remove worktree")
	fmt.Println("  " + green + "haive serve" + reset + "                         # Start worktree app")
	fmt.Println("  " + green + "haive serve stop" + reset + "                      # Stop worktree app")
	fmt.Println()
	fmt.Println(green + "Config file locations" + reset + " (checked in order):")
	fmt.Println("  1. " + bold + ".claude/project.json" + reset + " (recommended)")
	fmt.Println("  2. " + gray + ".haive/config.json" + reset)
	fmt.Println("  3. " + gray + ".haive.json" + reset)
	fmt.Println()
	fmt.Println(dim + "For more information:" + reset + " https://github.com/mkrowiarz/mcp-symfony-stack")
	fmt.Println()
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
			fmt.Fprintf(os.Stderr, "Remove it first or use 'haive init' to preview.\n")
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

func handleServe(args []string) {
	// ANSI color codes
	var (
		reset  = "\033[0m"
		red    = "\033[31m"
		green  = "\033[32m"
		yellow = "\033[33m"
		cyan   = "\033[36m"
		dim    = "\033[2m"
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
		// Check if it's a missing config error
		if cmdErr, ok := err.(*types.CommandError); ok {
			fmt.Println()
			fmt.Printf("%s✗ %s%s\n", red, cmdErr.Message, reset)
			fmt.Println()

			if cmdErr.Code == types.ErrConfigMissing && strings.Contains(cmdErr.Message, "compose.worktree.yaml") {
				fmt.Printf("%sSetup Instructions:%s\n", cyan, reset)
				fmt.Println()
				fmt.Println("1. Create " + yellow + "compose.worktree.yaml" + reset + " in your worktree root:")
				fmt.Println()
				fmt.Println(dim + "   services:" + reset)
				fmt.Println(dim + "     app:" + reset)
				fmt.Println(dim + "       ports: []  # OrbStack provides hostname" + reset)
				fmt.Println(dim + "       volumes:" + reset)
				fmt.Println(dim + "         - .:/app:delegated" + reset)
				fmt.Println(dim + "         - /app/var/cache" + reset)
				fmt.Println(dim + "         - /app/vendor" + reset)
				fmt.Println(dim + "       environment:" + reset)
				fmt.Println(dim + "         DATABASE_URL: \"mysql://user:pass@db:3306/mydb_wt_branch\"" + reset)
				fmt.Println(dim + "   networks:" + reset)
				fmt.Println(dim + "     local:" + reset)
				fmt.Println(dim + "       external: true" + reset)
				fmt.Println(dim + "       name: myproject_local" + reset)
				fmt.Println()
				fmt.Println("2. Run " + green + "composer install" + reset + " && " + green + "npm install" + reset + " in the worktree")
				fmt.Println()
				fmt.Println("3. Run " + green + "haive serve" + reset + " to start the container")
				fmt.Println()
				fmt.Printf("%sFor complete template:%s See README.md\n", dim, reset)
				fmt.Println()
			}

			os.Exit(1)
		}

		// Generic error
		fmt.Println()
		fmt.Printf("%s✗ Error:%s %v\n", red, reset, err)
		fmt.Println()
		os.Exit(1)
	}

	fmt.Printf("%s✓%s Worktree: %s\n", green, reset, result.Branch)
	fmt.Printf("%s✓%s Project: %s\n", green, reset, result.ProjectName)
	fmt.Printf("%s✓%s Started: app container\n", green, reset)
	fmt.Printf("%s✓%s URL: %s%s%s\n", green, reset, cyan, result.URL, reset)
}
