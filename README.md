# Guck

A Git diff review tool with a web interface, inspired by GitHub's pull request UI. Guck runs as a background daemon that automatically starts when you enter a git repository.

## Features

- 🤖 **Auto-start daemon** - Automatically starts a server when you cd into a git repo
- 🌐 **Web-based interface** - Review diffs in your browser with a GitHub-like UI
- 📁 **File-by-file diff viewing** - Expand and review individual files
- ✅ **Mark files as viewed** - Track your review progress
- 💾 **Persistent state** - Remembers what you've reviewed using XDG conventions
- 🎨 **GitHub-inspired dark theme** - Familiar and easy on the eyes
- ⚡ **Built with Rust** - Fast and efficient
- 🔌 **Automatic port allocation** - Each repository gets its own port

## Installation

### Using mise (recommended)

```bash
mise use -g guck@latest
```

### Download binary

Download the latest release for your platform from the [releases page](https://github.com/tuist/guck/releases).

### From source

```bash
cargo install --git https://github.com/tuist/guck
```

## Setup

After installing, add this to your shell configuration file (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
# For Bash/Zsh
eval "$(guck init)"
```

This enables automatic daemon management when entering/leaving git repositories.

## Usage

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

### Manual Commands

```bash
# Open the web interface for the current repo
guck

# Start the daemon manually
guck daemon start

# Stop the daemon for the current repo
guck daemon stop

# Stop all guck daemons
guck daemon stop-all

# List all running guck servers
guck daemon list

# Set the base branch (default: main)
guck config set base-branch develop
```

## How it works

1. **Shell Integration**: When you `cd` into a directory, guck checks if it's a git repository
2. **Daemon Management**: If it is, guck starts a background server (if not already running)
3. **Port Mapping**: Each repository is mapped to a unique port (stored in `~/.local/state/guck/`)
4. **Web Interface**: Run `guck` to open your browser to the appropriate port
5. **Diff Review**: Review changes against your base branch, mark files as viewed
6. **State Persistence**: Your review progress is saved and associated with the repo, branch, and commit

The viewed state is persisted locally using XDG conventions, associated with the repository path, branch name, and commit hash.

## Configuration

Guck stores its data in XDG-compliant directories:

- **State**: `~/.local/state/guck/` - Port mappings, daemon PIDs, viewed files
- **Config**: `~/.config/guck/` - User configuration

## Development

### Prerequisites

- Rust 1.70 or later
- Git

### Building

```bash
cargo build --release
```

### Running locally

```bash
cargo run -- daemon start
# In another terminal:
cargo run -- open
```

## License

MIT
