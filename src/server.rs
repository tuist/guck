use anyhow::Result;
use axum::{
    extract::State,
    http::StatusCode,
    response::{Html, IntoResponse, Json},
    routing::{get, post},
    Router,
};
use serde::{Deserialize, Serialize};
use std::sync::{Arc, Mutex};
use tower_http::trace::TraceLayer;

use crate::state::StateManager;

#[derive(Clone)]
struct AppState {
    repo_path: String,
    base_branch: String,
    state_manager: Arc<Mutex<StateManager>>,
}

#[derive(Serialize)]
struct DiffResponse {
    files: Vec<FileDiff>,
    branch: String,
    commit: String,
    repo_path: String,
}

#[derive(Serialize)]
struct FileDiff {
    path: String,
    status: String,
    additions: usize,
    deletions: usize,
    patch: String,
    viewed: bool,
}

#[derive(Deserialize)]
struct MarkViewedRequest {
    file_path: String,
}

#[derive(Deserialize)]
struct AddCommentRequest {
    file_path: String,
    line_number: Option<usize>,
    text: String,
}

#[derive(Deserialize)]
struct GetCommentsQuery {
    file_path: Option<String>,
}

#[derive(Deserialize)]
struct ResolveCommentRequest {
    comment_id: String,
}

pub async fn start(port: u16, base_branch: String) -> Result<()> {
    // Get repo path once at startup
    use crate::git::GitRepo;
    let git_repo = GitRepo::open(".")?;
    let repo_path = git_repo.repo_path()?;
    drop(git_repo); // Release the repository handle

    let state_manager = Arc::new(Mutex::new(StateManager::new()?));

    let app_state = AppState {
        repo_path,
        base_branch: base_branch.clone(),
        state_manager,
    };

    // Build the router
    let app = Router::new()
        .route("/", get(index_handler))
        .route("/api/diff", get(diff_handler))
        .route("/api/mark-viewed", post(mark_viewed_handler))
        .route("/api/unmark-viewed", post(unmark_viewed_handler))
        .route("/api/status", get(status_handler))
        .route("/api/comments", get(get_comments_handler))
        .route("/api/comments", post(add_comment_handler))
        .route("/api/comments/resolve", post(resolve_comment_handler))
        .with_state(app_state)
        .layer(TraceLayer::new_for_http());

    let addr = format!("127.0.0.1:{}", port);
    tracing::info!("Starting server on http://{}", addr);
    tracing::info!("Comparing against base branch: {}", base_branch);

    let listener = tokio::net::TcpListener::bind(&addr).await?;
    axum::serve(listener, app).await?;

    Ok(())
}

async fn index_handler() -> Html<&'static str> {
    Html(include_str!("../static/index.html"))
}

async fn diff_handler(State(state): State<AppState>) -> Result<Json<DiffResponse>, AppError> {
    use crate::git::GitRepo;

    // Create a new GitRepo instance for this request
    let git_repo = GitRepo::open(".")?;
    let current_branch = git_repo.current_branch()?;
    let current_commit = git_repo.current_commit()?;

    let files = git_repo.get_diff_files(&state.base_branch)?;

    let mut file_diffs = Vec::new();
    let state_manager = state.state_manager.lock().unwrap();
    for file in files {
        let viewed = state_manager.is_file_viewed(
            &state.repo_path,
            &current_branch,
            &current_commit,
            &file.path,
        )?;

        file_diffs.push(FileDiff {
            path: file.path,
            status: file.status,
            additions: file.additions,
            deletions: file.deletions,
            patch: file.patch,
            viewed,
        });
    }
    drop(state_manager);

    Ok(Json(DiffResponse {
        files: file_diffs,
        branch: current_branch,
        commit: current_commit,
        repo_path: state.repo_path.clone(),
    }))
}

async fn mark_viewed_handler(
    State(state): State<AppState>,
    Json(payload): Json<MarkViewedRequest>,
) -> Result<StatusCode, AppError> {
    use crate::git::GitRepo;

    let git_repo = GitRepo::open(".")?;
    let current_branch = git_repo.current_branch()?;
    let current_commit = git_repo.current_commit()?;

    let mut state_manager = state.state_manager.lock().unwrap();
    state_manager.mark_file_viewed(
        &state.repo_path,
        &current_branch,
        &current_commit,
        &payload.file_path,
    )?;

    Ok(StatusCode::OK)
}

async fn unmark_viewed_handler(
    State(state): State<AppState>,
    Json(payload): Json<MarkViewedRequest>,
) -> Result<StatusCode, AppError> {
    use crate::git::GitRepo;

    let git_repo = GitRepo::open(".")?;
    let current_branch = git_repo.current_branch()?;
    let current_commit = git_repo.current_commit()?;

    let mut state_manager = state.state_manager.lock().unwrap();
    state_manager.unmark_file_viewed(
        &state.repo_path,
        &current_branch,
        &current_commit,
        &payload.file_path,
    )?;

    Ok(StatusCode::OK)
}

#[derive(Serialize)]
struct StatusResponse {
    repo_path: String,
    branch: String,
    commit: String,
}

async fn status_handler(State(state): State<AppState>) -> Result<Json<StatusResponse>, AppError> {
    use crate::git::GitRepo;

    let git_repo = GitRepo::open(".")?;
    Ok(Json(StatusResponse {
        repo_path: state.repo_path.clone(),
        branch: git_repo.current_branch()?,
        commit: git_repo.current_commit()?,
    }))
}

async fn get_comments_handler(
    State(state): State<AppState>,
    axum::extract::Query(query): axum::extract::Query<GetCommentsQuery>,
) -> Result<Json<Vec<crate::state::Comment>>, AppError> {
    use crate::git::GitRepo;

    let git_repo = GitRepo::open(".")?;
    let current_branch = git_repo.current_branch()?;
    let current_commit = git_repo.current_commit()?;

    let state_manager = state.state_manager.lock().unwrap();
    let comments = state_manager.get_comments(
        &state.repo_path,
        &current_branch,
        &current_commit,
        query.file_path.as_deref(),
    )?;

    Ok(Json(comments))
}

async fn add_comment_handler(
    State(state): State<AppState>,
    Json(payload): Json<AddCommentRequest>,
) -> Result<Json<crate::state::Comment>, AppError> {
    use crate::git::GitRepo;

    let git_repo = GitRepo::open(".")?;
    let current_branch = git_repo.current_branch()?;
    let current_commit = git_repo.current_commit()?;

    let mut state_manager = state.state_manager.lock().unwrap();
    let comment = state_manager.add_comment(
        &state.repo_path,
        &current_branch,
        &current_commit,
        &payload.file_path,
        payload.line_number,
        payload.text,
    )?;

    Ok(Json(comment))
}

async fn resolve_comment_handler(
    State(state): State<AppState>,
    Json(payload): Json<ResolveCommentRequest>,
) -> Result<StatusCode, AppError> {
    use crate::git::GitRepo;

    let git_repo = GitRepo::open(".")?;
    let current_branch = git_repo.current_branch()?;
    let current_commit = git_repo.current_commit()?;

    let mut state_manager = state.state_manager.lock().unwrap();
    state_manager.resolve_comment(
        &state.repo_path,
        &current_branch,
        &current_commit,
        &payload.comment_id,
    )?;

    Ok(StatusCode::OK)
}

// Error handling
struct AppError(anyhow::Error);

impl IntoResponse for AppError {
    fn into_response(self) -> axum::response::Response {
        (
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("Error: {}", self.0),
        )
            .into_response()
    }
}

impl<E> From<E> for AppError
where
    E: Into<anyhow::Error>,
{
    fn from(err: E) -> Self {
        Self(err.into())
    }
}
