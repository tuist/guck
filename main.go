package main

import (
	"encoding/json"
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
			{
				Name:  "comments",
				Usage: "Code review comments management",
				Subcommands: []*cli.Command{
					{
						Name:  "list",
						Usage: "List code review comments",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "repo",
								Aliases: []string{"r"},
								Usage:   "Repository path (defaults to current directory)",
								Value:   ".",
							},
							&cli.StringFlag{
								Name:    "branch",
								Aliases: []string{"b"},
								Usage:   "Filter by branch name",
							},
							&cli.StringFlag{
								Name:    "commit",
								Aliases: []string{"c"},
								Usage:   "Filter by commit hash",
							},
							&cli.StringFlag{
								Name:    "file",
								Aliases: []string{"f"},
								Usage:   "Filter by file path",
							},
							&cli.BoolFlag{
								Name:    "resolved",
								Aliases: []string{"R"},
								Usage:   "Show only resolved comments",
							},
							&cli.BoolFlag{
								Name:    "unresolved",
								Aliases: []string{"U"},
								Usage:   "Show only unresolved comments",
							},
							&cli.StringFlag{
								Name:    "format",
								Aliases: []string{"o"},
								Usage:   "Output format: json, toon (default: human-readable)",
								Value:   "",
							},
						},
						Action: listComments,
					},
					{
						Name:      "resolve",
						Usage:     "Mark a comment as resolved",
						ArgsUsage: "<comment-id>",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "repo",
								Aliases: []string{"r"},
								Usage:   "Repository path (defaults to current directory)",
								Value:   ".",
							},
							&cli.StringFlag{
								Name:     "by",
								Aliases:  []string{"u"},
								Usage:    "Who is resolving the comment",
								Required: true,
							},
							&cli.StringFlag{
								Name:    "format",
								Aliases: []string{"o"},
								Usage:   "Output format: json, toon (default: human-readable)",
								Value:   "",
							},
						},
						Action: resolveComment,
					},
				},
			},
			{
				Name:  "notes",
				Usage: "AI agent notes management",
				Subcommands: []*cli.Command{
					{
						Name:  "add",
						Usage: "Add an AI agent note",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "repo",
								Aliases: []string{"r"},
								Usage:   "Repository path (defaults to current directory)",
								Value:   ".",
							},
							&cli.StringFlag{
								Name:     "file",
								Aliases:  []string{"f"},
								Usage:    "File path relative to repository root",
								Required: true,
							},
							&cli.IntFlag{
								Name:    "line",
								Aliases: []string{"l"},
								Usage:   "Line number for inline notes",
							},
							&cli.StringFlag{
								Name:     "text",
								Aliases:  []string{"t"},
								Usage:    "Note content (markdown supported)",
								Required: true,
							},
							&cli.StringFlag{
								Name:     "author",
								Aliases:  []string{"a"},
								Usage:    "Author identifier (e.g., 'claude', 'copilot', 'gpt-4')",
								Required: true,
							},
							&cli.StringFlag{
								Name:    "type",
								Aliases: []string{"T"},
								Usage:   "Note type (explanation, rationale, suggestion)",
								Value:   "explanation",
							},
							&cli.StringSliceFlag{
								Name:    "metadata",
								Aliases: []string{"m"},
								Usage:   "Metadata as key=value pairs",
							},
							&cli.StringFlag{
								Name:    "format",
								Aliases: []string{"o"},
								Usage:   "Output format: json, toon (default: human-readable)",
								Value:   "",
							},
						},
						Action: addNote,
					},
					{
						Name:  "list",
						Usage: "List AI agent notes",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "repo",
								Aliases: []string{"r"},
								Usage:   "Repository path (defaults to current directory)",
								Value:   ".",
							},
							&cli.StringFlag{
								Name:    "branch",
								Aliases: []string{"b"},
								Usage:   "Filter by branch name",
							},
							&cli.StringFlag{
								Name:    "commit",
								Aliases: []string{"c"},
								Usage:   "Filter by commit hash",
							},
							&cli.StringFlag{
								Name:    "file",
								Aliases: []string{"f"},
								Usage:   "Filter by file path",
							},
							&cli.StringFlag{
								Name:    "author",
								Aliases: []string{"a"},
								Usage:   "Filter by author",
							},
							&cli.BoolFlag{
								Name:    "dismissed",
								Aliases: []string{"D"},
								Usage:   "Show only dismissed notes",
							},
							&cli.BoolFlag{
								Name:    "active",
								Aliases: []string{"A"},
								Usage:   "Show only active (non-dismissed) notes",
							},
							&cli.StringFlag{
								Name:    "format",
								Aliases: []string{"o"},
								Usage:   "Output format: json, toon (default: human-readable)",
								Value:   "",
							},
						},
						Action: listNotes,
					},
					{
						Name:      "dismiss",
						Usage:     "Dismiss an AI agent note",
						ArgsUsage: "<note-id>",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "repo",
								Aliases: []string{"r"},
								Usage:   "Repository path (defaults to current directory)",
								Value:   ".",
							},
							&cli.StringFlag{
								Name:     "by",
								Aliases:  []string{"u"},
								Usage:    "Who is dismissing the note",
								Required: true,
							},
							&cli.StringFlag{
								Name:    "format",
								Aliases: []string{"o"},
								Usage:   "Output format: json, toon (default: human-readable)",
								Value:   "",
							},
						},
						Action: dismissNote,
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
        (guck daemon start >/dev/null 2>&1 &)
        if [ $? -eq 0 ]; then
            printf "\033[1;36m→\033[0m Run \033[1;34mguck\033[0m to inspect the project's diff\n"
        fi
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

// CLI handlers for comments and notes

func listComments(c *cli.Context) error {
	repoPath := c.String("repo")
	branch := c.String("branch")
	commit := c.String("commit")
	filePath := c.String("file")
	format := c.String("format")

	// Build params
	params := mcp.ListCommentsParams{
		RepoPath: repoPath,
	}

	if branch != "" {
		params.Branch = &branch
	}
	if commit != "" {
		params.Commit = &commit
	}
	if filePath != "" {
		params.FilePath = &filePath
	}

	// Handle resolved filter
	if c.Bool("resolved") {
		resolved := true
		params.Resolved = &resolved
	} else if c.Bool("unresolved") {
		resolved := false
		params.Resolved = &resolved
	}

	// Convert to JSON and call MCP function
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}

	result, err := mcp.ListComments(json.RawMessage(paramsJSON))
	if err != nil {
		return err
	}

	return outputResult(result, format)
}

func resolveComment(c *cli.Context) error {
	if c.NArg() != 1 {
		return fmt.Errorf("requires exactly 1 argument: comment-id")
	}

	commentID := c.Args().Get(0)
	repoPath := c.String("repo")
	resolvedBy := c.String("by")
	format := c.String("format")

	params := mcp.ResolveCommentParams{
		RepoPath:   repoPath,
		CommentID:  commentID,
		ResolvedBy: resolvedBy,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}

	result, err := mcp.ResolveComment(json.RawMessage(paramsJSON))
	if err != nil {
		return err
	}

	return outputResult(result, format)
}

func addNote(c *cli.Context) error {
	repoPath := c.String("repo")
	filePath := c.String("file")
	text := c.String("text")
	author := c.String("author")
	noteType := c.String("type")
	format := c.String("format")

	// Get current branch and commit
	gitRepo, err := git.Open(repoPath)
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

	params := mcp.AddNoteParams{
		RepoPath: repoPath,
		Branch:   branch,
		Commit:   commit,
		FilePath: filePath,
		Text:     text,
		Author:   author,
		Type:     noteType,
	}

	// Handle line number
	if c.IsSet("line") {
		line := c.Int("line")
		params.LineNumber = &line
	}

	// Handle metadata
	if c.IsSet("metadata") {
		metadata := make(map[string]string)
		for _, pair := range c.StringSlice("metadata") {
			parts := splitKeyValue(pair)
			if len(parts) == 2 {
				metadata[parts[0]] = parts[1]
			}
		}
		if len(metadata) > 0 {
			params.Metadata = metadata
		}
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}

	result, err := mcp.AddNote(json.RawMessage(paramsJSON))
	if err != nil {
		return err
	}

	return outputResult(result, format)
}

func listNotes(c *cli.Context) error {
	repoPath := c.String("repo")
	branch := c.String("branch")
	commit := c.String("commit")
	filePath := c.String("file")
	author := c.String("author")
	format := c.String("format")

	params := mcp.ListNotesParams{
		RepoPath: repoPath,
	}

	if branch != "" {
		params.Branch = &branch
	}
	if commit != "" {
		params.Commit = &commit
	}
	if filePath != "" {
		params.FilePath = &filePath
	}
	if author != "" {
		params.Author = &author
	}

	// Handle dismissed filter
	if c.Bool("dismissed") {
		dismissed := true
		params.Dismissed = &dismissed
	} else if c.Bool("active") {
		dismissed := false
		params.Dismissed = &dismissed
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}

	result, err := mcp.ListNotes(json.RawMessage(paramsJSON))
	if err != nil {
		return err
	}

	return outputResult(result, format)
}

func dismissNote(c *cli.Context) error {
	if c.NArg() != 1 {
		return fmt.Errorf("requires exactly 1 argument: note-id")
	}

	noteID := c.Args().Get(0)
	repoPath := c.String("repo")
	dismissedBy := c.String("by")
	format := c.String("format")

	params := mcp.DismissNoteParams{
		RepoPath:    repoPath,
		NoteID:      noteID,
		DismissedBy: dismissedBy,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}

	result, err := mcp.DismissNote(json.RawMessage(paramsJSON))
	if err != nil {
		return err
	}

	return outputResult(result, format)
}

// Helper functions

func outputResult(result interface{}, format string) error {
	switch format {
	case "json":
		return outputJSON(result)
	case "toon":
		return outputToon(result)
	default:
		return outputHumanReadable(result)
	}
}

func outputJSON(result interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func outputToon(result interface{}) error {
	// Toon format: https://github.com/toon-format/toon
	// This is a simple table format
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return fmt.Errorf("cannot convert result to toon format")
	}

	// Check if it's a list result with typed comments
	if commentsRaw, ok := resultMap["comments"]; ok {
		// Try typed slice first
		if comments, ok := commentsRaw.([]mcp.CommentResult); ok {
			return outputCommentResultsAsToon(comments)
		}
		// Try interface slice
		if comments, ok := commentsRaw.([]interface{}); ok {
			return outputCommentsAsToon(comments)
		}
	}

	// Check if it's a list result with typed notes
	if notesRaw, ok := resultMap["notes"]; ok {
		// Try typed slice first
		if notes, ok := notesRaw.([]mcp.NoteResult); ok {
			return outputNoteResultsAsToon(notes)
		}
		// Try interface slice
		if notes, ok := notesRaw.([]interface{}); ok {
			return outputNotesAsToon(notes)
		}
	}

	// For simple results, just output as key-value pairs
	for k, v := range resultMap {
		fmt.Printf("%s\t%v\n", k, v)
	}
	return nil
}

func outputCommentResultsAsToon(comments []mcp.CommentResult) error {
	if len(comments) == 0 {
		fmt.Println("# No comments found")
		return nil
	}

	fmt.Println("id\tfile\tline\tresolved\ttext")
	for _, comment := range comments {
		id := comment.ID
		file := comment.FilePath
		line := ""
		if comment.LineNumber != nil {
			line = fmt.Sprintf("%d", *comment.LineNumber)
		}
		resolved := comment.Resolved
		text := truncate(comment.Text, 50)

		fmt.Printf("%s\t%s\t%s\t%v\t%s\n", id, file, line, resolved, text)
	}
	return nil
}

func outputCommentsAsToon(comments []interface{}) error {
	if len(comments) == 0 {
		fmt.Println("# No comments found")
		return nil
	}

	fmt.Println("id\tfile\tline\tresolved\ttext")
	for _, item := range comments {
		comment, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		id := comment["id"]
		file := comment["file_path"]
		line := ""
		if ln, ok := comment["line_number"]; ok && ln != nil {
			line = fmt.Sprintf("%v", ln)
		}
		resolved := comment["resolved"]
		text := truncate(fmt.Sprintf("%v", comment["text"]), 50)

		fmt.Printf("%s\t%s\t%s\t%v\t%s\n", id, file, line, resolved, text)
	}
	return nil
}

func outputNoteResultsAsToon(notes []mcp.NoteResult) error {
	if len(notes) == 0 {
		fmt.Println("# No notes found")
		return nil
	}

	fmt.Println("id\tfile\tline\tauthor\ttype\tdismissed\ttext")
	for _, note := range notes {
		id := note.ID
		file := note.FilePath
		line := ""
		if note.LineNumber != nil {
			line = fmt.Sprintf("%d", *note.LineNumber)
		}
		author := note.Author
		noteType := note.Type
		dismissed := note.Dismissed
		text := truncate(note.Text, 50)

		fmt.Printf("%s\t%s\t%s\t%s\t%s\t%v\t%s\n", id, file, line, author, noteType, dismissed, text)
	}
	return nil
}

func outputNotesAsToon(notes []interface{}) error {
	if len(notes) == 0 {
		fmt.Println("# No notes found")
		return nil
	}

	fmt.Println("id\tfile\tline\tauthor\ttype\tdismissed\ttext")
	for _, item := range notes {
		note, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		id := note["id"]
		file := note["file_path"]
		line := ""
		if ln, ok := note["line_number"]; ok && ln != nil {
			line = fmt.Sprintf("%v", ln)
		}
		author := note["author"]
		noteType := note["type"]
		dismissed := note["dismissed"]
		text := truncate(fmt.Sprintf("%v", note["text"]), 50)

		fmt.Printf("%s\t%s\t%s\t%s\t%s\t%v\t%s\n", id, file, line, author, noteType, dismissed, text)
	}
	return nil
}

func outputHumanReadable(result interface{}) error {
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return outputJSON(result)
	}

	// Check if it's a list result with comments
	if comments, ok := resultMap["comments"].([]mcp.CommentResult); ok {
		count := resultMap["count"]
		infoColor.Printf("Found %v comment(s):\n\n", count)

		for _, comment := range comments {
			if comment.Resolved {
				successColor.Print("✓ ")
			} else {
				warningColor.Print("• ")
			}

			fmt.Printf("[%s] ", comment.ID[:8])
			urlColor.Print(comment.FilePath)
			if comment.LineNumber != nil {
				fmt.Printf(":%d", *comment.LineNumber)
			}
			fmt.Println()

			fmt.Printf("  %s\n", comment.Text)

			if comment.Resolved {
				infoColor.Printf("  Resolved by %s\n", comment.ResolvedBy)
			}
			fmt.Println()
		}
		return nil
	}

	// Check if it's a list result with notes
	if notes, ok := resultMap["notes"].([]mcp.NoteResult); ok {
		count := resultMap["count"]
		infoColor.Printf("Found %v note(s):\n\n", count)

		for _, note := range notes {
			if note.Dismissed {
				color.New(color.Faint).Print("✗ ")
			} else {
				infoColor.Print("📝 ")
			}

			fmt.Printf("[%s] ", note.ID[:8])
			urlColor.Print(note.FilePath)
			if note.LineNumber != nil {
				fmt.Printf(":%d", *note.LineNumber)
			}
			fmt.Printf(" (%s)\n", note.Author)

			fmt.Printf("  Type: %s\n", note.Type)
			fmt.Printf("  %s\n", note.Text)

			if note.Dismissed {
				infoColor.Printf("  Dismissed by %s\n", note.DismissedBy)
			}
			fmt.Println()
		}
		return nil
	}

	// For simple success results
	if success, ok := resultMap["success"].(bool); ok && success {
		successColor.Println("✓ Operation completed successfully")
		for k, v := range resultMap {
			if k != "success" {
				infoColor.Printf("  %s: %v\n", k, v)
			}
		}
		return nil
	}

	// Default: output as JSON
	return outputJSON(result)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func splitKeyValue(pair string) []string {
	parts := make([]string, 0, 2)
	idx := -1
	for i, ch := range pair {
		if ch == '=' {
			idx = i
			break
		}
	}
	if idx == -1 {
		return []string{pair}
	}
	parts = append(parts, pair[:idx])
	parts = append(parts, pair[idx+1:])
	return parts
}
