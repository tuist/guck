package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/fatih/color"
	"github.com/tuist/guck/internal/config"
	"github.com/tuist/guck/internal/daemon"
	"github.com/tuist/guck/internal/git"
	"github.com/tuist/guck/internal/mcp"
	"github.com/tuist/guck/internal/server"
	"github.com/tuist/guck/internal/state"
	"github.com/urfave/cli/v2"
)

var (
	successColor = color.New(color.FgGreen, color.Bold)
	infoColor    = color.New(color.FgCyan)
	errorColor   = color.New(color.FgRed, color.Bold)
	warningColor = color.New(color.FgYellow)
	urlColor     = color.New(color.FgBlue, color.Underline)
)

func main() {
	app := &cli.App{
		Name:  "guck",
		Usage: "A Git diff review tool with a web interface",
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "Start the server (run in foreground or use & to background)",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "port",
						Aliases: []string{"p"},
						Usage:   "Port to run the server on (defaults to random available port)",
					},
					&cli.StringFlag{
						Name:    "base",
						Aliases: []string{"b"},
						Usage:   "Base branch to compare against",
					},
				},
				Action: startServerForeground,
			},
			{
				Name:   "init",
				Usage:  "Initialize shell integration (outputs shell script to eval)",
				Action: printShellIntegration,
			},
			{
				Name:  "daemon",
				Usage: "Daemon management commands",
				Subcommands: []*cli.Command{
					{
						Name:  "start",
						Usage: "Start daemon for current repository",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "base",
								Aliases: []string{"b"},
								Usage:   "Override base branch",
							},
						},
						Action: startDaemon,
					},
					{
						Name:   "stop",
						Usage:  "Stop daemon for current repository",
						Action: stopDaemon,
					},
					{
						Name:   "stop-all",
						Usage:  "Stop all running daemons",
						Action: stopAllDaemons,
					},
					{
						Name:   "list",
						Usage:  "List all running daemons",
						Action: listDaemons,
					},
					{
						Name:   "cleanup",
						Usage:  "Clean up stale daemon entries",
						Action: cleanupDaemons,
					},
				},
			},
			{
				Name:  "config",
				Usage: "Configuration commands",
				Subcommands: []*cli.Command{
					{
						Name:      "set",
						Usage:     "Set a configuration value",
						ArgsUsage: "<key> <value>",
						Action:    setConfig,
					},
					{
						Name:      "get",
						Usage:     "Get a configuration value",
						ArgsUsage: "<key>",
						Action:    getConfig,
					},
					{
						Name:   "show",
						Usage:  "Show all configuration",
						Action: showConfig,
					},
				},
			},
			{
				Name:   "mcp",
				Usage:  "Start MCP (Model Context Protocol) server for LLM integrations",
				Action: mcpStdio,
			},
			{
				Name:  "dev",
				Usage: "Development utilities",
				Subcommands: []*cli.Command{
					{
						Name:  "sample-notes",
						Usage: "Add sample AI agent notes for testing/preview",
						Flags: []cli.Flag{
							&cli.IntFlag{
								Name:    "count",
								Aliases: []string{"c"},
								Usage:   "Number of sample notes to generate",
								Value:   5,
							},
						},
						Action: addSampleNotes,
					},
				},
			},
		},
		Action: openBrowser,
	}

	if err := app.Run(os.Args); err != nil {
		errorColor.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func startServerForeground(c *cli.Context) error {
	gitRepo, err := git.Open(".")
	if err != nil {
		return err
	}

	repoPath, err := gitRepo.RepoPath()
	if err != nil {
		return err
	}

	daemonMgr, err := daemon.NewManager()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	baseBranch := c.String("base")
	if baseBranch == "" {
		baseBranch = cfg.BaseBranch
	}

	port := c.Int("port")
	if port == 0 {
		port, err = daemonMgr.FindAvailablePort()
		if err != nil {
			return err
		}
	}

	daemonInfo := &daemon.Info{
		PID:        os.Getpid(),
		Port:       port,
		RepoPath:   repoPath,
		BaseBranch: baseBranch,
	}

	if err := daemonMgr.RegisterDaemon(daemonInfo); err != nil {
		return err
	}

	successColor.Printf("✓ Starting guck server for %s\n", repoPath)
	infoColor.Print("Server running on ")
	urlColor.Printf("http://localhost:%d\n", port)
	infoColor.Println("Press Ctrl+C to stop")

	return server.Start(port, baseBranch)
}

func printShellIntegration(c *cli.Context) error {
	script := `
# Guck shell integration

# Track the current git repository path
_GUCK_CURRENT_REPO=""

# Get the repository path for the current directory
_guck_get_repo_path() {
    if git rev-parse --show-toplevel >/dev/null 2>&1; then
        git rev-parse --show-toplevel 2>/dev/null
    fi
}

# Auto-start/stop daemons based on directory changes
_guck_auto_manage() {
    if ! command -v guck >/dev/null 2>&1; then
        return
    fi

    local new_repo
    new_repo=$(_guck_get_repo_path)

    # If we left a git repo, stop its daemon
    if [ -n "$_GUCK_CURRENT_REPO" ] && [ "$_GUCK_CURRENT_REPO" != "$new_repo" ]; then
        (cd "$_GUCK_CURRENT_REPO" && guck daemon stop >/dev/null 2>&1 &)
    fi

    # If we entered a git repo, start its daemon
    if [ -n "$new_repo" ] && [ "$_GUCK_CURRENT_REPO" != "$new_repo" ]; then
        guck daemon start >/dev/null 2>&1 &
    fi

    # Update the tracked repo path
    _GUCK_CURRENT_REPO="$new_repo"
}

# Hook into cd command
if [ -n "$ZSH_VERSION" ]; then
    chpwd_functions+=(_guck_auto_manage)
elif [ -n "$BASH_VERSION" ]; then
    _guck_original_cd=$(declare -f cd)
    cd() {
        builtin cd "$@"
        _guck_auto_manage
    }
fi

# Initialize for current directory if it's a git repo
_guck_auto_manage
`
	fmt.Println(script)
	return nil
}

func startDaemon(c *cli.Context) error {
	// Implementation similar to Rust version
	gitRepo, err := git.Open(".")
	if err != nil {
		return err
	}

	repoPath, err := gitRepo.RepoPath()
	if err != nil {
		return err
	}

	daemonMgr, err := daemon.NewManager()
	if err != nil {
		return err
	}

	// Check if daemon already running
	if info, _ := daemonMgr.GetDaemonForRepo(repoPath); info != nil {
		if daemonMgr.IsDaemonRunning(info.PID) {
			return nil
		}
		_ = daemonMgr.UnregisterDaemon(repoPath) // Ignore error, we'll register a new one
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	baseBranch := c.String("base")
	if baseBranch == "" {
		baseBranch = cfg.BaseBranch
	}

	port, err := daemonMgr.FindAvailablePort()
	if err != nil {
		return err
	}

	// Check if we're the daemon process
	if os.Getenv("GUCK_DAEMON") == "1" {
		daemonInfo := &daemon.Info{
			PID:        os.Getpid(),
			Port:       port,
			RepoPath:   repoPath,
			BaseBranch: baseBranch,
		}

		if err := daemonMgr.RegisterDaemon(daemonInfo); err != nil {
			return err
		}

		return server.Start(port, baseBranch)
	}

	// Spawn daemon process
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	logPath := daemonMgr.GetLogPath(repoPath)
	logFile, err := os.Create(logPath)
	if err != nil {
		return err
	}
	defer logFile.Close()

	args := []string{"daemon", "start"}
	if baseBranch != "" {
		args = append(args, "--base", baseBranch)
	}

	cmd := exec.Command(exe, args...)
	cmd.Env = append(os.Environ(), "GUCK_DAEMON=1")
	cmd.Dir = repoPath
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return err
	}

	successColor.Printf("✓ Started daemon for %s\n", repoPath)
	infoColor.Printf("  Port: %d | PID: %d\n", port, cmd.Process.Pid)
	return nil
}

func stopDaemon(c *cli.Context) error {
	gitRepo, err := git.Open(".")
	if err != nil {
		return err
	}

	repoPath, err := gitRepo.RepoPath()
	if err != nil {
		return err
	}

	daemonMgr, err := daemon.NewManager()
	if err != nil {
		return err
	}

	info, err := daemonMgr.GetDaemonForRepo(repoPath)
	if err != nil || info == nil {
		warningColor.Println("⚠ No daemon running for this repository")
		return nil
	}

	if err := daemonMgr.StopDaemon(info.PID); err != nil {
		return err
	}

	if err := daemonMgr.UnregisterDaemon(repoPath); err != nil {
		return err
	}

	successColor.Printf("✓ Stopped daemon for %s\n", repoPath)
	return nil
}

func stopAllDaemons(c *cli.Context) error {
	daemonMgr, err := daemon.NewManager()
	if err != nil {
		return err
	}

	daemons, err := daemonMgr.ListDaemons()
	if err != nil {
		return err
	}

	for _, info := range daemons {
		if daemonMgr.IsDaemonRunning(info.PID) {
			_ = daemonMgr.StopDaemon(info.PID)            // Best effort stop
			_ = daemonMgr.UnregisterDaemon(info.RepoPath) // Best effort cleanup
			successColor.Printf("✓ Stopped daemon for %s\n", info.RepoPath)
		}
	}

	return nil
}

func listDaemons(c *cli.Context) error {
	daemonMgr, err := daemon.NewManager()
	if err != nil {
		return err
	}

	if err := daemonMgr.CleanupStaleDaemons(); err != nil {
		return err
	}

	daemons, err := daemonMgr.ListDaemons()
	if err != nil {
		return err
	}

	if len(daemons) == 0 {
		warningColor.Println("⚠ No running daemons")
		return nil
	}

	infoColor.Println("Running daemons:")
	for _, info := range daemons {
		fmt.Printf("  %s - ", info.RepoPath)
		urlColor.Printf("http://localhost:%d", info.Port)
		fmt.Printf(" (PID: %d)\n", info.PID)
	}

	return nil
}

func cleanupDaemons(c *cli.Context) error {
	daemonMgr, err := daemon.NewManager()
	if err != nil {
		return err
	}

	if err := daemonMgr.CleanupStaleDaemons(); err != nil {
		return err
	}

	successColor.Println("✓ Cleaned up stale daemon entries")
	return nil
}

func openBrowser(c *cli.Context) error {
	gitRepo, err := git.Open(".")
	if err != nil {
		return err
	}

	repoPath, err := gitRepo.RepoPath()
	if err != nil {
		return err
	}

	daemonMgr, err := daemon.NewManager()
	if err != nil {
		return err
	}

	if err := daemonMgr.CleanupStaleDaemons(); err != nil {
		return err
	}

	info, err := daemonMgr.GetDaemonForRepo(repoPath)
	if err != nil || info == nil {
		return fmt.Errorf("no daemon running for this repository. Run 'guck daemon start' first")
	}

	if !daemonMgr.IsDaemonRunning(info.PID) {
		_ = daemonMgr.UnregisterDaemon(repoPath) // Clean up stale registration
		return fmt.Errorf("daemon is not running. Run 'guck daemon start' first")
	}

	url := fmt.Sprintf("http://localhost:%d", info.Port)
	infoColor.Print("Opening ")
	urlColor.Print(url)
	infoColor.Println(" in your browser...")

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/C", "start", url)
	default: // linux, freebsd, etc.
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}

func setConfig(c *cli.Context) error {
	if c.NArg() != 2 {
		return fmt.Errorf("requires exactly 2 arguments: key and value")
	}

	key := c.Args().Get(0)
	value := c.Args().Get(1)

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch key {
	case "base-branch":
		cfg.BaseBranch = value
		if err := cfg.Save(); err != nil {
			return err
		}
		successColor.Print("✓ Set ")
		infoColor.Print("base-branch")
		successColor.Printf(" to '%s'\n", value)
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	return nil
}

func getConfig(c *cli.Context) error {
	if c.NArg() != 1 {
		return fmt.Errorf("requires exactly 1 argument: key")
	}

	key := c.Args().Get(0)

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	switch key {
	case "base-branch":
		fmt.Println(cfg.BaseBranch)
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	return nil
}

func showConfig(c *cli.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	infoColor.Print("base-branch = ")
	successColor.Println(cfg.BaseBranch)
	return nil
}

func mcpStdio(c *cli.Context) error {
	// Start MCP server with stdio transport
	return mcp.StartStdioServer()
}

func addSampleNotes(c *cli.Context) error {
	gitRepo, err := git.Open(".")
	if err != nil {
		return err
	}

	repoPath, err := gitRepo.RepoPath()
	if err != nil {
		return err
	}

	branch, err := gitRepo.CurrentBranch()
	if err != nil {
		return err
	}

	commit, err := gitRepo.CurrentCommit()
	if err != nil {
		return err
	}

	count := c.Int("count")
	if count <= 0 {
		count = 5
	}

	mgr, err := state.NewManager()
	if err != nil {
		return err
	}

	sampleNotes := []struct {
		filePath   string
		lineNumber *int
		text       string
		author     string
		noteType   string
		metadata   map[string]string
	}{
		{
			filePath:   "main.go",
			lineNumber: intPtr(42),
			text:       "This function could benefit from better error handling. Consider wrapping errors with context using fmt.Errorf with %w verb for better error tracing.",
			author:     "claude",
			noteType:   "suggestion",
			metadata: map[string]string{
				"model":      "claude-sonnet-4",
				"context":    "code_review",
				"confidence": "high",
			},
		},
		{
			filePath:   "internal/server/server.go",
			lineNumber: intPtr(120),
			text:       "The HTTP handler implements a proper REST API pattern. The use of gorilla/mux provides clean routing and the error handling follows Go best practices.",
			author:     "claude",
			noteType:   "explanation",
			metadata: map[string]string{
				"model":   "claude-sonnet-4",
				"context": "documentation",
			},
		},
		{
			filePath:   "internal/git/git.go",
			lineNumber: nil,
			text:       "This module abstracts Git operations effectively. The design allows for easy testing and mocking. Consider adding integration tests for complex Git scenarios.",
			author:     "copilot",
			noteType:   "rationale",
			metadata: map[string]string{
				"model":   "gpt-4",
				"context": "architecture_review",
			},
		},
		{
			filePath:   "internal/state/state.go",
			lineNumber: intPtr(85),
			text:       "The state management uses a file-based approach which is simple and reliable. For larger datasets, consider adding indexing or using a lightweight database like SQLite.",
			author:     "claude",
			noteType:   "suggestion",
			metadata: map[string]string{
				"model":    "claude-sonnet-4",
				"context":  "performance_review",
				"priority": "low",
			},
		},
		{
			filePath:   "README.md",
			lineNumber: nil,
			text:       "Documentation is clear and well-structured. The installation instructions cover all major platforms and the usage examples are practical.",
			author:     "copilot",
			noteType:   "explanation",
			metadata: map[string]string{
				"model":   "gpt-4",
				"context": "documentation_review",
			},
		},
		{
			filePath:   "internal/mcp/mcp.go",
			lineNumber: intPtr(200),
			text:       "The MCP implementation follows the protocol specification correctly. This enables seamless integration with AI agents like Claude and GitHub Copilot for code review automation.",
			author:     "claude",
			noteType:   "explanation",
			metadata: map[string]string{
				"model":      "claude-sonnet-4",
				"context":    "integration_review",
				"importance": "high",
			},
		},
	}

	added := 0
	for i := 0; i < count && i < len(sampleNotes); i++ {
		note := sampleNotes[i]
		_, err := mgr.AddNote(
			repoPath,
			branch,
			commit,
			note.filePath,
			note.lineNumber,
			note.text,
			note.author,
			note.noteType,
			note.metadata,
		)
		if err != nil {
			warningColor.Printf("⚠ Failed to add note: %v\n", err)
			continue
		}
		added++
	}

	successColor.Printf("✓ Added %d sample AI agent note(s)\n", added)
	infoColor.Printf("  Repository: %s\n", repoPath)
	infoColor.Printf("  Branch: %s\n", branch)
	infoColor.Printf("  Commit: %s\n", commit[:7])
	infoColor.Println("\nRefresh your browser to see the notes in the UI")

	return nil
}

func intPtr(i int) *int {
	return &i
}
