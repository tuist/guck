package mcp

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/tuist/guck/internal/state"
)

type ListCommentsParams struct {
	RepoPath string  `json:"repo_path"`
	Branch   *string `json:"branch,omitempty"`
	Commit   *string `json:"commit,omitempty"`
	FilePath *string `json:"file_path,omitempty"`
	Resolved *bool   `json:"resolved,omitempty"`
}

type ResolveCommentParams struct {
	RepoPath   string `json:"repo_path"`
	CommentID  string `json:"comment_id"`
	ResolvedBy string `json:"resolved_by"`
}

type AddCommentParams struct {
	RepoPath   string            `json:"repo_path"`
	Branch     string            `json:"branch"`
	Commit     string            `json:"commit"`
	FilePath   string            `json:"file_path"`
	LineNumber *int              `json:"line_number,omitempty"`
	Text       string            `json:"text"`
	Author     string            `json:"author,omitempty"`
	Type       string            `json:"type,omitempty"`
	ParentID   string            `json:"parent_id,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type AddNoteParams struct {
	RepoPath   string            `json:"repo_path"`
	Branch     string            `json:"branch"`
	Commit     string            `json:"commit"`
	FilePath   string            `json:"file_path"`
	LineNumber *int              `json:"line_number,omitempty"`
	Text       string            `json:"text"`
	Author     string            `json:"author"`
	Type       string            `json:"type,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type ListNotesParams struct {
	RepoPath  string  `json:"repo_path"`
	Branch    *string `json:"branch,omitempty"`
	Commit    *string `json:"commit,omitempty"`
	FilePath  *string `json:"file_path,omitempty"`
	Dismissed *bool   `json:"dismissed,omitempty"`
	Author    *string `json:"author,omitempty"`
}

type DismissNoteParams struct {
	RepoPath    string `json:"repo_path"`
	NoteID      string `json:"note_id"`
	DismissedBy string `json:"dismissed_by"`
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

type NoteResult struct {
	ID          string            `json:"id"`
	FilePath    string            `json:"file_path"`
	LineNumber  *int              `json:"line_number,omitempty"`
	Text        string            `json:"text"`
	Timestamp   int64             `json:"timestamp"`
	Branch      string            `json:"branch"`
	Commit      string            `json:"commit"`
	Author      string            `json:"author"`
	Type        string            `json:"type"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Dismissed   bool              `json:"dismissed"`
	DismissedBy string            `json:"dismissed_by,omitempty"`
	DismissedAt int64             `json:"dismissed_at,omitempty"`
}

func ListTools() map[string]interface{} {
	tools := []map[string]interface{}{
		{
			"name":        "list_comments",
			"description": "List all code review comments for a repository. Can filter by branch, commit, file, and resolution status.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo_path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the git repository",
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
				"required": []string{"repo_path"},
			},
		},
		{
			"name":        "resolve_comment",
			"description": "Mark a code review comment as resolved, tracking who resolved it and when.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo_path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the git repository",
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
				"required": []string{"repo_path", "comment_id", "resolved_by"},
			},
		},
		{
			"name":        "add_comment",
			"description": "Add a code review comment or agent explanation. Can be used to start a conversation or leave rationale.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo_path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the git repository",
					},
					"branch": map[string]interface{}{
						"type":        "string",
						"description": "Branch name",
					},
					"commit": map[string]interface{}{
						"type":        "string",
						"description": "Commit hash",
					},
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "File path relative to repository root",
					},
					"line_number": map[string]interface{}{
						"type":        "integer",
						"description": "Optional: Line number",
					},
					"text": map[string]interface{}{
						"type":        "string",
						"description": "Comment text",
					},
					"author": map[string]interface{}{
						"type":        "string",
						"description": "Optional: Author identifier (e.g., 'agent:claude')",
					},
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Optional: Comment type (e.g., 'explanation', 'rationale')",
					},
					"parent_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional: ID of parent comment for threading",
					},
				},
				"required": []string{"repo_path", "branch", "commit", "file_path", "text"},
			},
		},
		{
			"name":        "add_note",
			"description": "Add an AI agent note to explain code decisions, rationale, or suggestions. Notes are distinct from review comments and represent AI-generated explanations.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo_path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the git repository",
					},
					"branch": map[string]interface{}{
						"type":        "string",
						"description": "Branch name where the note applies",
					},
					"commit": map[string]interface{}{
						"type":        "string",
						"description": "Commit hash where the note applies",
					},
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "File path relative to repository root",
					},
					"line_number": map[string]interface{}{
						"type":        "integer",
						"description": "Optional: Line number for inline notes",
					},
					"text": map[string]interface{}{
						"type":        "string",
						"description": "The note content (markdown supported)",
					},
					"author": map[string]interface{}{
						"type":        "string",
						"description": "Author identifier (e.g., 'claude', 'copilot', 'gpt-4')",
					},
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Optional: Note type (e.g., 'explanation', 'rationale', 'suggestion'). Defaults to 'explanation'",
					},
					"metadata": map[string]interface{}{
						"type":        "object",
						"description": "Optional: Additional metadata as key-value pairs",
					},
				},
				"required": []string{"repo_path", "branch", "commit", "file_path", "text", "author"},
			},
		},
		{
			"name":        "list_notes",
			"description": "List AI agent notes with optional filtering by branch, commit, file, author, or dismissal status.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo_path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the git repository",
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
					"dismissed": map[string]interface{}{
						"type":        "boolean",
						"description": "Optional: Filter by dismissal status (true=dismissed, false=active)",
					},
					"author": map[string]interface{}{
						"type":        "string",
						"description": "Optional: Filter by author",
					},
				},
				"required": []string{"repo_path"},
			},
		},
		{
			"name":        "dismiss_note",
			"description": "Mark an AI agent note as dismissed, indicating the user has acknowledged it.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo_path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute path to the git repository",
					},
					"note_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the note to dismiss",
					},
					"dismissed_by": map[string]interface{}{
						"type":        "string",
						"description": "Identifier of who is dismissing the note",
					},
				},
				"required": []string{"repo_path", "note_id", "dismissed_by"},
			},
		},
	}

	return map[string]interface{}{
		"tools": tools,
	}
}

func ListComments(paramsRaw json.RawMessage) (interface{}, error) {
	stateMgr, err := state.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}
	return ListCommentsWithManager(paramsRaw, stateMgr)
}

func ListCommentsWithManager(paramsRaw json.RawMessage, stateMgr *state.Manager) (interface{}, error) {
	var params ListCommentsParams
	if err := json.Unmarshal(paramsRaw, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if params.RepoPath == "" {
		return nil, fmt.Errorf("repo_path is required")
	}

	repoPath := params.RepoPath

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

func ResolveComment(paramsRaw json.RawMessage) (interface{}, error) {
	stateMgr, err := state.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}
	return ResolveCommentWithManager(paramsRaw, stateMgr)
}

func ResolveCommentWithManager(paramsRaw json.RawMessage, stateMgr *state.Manager) (interface{}, error) {
	var params ResolveCommentParams
	if err := json.Unmarshal(paramsRaw, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if params.RepoPath == "" {
		return nil, fmt.Errorf("repo_path is required")
	}

	if params.CommentID == "" {
		return nil, fmt.Errorf("comment_id is required")
	}

	if params.ResolvedBy == "" {
		return nil, fmt.Errorf("resolved_by is required")
	}

	repoPath := params.RepoPath

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

func AddComment(paramsRaw json.RawMessage) (interface{}, error) {
	stateMgr, err := state.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}
	return AddCommentWithManager(paramsRaw, stateMgr)
}

func AddCommentWithManager(paramsRaw json.RawMessage, stateMgr *state.Manager) (interface{}, error) {
	var params AddCommentParams
	if err := json.Unmarshal(paramsRaw, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if params.RepoPath == "" {
		return nil, fmt.Errorf("repo_path is required")
	}
	if params.Branch == "" {
		return nil, fmt.Errorf("branch is required")
	}
	if params.Commit == "" {
		return nil, fmt.Errorf("commit is required")
	}
	if params.FilePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}
	if params.Text == "" {
		return nil, fmt.Errorf("text is required")
	}

	repoPath := params.RepoPath

	// Make path absolute
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("invalid repo_path: %w", err)
	}

	comment, err := stateMgr.AddComment(
		absPath,
		params.Branch,
		params.Commit,
		params.FilePath,
		params.LineNumber,
		params.Text,
		params.Author,
		params.Type,
		params.ParentID,
		params.Metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add comment: %w", err)
	}

	return map[string]interface{}{
		"success":     true,
		"comment_id":  comment.ID,
		"author":      comment.Author,
		"type":        comment.Type,
		"parent_id":   comment.ParentID,
		"repo_path":   absPath,
	}, nil
}

func AddNote(paramsRaw json.RawMessage) (interface{}, error) {
	stateMgr, err := state.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}
	return AddNoteWithManager(paramsRaw, stateMgr)
}

func AddNoteWithManager(paramsRaw json.RawMessage, stateMgr *state.Manager) (interface{}, error) {
	var params AddNoteParams
	if err := json.Unmarshal(paramsRaw, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if params.RepoPath == "" {
		return nil, fmt.Errorf("repo_path is required")
	}

	if params.Branch == "" {
		return nil, fmt.Errorf("branch is required")
	}

	if params.Commit == "" {
		return nil, fmt.Errorf("commit is required")
	}

	if params.FilePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}

	if params.Text == "" {
		return nil, fmt.Errorf("text is required")
	}

	if params.Author == "" {
		return nil, fmt.Errorf("author is required")
	}

	// Default type to "explanation" if not provided
	noteType := params.Type
	if noteType == "" {
		noteType = "explanation"
	}

	// Make path absolute
	absPath, err := filepath.Abs(params.RepoPath)
	if err != nil {
		return nil, fmt.Errorf("invalid repo_path: %w", err)
	}

	note, err := stateMgr.AddNote(
		absPath,
		params.Branch,
		params.Commit,
		params.FilePath,
		params.LineNumber,
		params.Text,
		params.Author,
		noteType,
		params.Metadata,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to add note: %w", err)
	}

	return map[string]interface{}{
		"success":   true,
		"note_id":   note.ID,
		"author":    note.Author,
		"type":      note.Type,
		"repo_path": absPath,
	}, nil
}

func ListNotes(paramsRaw json.RawMessage) (interface{}, error) {
	stateMgr, err := state.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}
	return ListNotesWithManager(paramsRaw, stateMgr)
}

func ListNotesWithManager(paramsRaw json.RawMessage, stateMgr *state.Manager) (interface{}, error) {
	var params ListNotesParams
	if err := json.Unmarshal(paramsRaw, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if params.RepoPath == "" {
		return nil, fmt.Errorf("repo_path is required")
	}

	// Make path absolute
	absPath, err := filepath.Abs(params.RepoPath)
	if err != nil {
		return nil, fmt.Errorf("invalid repo_path: %w", err)
	}

	var notes []*state.Note

	// If branch and commit are specified, get notes for that specific state
	if params.Branch != nil && params.Commit != nil {
		notes = stateMgr.GetNotes(absPath, *params.Branch, *params.Commit, params.FilePath)
	} else {
		// Otherwise get all notes for the repo
		notes = stateMgr.GetAllNotes(absPath)
	}

	// Filter by dismissal status if specified
	if params.Dismissed != nil {
		filtered := []*state.Note{}
		for _, n := range notes {
			if n.Dismissed == *params.Dismissed {
				filtered = append(filtered, n)
			}
		}
		notes = filtered
	}

	// Filter by author if specified
	if params.Author != nil {
		filtered := []*state.Note{}
		for _, n := range notes {
			if n.Author == *params.Author {
				filtered = append(filtered, n)
			}
		}
		notes = filtered
	}

	// Filter by file path if specified (and not already filtered by GetNotes)
	if params.FilePath != nil && (params.Branch == nil || params.Commit == nil) {
		filtered := []*state.Note{}
		for _, n := range notes {
			if n.FilePath == *params.FilePath {
				filtered = append(filtered, n)
			}
		}
		notes = filtered
	}

	// Convert to result format
	results := make([]NoteResult, len(notes))
	for i, n := range notes {
		results[i] = NoteResult{
			ID:          n.ID,
			FilePath:    n.FilePath,
			LineNumber:  n.LineNumber,
			Text:        n.Text,
			Timestamp:   n.Timestamp,
			Branch:      n.Branch,
			Commit:      n.Commit,
			Author:      n.Author,
			Type:        n.Type,
			Metadata:    n.Metadata,
			Dismissed:   n.Dismissed,
			DismissedBy: n.DismissedBy,
			DismissedAt: n.DismissedAt,
		}
	}

	return map[string]interface{}{
		"notes":     results,
		"count":     len(results),
		"repo_path": absPath,
	}, nil
}

func DismissNote(paramsRaw json.RawMessage) (interface{}, error) {
	stateMgr, err := state.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}
	return DismissNoteWithManager(paramsRaw, stateMgr)
}

func DismissNoteWithManager(paramsRaw json.RawMessage, stateMgr *state.Manager) (interface{}, error) {
	var params DismissNoteParams
	if err := json.Unmarshal(paramsRaw, &params); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if params.RepoPath == "" {
		return nil, fmt.Errorf("repo_path is required")
	}

	if params.NoteID == "" {
		return nil, fmt.Errorf("note_id is required")
	}

	if params.DismissedBy == "" {
		return nil, fmt.Errorf("dismissed_by is required")
	}

	// Make path absolute
	absPath, err := filepath.Abs(params.RepoPath)
	if err != nil {
		return nil, fmt.Errorf("invalid repo_path: %w", err)
	}

	// Get all notes to find the one to dismiss
	allNotes := stateMgr.GetAllNotes(absPath)

	var targetNote *state.Note
	for _, n := range allNotes {
		if n.ID == params.NoteID {
			targetNote = n
			break
		}
	}

	if targetNote == nil {
		return nil, fmt.Errorf("note not found: %s", params.NoteID)
	}

	// Dismiss the note
	if err := stateMgr.DismissNote(absPath, targetNote.Branch, targetNote.Commit, params.NoteID, params.DismissedBy); err != nil {
		return nil, fmt.Errorf("failed to dismiss note: %w", err)
	}

	return map[string]interface{}{
		"success":      true,
		"note_id":      params.NoteID,
		"dismissed_by": params.DismissedBy,
		"repo_path":    absPath,
	}, nil
}
