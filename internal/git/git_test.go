package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v5"
)

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()

	// Initialize git repo
	runGit(t, tempDir, "init")
	runGit(t, tempDir, "config", "user.email", "test@test.com")
	runGit(t, tempDir, "config", "user.name", "Test User")

	// Create initial commit
	testFile := filepath.Join(tempDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test Repo\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	runGit(t, tempDir, "add", ".")
	runGit(t, tempDir, "commit", "-m", "Initial commit")

	return tempDir
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\nOutput: %s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func TestOpen(t *testing.T) {
	tempDir := setupTestRepo(t)

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	if repo == nil {
		t.Fatal("Expected non-nil repo")
	}
}

func TestOpenNonGitDirectory(t *testing.T) {
	tempDir := t.TempDir()

	_, err := Open(tempDir)
	if err == nil {
		t.Fatal("Expected error when opening non-git directory")
	}
}

func TestCurrentBranch(t *testing.T) {
	tempDir := setupTestRepo(t)

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	branch, err := repo.CurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	// Git default branch could be main or master depending on git config
	if branch != "main" && branch != "master" {
		t.Errorf("Expected branch to be 'main' or 'master', got '%s'", branch)
	}
}

func TestCurrentCommit(t *testing.T) {
	tempDir := setupTestRepo(t)

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	commit, err := repo.CurrentCommit()
	if err != nil {
		t.Fatalf("Failed to get current commit: %v", err)
	}

	if len(commit) != 40 {
		t.Errorf("Expected 40-character commit hash, got %d characters: %s", len(commit), commit)
	}
}

func TestRepoPath(t *testing.T) {
	tempDir := setupTestRepo(t)

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	path, err := repo.RepoPath()
	if err != nil {
		t.Fatalf("Failed to get repo path: %v", err)
	}

	// The path should be the absolute path to the repo
	if !filepath.IsAbs(path) {
		t.Errorf("Expected absolute path, got %s", path)
	}
}

func TestStagingStatusConstants(t *testing.T) {
	if StagingStatusCommitted != "committed" {
		t.Errorf("Expected 'committed', got '%s'", StagingStatusCommitted)
	}
	if StagingStatusStaged != "staged" {
		t.Errorf("Expected 'staged', got '%s'", StagingStatusStaged)
	}
	if StagingStatusUnstaged != "unstaged" {
		t.Errorf("Expected 'unstaged', got '%s'", StagingStatusUnstaged)
	}
}

func TestGetUncommittedChangesCleanRepo(t *testing.T) {
	tempDir := setupTestRepo(t)

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	files, err := repo.GetUncommittedChanges()
	if err != nil {
		t.Fatalf("Failed to get uncommitted changes: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 uncommitted changes in clean repo, got %d", len(files))
	}
}

func TestGetUncommittedChangesUnstagedModification(t *testing.T) {
	tempDir := setupTestRepo(t)

	// Modify the README.md file without staging
	readmePath := filepath.Join(tempDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Modified Repo\n\nNew content.\n"), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	files, err := repo.GetUncommittedChanges()
	if err != nil {
		t.Fatalf("Failed to get uncommitted changes: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 uncommitted change, got %d", len(files))
	}

	file := files[0]
	if file.Path != "README.md" {
		t.Errorf("Expected path 'README.md', got '%s'", file.Path)
	}

	if file.StagingStatus != StagingStatusUnstaged {
		t.Errorf("Expected staging status '%s', got '%s'", StagingStatusUnstaged, file.StagingStatus)
	}

	if file.Status != "modified" {
		t.Errorf("Expected status 'modified', got '%s'", file.Status)
	}
}

func TestGetUncommittedChangesStagedModification(t *testing.T) {
	tempDir := setupTestRepo(t)

	// Modify and stage the README.md file
	readmePath := filepath.Join(tempDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Modified Repo\n\nStaged content.\n"), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}
	runGit(t, tempDir, "add", "README.md")

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	files, err := repo.GetUncommittedChanges()
	if err != nil {
		t.Fatalf("Failed to get uncommitted changes: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 uncommitted change, got %d", len(files))
	}

	file := files[0]
	if file.Path != "README.md" {
		t.Errorf("Expected path 'README.md', got '%s'", file.Path)
	}

	if file.StagingStatus != StagingStatusStaged {
		t.Errorf("Expected staging status '%s', got '%s'", StagingStatusStaged, file.StagingStatus)
	}

	if file.Status != "modified" {
		t.Errorf("Expected status 'modified', got '%s'", file.Status)
	}
}

func TestGetUncommittedChangesUntrackedFile(t *testing.T) {
	tempDir := setupTestRepo(t)

	// Create a new untracked file
	newFilePath := filepath.Join(tempDir, "new_file.txt")
	if err := os.WriteFile(newFilePath, []byte("New file content\n"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	files, err := repo.GetUncommittedChanges()
	if err != nil {
		t.Fatalf("Failed to get uncommitted changes: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 uncommitted change, got %d", len(files))
	}

	file := files[0]
	if file.Path != "new_file.txt" {
		t.Errorf("Expected path 'new_file.txt', got '%s'", file.Path)
	}

	if file.StagingStatus != StagingStatusUnstaged {
		t.Errorf("Expected staging status '%s', got '%s'", StagingStatusUnstaged, file.StagingStatus)
	}

	if file.Status != "added" {
		t.Errorf("Expected status 'added', got '%s'", file.Status)
	}

	if file.Additions != 1 {
		t.Errorf("Expected 1 addition, got %d", file.Additions)
	}
}

func TestGetUncommittedChangesStagedNewFile(t *testing.T) {
	tempDir := setupTestRepo(t)

	// Create and stage a new file
	newFilePath := filepath.Join(tempDir, "staged_file.txt")
	if err := os.WriteFile(newFilePath, []byte("Staged file content\nLine 2\n"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	runGit(t, tempDir, "add", "staged_file.txt")

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	files, err := repo.GetUncommittedChanges()
	if err != nil {
		t.Fatalf("Failed to get uncommitted changes: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 uncommitted change, got %d", len(files))
	}

	file := files[0]
	if file.Path != "staged_file.txt" {
		t.Errorf("Expected path 'staged_file.txt', got '%s'", file.Path)
	}

	if file.StagingStatus != StagingStatusStaged {
		t.Errorf("Expected staging status '%s', got '%s'", StagingStatusStaged, file.StagingStatus)
	}

	if file.Status != "added" {
		t.Errorf("Expected status 'added', got '%s'", file.Status)
	}
}

func TestGetUncommittedChangesMixedStagedAndUnstaged(t *testing.T) {
	tempDir := setupTestRepo(t)

	// Create and stage a new file
	stagedFilePath := filepath.Join(tempDir, "staged.txt")
	if err := os.WriteFile(stagedFilePath, []byte("Staged content\n"), 0644); err != nil {
		t.Fatalf("Failed to create staged file: %v", err)
	}
	runGit(t, tempDir, "add", "staged.txt")

	// Create an untracked file
	untrackedFilePath := filepath.Join(tempDir, "untracked.txt")
	if err := os.WriteFile(untrackedFilePath, []byte("Untracked content\n"), 0644); err != nil {
		t.Fatalf("Failed to create untracked file: %v", err)
	}

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	files, err := repo.GetUncommittedChanges()
	if err != nil {
		t.Fatalf("Failed to get uncommitted changes: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("Expected 2 uncommitted changes, got %d", len(files))
	}

	// Count staged and unstaged files
	stagedCount := 0
	unstagedCount := 0
	for _, file := range files {
		if file.StagingStatus == StagingStatusStaged {
			stagedCount++
		} else if file.StagingStatus == StagingStatusUnstaged {
			unstagedCount++
		}
	}

	if stagedCount != 1 {
		t.Errorf("Expected 1 staged file, got %d", stagedCount)
	}

	if unstagedCount != 1 {
		t.Errorf("Expected 1 unstaged file, got %d", unstagedCount)
	}
}

func TestGetUncommittedChangesStagedDeletion(t *testing.T) {
	tempDir := setupTestRepo(t)

	// Stage deletion of README.md
	runGit(t, tempDir, "rm", "README.md")

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	files, err := repo.GetUncommittedChanges()
	if err != nil {
		t.Fatalf("Failed to get uncommitted changes: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 uncommitted change, got %d", len(files))
	}

	file := files[0]
	if file.Path != "README.md" {
		t.Errorf("Expected path 'README.md', got '%s'", file.Path)
	}

	if file.StagingStatus != StagingStatusStaged {
		t.Errorf("Expected staging status '%s', got '%s'", StagingStatusStaged, file.StagingStatus)
	}

	if file.Status != "deleted" {
		t.Errorf("Expected status 'deleted', got '%s'", file.Status)
	}
}

func TestFileInfoPatchContainsContent(t *testing.T) {
	tempDir := setupTestRepo(t)

	// Modify README.md
	readmePath := filepath.Join(tempDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Modified Title\n"), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	files, err := repo.GetUncommittedChanges()
	if err != nil {
		t.Fatalf("Failed to get uncommitted changes: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 uncommitted change, got %d", len(files))
	}

	file := files[0]

	// Check that the patch contains diff markers
	if !strings.Contains(file.Patch, "diff --git") {
		t.Error("Patch should contain 'diff --git' header")
	}

	if !strings.Contains(file.Patch, "README.md") {
		t.Error("Patch should contain the filename")
	}
}

func TestGenerateUnifiedDiff(t *testing.T) {
	tests := []struct {
		name       string
		filePath   string
		oldContent string
		newContent string
		status     string
		wantHeader string
	}{
		{
			name:       "added file",
			filePath:   "new.txt",
			oldContent: "",
			newContent: "hello\n",
			status:     "added",
			wantHeader: "new file mode",
		},
		{
			name:       "deleted file",
			filePath:   "old.txt",
			oldContent: "goodbye\n",
			newContent: "",
			status:     "deleted",
			wantHeader: "deleted file mode",
		},
		{
			name:       "modified file",
			filePath:   "mod.txt",
			oldContent: "old content\n",
			newContent: "new content\n",
			status:     "modified",
			wantHeader: "--- a/mod.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patch := generateUnifiedDiff(tt.filePath, tt.oldContent, tt.newContent, tt.status)

			if !strings.Contains(patch, tt.wantHeader) {
				t.Errorf("Patch should contain '%s', got:\n%s", tt.wantHeader, patch)
			}

			if !strings.Contains(patch, "diff --git") {
				t.Error("Patch should contain 'diff --git' header")
			}
		})
	}
}

func TestPorcelainToStatusCode(t *testing.T) {
	tests := []struct {
		input byte
		want  gogit.StatusCode
	}{
		{'M', gogit.Modified},
		{'A', gogit.Added},
		{'D', gogit.Deleted},
		{'R', gogit.Renamed},
		{'C', gogit.Copied},
		{'?', gogit.Modified}, // Unknown defaults to Modified
		{' ', gogit.Modified}, // Space defaults to Modified
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := porcelainToStatusCode(tt.input)
			if got != tt.want {
				t.Errorf("porcelainToStatusCode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDiffResultContainsCommitHashes(t *testing.T) {
	tempDir := setupTestRepo(t)

	// Get the current branch name (could be main or master)
	repo, err := Open(tempDir)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	baseBranch, err := repo.CurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	// Create a feature branch with some changes
	runGit(t, tempDir, "checkout", "-b", "feature")

	// Add a new file on the feature branch
	newFile := filepath.Join(tempDir, "feature.txt")
	if err := os.WriteFile(newFile, []byte("feature content\n"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	runGit(t, tempDir, "add", "feature.txt")
	runGit(t, tempDir, "commit", "-m", "Add feature file")

	result, err := repo.GetDiffFiles(baseBranch)
	if err != nil {
		t.Fatalf("GetDiffFiles failed: %v", err)
	}

	// Check that commit hashes are populated
	if result.BaseCommit == "" {
		t.Error("BaseCommit should not be empty")
	}

	if result.HeadCommit == "" {
		t.Error("HeadCommit should not be empty")
	}

	// Commit hashes should be 40 characters
	if len(result.BaseCommit) != 40 {
		t.Errorf("BaseCommit should be 40 characters, got %d: %s", len(result.BaseCommit), result.BaseCommit)
	}

	if len(result.HeadCommit) != 40 {
		t.Errorf("HeadCommit should be 40 characters, got %d: %s", len(result.HeadCommit), result.HeadCommit)
	}

	// Should have one file in the diff
	if len(result.Files) != 1 {
		t.Errorf("Expected 1 file in diff, got %d", len(result.Files))
	}

	if len(result.Files) > 0 && result.Files[0].Path != "feature.txt" {
		t.Errorf("Expected file path 'feature.txt', got '%s'", result.Files[0].Path)
	}
}
