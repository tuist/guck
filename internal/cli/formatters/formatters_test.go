package formatters

import (
	"os"
	"testing"

	"github.com/tuist/guck/internal/mcp"
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

	err := OutputJSON(result)
	if err != nil {
		t.Fatalf("OutputJSON failed: %v", err)
	}

	w.Close()
	os.Stdout = old

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Verify it contains expected data
	if len(output) == 0 {
		t.Fatal("Output is empty")
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

	err := OutputCommentResultsAsToon(comments)
	if err != nil {
		t.Fatalf("OutputCommentResultsAsToon failed: %v", err)
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
	if !contains(output, "id\tfile\tline\tresolved\ttext") {
		t.Error("Output missing header")
	}

	// Should contain test data
	if !contains(output, "test-id-1") {
		t.Error("Output missing test-id-1")
	}
	if !contains(output, "test.go") {
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

	err := OutputNoteResultsAsToon(notes)
	if err != nil {
		t.Fatalf("OutputNoteResultsAsToon failed: %v", err)
	}

	w.Close()
	os.Stdout = old

	// Read captured output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Verify output contains expected data
	if !contains(output, "note-1") {
		t.Error("Output missing note-1")
	}
	if !contains(output, "claude") {
		t.Error("Output missing author")
	}
	if !contains(output, "explanation") {
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

func contains(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
