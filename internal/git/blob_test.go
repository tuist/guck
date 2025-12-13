package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsLFSPointer(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    bool
	}{
		{
			name: "valid LFS pointer",
			content: []byte(`version https://git-lfs.github.com/spec/v1
oid sha256:4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393
size 12345
`),
			want: true,
		},
		{
			name: "LFS pointer without trailing newline",
			content: []byte(`version https://git-lfs.github.com/spec/v1
oid sha256:4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393
size 12345`),
			want: true,
		},
		{
			name:    "regular text file",
			content: []byte("Hello, world!\n"),
			want:    false,
		},
		{
			name:    "binary content",
			content: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, // PNG header
			want:    false,
		},
		{
			name:    "empty content",
			content: []byte{},
			want:    false,
		},
		{
			name:    "content starting with version but not LFS",
			content: []byte("version 1.0.0\nsome other content"),
			want:    false,
		},
		{
			name:    "large file should not be LFS pointer",
			content: make([]byte, 600), // LFS pointers are < 500 bytes
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsLFSPointer(tt.content)
			if got != tt.want {
				t.Errorf("IsLFSPointer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMIMEType(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"image.png", "image/png"},
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"animation.gif", "image/gif"},
		{"modern.webp", "image/webp"},
		{"legacy.bmp", "image/bmp"},
		{"vector.svg", "image/svg+xml"},
		{"favicon.ico", "image/x-icon"},
		{"document.pdf", "application/octet-stream"},
		{"script.js", "application/octet-stream"},
		{"noextension", "application/octet-stream"},
		// Case insensitivity
		{"IMAGE.PNG", "image/png"},
		{"Photo.JPG", "image/jpeg"},
		{"path/to/nested/image.png", "image/png"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := GetMIMEType(tt.path)
			if got != tt.want {
				t.Errorf("GetMIMEType(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsImagePath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"image.png", true},
		{"photo.jpg", true},
		{"photo.jpeg", true},
		{"animation.gif", true},
		{"modern.webp", true},
		{"legacy.bmp", true},
		{"vector.svg", true},
		{"favicon.ico", true},
		{"document.pdf", false},
		{"script.js", false},
		{"style.css", false},
		{"noextension", false},
		{"", false},
		// Case insensitivity
		{"IMAGE.PNG", true},
		{"Photo.JPG", true},
		// Nested paths
		{"path/to/nested/image.png", true},
		{"assets/icons/logo.svg", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsImagePath(tt.path)
			if got != tt.want {
				t.Errorf("IsImagePath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestReadBlobWorktree(t *testing.T) {
	tempDir := setupTestRepo(t)

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	// Create a test file
	testContent := []byte("test file content\n")
	testPath := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testPath, testContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Read the file through ReadBlobWorktree
	content, err := repo.ReadBlobWorktree("test.txt")
	if err != nil {
		t.Fatalf("ReadBlobWorktree failed: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("ReadBlobWorktree returned %q, want %q", string(content), string(testContent))
	}
}

func TestReadBlobWorktreeNonExistent(t *testing.T) {
	tempDir := setupTestRepo(t)

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	// Try to read a non-existent file
	_, err = repo.ReadBlobWorktree("nonexistent.txt")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestReadBlobWorktreePathTraversal(t *testing.T) {
	tempDir := setupTestRepo(t)

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	// Try to escape the repository
	_, err = repo.ReadBlobWorktree("../../../etc/passwd")
	if err == nil {
		t.Error("Expected error for path traversal, got nil")
	}
}

func TestReadBlobWorktreeNestedPath(t *testing.T) {
	tempDir := setupTestRepo(t)

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	// Create a nested directory and file
	nestedDir := filepath.Join(tempDir, "subdir", "nested")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	testContent := []byte("nested content\n")
	testPath := filepath.Join(nestedDir, "file.txt")
	if err := os.WriteFile(testPath, testContent, 0644); err != nil {
		t.Fatalf("Failed to write nested file: %v", err)
	}

	// Read the nested file
	content, err := repo.ReadBlobWorktree("subdir/nested/file.txt")
	if err != nil {
		t.Fatalf("ReadBlobWorktree failed for nested file: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("ReadBlobWorktree returned %q, want %q", string(content), string(testContent))
	}
}

func TestReadBlobCommit(t *testing.T) {
	tempDir := setupTestRepo(t)

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	// The initial commit created README.md with "# Test Repo\n"
	content, err := repo.ReadBlobCommit("HEAD", "README.md")
	if err != nil {
		t.Fatalf("ReadBlobCommit failed: %v", err)
	}

	expected := "# Test Repo\n"
	if string(content) != expected {
		t.Errorf("ReadBlobCommit returned %q, want %q", string(content), expected)
	}
}

func TestReadBlobCommitNonExistent(t *testing.T) {
	tempDir := setupTestRepo(t)

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	// Try to read a non-existent file
	_, err = repo.ReadBlobCommit("HEAD", "nonexistent.txt")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestReadBlobCommitInvalidRef(t *testing.T) {
	tempDir := setupTestRepo(t)

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	// Try to read from an invalid ref
	_, err = repo.ReadBlobCommit("invalid-ref-12345", "README.md")
	if err == nil {
		t.Error("Expected error for invalid ref, got nil")
	}
}

func TestReadBlobIndex(t *testing.T) {
	tempDir := setupTestRepo(t)

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	// Create and stage a new file
	testContent := []byte("staged content\n")
	testPath := filepath.Join(tempDir, "staged.txt")
	if err := os.WriteFile(testPath, testContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	runGit(t, tempDir, "add", "staged.txt")

	// Read from index
	content, err := repo.ReadBlobIndex("staged.txt")
	if err != nil {
		t.Fatalf("ReadBlobIndex failed: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("ReadBlobIndex returned %q, want %q", string(content), string(testContent))
	}
}

func TestReadBlobIndexModifiedStaged(t *testing.T) {
	tempDir := setupTestRepo(t)

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	// Modify and stage README.md
	stagedContent := []byte("# Modified and Staged\n")
	readmePath := filepath.Join(tempDir, "README.md")
	if err := os.WriteFile(readmePath, stagedContent, 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}
	runGit(t, tempDir, "add", "README.md")

	// Now modify it again in worktree (but don't stage)
	worktreeContent := []byte("# Modified in Worktree\n")
	if err := os.WriteFile(readmePath, worktreeContent, 0644); err != nil {
		t.Fatalf("Failed to modify file again: %v", err)
	}

	// ReadBlobIndex should return the staged version
	content, err := repo.ReadBlobIndex("README.md")
	if err != nil {
		t.Fatalf("ReadBlobIndex failed: %v", err)
	}

	if string(content) != string(stagedContent) {
		t.Errorf("ReadBlobIndex returned %q, want staged content %q", string(content), string(stagedContent))
	}

	// ReadBlobWorktree should return the worktree version
	worktreeRead, err := repo.ReadBlobWorktree("README.md")
	if err != nil {
		t.Fatalf("ReadBlobWorktree failed: %v", err)
	}

	if string(worktreeRead) != string(worktreeContent) {
		t.Errorf("ReadBlobWorktree returned %q, want worktree content %q", string(worktreeRead), string(worktreeContent))
	}
}

func TestReadBlobIndexNonExistent(t *testing.T) {
	tempDir := setupTestRepo(t)

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	// Try to read a file that's not in the index
	_, err = repo.ReadBlobIndex("nonexistent.txt")
	if err == nil {
		t.Error("Expected error for non-existent file in index, got nil")
	}
}
