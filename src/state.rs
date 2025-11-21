use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;

#[derive(Serialize, Deserialize, Clone, Debug)]
pub struct Comment {
    pub id: String,
    pub file_path: String,
    pub line_number: Option<usize>,
    pub text: String,
    pub timestamp: u64,
    pub branch: String,
    pub commit: String,
    pub resolved: bool,
}

#[derive(Serialize, Deserialize, Default)]
struct RepoState {
    viewed_files: Vec<String>,
    comments: Vec<Comment>,
}

#[derive(Serialize, Deserialize, Default)]
struct ViewedState {
    // repo_path -> branch -> commit -> RepoState
    repos: HashMap<String, HashMap<String, HashMap<String, RepoState>>>,
}

pub struct StateManager {
    state_file: PathBuf,
    state: ViewedState,
}

impl StateManager {
    pub fn new() -> Result<Self> {
        let state_dir = dirs::state_dir()
            .or_else(|| dirs::data_local_dir())
            .context("Failed to determine state directory")?
            .join("guck");

        fs::create_dir_all(&state_dir).context("Failed to create state directory")?;

        let state_file = state_dir.join("viewed.json");

        let state = if state_file.exists() {
            let contents = fs::read_to_string(&state_file).context("Failed to read state file")?;
            serde_json::from_str(&contents).unwrap_or_default()
        } else {
            ViewedState::default()
        };

        Ok(Self { state_file, state })
    }

    pub fn is_file_viewed(
        &self,
        repo_path: &str,
        branch: &str,
        commit: &str,
        file_path: &str,
    ) -> Result<bool> {
        Ok(self
            .state
            .repos
            .get(repo_path)
            .and_then(|branches| branches.get(branch))
            .and_then(|commits| commits.get(commit))
            .map(|repo_state| repo_state.viewed_files.contains(&file_path.to_string()))
            .unwrap_or(false))
    }

    pub fn mark_file_viewed(
        &mut self,
        repo_path: &str,
        branch: &str,
        commit: &str,
        file_path: &str,
    ) -> Result<()> {
        let repo = self
            .state
            .repos
            .entry(repo_path.to_string())
            .or_insert_with(HashMap::new);

        let branch_map = repo.entry(branch.to_string()).or_insert_with(HashMap::new);

        let repo_state = branch_map
            .entry(commit.to_string())
            .or_insert_with(RepoState::default);

        if !repo_state.viewed_files.contains(&file_path.to_string()) {
            repo_state.viewed_files.push(file_path.to_string());
        }

        self.save()?;
        Ok(())
    }

    pub fn unmark_file_viewed(
        &mut self,
        repo_path: &str,
        branch: &str,
        commit: &str,
        file_path: &str,
    ) -> Result<()> {
        if let Some(repo) = self.state.repos.get_mut(repo_path) {
            if let Some(branch_map) = repo.get_mut(branch) {
                if let Some(repo_state) = branch_map.get_mut(commit) {
                    repo_state.viewed_files.retain(|f| f != file_path);
                }
            }
        }

        self.save()?;
        Ok(())
    }

    pub fn add_comment(
        &mut self,
        repo_path: &str,
        branch: &str,
        commit: &str,
        file_path: &str,
        line_number: Option<usize>,
        text: String,
    ) -> Result<Comment> {
        use std::time::{SystemTime, UNIX_EPOCH};

        let repo = self
            .state
            .repos
            .entry(repo_path.to_string())
            .or_insert_with(HashMap::new);

        let branch_map = repo.entry(branch.to_string()).or_insert_with(HashMap::new);

        let repo_state = branch_map
            .entry(commit.to_string())
            .or_insert_with(RepoState::default);

        let timestamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs();

        let comment = Comment {
            id: format!("{}-{}", timestamp, repo_state.comments.len()),
            file_path: file_path.to_string(),
            line_number,
            text,
            timestamp,
            branch: branch.to_string(),
            commit: commit.to_string(),
            resolved: false,
        };

        repo_state.comments.push(comment.clone());
        self.save()?;
        Ok(comment)
    }

    pub fn get_comments(
        &self,
        repo_path: &str,
        branch: &str,
        commit: &str,
        file_path: Option<&str>,
    ) -> Result<Vec<Comment>> {
        let comments = self
            .state
            .repos
            .get(repo_path)
            .and_then(|branches| branches.get(branch))
            .and_then(|commits| commits.get(commit))
            .map(|repo_state| {
                if let Some(fp) = file_path {
                    repo_state
                        .comments
                        .iter()
                        .filter(|c| c.file_path == fp)
                        .cloned()
                        .collect()
                } else {
                    repo_state.comments.clone()
                }
            })
            .unwrap_or_default();

        Ok(comments)
    }

    pub fn resolve_comment(
        &mut self,
        repo_path: &str,
        branch: &str,
        commit: &str,
        comment_id: &str,
    ) -> Result<()> {
        if let Some(repo) = self.state.repos.get_mut(repo_path) {
            if let Some(branch_map) = repo.get_mut(branch) {
                if let Some(repo_state) = branch_map.get_mut(commit) {
                    if let Some(comment) =
                        repo_state.comments.iter_mut().find(|c| c.id == comment_id)
                    {
                        comment.resolved = true;
                    }
                }
            }
        }
        self.save()?;
        Ok(())
    }

    fn save(&self) -> Result<()> {
        let contents =
            serde_json::to_string_pretty(&self.state).context("Failed to serialize state")?;
        fs::write(&self.state_file, contents).context("Failed to write state file")?;
        Ok(())
    }
}
