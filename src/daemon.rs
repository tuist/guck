use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;
use std::process::{Child, Command};

#[derive(Serialize, Deserialize, Default)]
pub struct DaemonRegistry {
    // repo_path -> DaemonInfo
    daemons: HashMap<String, DaemonInfo>,
}

#[derive(Serialize, Deserialize, Clone)]
pub struct DaemonInfo {
    pub pid: u32,
    pub port: u16,
    pub repo_path: String,
    pub base_branch: String,
}

pub struct DaemonManager {
    registry_path: PathBuf,
    state_dir: PathBuf,
}

impl DaemonManager {
    pub fn new() -> Result<Self> {
        let state_dir = dirs::state_dir()
            .or_else(|| dirs::data_local_dir())
            .context("Failed to determine state directory")?
            .join("guck");

        fs::create_dir_all(&state_dir).context("Failed to create state directory")?;

        let registry_path = state_dir.join("daemon-registry.json");

        Ok(Self {
            registry_path,
            state_dir,
        })
    }

    fn load_registry(&self) -> Result<DaemonRegistry> {
        if self.registry_path.exists() {
            let contents = fs::read_to_string(&self.registry_path)?;
            Ok(serde_json::from_str(&contents).unwrap_or_default())
        } else {
            Ok(DaemonRegistry::default())
        }
    }

    fn save_registry(&self, registry: &DaemonRegistry) -> Result<()> {
        let contents = serde_json::to_string_pretty(registry)?;
        fs::write(&self.registry_path, contents)?;
        Ok(())
    }

    pub fn find_available_port(&self) -> Result<u16> {
        let registry = self.load_registry()?;
        let used_ports: Vec<u16> = registry.daemons.values().map(|d| d.port).collect();

        // Start from 3000 and find the first available port
        for port in 3000..4000 {
            if !used_ports.contains(&port) {
                return Ok(port);
            }
        }

        anyhow::bail!("No available ports in range 3000-4000")
    }

    pub fn get_daemon_for_repo(&self, repo_path: &str) -> Result<Option<DaemonInfo>> {
        let registry = self.load_registry()?;
        Ok(registry.daemons.get(repo_path).cloned())
    }

    pub fn register_daemon(&self, info: DaemonInfo) -> Result<()> {
        let mut registry = self.load_registry()?;
        registry.daemons.insert(info.repo_path.clone(), info);
        self.save_registry(&registry)?;
        Ok(())
    }

    pub fn unregister_daemon(&self, repo_path: &str) -> Result<()> {
        let mut registry = self.load_registry()?;
        registry.daemons.remove(repo_path);
        self.save_registry(&registry)?;
        Ok(())
    }

    pub fn list_daemons(&self) -> Result<Vec<DaemonInfo>> {
        let registry = self.load_registry()?;
        Ok(registry.daemons.values().cloned().collect())
    }

    pub fn is_daemon_running(&self, pid: u32) -> bool {
        #[cfg(unix)]
        {
            use nix::sys::signal::{kill, Signal};
            use nix::unistd::Pid;

            match kill(Pid::from_raw(pid as i32), Signal::SIGTERM) {
                Ok(_) => true,
                Err(_) => false,
            }
        }

        #[cfg(not(unix))]
        {
            // On Windows, use tasklist
            let output = Command::new("tasklist")
                .arg("/FI")
                .arg(format!("PID eq {}", pid))
                .output();

            if let Ok(output) = output {
                let stdout = String::from_utf8_lossy(&output.stdout);
                stdout.contains(&pid.to_string())
            } else {
                false
            }
        }
    }

    pub fn stop_daemon(&self, pid: u32) -> Result<()> {
        #[cfg(unix)]
        {
            use nix::sys::signal::{kill, Signal};
            use nix::unistd::Pid;

            kill(Pid::from_raw(pid as i32), Signal::SIGTERM)
                .context("Failed to send SIGTERM to daemon")?;
        }

        #[cfg(not(unix))]
        {
            Command::new("taskkill")
                .arg("/PID")
                .arg(pid.to_string())
                .arg("/F")
                .output()
                .context("Failed to kill daemon process")?;
        }

        Ok(())
    }

    pub fn cleanup_stale_daemons(&self) -> Result<()> {
        let mut registry = self.load_registry()?;
        let mut to_remove = Vec::new();

        for (repo_path, info) in &registry.daemons {
            if !self.is_daemon_running(info.pid) {
                to_remove.push(repo_path.clone());
            }
        }

        for repo_path in to_remove {
            registry.daemons.remove(&repo_path);
        }

        self.save_registry(&registry)?;
        Ok(())
    }

    pub fn get_log_path(&self, repo_path: &str) -> PathBuf {
        // Create a safe filename from repo path
        let safe_name = repo_path
            .replace("/", "_")
            .replace("\\", "_")
            .replace(":", "_");
        self.state_dir.join(format!("{}.log", safe_name))
    }
}
