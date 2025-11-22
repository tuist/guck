# Guck

A Git diff review tool with a web interface, inspired by GitHub's pull request UI. Guck runs as a background daemon that automatically starts when you enter a git repository.

## Features

- 🤖 **Auto-start daemon** - Automatically starts a server when you cd into a git repo
- 🌐 **Web-based interface** - Review diffs in your browser with a GitHub-like UI
- 📁 **File-by-file diff viewing** - Expand and review individual files
- ✅ **Mark files as viewed** - Track your review progress
- 💬 **Inline comments** - Add comments to specific lines of code
- 💾 **Persistent state** - Remembers what you've reviewed using XDG conventions
- 🎨 **GitHub-inspired dark theme** - Familiar and easy on the eyes
- ⚡ **Built with Go** - Fast, simple, and efficient
- 🔌 **Automatic port allocation** - Each repository gets its own port
- 🤖 **MCP Server Integration** - Allows LLMs like Claude to query and resolve review comments

## Quick Start

### Installation

```bash
# Using mise (recommended)
mise use -g guck@latest

# Or download from releases
# https://github.com/tuist/guck/releases
```

### Setup

Add to your shell configuration (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
eval "$(guck init)"
```

### Usage

```bash
cd /path/to/your/repo  # Daemon starts automatically
guck                   # Opens browser to review diffs
```

## MCP Integration with Claude Code

Add to your Claude Desktop configuration:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`

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

Restart Claude Code, and you can ask Claude to:
- "List all unresolved comments in this repository"
- "Show me comments on main.go"
- "Resolve comment with ID xyz"

## Documentation

For comprehensive documentation, see [docs/README.md](docs/README.md):

- [Installation & Setup](docs/README.md#installation)
- [Usage Guide](docs/README.md#usage)
- [MCP Server Integration](docs/README.md#mcp-server-integration)
- [Development](docs/README.md#development)
- [Architecture](docs/README.md#architecture)
- [Troubleshooting](docs/README.md#troubleshooting)

## Common Commands

```bash
# Daemon management
guck daemon start      # Start daemon manually
guck daemon stop       # Stop current repo's daemon
guck daemon list       # List all running daemons
guck daemon stop-all   # Stop all daemons

# Configuration
guck config set base-branch develop
guck config show
```

## How It Works

1. **Shell Integration**: Automatically starts a server when you `cd` into a git repository
2. **Daemon Management**: Each repository gets its own background server with a unique port
3. **Web Interface**: Review diffs in your browser, mark files as viewed, add inline comments
4. **State Persistence**: Everything is saved locally and associated with repo/branch/commit
5. **MCP Integration**: LLMs like Claude can query and resolve comments through the MCP protocol

## License

MIT
