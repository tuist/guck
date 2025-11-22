package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/tuist/guck/internal/config"
	"github.com/tuist/guck/internal/daemon"
	"github.com/tuist/guck/internal/git"
	"github.com/tuist/guck/internal/mcp"
	"github.com/tuist/guck/internal/server"
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
				Name:  "mcp",
				Usage: "MCP (Model Context Protocol) server for LLM integrations",
				Subcommands: []*cli.Command{
					{
						Name:   "stdio",
						Usage:  "Start MCP server with stdio transport (for integration with Claude Code, etc.)",
						Action: mcpStdio,
					},
					{
						Name:   "list-tools",
						Usage:  "List available MCP tools (legacy)",
						Action: mcpListTools,
					},
					{
						Name:      "call-tool",
						Usage:     "Call an MCP tool (legacy)",
						ArgsUsage: "<tool-name>",
						Action:    mcpCallTool,
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
_guck_auto_start() {
    if command -v guck >/dev/null 2>&1; then
        if git rev-parse --git-dir >/dev/null 2>&1; then
            guck daemon start >/dev/null 2>&1 &
        fi
    fi
}

# Hook into cd command
if [ -n "$ZSH_VERSION" ]; then
    chpwd_functions+=(/_guck_auto_start)
elif [ -n "$BASH_VERSION" ]; then
    _guck_original_cd=$(declare -f cd)
    cd() {
        builtin cd "$@"
        _guck_auto_start
    }
fi

# Start daemon for current directory if it's a git repo
_guck_auto_start
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
			_ = daemonMgr.StopDaemon(info.PID)               // Best effort stop
			_ = daemonMgr.UnregisterDaemon(info.RepoPath)     // Best effort cleanup
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
	switch {
	case exec.Command("xdg-open", "--version").Run() == nil:
		cmd = exec.Command("xdg-open", url)
	case exec.Command("open", "--version").Run() == nil:
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("cmd", "/C", "start", url)
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
	// Get current working directory
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Start MCP server with stdio transport
	return mcp.StartStdioServer(workingDir)
}

func mcpListTools(c *cli.Context) error {
	tools := mcp.ListTools()
	data, err := json.MarshalIndent(tools, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode tools: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func mcpCallTool(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("tool name required")
	}

	toolName := c.Args().Get(0)

	// Get current working directory
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Read params from stdin
	var params json.RawMessage
	if err := json.NewDecoder(os.Stdin).Decode(&params); err != nil {
		return fmt.Errorf("failed to parse params: %w", err)
	}

	var result interface{}
	var toolErr error

	switch toolName {
	case "list_comments":
		result, toolErr = mcp.ListComments(params, workingDir)
	case "resolve_comment":
		result, toolErr = mcp.ResolveComment(params, workingDir)
	default:
		return fmt.Errorf("unknown tool: %s", toolName)
	}

	response := map[string]interface{}{}
	if toolErr != nil {
		response["error"] = map[string]interface{}{
			"code":    500,
			"message": toolErr.Error(),
		}
	} else {
		response["result"] = result
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode response: %w", err)
	}

	fmt.Println(string(data))
	return nil
}
