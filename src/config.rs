use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::PathBuf;

#[derive(Serialize, Deserialize, Default)]
pub struct Config {
    #[serde(default = "default_base_branch")]
    pub base_branch: String,
}

fn default_base_branch() -> String {
    "main".to_string()
}

impl Config {
    pub fn load() -> Result<Self> {
        let config_path = Self::config_path()?;

        if config_path.exists() {
            let contents = fs::read_to_string(&config_path)?;
            Ok(toml::from_str(&contents).unwrap_or_default())
        } else {
            Ok(Config::default())
        }
    }

    pub fn save(&self) -> Result<()> {
        let config_path = Self::config_path()?;
        let config_dir = config_path.parent().context("Invalid config path")?;

        fs::create_dir_all(config_dir).context("Failed to create config directory")?;

        let contents = toml::to_string_pretty(self)?;
        fs::write(&config_path, contents)?;

        Ok(())
    }

    fn config_path() -> Result<PathBuf> {
        let config_dir = dirs::config_dir()
            .context("Failed to determine config directory")?
            .join("guck");

        Ok(config_dir.join("config.toml"))
    }
}
