// ABOUTME: Tests for the export package that handles JSON export of comments and notes.
// ABOUTME: Tests cover empty state, comments, notes, summary counts, and file output.

package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExportEmptyState(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "comments_export.json")

	err := Export("/test/repo", nil, nil, outputPath)
	if err != nil {
		t.Fatalf("Failed to export empty state: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("Export file should exist")
	}

	// Read and parse the file
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	var exportData ExportData
	if err := json.Unmarshal(data, &exportData); err != nil {
		t.Fatalf("Failed to parse export JSON: %v", err)
	}

	// Verify structure
	if exportData.GeneratedAt == "" {
		t.Error("GeneratedAt should be set")
	}

	if exportData.Comments == nil {
		t.Error("Comments should be initialized (not nil)")
	}

	if len(exportData.Comments) != 0 {
		t.Errorf("Expected 0 comments, got %d", len(exportData.Comments))
	}

	if exportData.Notes == nil {
		t.Error("Notes should be initialized (not nil)")
	}

	if len(exportData.Notes) != 0 {
		t.Errorf("Expected 0 notes, got %d", len(exportData.Notes))
	}

	if exportData.Summary.TotalComments != 0 {
		t.Errorf("Expected 0 total comments, got %d", exportData.Summary.TotalComments)
	}
}

func TestExportWithComments(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "comments_export.json")

	lineNum := 42
	comment := &Comment{
		ID:         "123-0",
		FilePath:   "internal/server/server.go",
		LineNumber: &lineNum,
		Text:       "This needs better error handling",
		Timestamp:  time.Now().Unix(),
		Branch:     "feat/test",
		Commit:     "abc1234",
		Resolved:   false,
	}
	comments := []*Comment{comment}

	err := Export("/test/repo", comments, nil, outputPath)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	var exportData ExportData
	if err := json.Unmarshal(data, &exportData); err != nil {
		t.Fatalf("Failed to parse export JSON: %v", err)
	}

	if exportData.RepoPath != "/test/repo" {
		t.Errorf("Expected RepoPath /test/repo, got %s", exportData.RepoPath)
	}

	if len(exportData.Comments) != 1 {
		t.Fatalf("Expected 1 comment, got %d", len(exportData.Comments))
	}

	exported := exportData.Comments[0]
	if exported.ID != comment.ID {
		t.Errorf("Expected ID %s, got %s", comment.ID, exported.ID)
	}

	if exported.FilePath != comment.FilePath {
		t.Errorf("Expected FilePath %s, got %s", comment.FilePath, exported.FilePath)
	}

	if exported.Text != comment.Text {
		t.Errorf("Expected Text %s, got %s", comment.Text, exported.Text)
	}

	if exported.Resolved != comment.Resolved {
		t.Errorf("Expected Resolved %v, got %v", comment.Resolved, exported.Resolved)
	}
}

func TestExportWithNotes(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "comments_export.json")

	lineNum := 85
	note := &Note{
		ID:         "456-0",
		FilePath:   "internal/state/state.go",
		LineNumber: &lineNum,
		Text:       "Consider adding indexing for larger datasets",
		Timestamp:  time.Now().Unix(),
		Branch:     "feat/test",
		Commit:     "abc1234",
		Author:     "claude",
		Type:       "suggestion",
		Dismissed:  false,
	}
	notes := []*Note{note}

	err := Export("/test/repo", nil, notes, outputPath)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	var exportData ExportData
	if err := json.Unmarshal(data, &exportData); err != nil {
		t.Fatalf("Failed to parse export JSON: %v", err)
	}

	if len(exportData.Notes) != 1 {
		t.Fatalf("Expected 1 note, got %d", len(exportData.Notes))
	}

	exported := exportData.Notes[0]
	if exported.ID != note.ID {
		t.Errorf("Expected ID %s, got %s", note.ID, exported.ID)
	}

	if exported.Author != note.Author {
		t.Errorf("Expected Author %s, got %s", note.Author, exported.Author)
	}

	if exported.Type != note.Type {
		t.Errorf("Expected Type %s, got %s", note.Type, exported.Type)
	}
}

func TestExportMixedContent(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "comments_export.json")

	lineNum := 10
	comment1 := &Comment{
		ID:       "c1",
		FilePath: "file1.go",
		Text:     "Comment 1",
		Branch:   "main",
		Commit:   "commit1",
		Resolved: false,
	}
	comment2 := &Comment{
		ID:         "c2",
		FilePath:   "file2.go",
		LineNumber: &lineNum,
		Text:       "Comment 2",
		Branch:     "main",
		Commit:     "commit1",
		Resolved:   true,
		ResolvedBy: "user",
	}
	note1 := &Note{
		ID:        "n1",
		FilePath:  "file1.go",
		Text:      "Note 1",
		Branch:    "main",
		Commit:    "commit1",
		Author:    "claude",
		Type:      "explanation",
		Dismissed: false,
	}

	comments := []*Comment{comment1, comment2}
	notes := []*Note{note1}

	err := Export("/test/repo", comments, notes, outputPath)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	var exportData ExportData
	if err := json.Unmarshal(data, &exportData); err != nil {
		t.Fatalf("Failed to parse export JSON: %v", err)
	}

	if len(exportData.Comments) != 2 {
		t.Errorf("Expected 2 comments, got %d", len(exportData.Comments))
	}

	if len(exportData.Notes) != 1 {
		t.Errorf("Expected 1 note, got %d", len(exportData.Notes))
	}
}

func TestExportSummary(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "comments_export.json")

	// Add 3 comments: 2 unresolved, 1 resolved
	comments := []*Comment{
		{ID: "c1", Resolved: false},
		{ID: "c2", Resolved: false},
		{ID: "c3", Resolved: true},
	}

	// Add 2 notes: 1 active, 1 dismissed
	notes := []*Note{
		{ID: "n1", Dismissed: false},
		{ID: "n2", Dismissed: true},
	}

	err := Export("/test/repo", comments, notes, outputPath)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	var exportData ExportData
	if err := json.Unmarshal(data, &exportData); err != nil {
		t.Fatalf("Failed to parse export JSON: %v", err)
	}

	if exportData.Summary.TotalComments != 3 {
		t.Errorf("Expected 3 total comments, got %d", exportData.Summary.TotalComments)
	}

	if exportData.Summary.UnresolvedComments != 2 {
		t.Errorf("Expected 2 unresolved comments, got %d", exportData.Summary.UnresolvedComments)
	}

	if exportData.Summary.TotalNotes != 2 {
		t.Errorf("Expected 2 total notes, got %d", exportData.Summary.TotalNotes)
	}

	if exportData.Summary.ActiveNotes != 1 {
		t.Errorf("Expected 1 active note, got %d", exportData.Summary.ActiveNotes)
	}
}

func TestExportCustomPath(t *testing.T) {
	tempDir := t.TempDir()
	customPath := filepath.Join(tempDir, "custom", "nested", "export.json")

	comments := []*Comment{{ID: "c1", Text: "test"}}

	err := Export("/test/repo", comments, nil, customPath)
	if err != nil {
		t.Fatalf("Failed to export to custom path: %v", err)
	}

	// Verify file was created at custom path
	if _, err := os.Stat(customPath); os.IsNotExist(err) {
		t.Fatal("Export file should exist at custom path")
	}

	// Verify content
	data, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	var exportData ExportData
	if err := json.Unmarshal(data, &exportData); err != nil {
		t.Fatalf("Failed to parse export JSON: %v", err)
	}

	if len(exportData.Comments) != 1 {
		t.Errorf("Expected 1 comment, got %d", len(exportData.Comments))
	}
}

func TestExportIsValidJSON(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "comments_export.json")

	lineNum := 42
	metadata := map[string]string{"model": "claude-sonnet-4", "context": "review"}
	comments := []*Comment{{
		ID:         "c1",
		FilePath:   "file.go",
		LineNumber: &lineNum,
		Text:       "Comment with \"quotes\" and\nnewlines",
		Branch:     "main",
		Commit:     "c1",
		Resolved:   true,
		ResolvedBy: "user",
		ResolvedAt: time.Now().Unix(),
	}}
	notes := []*Note{{
		ID:          "n1",
		FilePath:    "file.go",
		LineNumber:  &lineNum,
		Text:        "Note with special chars: <>&",
		Branch:      "main",
		Commit:      "c1",
		Author:      "claude",
		Type:        "suggestion",
		Metadata:    metadata,
		Dismissed:   true,
		DismissedBy: "user",
		DismissedAt: time.Now().Unix(),
	}}

	err := Export("/test/repo", comments, notes, outputPath)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	// Should be valid JSON
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Export is not valid JSON: %v", err)
	}

	// Should be able to round-trip back to ExportData
	var exportData ExportData
	if err := json.Unmarshal(data, &exportData); err != nil {
		t.Fatalf("Failed to unmarshal to ExportData: %v", err)
	}

	if exportData.Comments[0].Text != "Comment with \"quotes\" and\nnewlines" {
		t.Error("Comment text not preserved correctly")
	}

	if exportData.Notes[0].Metadata["model"] != "claude-sonnet-4" {
		t.Error("Note metadata not preserved correctly")
	}
}

func TestExportMultipleRepos(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "comments_export.json")

	// Simulate comments/notes from multiple repos (already flattened by caller)
	comments := []*Comment{
		{ID: "c1", Text: "Repo 1 comment"},
		{ID: "c2", Text: "Repo 2 comment"},
	}
	notes := []*Note{
		{ID: "n1", Text: "Repo 1 note"},
	}

	err := Export("/test/repo", comments, notes, outputPath)
	if err != nil {
		t.Fatalf("Failed to export: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	var exportData ExportData
	if err := json.Unmarshal(data, &exportData); err != nil {
		t.Fatalf("Failed to parse export JSON: %v", err)
	}

	// Should have all comments/notes
	if len(exportData.Comments) != 2 {
		t.Errorf("Expected 2 comments, got %d", len(exportData.Comments))
	}

	if len(exportData.Notes) != 1 {
		t.Errorf("Expected 1 note, got %d", len(exportData.Notes))
	}
}

func TestGetExportPathForRepo(t *testing.T) {
	path1, err := GetExportPathForRepo("/test/repo1")
	if err != nil {
		t.Fatalf("Failed to get export path: %v", err)
	}

	path2, err := GetExportPathForRepo("/test/repo2")
	if err != nil {
		t.Fatalf("Failed to get export path: %v", err)
	}

	// Different repos should have different paths
	if path1 == path2 {
		t.Error("Different repos should have different export paths")
	}

	// Same repo should have same path
	path1Again, err := GetExportPathForRepo("/test/repo1")
	if err != nil {
		t.Fatalf("Failed to get export path: %v", err)
	}

	if path1 != path1Again {
		t.Error("Same repo should have same export path")
	}

	// Should end with comments_export.json
	if filepath.Base(path1) != "comments_export.json" {
		t.Errorf("Expected filename comments_export.json, got %s", filepath.Base(path1))
	}
}

func TestHashRepoPath(t *testing.T) {
	hash1 := hashRepoPath("/test/repo1")
	hash2 := hashRepoPath("/test/repo2")

	if hash1 == hash2 {
		t.Error("Different paths should have different hashes")
	}

	// Should be consistent
	hash1Again := hashRepoPath("/test/repo1")
	if hash1 != hash1Again {
		t.Error("Same path should have same hash")
	}

	// Should be 16 chars (8 bytes hex encoded)
	if len(hash1) != 16 {
		t.Errorf("Expected hash length 16, got %d", len(hash1))
	}
}
