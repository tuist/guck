package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/tuist/guck/internal/mcp"
	"github.com/tuist/guck/internal/state"
)

func TestOutputJSON(t *testing.T) {
	result := map[string]interface{}{
		"success": true,
		"test":    "data",
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputJSON(result)
	if err != nil {
		t.Fatalf("outputJSON failed: %v", err)
	}

	w.Close()
	os.Stdout = old

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if parsed["success"] != true {
		t.Errorf("Expected success=true, got %v", parsed["success"])
	}
}

func TestOutputCommentResultsAsToon(t *testing.T) {
	line := 42
	comments := []mcp.CommentResult{
		{
			ID:         "test-id-1",
			FilePath:   "test.go",
			LineNumber: &line,
			Text:       "This is a test comment",
			Resolved:   false,
		},
		{
			ID:       "test-id-2",
			FilePath: "other.go",
			Text:     "Another comment without line number",
			Resolved: true,
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputCommentResultsAsToon(comments)
	if err != nil {
		t.Fatalf("outputCommentResultsAsToon failed: %v", err)
	}

	w.Close()
	os.Stdout = old

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Verify output contains expected data
	if len(output) == 0 {
		t.Fatal("Output is empty")
	}

	// Should contain header
	if !containsString(output, "id\tfile\tline\tresolved\ttext") {
		t.Error("Output missing header")
	}

	// Should contain test data
	if !containsString(output, "test-id-1") {
		t.Error("Output missing test-id-1")
	}
	if !containsString(output, "test.go") {
		t.Error("Output missing test.go")
	}
}

func TestOutputNoteResultsAsToon(t *testing.T) {
	line := 100
	notes := []mcp.NoteResult{
		{
			ID:         "note-1",
			FilePath:   "main.go",
			LineNumber: &line,
			Text:       "Test note",
			Author:     "claude",
			Type:       "explanation",
			Dismissed:  false,
		},
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputNoteResultsAsToon(notes)
	if err != nil {
		t.Fatalf("outputNoteResultsAsToon failed: %v", err)
	}

	w.Close()
	os.Stdout = old

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Verify output contains expected data
	if !containsString(output, "note-1") {
		t.Error("Output missing note-1")
	}
	if !containsString(output, "claude") {
		t.Error("Output missing author")
	}
	if !containsString(output, "explanation") {
		t.Error("Output missing type")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly ten!", 12, "exactly ten!"},
		{"this is a very long string that needs truncating", 20, "this is a very lo..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestSplitKeyValue(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"key=value", []string{"key", "value"}},
		{"name=john", []string{"name", "john"}},
		{"single", []string{"single"}},
		{"with=equals=sign", []string{"with", "equals=sign"}},
		{"empty=", []string{"empty", ""}},
	}

	for _, tt := range tests {
		result := splitKeyValue(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("splitKeyValue(%q) returned %d parts, want %d", tt.input, len(result), len(tt.expected))
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("splitKeyValue(%q)[%d] = %q, want %q", tt.input, i, result[i], tt.expected[i])
			}
		}
	}
}

// Integration tests with actual state management
func TestListCommentsIntegration(t *testing.T) {
	// Create a temporary directory for test state
	tmpDir := t.TempDir()
	testRepo := filepath.Join(tmpDir, "test-repo")
	if err := os.MkdirAll(testRepo, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a state manager with test data
	stateMgr, err := state.NewManager()
	if err != nil {
		t.Fatal(err)
	}

	// Add a test comment
	branch := "test-branch"
	commit := "abc123"
	filePath := "test.go"
	line := 10
	_, err = stateMgr.AddComment(testRepo, branch, commit, filePath, &line, "Test comment")
	if err != nil {
		t.Fatal(err)
	}

	// Test list comments
	params := mcp.ListCommentsParams{
		RepoPath: testRepo,
	}
	paramsJSON, _ := json.Marshal(params)

	result, err := mcp.ListCommentsWithManager(json.RawMessage(paramsJSON), stateMgr)
	if err != nil {
		t.Fatalf("ListComments failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	count := resultMap["count"].(int)
	if count != 1 {
		t.Errorf("Expected 1 comment, got %d", count)
	}
}

func TestAddNoteIntegration(t *testing.T) {
	// Create a temporary directory for test state
	tmpDir := t.TempDir()
	testRepo := filepath.Join(tmpDir, "test-repo")
	if err := os.MkdirAll(testRepo, 0755); err != nil {
		t.Fatal(err)
	}

	stateMgr, err := state.NewManager()
	if err != nil {
		t.Fatal(err)
	}

	// Add a test note
	params := mcp.AddNoteParams{
		RepoPath: testRepo,
		Branch:   "main",
		Commit:   "def456",
		FilePath: "test.go",
		Text:     "Test note",
		Author:   "test-author",
		Type:     "suggestion",
		Metadata: map[string]string{
			"key": "value",
		},
	}
	line := 20
	params.LineNumber = &line

	paramsJSON, _ := json.Marshal(params)
	result, err := mcp.AddNoteWithManager(json.RawMessage(paramsJSON), stateMgr)
	if err != nil {
		t.Fatalf("AddNote failed: %v", err)
	}

	resultMap := result.(map[string]interface{})
	if !resultMap["success"].(bool) {
		t.Error("Expected success=true")
	}
	if resultMap["author"] != "test-author" {
		t.Errorf("Expected author=test-author, got %v", resultMap["author"])
	}
}

// Helper function
func containsString(haystack, needle string) bool {
	return len(haystack) >= len(needle) &&
		(haystack == needle ||
			len(haystack) > len(needle) &&
				(haystack[:len(needle)] == needle ||
					haystack[len(haystack)-len(needle):] == needle ||
					containsSubstring(haystack, needle)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
