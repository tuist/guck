package mcp

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/tuist/guck/internal/state"
)

// Create a test manager with a temporary state file
func createTestManager(t *testing.T) (*state.Manager, string) {
	t.Helper()
	tempDir := t.TempDir()
	testRepoPath := filepath.Join(tempDir, "test-repo")

	// Override XDG_STATE_HOME to use temp directory
	t.Setenv("XDG_STATE_HOME", tempDir)

	manager, err := state.NewManager()
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	return manager, testRepoPath
}

func TestListTools(t *testing.T) {
	tools := ListTools()

	toolsList, ok := tools["tools"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected tools to be a slice of maps")
	}

	if len(toolsList) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(toolsList))
	}

	// Check list_comments tool
	listCommentsTool := toolsList[0]
	if listCommentsTool["name"] != "list_comments" {
		t.Errorf("Expected first tool to be list_comments, got %s", listCommentsTool["name"])
	}

	// Check resolve_comment tool
	resolveCommentTool := toolsList[1]
	if resolveCommentTool["name"] != "resolve_comment" {
		t.Errorf("Expected second tool to be resolve_comment, got %s", resolveCommentTool["name"])
	}
}

func TestListCommentsWithManager_EmptyRepo(t *testing.T) {
	manager, repoPath := createTestManager(t)

	params := ListCommentsParams{
		RepoPath: repoPath,
	}
	paramsJSON, _ := json.Marshal(params)

	result, err := ListCommentsWithManager(paramsJSON, manager)
	if err != nil {
		t.Fatalf("ListCommentsWithManager failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected result to be a map")
	}

	count, ok := resultMap["count"].(int)
	if !ok {
		t.Fatal("Expected count to be an int")
	}

	if count != 0 {
		t.Errorf("Expected 0 comments, got %d", count)
	}
}

func TestListCommentsWithManager_WithComments(t *testing.T) {
	manager, repoPath := createTestManager(t)

	// Add some test comments
	branch := "main"
	commit := "abc123"
	filePath := "test.go"
	lineNumber := 42

	_, err := manager.AddComment(repoPath, branch, commit, filePath, &lineNumber, "Test comment 1")
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	_, err = manager.AddComment(repoPath, branch, commit, filePath, &lineNumber, "Test comment 2")
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	// List all comments
	params := ListCommentsParams{
		RepoPath: repoPath,
	}
	paramsJSON, _ := json.Marshal(params)

	result, err := ListCommentsWithManager(paramsJSON, manager)
	if err != nil {
		t.Fatalf("ListCommentsWithManager failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	count := resultMap["count"].(int)

	if count != 2 {
		t.Errorf("Expected 2 comments, got %d", count)
	}

	comments := resultMap["comments"].([]CommentResult)
	if len(comments) != 2 {
		t.Errorf("Expected 2 comments in array, got %d", len(comments))
	}
}

func TestListCommentsWithManager_FilterByBranchAndCommit(t *testing.T) {
	manager, repoPath := createTestManager(t)

	lineNumber := 42

	// Add comments to different branches/commits
	_, err := manager.AddComment(repoPath, "main", "commit1", "file.go", &lineNumber, "Comment 1")
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	_, err = manager.AddComment(repoPath, "main", "commit2", "file.go", &lineNumber, "Comment 2")
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	_, err = manager.AddComment(repoPath, "feature", "commit3", "file.go", &lineNumber, "Comment 3")
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	// Filter by branch and commit
	branch := "main"
	commit := "commit1"
	params := ListCommentsParams{
		RepoPath: repoPath,
		Branch:   &branch,
		Commit:   &commit,
	}
	paramsJSON, _ := json.Marshal(params)

	result, err := ListCommentsWithManager(paramsJSON, manager)
	if err != nil {
		t.Fatalf("ListCommentsWithManager failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	count := resultMap["count"].(int)

	if count != 1 {
		t.Errorf("Expected 1 comment for main/commit1, got %d", count)
	}
}

func TestListCommentsWithManager_FilterByResolved(t *testing.T) {
	manager, repoPath := createTestManager(t)

	branch := "main"
	commit := "abc123"
	lineNumber := 42

	// Add comments
	comment1, err := manager.AddComment(repoPath, branch, commit, "file.go", &lineNumber, "Comment 1")
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	_, err = manager.AddComment(repoPath, branch, commit, "file.go", &lineNumber, "Comment 2")
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	// Resolve one comment
	err = manager.ResolveComment(repoPath, branch, commit, comment1.ID, "test-user")
	if err != nil {
		t.Fatalf("Failed to resolve comment: %v", err)
	}

	// Filter by resolved=true
	resolvedTrue := true
	params := ListCommentsParams{
		RepoPath: repoPath,
		Resolved: &resolvedTrue,
	}
	paramsJSON, _ := json.Marshal(params)

	result, err := ListCommentsWithManager(paramsJSON, manager)
	if err != nil {
		t.Fatalf("ListCommentsWithManager failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	count := resultMap["count"].(int)

	if count != 1 {
		t.Errorf("Expected 1 resolved comment, got %d", count)
	}

	// Filter by resolved=false
	resolvedFalse := false
	params = ListCommentsParams{
		RepoPath: repoPath,
		Resolved: &resolvedFalse,
	}
	paramsJSON, _ = json.Marshal(params)

	result, err = ListCommentsWithManager(paramsJSON, manager)
	if err != nil {
		t.Fatalf("ListCommentsWithManager failed: %v", err)
	}

	resultMap = result.(map[string]interface{})
	count = resultMap["count"].(int)

	if count != 1 {
		t.Errorf("Expected 1 unresolved comment, got %d", count)
	}
}

func TestListCommentsWithManager_FilterByFilePath(t *testing.T) {
	manager, repoPath := createTestManager(t)

	branch := "main"
	commit := "abc123"
	lineNumber := 42

	// Add comments to different files
	_, err := manager.AddComment(repoPath, branch, commit, "file1.go", &lineNumber, "Comment 1")
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	_, err = manager.AddComment(repoPath, branch, commit, "file2.go", &lineNumber, "Comment 2")
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	// Filter by file path
	filePath := "file1.go"
	params := ListCommentsParams{
		RepoPath: repoPath,
		FilePath: &filePath,
	}
	paramsJSON, _ := json.Marshal(params)

	result, err := ListCommentsWithManager(paramsJSON, manager)
	if err != nil {
		t.Fatalf("ListCommentsWithManager failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	count := resultMap["count"].(int)

	if count != 1 {
		t.Errorf("Expected 1 comment for file1.go, got %d", count)
	}
}

func TestResolveCommentWithManager_Success(t *testing.T) {
	manager, repoPath := createTestManager(t)

	branch := "main"
	commit := "abc123"
	lineNumber := 42

	// Add a comment
	comment, err := manager.AddComment(repoPath, branch, commit, "file.go", &lineNumber, "Test comment")
	if err != nil {
		t.Fatalf("Failed to add comment: %v", err)
	}

	// Resolve it
	params := ResolveCommentParams{
		RepoPath:   repoPath,
		CommentID:  comment.ID,
		ResolvedBy: "test-user",
	}
	paramsJSON, _ := json.Marshal(params)

	result, err := ResolveCommentWithManager(paramsJSON, manager)
	if err != nil {
		t.Fatalf("ResolveCommentWithManager failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	success := resultMap["success"].(bool)

	if !success {
		t.Error("Expected success to be true")
	}

	// Verify comment is resolved
	comments := manager.GetComments(repoPath, branch, commit, nil)
	if len(comments) != 1 {
		t.Fatalf("Expected 1 comment, got %d", len(comments))
	}

	if !comments[0].Resolved {
		t.Error("Comment should be resolved")
	}

	if comments[0].ResolvedBy != "test-user" {
		t.Errorf("Expected resolved by test-user, got %s", comments[0].ResolvedBy)
	}
}

func TestResolveCommentWithManager_MissingCommentID(t *testing.T) {
	manager, repoPath := createTestManager(t)

	params := ResolveCommentParams{
		RepoPath:   repoPath,
		CommentID:  "",
		ResolvedBy: "test-user",
	}
	paramsJSON, _ := json.Marshal(params)

	_, err := ResolveCommentWithManager(paramsJSON, manager)
	if err == nil {
		t.Error("Expected error for missing comment_id")
	}
}

func TestResolveCommentWithManager_MissingResolvedBy(t *testing.T) {
	manager, repoPath := createTestManager(t)

	params := ResolveCommentParams{
		RepoPath:   repoPath,
		CommentID:  "some-id",
		ResolvedBy: "",
	}
	paramsJSON, _ := json.Marshal(params)

	_, err := ResolveCommentWithManager(paramsJSON, manager)
	if err == nil {
		t.Error("Expected error for missing resolved_by")
	}
}

func TestResolveCommentWithManager_CommentNotFound(t *testing.T) {
	manager, repoPath := createTestManager(t)

	params := ResolveCommentParams{
		RepoPath:   repoPath,
		CommentID:  "nonexistent-id",
		ResolvedBy: "test-user",
	}
	paramsJSON, _ := json.Marshal(params)

	_, err := ResolveCommentWithManager(paramsJSON, manager)
	if err == nil {
		t.Error("Expected error for nonexistent comment")
	}
}

func TestListCommentsWithManager_InvalidJSON(t *testing.T) {
	manager, _ := createTestManager(t)

	invalidJSON := []byte(`{"invalid": json}`)

	_, err := ListCommentsWithManager(invalidJSON, manager)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestListCommentsWithManager_MissingRepoPath(t *testing.T) {
	manager, _ := createTestManager(t)

	// List without specifying repo_path (should error)
	params := ListCommentsParams{}
	paramsJSON, _ := json.Marshal(params)

	_, err := ListCommentsWithManager(paramsJSON, manager)
	if err == nil {
		t.Error("Expected error for missing repo_path")
	}
}
