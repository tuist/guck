package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// LFS pointer signature
const lfsPointerSignature = "version https://git-lfs.github.com/spec/v1"

// validRefRegex matches valid git reference patterns
var validRefRegex = regexp.MustCompile(`^[a-zA-Z0-9._\-/]+$`)

// ValidateGitRef validates that a git reference is safe to use
func ValidateGitRef(ref string) error {
	if ref == "" {
		return fmt.Errorf("ref cannot be empty")
	}
	if len(ref) > 256 {
		return fmt.Errorf("ref too long")
	}
	if !validRefRegex.MatchString(ref) {
		return fmt.Errorf("ref contains invalid characters")
	}
	// Reject ".." to prevent traversal
	if strings.Contains(ref, "..") {
		return fmt.Errorf("ref cannot contain '..'")
	}
	return nil
}

// ReadBlobCommit reads blob content from a specific commit using git show
func (r *Repo) ReadBlobCommit(ref, path string) ([]byte, error) {
	repoPath, err := r.RepoPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo path: %w", err)
	}

	// Use git show <ref>:<path> to get the blob content
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", ref, path))
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read blob at %s:%s: %w", ref, path, err)
	}

	// Check if this is an LFS pointer and smudge if needed
	if IsLFSPointer(output) {
		smudged, err := r.SmudgeLFS(output, path)
		if err != nil {
			// If smudge fails, return the original pointer content
			// This allows the UI to show something rather than failing completely
			return output, nil
		}
		return smudged, nil
	}

	return output, nil
}

// ReadBlobIndex reads blob content from the git index (staged version)
func (r *Repo) ReadBlobIndex(path string) ([]byte, error) {
	repoPath, err := r.RepoPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo path: %w", err)
	}

	// Use git show :<path> to get the index version
	cmd := exec.Command("git", "show", fmt.Sprintf(":%s", path))
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read blob from index for %s: %w", path, err)
	}

	// Check if this is an LFS pointer and smudge if needed
	if IsLFSPointer(output) {
		smudged, err := r.SmudgeLFS(output, path)
		if err != nil {
			return output, nil
		}
		return smudged, nil
	}

	return output, nil
}

// ReadBlobWorktree reads file content directly from the working directory
func (r *Repo) ReadBlobWorktree(path string) ([]byte, error) {
	repoPath, err := r.RepoPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo path: %w", err)
	}

	fullPath := filepath.Join(repoPath, path)

	// Security check: ensure the path is within the repo
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}
	rel, err := filepath.Rel(repoPath, absPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return nil, fmt.Errorf("path escapes repository: %s", path)
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read worktree file %s: %w", path, err)
	}

	// Worktree files should already be smudged by git, but check just in case
	// (this can happen if someone manually created an LFS pointer file)
	if IsLFSPointer(content) {
		smudged, err := r.SmudgeLFS(content, path)
		if err != nil {
			return content, nil
		}
		return smudged, nil
	}

	return content, nil
}

// IsLFSPointer checks if the given content is a Git LFS pointer
func IsLFSPointer(content []byte) bool {
	// LFS pointers are small text files starting with the LFS signature
	// They're typically under 200 bytes
	if len(content) > 500 {
		return false
	}

	return bytes.HasPrefix(content, []byte(lfsPointerSignature))
}

// SmudgeLFS converts an LFS pointer to its actual content by running git lfs smudge
func (r *Repo) SmudgeLFS(pointer []byte, path string) ([]byte, error) {
	repoPath, err := r.RepoPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo path: %w", err)
	}

	// Check if git-lfs is available
	if _, err := exec.LookPath("git-lfs"); err != nil {
		return nil, fmt.Errorf("git-lfs is not installed: %w", err)
	}

	// Run git lfs smudge with the pointer content on stdin
	cmd := exec.Command("git", "lfs", "smudge", path)
	cmd.Dir = repoPath
	cmd.Stdin = bytes.NewReader(pointer)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git lfs smudge failed: %s: %w", stderr.String(), err)
	}

	return stdout.Bytes(), nil
}

// GetMIMEType returns the MIME type for a file based on its extension
func GetMIMEType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".bmp":
		return "image/bmp"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}

// IsImagePath checks if the path points to an image file based on extension
func IsImagePath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg", ".ico":
		return true
	default:
		return false
	}
}
