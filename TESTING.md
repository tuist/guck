# Testing Guck Locally

This guide shows you how to test guck in this repository.

## Quick Start

### 1. Build the project

```bash
# If you have mise installed:
mise exec -- cargo build --release

# Or directly with cargo:
cargo build --release
```

The binary will be at `./target/release/guck`

### 2. Test basic functionality

```bash
# Start the daemon for this repository
./target/release/guck daemon start

# Wait a moment for it to start, then list running daemons
./target/release/guck daemon list

# Open the browser (opens http://localhost:3000 or whatever port was allocated)
./target/release/guck

# You should see a web interface showing the diff between your current branch and main
```

### 3. Test configuration

```bash
# Show current config
./target/release/guck config show

# Set base branch to something else
./target/release/guck config set base-branch develop

# Get the value back
./target/release/guck config get base-branch
```

### 4. Test shell integration

```bash
# See what the shell integration script looks like
./target/release/guck init

# To actually enable it (temporarily for testing):
eval "$(./target/release/guck init)"

# Now cd to another git repository and it should auto-start a daemon
cd /path/to/another/repo
./target/release/guck daemon list  # Should show both repos
```

### 5. Cleanup

```bash
# Stop daemon for current repo
./target/release/guck daemon stop

# Or stop all daemons
./target/release/guck daemon stop-all

# Clean up stale daemon entries
./target/release/guck daemon cleanup
```

## Testing Scenarios

### Scenario 1: Review your changes in this repo

1. Make some changes to files in this repo
2. Create a branch: `git checkout -b test-feature`
3. Commit your changes
4. Start guck: `./target/release/guck daemon start`
5. Open browser: `./target/release/guck`
6. You should see your changes vs main

### Scenario 2: Multiple repositories

1. Start daemon in this repo: `./target/release/guck daemon start`
2. Go to another git repo: `cd /path/to/another/repo`
3. Start daemon there: `/Users/pepicrft/src/github.com/tuist/guck/target/release/guck daemon start`
4. List daemons: `/Users/pepicrft/src/github.com/tuist/guck/target/release/guck daemon list`
5. Each repo should have its own port

### Scenario 3: Mark files as viewed

1. Open the web interface
2. Click on a file to expand it and see the diff
3. Click "Mark as viewed" button
4. Refresh the page - the file should still show as viewed
5. Switch branches or make new commits - viewed state is per-commit

## Troubleshooting

### Daemon won't start

Check the logs:
```bash
ls -la ~/.local/state/guck/
cat ~/.local/state/guck/*.log
```

### Port already in use

List and stop daemons:
```bash
./target/release/guck daemon list
./target/release/guck daemon stop-all
```

### Can't see changes

Make sure you're on a different branch than main:
```bash
git branch  # Should show you're not on main
git diff main  # Should show some differences
```

## File Locations

When testing, guck stores data in these locations:

- **Daemon registry**: `~/.local/state/guck/daemon-registry.json`
- **Viewed files**: `~/.local/state/guck/viewed.json`
- **Config**: `~/.config/guck/config.toml`
- **Logs**: `~/.local/state/guck/*.log`

You can inspect these files to see how guck is working:

```bash
# See what daemons are registered
cat ~/.local/state/guck/daemon-registry.json | jq .

# See what files you've marked as viewed
cat ~/.local/state/guck/viewed.json | jq .

# See your config
cat ~/.config/guck/config.toml
```

## Development Workflow

For development, you might want to use `cargo run` instead:

```bash
# Start daemon
cargo run -- daemon start

# Open browser
cargo run

# List daemons
cargo run -- daemon list

# Stop daemon
cargo run -- daemon stop
```
