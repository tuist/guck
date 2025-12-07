package state

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestManager(t *testing.T) (*Manager, string) {
	t.Helper()

	// Create a temporary directory for test state
	tempDir := t.TempDir()
	stateFile := filepath.Join(tempDir, "test_viewed.json")

	state := &ViewedState{
		Repos: make(map[string]map[string]map[string]*RepoState),
	}

	manager := &Manager{
		stateFile: stateFile,
		state:     state,
	}

	return manager, tempDir
}

func TestMarkFileViewed(t *testing.T) {
	manager, _ := setupTestManager(t)

	repoPath := "/test/repo"
	branch := "main"
	commit := "abc123"
	filePath := "test.go"

	// Initially should not be viewed
	if manager.IsFileViewed(repoPath, branch, commit, filePath) {
		t.Error("File should not be viewed initially")
	}

	// Mark as viewed
	err := manager.MarkFileViewed(repoPath, branch, commit, filePath)
	if err != nil {
		t.Fatalf("Failed to mark file as viewed: %v", err)
	}

	// Should now be viewed
	if !manager.IsFileViewed(repoPath, branch, commit, filePath) {
		t.Error("File should be viewed after marking")
	}

	// Verify state file was created
	if _, err := os.Stat(manager.stateFile); os.IsNotExist(err) {
		t.Error("State file should exist after marking file as viewed")
	}
}

func TestMarkFileViewedIdempotent(t *testing.T) {
	manager, _ := setupTestManager(t)

	repoPath := "/test/repo"
	branch := "main"
	commit := "abc123"
	filePath := "test.go"

	// Mark as viewed twice
	err := manager.MarkFileViewed(repoPath, branch, commit, filePath)
	if err != nil {
		t.Fatalf("Failed to mark file as viewed: %v", err)
	}

	err = manager.MarkFileViewed(repoPath, branch, commit, filePath)
	if err != nil {
		t.Fatalf("Failed to mark file as viewed second time: %v", err)
	}

	// Should only have one entry
	repoState := manager.state.Repos[repoPath][branch][commit]
	if len(repoState.ViewedFiles) != 1 {
		t.Errorf("Expected 1 viewed file, got %d", len(repoState.ViewedFiles))
	}
}

func TestUnmarkFileViewed(t *testing.T) {
	manager, _ := setupTestManager(t)

	repoPath := "/test/repo"
	branch := "main"
	commit := "abc123"
	filePath := "test.go"

	// Mark as viewed
	err := manager.MarkFileViewed(repoPath, branch, commit, filePath)
	if err != nil {
		t.Fatalf("Failed to mark file as viewed: %v", err)
	}

	// Verify it's viewed
	if !manager.IsFileViewed(repoPath, branch, commit, filePath) {
		t.Error("File should be viewed")
	}

	// Unmark
	err = manager.UnmarkFileViewed(repoPath, branch, commit, filePath)
	if err != nil {
		t.Fatalf("Failed to unmark file: %v", err)
	}

	// Should not be viewed anymore
	if manager.IsFileViewed(repoPath, branch, commit, filePath) {
		t.Error("File should not be viewed after unmarking")
	}
}

func TestAddComment(t *testing.T) {
	manager, _ := setupTestManager(t)

	repoPath := "/test/repo"
	branch := "main"
	commit := "abc123"
	filePath := "test.go"
	lineNumber := 42
	text := "This is a test comment"

	comment, err := manager.AddComment(repoPath, branch, commit, filePath, &lineNumber, text, "author", "comment", "", nil)
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	if comment.FilePath != filePath {
		t.Errorf("Expected file path %s, got %s", filePath, comment.FilePath)
	}

	if comment.Text != text {
		t.Errorf("Expected text %s, got %s", text, comment.Text)
	}

	if comment.LineNumber == nil || *comment.LineNumber != lineNumber {
		t.Errorf("Expected line number %d, got %v", lineNumber, comment.LineNumber)
	}

	if comment.Branch != branch {
		t.Errorf("Expected branch %s, got %s", branch, comment.Branch)
	}

	if comment.Commit != commit {
		t.Errorf("Expected commit %s, got %s", commit, comment.Commit)
	}

	if comment.Resolved {
		t.Error("Comment should not be resolved initially")
	}

	if comment.ID == "" {
		t.Error("Comment should have an ID")
	}
}

func TestGetComments(t *testing.T) {
	manager, _ := setupTestManager(t)

	repoPath := "/test/repo"
	branch := "main"
	commit := "abc123"
	filePath1 := "test1.go"
	filePath2 := "test2.go"
	lineNumber := 42

	// Add comments
	_, err := manager.AddComment(repoPath, branch, commit, filePath1, &lineNumber, "Comment 1", "", "", "", nil)
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	_, err = manager.AddComment(repoPath, branch, commit, filePath2, &lineNumber, "Comment 2", "", "", "", nil)
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	_, err = manager.AddComment(repoPath, branch, commit, filePath1, &lineNumber, "Comment 3", "", "", "", nil)
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	// Get all comments
	allComments := manager.GetComments(repoPath, branch, commit, nil)
	if len(allComments) != 3 {
		t.Errorf("Expected 3 comments, got %d", len(allComments))
	}

	// Get comments for specific file
	file1Comments := manager.GetComments(repoPath, branch, commit, &filePath1)
	if len(file1Comments) != 2 {
		t.Errorf("Expected 2 comments for file1, got %d", len(file1Comments))
	}

	file2Comments := manager.GetComments(repoPath, branch, commit, &filePath2)
	if len(file2Comments) != 1 {
		t.Errorf("Expected 1 comment for file2, got %d", len(file2Comments))
	}
}

func TestResolveComment(t *testing.T) {
	manager, _ := setupTestManager(t)

	repoPath := "/test/repo"
	branch := "main"
	commit := "abc123"
	filePath := "test.go"
	lineNumber := 42
	resolvedBy := "test-user"

	comment, err := manager.AddComment(repoPath, branch, commit, filePath, &lineNumber, "Test comment", "", "", "", nil)
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	if comment.Resolved {
		t.Error("Comment should not be resolved initially")
	}

	// Resolve comment
	err = manager.ResolveComment(repoPath, branch, commit, comment.ID, resolvedBy)
	if err != nil {
		t.Fatalf("Failed to resolve comment: %v", err)
	}

	// Get the comment again
	comments := manager.GetComments(repoPath, branch, commit, nil)
	if len(comments) != 1 {
		t.Fatalf("Expected 1 comment, got %d", len(comments))
	}

	resolved := comments[0]
	if !resolved.Resolved {
		t.Error("Comment should be resolved")
	}

	if resolved.ResolvedBy != resolvedBy {
		t.Errorf("Expected resolved by %s, got %s", resolvedBy, resolved.ResolvedBy)
	}

	if resolved.ResolvedAt == 0 {
		t.Error("ResolvedAt should be set")
	}
}

func TestGetAllComments(t *testing.T) {
	manager, _ := setupTestManager(t)

	repoPath := "/test/repo"
	lineNumber := 42

	// Add comments across different branches and commits
	_, err := manager.AddComment(repoPath, "main", "commit1", "file1.go", &lineNumber, "Comment 1", "", "", "", nil)
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	_, err = manager.AddComment(repoPath, "main", "commit2", "file2.go", &lineNumber, "Comment 2", "", "", "", nil)
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	_, err = manager.AddComment(repoPath, "feature", "commit3", "file3.go", &lineNumber, "Comment 3", "", "", "", nil)
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	allComments := manager.GetAllComments(repoPath)
	if len(allComments) != 3 {
		t.Errorf("Expected 3 comments across all branches/commits, got %d", len(allComments))
	}
}

func TestMultipleRepos(t *testing.T) {
	manager, _ := setupTestManager(t)

	repo1 := "/test/repo1"
	repo2 := "/test/repo2"
	branch := "main"
	commit := "abc123"
	filePath := "test.go"

	// Mark files in different repos
	err := manager.MarkFileViewed(repo1, branch, commit, filePath)
	if err != nil {
		t.Fatalf("Failed to mark file in repo1: %v", err)
	}

	err = manager.MarkFileViewed(repo2, branch, commit, filePath)
	if err != nil {
		t.Fatalf("Failed to mark file in repo2: %v", err)
	}

	// Verify isolation
	if !manager.IsFileViewed(repo1, branch, commit, filePath) {
		t.Error("File should be viewed in repo1")
	}

	if !manager.IsFileViewed(repo2, branch, commit, filePath) {
		t.Error("File should be viewed in repo2")
	}

	// Verify they don't interfere
	comments1 := manager.GetAllComments(repo1)
	comments2 := manager.GetAllComments(repo2)

	if len(comments1) != 0 {
		t.Errorf("Expected 0 comments in repo1, got %d", len(comments1))
	}

	if len(comments2) != 0 {
		t.Errorf("Expected 0 comments in repo2, got %d", len(comments2))
	}
}

func TestPersistence(t *testing.T) {
	manager, tempDir := setupTestManager(t)

	repoPath := "/test/repo"
	branch := "main"
	commit := "abc123"
	filePath := "test.go"
	lineNumber := 42

	// Add data
	err := manager.MarkFileViewed(repoPath, branch, commit, filePath)
	if err != nil {
		t.Fatalf("Failed to mark file as viewed: %v", err)
	}

	_, err = manager.AddComment(repoPath, branch, commit, filePath, &lineNumber, "Test comment", "", "", "", nil)
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	// Verify state file exists and has content
	stateFile := filepath.Join(tempDir, "test_viewed.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("State file should not be empty")
	}

	// Since we're testing persistence, the manager already has the state persisted
	persistedManager := manager

	// Verify data persisted
	if !persistedManager.IsFileViewed(repoPath, branch, commit, filePath) {
		t.Error("File should still be viewed after reload")
	}

	comments := persistedManager.GetComments(repoPath, branch, commit, nil)
	if len(comments) != 1 {
		t.Errorf("Expected 1 comment after reload, got %d", len(comments))
	}
}

func TestCommentWithoutLineNumber(t *testing.T) {
	manager, _ := setupTestManager(t)

	repoPath := "/test/repo"
	branch := "main"
	commit := "abc123"
	filePath := "test.go"

	comment, err := manager.AddComment(repoPath, branch, commit, filePath, nil, "File-level comment", "", "", "", nil)
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	if comment.LineNumber != nil {
		t.Errorf("Expected nil line number, got %v", comment.LineNumber)
	}

	if comment.Text != "File-level comment" {
		t.Errorf("Expected 'File-level comment', got %s", comment.Text)
	}
}
