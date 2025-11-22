package mcp

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/tuist/guck/internal/state"
)

type ListCommentsParams struct {
	RepoPath string  `json:"repo_path,omitempty"`
	Branch   *string `json:"branch,omitempty"`
	Commit   *string `json:"commit,omitempty"`
	FilePath *string `json:"file_path,omitempty"`
	Resolved *bool   `json:"resolved,omitempty"`
}

type ResolveCommentParams struct {
	RepoPath   string `json:"repo_path,omitempty"`
	CommentID  string `json:"comment_id"`
	ResolvedBy string `json:"resolved_by"`
}

type CommentResult struct {
	ID         string `json:"id"`
	FilePath   string `json:"file_path"`
	LineNumber *int   `json:"line_number,omitempty"`
	Text       string `json:"text"`
	Timestamp  int64  `json:"timestamp"`
	Branch     string `json:"branch"`
	Commit     string `json:"commit"`
	Resolved   bool   `json:"resolved"`
	ResolvedBy string `json:"resolved_by,omitempty"`
	ResolvedAt int64  `json:"resolved_at,omitempty"`
}

func ListTools() map[string]interface{} {
	tools := []map[string]interface{}{
		{
			"name":        "list_comments",
			"description": "List all code review comments for a repository. Can filter by branch, commit, file, and resolution status. If repo_path is not provided, uses the current working directory.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo_path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the git repository (defaults to current working directory)",
					},
					"branch": map[string]interface{}{
						"type":        "string",
						"description": "Optional: Filter by branch name",
					},
					"commit": map[string]interface{}{
						"type":        "string",
						"description": "Optional: Filter by commit hash",
					},
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Optional: Filter by file path",
					},
					"resolved": map[string]interface{}{
						"type":        "boolean",
						"description": "Optional: Filter by resolution status (true=resolved, false=unresolved)",
					},
				},
			},
		},
		{
			"name":        "resolve_comment",
			"description": "Mark a code review comment as resolved, tracking who resolved it and when. If repo_path is not provided, uses the current working directory.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo_path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the git repository (defaults to current working directory)",
					},
					"comment_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the comment to resolve",
					},
					"resolved_by": map[string]interface{}{
						"type":        "string",
						"description": "Name or identifier of who is resolving the comment",
					},
				},
				"required": []string{"comment_id", "resolved_by"},
			},
		},
	}

	return map[string]interface{}{
		"tools": tools,
	}
}

func ListComments(paramsRaw json.RawMessage, workingDir string) (interface{}, error) {
	stateMgr, err := state.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}
	return ListCommentsWithManager(paramsRaw, workingDir, stateMgr)
}

func ListCommentsWithManager(paramsRaw json.RawMessage, workingDir string, stateMgr *state.Manager) (interface{}, error) {
	var params ListCommentsParams
	if err := json.Unmarshal(paramsRaw, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	// Use working directory if repo_path not specified
	repoPath := params.RepoPath
	if repoPath == "" {
		repoPath = workingDir
	}

	// Make path absolute
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("invalid repo_path: %w", err)
	}

	var comments []*state.Comment

	// If branch and commit are specified, get comments for that specific state
	if params.Branch != nil && params.Commit != nil {
		comments = stateMgr.GetComments(absPath, *params.Branch, *params.Commit, params.FilePath)
	} else {
		// Otherwise get all comments for the repo
		comments = stateMgr.GetAllComments(absPath)
	}

	// Filter by resolution status if specified
	if params.Resolved != nil {
		filtered := []*state.Comment{}
		for _, c := range comments {
			if c.Resolved == *params.Resolved {
				filtered = append(filtered, c)
			}
		}
		comments = filtered
	}

	// Filter by file path if specified (and not already filtered by GetComments)
	if params.FilePath != nil && (params.Branch == nil || params.Commit == nil) {
		filtered := []*state.Comment{}
		for _, c := range comments {
			if c.FilePath == *params.FilePath {
				filtered = append(filtered, c)
			}
		}
		comments = filtered
	}

	// Convert to result format
	results := make([]CommentResult, len(comments))
	for i, c := range comments {
		results[i] = CommentResult{
			ID:         c.ID,
			FilePath:   c.FilePath,
			LineNumber: c.LineNumber,
			Text:       c.Text,
			Timestamp:  c.Timestamp,
			Branch:     c.Branch,
			Commit:     c.Commit,
			Resolved:   c.Resolved,
			ResolvedBy: c.ResolvedBy,
			ResolvedAt: c.ResolvedAt,
		}
	}

	return map[string]interface{}{
		"comments":  results,
		"count":     len(results),
		"repo_path": absPath,
	}, nil
}

func ResolveComment(paramsRaw json.RawMessage, workingDir string) (interface{}, error) {
	stateMgr, err := state.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}
	return ResolveCommentWithManager(paramsRaw, workingDir, stateMgr)
}

func ResolveCommentWithManager(paramsRaw json.RawMessage, workingDir string, stateMgr *state.Manager) (interface{}, error) {
	var params ResolveCommentParams
	if err := json.Unmarshal(paramsRaw, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if params.CommentID == "" {
		return nil, fmt.Errorf("comment_id is required")
	}

	if params.ResolvedBy == "" {
		return nil, fmt.Errorf("resolved_by is required")
	}

	// Use working directory if repo_path not specified
	repoPath := params.RepoPath
	if repoPath == "" {
		repoPath = workingDir
	}

	// Make path absolute
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("invalid repo_path: %w", err)
	}

	// Get all comments to find the one to resolve
	allComments := stateMgr.GetAllComments(absPath)

	var targetComment *state.Comment
	for _, c := range allComments {
		if c.ID == params.CommentID {
			targetComment = c
			break
		}
	}

	if targetComment == nil {
		return nil, fmt.Errorf("comment not found: %s", params.CommentID)
	}

	// Resolve the comment
	if err := stateMgr.ResolveComment(absPath, targetComment.Branch, targetComment.Commit, params.CommentID, params.ResolvedBy); err != nil {
		return nil, fmt.Errorf("failed to resolve comment: %w", err)
	}

	return map[string]interface{}{
		"success":     true,
		"comment_id":  params.CommentID,
		"resolved_by": params.ResolvedBy,
		"repo_path":   absPath,
	}, nil
}
