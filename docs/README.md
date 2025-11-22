# Guck Documentation

Comprehensive documentation for Guck, a Git diff review tool with web interface and MCP server integration.

## Table of Contents

- [Installation](#installation)
- [Setup](#setup)
- [Usage](#usage)
  - [Web Interface](#web-interface)
  - [Daemon Management](#daemon-management)
  - [Configuration](#configuration)
- [MCP Server Integration](#mcp-server-integration)
  - [Claude Code Integration](#claude-code-integration)
  - [Available Tools](#available-tools)
  - [Usage Examples](#usage-examples)
- [Development](#development)
- [Architecture](#architecture)

## Installation

### Using mise (recommended)

```bash
mise use -g guck@latest
```

### Download binary

Download the latest release for your platform from the [releases page](https://github.com/tuist/guck/releases).

Binaries are available for:
- Linux (amd64, arm64)
- macOS (amd64/Intel, arm64/Apple Silicon)
- Windows (amd64)

### From source

```bash
git clone https://github.com/tuist/guck
cd guck
go build -o guck .
```

## Setup

After installing, add this to your shell configuration file (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
# For Bash/Zsh
eval "$(guck init)"
```

This enables automatic daemon management when entering/leaving git repositories.

## Usage

### Web Interface

Once set up, simply navigate to any git repository:

```bash
cd /path/to/your/repo
# Guck daemon automatically starts in the background
```

Then open the web interface:

```bash
guck
# Opens your default browser to view the diff
```

The daemon will:
- Start automatically when you `cd` into a git repository
- Allocate a unique port for each repository
- Keep running in the background
- Persist across terminal sessions

### Daemon Management

```bash
# Start the daemon manually
guck daemon start

# Stop the daemon for the current repo
guck daemon stop

# Stop all guck daemons
guck daemon stop-all

# List all running guck servers
guck daemon list

# Clean up stale daemon entries
guck daemon cleanup
```

### Configuration

```bash
# Set the base branch (default: main)
guck config set base-branch develop

# Get current base branch
guck config get base-branch

# Show all configuration
guck config show
```

#### Configuration Files

Guck stores its data in XDG-compliant directories:

- **State**: `~/.local/state/guck/` - Port mappings, daemon PIDs, viewed files, comments
- **Config**: `~/.config/guck/` - User configuration (base branch, etc.)

## MCP Server Integration

Guck includes a Model Context Protocol (MCP) server that allows LLMs like Claude to interact with code review comments. This enables AI assistants to query comments, resolve issues, and integrate with your code review workflow.

### Claude Code Integration

To integrate Guck with Claude Code (or Claude Desktop), add the following to your MCP configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`

**Linux**: `~/.config/Claude/claude_desktop_config.json`

**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

#### Using mise (recommended)

If you have [mise](https://mise.jdx.dev/) installed:

```json
{
  "mcpServers": {
    "guck": {
      "command": "mise",
      "args": ["x", "ubi:tuist/guck@latest", "--", "guck", "mcp", "stdio"]
    }
  }
}
```

This ensures you always use the latest version of Guck without needing to specify a full path.

#### Using a direct path

```json
{
  "mcpServers": {
    "guck": {
      "command": "/path/to/guck",
      "args": ["mcp", "stdio"]
    }
  }
}
```

After adding this configuration, restart Claude Code/Desktop. Guck will be available as an MCP server, allowing Claude to:
- List code review comments from your repositories
- Filter comments by file, branch, commit, or resolution status
- Mark comments as resolved
- Track who resolved comments and when

### Available Tools

#### `list_comments`

Lists code review comments with optional filtering.

**Parameters:**
- `repo_path` (optional): Path to the repository (defaults to current working directory)
- `branch` (optional): Filter by branch name
- `commit` (optional): Filter by commit hash
- `file_path` (optional): Filter by file path
- `resolved` (optional): Filter by resolution status (true/false)

**Example Request:**
```json
{
  "name": "list_comments",
  "arguments": {
    "file_path": "main.go",
    "resolved": false
  }
}
```

**Example Response:**
```json
{
  "comments": [
    {
      "id": "1234567890-0",
      "file_path": "main.go",
      "line_number": 42,
      "text": "Consider adding error handling here",
      "timestamp": 1234567890,
      "branch": "feature/new-feature",
      "commit": "abc123def456...",
      "resolved": false
    }
  ],
  "count": 1,
  "repo_path": "/path/to/repo"
}
```

#### `resolve_comment`

Marks a comment as resolved and tracks who resolved it.

**Parameters:**
- `comment_id` (required): The ID of the comment to resolve
- `resolved_by` (required): Identifier of who/what is resolving the comment (e.g., "claude", "copilot", "user-name")
- `repo_path` (optional): Path to the repository (defaults to current working directory)

**Example Request:**
```json
{
  "name": "resolve_comment",
  "arguments": {
    "comment_id": "1234567890-0",
    "resolved_by": "claude"
  }
}
```

**Example Response:**
```json
{
  "success": true,
  "comment_id": "1234567890-0",
  "repo_path": "/path/to/repo",
  "resolved_by": "claude"
}
```

### Usage Examples

#### Using with Claude Code

Once configured, you can ask Claude to interact with your code reviews:

- "List all unresolved comments in this repository"
- "Show me comments on main.go"
- "Resolve the comment with ID 1234567890-0"
- "What comments were added on the feature/auth branch?"

#### Legacy Command-line Usage

For testing or scripting, you can also use the legacy command-line interface:

```bash
# List available tools
guck mcp list-tools

# List all comments (JSON via stdin)
echo '{}' | guck mcp call-tool list_comments

# Filter by file
echo '{"file_path": "main.go"}' | guck mcp call-tool list_comments

# Filter unresolved comments
echo '{"resolved": false}' | guck mcp call-tool list_comments

# Resolve a comment
echo '{"comment_id": "1234567890-0", "resolved_by": "me"}' | guck mcp call-tool resolve_comment
```

## Development

### Prerequisites

- Go 1.23 or later
- Git

### Building

```bash
go build -o guck .
```

### Running locally

```bash
# Start server in foreground
go run . start --port 3456

# Or use daemon mode
go run . daemon start
# In another terminal:
go run .
```

### Running tests

```bash
go test ./...
```

## Architecture

### Project Structure

```
.
├── main.go              # CLI entry point and command handlers
├── internal/
│   ├── config/          # Configuration management (XDG-compliant)
│   ├── daemon/          # Daemon process management and port allocation
│   ├── git/             # Git operations and diff parsing (using go-git)
│   ├── mcp/             # MCP server implementation
│   │   ├── mcp.go      # Legacy tool functions (list_comments, resolve_comment)
│   │   └── server.go   # JSON-RPC 2.0 stdio server for MCP protocol
│   ├── server/          # HTTP server and REST API
│   │   ├── server.go   # Server logic and handlers
│   │   └── static/     # Web UI (HTML/CSS/React)
│   └── state/           # State persistence (comments, viewed files)
├── docs/                # Documentation
└── .github/workflows/   # CI/CD for releases
```

### Key Components

#### Git Integration (`internal/git`)
- Uses `go-git` library for Git operations
- Parses diffs between current branch and base branch
- Provides file-by-file diff information

#### State Management (`internal/state`)
- XDG-compliant state storage
- Persists comments, viewed files, and daemon info
- State is scoped by repository path, branch, and commit

#### Web Server (`internal/server`)
- Gorilla Mux for HTTP routing
- REST API for comments, viewed status, and diff data
- Single-page React application embedded in binary

#### MCP Server (`internal/mcp`)
- JSON-RPC 2.0 over stdio transport
- Implements Model Context Protocol specification
- Exposes tools for comment management to LLMs

#### Daemon Management (`internal/daemon`)
- Automatic port allocation (3000-4000 range)
- PID tracking and stale daemon cleanup
- Per-repository daemon instances

### Web Interface Features

- **File-by-file review**: Expand individual files to see diffs
- **Syntax highlighting**: Prism.js for code highlighting
- **Inline comments**: Click the + button on any line to add a comment
- **Resolution tracking**: Mark comments as resolved from the UI
- **View tracking**: Mark files as viewed to track review progress
- **GitHub-like UI**: Dark theme using Primer CSS

### MCP Protocol Implementation

Guck implements the Model Context Protocol (MCP) specification:

1. **Transport**: stdio (standard input/output streams)
2. **Encoding**: JSON-RPC 2.0
3. **Initialization**: Handshake with protocol version and capabilities
4. **Tools**: List and call operations for comment management

The MCP server runs as a subprocess when invoked by LLM applications like Claude Code, maintaining a persistent connection over stdio.

### Security Considerations

- **Local-only**: Web server binds to 127.0.0.1 (localhost only)
- **State isolation**: Each repository's state is independent
- **No authentication**: Assumes trusted local environment
- **File system access**: Limited to configured repository paths

## Troubleshooting

### Port already in use

If you see a "port already in use" error:

```bash
# List running daemons
guck daemon list

# Clean up stale entries
guck daemon cleanup

# Or stop all daemons
guck daemon stop-all
```

### MCP server not appearing in Claude Code

1. Check the configuration file path for your platform
2. Ensure the `command` path points to the correct guck binary
3. Restart Claude Code/Desktop after modifying the configuration
4. Check Claude's logs for MCP server errors

### Comments not persisting

Comments are stored in `~/.local/state/guck/state.json`. If they're not persisting:

1. Check file permissions
2. Ensure the directory exists and is writable
3. Look for errors in the server logs

## License

MIT
