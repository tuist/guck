use anyhow::Result;
use clap::{Parser, Subcommand};

mod config;
mod daemon;
mod git;
mod server;
mod state;

#[derive(Parser)]
#[command(name = "guck")]
#[command(about = "A Git diff review tool with a web interface", long_about = None)]
struct Cli {
    #[command(subcommand)]
    command: Option<Commands>,
}

#[derive(Subcommand)]
enum Commands {
    /// Initialize shell integration (outputs shell script to eval)
    Init,

    /// Daemon management commands
    Daemon {
        #[command(subcommand)]
        command: DaemonCommands,
    },

    /// Configuration commands
    Config {
        #[command(subcommand)]
        command: ConfigCommands,
    },
}

#[derive(Subcommand)]
enum DaemonCommands {
    /// Start daemon for current repository
    Start {
        /// Override base branch
        #[arg(short, long)]
        base: Option<String>,
    },

    /// Stop daemon for current repository
    Stop,

    /// Stop all running daemons
    StopAll,

    /// List all running daemons
    List,

    /// Clean up stale daemon entries
    Cleanup,
}

#[derive(Subcommand)]
enum ConfigCommands {
    /// Set a configuration value
    Set {
        /// Configuration key (e.g., base-branch)
        key: String,
        /// Configuration value
        value: String,
    },

    /// Get a configuration value
    Get {
        /// Configuration key
        key: String,
    },

    /// Show all configuration
    Show,
}

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize tracing
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "guck=info,tower_http=debug".into()),
        )
        .init();

    let cli = Cli::parse();

    match cli.command {
        Some(Commands::Init) => {
            print_shell_integration();
        }
        Some(Commands::Daemon { command }) => {
            handle_daemon_command(command).await?;
        }
        Some(Commands::Config { command }) => {
            handle_config_command(command)?;
        }
        None => {
            // Default action: open browser for current repo
            open_browser().await?;
        }
    }

    Ok(())
}

fn print_shell_integration() {
    let script = r#"
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
"#;
    println!("{}", script);
}

async fn handle_daemon_command(command: DaemonCommands) -> Result<()> {
    let daemon_manager = daemon::DaemonManager::new()?;

    match command {
        DaemonCommands::Start { base } => {
            start_daemon(base).await?;
        }
        DaemonCommands::Stop => {
            let git_repo = git::GitRepo::open(".")?;
            let repo_path = git_repo.repo_path()?;

            if let Some(info) = daemon_manager.get_daemon_for_repo(&repo_path)? {
                daemon_manager.stop_daemon(info.pid)?;
                daemon_manager.unregister_daemon(&repo_path)?;
                println!("Stopped daemon for {}", repo_path);
            } else {
                println!("No daemon running for this repository");
            }
        }
        DaemonCommands::StopAll => {
            let daemons = daemon_manager.list_daemons()?;
            for daemon_info in daemons {
                if daemon_manager.is_daemon_running(daemon_info.pid) {
                    daemon_manager.stop_daemon(daemon_info.pid)?;
                    daemon_manager.unregister_daemon(&daemon_info.repo_path)?;
                    println!("Stopped daemon for {}", daemon_info.repo_path);
                }
            }
        }
        DaemonCommands::List => {
            daemon_manager.cleanup_stale_daemons()?;
            let daemons = daemon_manager.list_daemons()?;

            if daemons.is_empty() {
                println!("No running daemons");
            } else {
                println!("Running daemons:");
                for info in daemons {
                    println!(
                        "  {} - http://localhost:{} (PID: {})",
                        info.repo_path, info.port, info.pid
                    );
                }
            }
        }
        DaemonCommands::Cleanup => {
            daemon_manager.cleanup_stale_daemons()?;
            println!("Cleaned up stale daemon entries");
        }
    }

    Ok(())
}

async fn start_daemon(base_branch_override: Option<String>) -> Result<()> {
    use std::process;

    let git_repo = git::GitRepo::open(".")?;
    let repo_path = git_repo.repo_path()?;
    let daemon_manager = daemon::DaemonManager::new()?;

    // Check if daemon already running
    if let Some(info) = daemon_manager.get_daemon_for_repo(&repo_path)? {
        if daemon_manager.is_daemon_running(info.pid) {
            tracing::info!("Daemon already running on port {}", info.port);
            return Ok(());
        } else {
            // Clean up stale entry
            daemon_manager.unregister_daemon(&repo_path)?;
        }
    }

    // Get config
    let config = config::Config::load()?;
    let base_branch = base_branch_override.unwrap_or(config.base_branch);

    // Find available port
    let port = daemon_manager.find_available_port()?;

    // Check if we're already running as a daemon
    let is_daemon = std::env::var("GUCK_DAEMON").is_ok();

    if is_daemon {
        // We're the daemon process, start the server
        let daemon_info = daemon::DaemonInfo {
            pid: process::id(),
            port,
            repo_path: repo_path.clone(),
            base_branch: base_branch.clone(),
        };

        daemon_manager.register_daemon(daemon_info)?;

        tracing::info!("Starting daemon for {} on port {}", repo_path, port);

        server::start(port, base_branch).await?;
    } else {
        // Fork daemon process
        #[cfg(unix)]
        {
            use daemonize::Daemonize;

            let log_path = daemon_manager.get_log_path(&repo_path);
            let stdout = std::fs::File::create(&log_path)?;
            let stderr = std::fs::File::create(&log_path)?;

            let daemonize = Daemonize::new()
                .stdout(stdout)
                .stderr(stderr)
                .env("GUCK_DAEMON", "1")
                .env("GUCK_REPO_PATH", &repo_path)
                .env("GUCK_PORT", port.to_string())
                .env("GUCK_BASE_BRANCH", &base_branch);

            match daemonize.start() {
                Ok(_) => {
                    // We're now in the daemon process
                    let daemon_info = daemon::DaemonInfo {
                        pid: process::id(),
                        port,
                        repo_path: repo_path.clone(),
                        base_branch: base_branch.clone(),
                    };

                    daemon_manager.register_daemon(daemon_info)?;
                    server::start(port, base_branch).await?;
                }
                Err(e) => {
                    eprintln!("Failed to daemonize: {}", e);
                }
            }
        }

        #[cfg(not(unix))]
        {
            // On Windows, spawn a detached process
            let exe = std::env::current_exe()?;
            std::process::Command::new(exe)
                .arg("daemon")
                .arg("start")
                .env("GUCK_DAEMON", "1")
                .spawn()?;
        }
    }

    Ok(())
}

async fn open_browser() -> Result<()> {
    let git_repo = git::GitRepo::open(".")?;
    let repo_path = git_repo.repo_path()?;
    let daemon_manager = daemon::DaemonManager::new()?;

    // Clean up stale daemons first
    daemon_manager.cleanup_stale_daemons()?;

    // Get daemon info for current repo
    let info = daemon_manager
        .get_daemon_for_repo(&repo_path)?
        .ok_or_else(|| {
            anyhow::anyhow!("No daemon running for this repository. Run 'guck daemon start' first.")
        })?;

    // Verify daemon is actually running
    if !daemon_manager.is_daemon_running(info.pid) {
        daemon_manager.unregister_daemon(&repo_path)?;
        anyhow::bail!("Daemon is not running. Run 'guck daemon start' first.");
    }

    let url = format!("http://localhost:{}", info.port);
    println!("Opening {} in your browser...", url);

    // Open browser
    #[cfg(target_os = "macos")]
    std::process::Command::new("open").arg(&url).spawn()?;

    #[cfg(target_os = "linux")]
    std::process::Command::new("xdg-open").arg(&url).spawn()?;

    #[cfg(target_os = "windows")]
    std::process::Command::new("cmd")
        .args(&["/C", "start", &url])
        .spawn()?;

    Ok(())
}

fn handle_config_command(command: ConfigCommands) -> Result<()> {
    match command {
        ConfigCommands::Set { key, value } => {
            let mut config = config::Config::load()?;

            match key.as_str() {
                "base-branch" => {
                    config.base_branch = value.clone();
                    config.save()?;
                    println!("Set base-branch to '{}'", value);
                }
                _ => {
                    anyhow::bail!("Unknown configuration key: {}", key);
                }
            }
        }
        ConfigCommands::Get { key } => {
            let config = config::Config::load()?;

            match key.as_str() {
                "base-branch" => {
                    println!("{}", config.base_branch);
                }
                _ => {
                    anyhow::bail!("Unknown configuration key: {}", key);
                }
            }
        }
        ConfigCommands::Show => {
            let config = config::Config::load()?;
            println!("base-branch = {}", config.base_branch);
        }
    }

    Ok(())
}
