package state

import (
	"testing"
)

func TestAddNote(t *testing.T) {
	manager, repoPath := setupTestManager(t)

	branch := "main"
	commit := "abc123"
	filePath := "test.go"
	lineNumber := 42
	text := "This implementation uses a binary search algorithm for O(log n) performance"
	author := "claude"
	noteType := "explanation"
	metadata := map[string]string{"model": "claude-sonnet-4"}

	note, err := manager.AddNote(repoPath, branch, commit, filePath, &lineNumber, text, author, noteType, metadata)
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	if note.Text != text {
		t.Errorf("Expected text %s, got %s", text, note.Text)
	}

	if note.Author != author {
		t.Errorf("Expected author %s, got %s", author, note.Author)
	}

	if note.Type != noteType {
		t.Errorf("Expected type %s, got %s", noteType, note.Type)
	}

	if note.Dismissed {
		t.Error("Note should not be dismissed initially")
	}
}

func TestGetNotes(t *testing.T) {
	manager, repoPath := setupTestManager(t)

	branch := "main"
	commit := "abc123"
	lineNumber := 42

	// Add multiple notes
	_, err := manager.AddNote(repoPath, branch, commit, "file1.go", &lineNumber, "Note 1", "claude", "explanation", nil)
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	_, err = manager.AddNote(repoPath, branch, commit, "file2.go", &lineNumber, "Note 2", "copilot", "suggestion", nil)
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	// Get all notes
	notes := manager.GetNotes(repoPath, branch, commit, nil)
	if len(notes) != 2 {
		t.Errorf("Expected 2 notes, got %d", len(notes))
	}

	// Get notes for specific file
	file1 := "file1.go"
	notes = manager.GetNotes(repoPath, branch, commit, &file1)
	if len(notes) != 1 {
		t.Errorf("Expected 1 note for file1.go, got %d", len(notes))
	}

	if notes[0].Text != "Note 1" {
		t.Errorf("Expected 'Note 1', got %s", notes[0].Text)
	}
}

func TestGetAllNotes(t *testing.T) {
	manager, repoPath := setupTestManager(t)

	lineNumber := 42

	// Add notes to different branches/commits
	_, err := manager.AddNote(repoPath, "main", "commit1", "file.go", &lineNumber, "Note on main", "claude", "explanation", nil)
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	_, err = manager.AddNote(repoPath, "feature", "commit2", "file.go", &lineNumber, "Note on feature", "claude", "explanation", nil)
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	allNotes := manager.GetAllNotes(repoPath)
	if len(allNotes) != 2 {
		t.Errorf("Expected 2 notes across all branches, got %d", len(allNotes))
	}
}

func TestDismissNote(t *testing.T) {
	manager, repoPath := setupTestManager(t)

	branch := "main"
	commit := "abc123"
	lineNumber := 42

	note, err := manager.AddNote(repoPath, branch, commit, "file.go", &lineNumber, "Test note", "claude", "explanation", nil)
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	// Dismiss the note
	err = manager.DismissNote(repoPath, branch, commit, note.ID, "test-user")
	if err != nil {
		t.Fatalf("Failed to dismiss note: %v", err)
	}

	// Verify note is dismissed
	notes := manager.GetNotes(repoPath, branch, commit, nil)
	if len(notes) != 1 {
		t.Fatalf("Expected 1 note, got %d", len(notes))
	}

	if !notes[0].Dismissed {
		t.Error("Note should be dismissed")
	}

	if notes[0].DismissedBy != "test-user" {
		t.Errorf("Expected dismissed by test-user, got %s", notes[0].DismissedBy)
	}

	if notes[0].DismissedAt == 0 {
		t.Error("DismissedAt should be set")
	}
}

func TestNoteMetadata(t *testing.T) {
	manager, repoPath := setupTestManager(t)

	branch := "main"
	commit := "abc123"
	lineNumber := 42
	metadata := map[string]string{
		"model":       "claude-sonnet-4",
		"temperature": "0.7",
		"context":     "code-review",
	}

	note, err := manager.AddNote(repoPath, branch, commit, "file.go", &lineNumber, "Test note", "claude", "explanation", metadata)
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	if note.Metadata["model"] != "claude-sonnet-4" {
		t.Errorf("Expected metadata model=claude-sonnet-4, got %s", note.Metadata["model"])
	}

	if len(note.Metadata) != 3 {
		t.Errorf("Expected 3 metadata entries, got %d", len(note.Metadata))
	}
}

func TestNoteWithoutLineNumber(t *testing.T) {
	manager, repoPath := setupTestManager(t)

	branch := "main"
	commit := "abc123"
	text := "General note about this file's architecture"

	note, err := manager.AddNote(repoPath, branch, commit, "file.go", nil, text, "claude", "explanation", nil)
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	if note.LineNumber != nil {
		t.Error("LineNumber should be nil for general notes")
	}

	if note.Text != text {
		t.Errorf("Expected text %s, got %s", text, note.Text)
	}
}
