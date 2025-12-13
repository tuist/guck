package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

type Repo struct {
	repo *git.Repository
}

// StagingStatus indicates whether a file change is staged, unstaged, or committed
type StagingStatus string

const (
	StagingStatusCommitted StagingStatus = "committed"
	StagingStatusStaged    StagingStatus = "staged"
	StagingStatusUnstaged  StagingStatus = "unstaged"
)

type FileInfo struct {
	Path          string        `json:"path"`
	FromPath      string        `json:"from_path,omitempty"` // Original path for renames/deletes
	ToPath        string        `json:"to_path,omitempty"`   // New path for renames/adds
	Status        string        `json:"status"`
	Additions     int           `json:"additions"`
	Deletions     int           `json:"deletions"`
	Patch         string        `json:"patch"`
	StagingStatus StagingStatus `json:"staging_status,omitempty"`
}

// DiffResult contains the result of a diff operation including commit references
type DiffResult struct {
	BaseCommit string     // The merge-base commit hash used for comparison
	HeadCommit string     // The HEAD commit hash
	Files      []FileInfo // Changed files
}

func Open(path string) (*Repo, error) {
	repo, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find git repository: %w", err)
	}

	return &Repo{repo: repo}, nil
}

func (r *Repo) CurrentBranch() (string, error) {
	head, err := r.repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	if !head.Name().IsBranch() {
		return "HEAD", nil
	}

	return head.Name().Short(), nil
}

func (r *Repo) CurrentCommit() (string, error) {
	head, err := r.repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	return head.Hash().String(), nil
}

func (r *Repo) RepoPath() (string, error) {
	wt, err := r.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	absPath, err := filepath.Abs(wt.Filesystem.Root())
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return absPath, nil
}

// GetRemoteURL returns the URL of the origin remote, or empty string if not found
func (r *Repo) GetRemoteURL() (string, error) {
	remote, err := r.repo.Remote("origin")
	if err != nil {
		// No origin remote, return empty string
		return "", nil
	}

	if len(remote.Config().URLs) == 0 {
		return "", nil
	}

	return remote.Config().URLs[0], nil
}

func (r *Repo) GetDiffFiles(baseBranch string) (DiffResult, error) {
	// Try to get the remote tracking branch first (origin/baseBranch)
	// This ensures we compare against the remote version even if local is outdated
	remoteBranchRef, err := r.repo.Reference(plumbing.NewRemoteReferenceName("origin", baseBranch), true)

	var baseCommit *object.Commit
	if err == nil {
		// Remote tracking branch exists, use it
		baseCommit, err = r.repo.CommitObject(remoteBranchRef.Hash())
		if err != nil {
			return DiffResult{}, fmt.Errorf("failed to get remote base commit: %w", err)
		}
	} else {
		// Fall back to local branch if remote tracking branch doesn't exist
		baseBranchRef, err := r.repo.Reference(plumbing.NewBranchReferenceName(baseBranch), true)
		if err != nil {
			return DiffResult{}, fmt.Errorf("failed to find branch %s: %w", baseBranch, err)
		}

		baseCommit, err = r.repo.CommitObject(baseBranchRef.Hash())
		if err != nil {
			return DiffResult{}, fmt.Errorf("failed to get base commit: %w", err)
		}
	}

	// Get the current HEAD commit
	head, err := r.repo.Head()
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to get HEAD: %w", err)
	}

	headCommit, err := r.repo.CommitObject(head.Hash())
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to get HEAD commit: %w", err)
	}

	// Find the merge base between base branch and HEAD
	mergeBase, err := headCommit.MergeBase(baseCommit)
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to find merge base: %w", err)
	}

	// Use the merge base as the comparison point
	var baseTree *object.Tree
	var baseCommitHash string
	if len(mergeBase) > 0 {
		baseTree, err = mergeBase[0].Tree()
		if err != nil {
			return DiffResult{}, fmt.Errorf("failed to get merge base tree: %w", err)
		}
		baseCommitHash = mergeBase[0].Hash.String()
	} else {
		// Fallback to base branch if no merge base found
		baseTree, err = baseCommit.Tree()
		if err != nil {
			return DiffResult{}, fmt.Errorf("failed to get base tree: %w", err)
		}
		baseCommitHash = baseCommit.Hash.String()
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to get HEAD tree: %w", err)
	}

	// Get the diff
	changes, err := baseTree.Diff(headTree)
	if err != nil {
		return DiffResult{}, fmt.Errorf("failed to create diff: %w", err)
	}

	files := []FileInfo{}

	for _, change := range changes {
		patch, err := change.Patch()
		if err != nil {
			continue
		}

		// Set FromPath and ToPath for proper rename/add/delete tracking
		fromPath := change.From.Name
		toPath := change.To.Name

		// Path is the "canonical" path for display (prefer ToPath, fallback to FromPath)
		filePath := toPath
		if filePath == "" {
			filePath = fromPath
		}

		status := "modified"
		switch {
		case fromPath == "":
			status = "added"
		case toPath == "":
			status = "deleted"
		case fromPath != toPath:
			status = "renamed"
		}

		// Count additions and deletions from the patch string
		additions := 0
		deletions := 0
		patchStr := patch.String()

		lines := strings.Split(patchStr, "\n")
		for _, line := range lines {
			if len(line) == 0 {
				continue
			}
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				additions++
			} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				deletions++
			}
		}

		files = append(files, FileInfo{
			Path:      filePath,
			FromPath:  fromPath,
			ToPath:    toPath,
			Status:    status,
			Additions: additions,
			Deletions: deletions,
			Patch:     patchStr,
		})
	}

	return DiffResult{
		BaseCommit: baseCommitHash,
		HeadCommit: headCommit.Hash.String(),
		Files:      files,
	}, nil
}

// GetUncommittedChanges returns all uncommitted changes (both staged and unstaged)
// Uses native git status to properly handle LFS-tracked files
func (r *Repo) GetUncommittedChanges() ([]FileInfo, error) {
	repoPath, err := r.RepoPath()
	if err != nil {
		return nil, err
	}

	// Use native git status --porcelain to properly handle LFS files
	// go-git's Status() compares raw content (LFS pointer vs smudged content)
	// which incorrectly reports LFS files as having changes
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get git status: %w", err)
	}

	files := []FileInfo{}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}

		// Porcelain format: XY filename
		// X = index status, Y = worktree status
		indexStatus := line[0]
		worktreeStatus := line[1]
		filePath := strings.TrimSpace(line[3:])

		// Handle renamed files (format: "R  old -> new")
		if strings.Contains(filePath, " -> ") {
			parts := strings.Split(filePath, " -> ")
			if len(parts) == 2 {
				filePath = parts[1] // Use the new path
			}
		}

		// Check for staged changes (index vs HEAD)
		if indexStatus != ' ' && indexStatus != '?' {
			statusCode := porcelainToStatusCode(indexStatus)
			fileInfo, err := r.getFileInfoWithGitDiff(repoPath, filePath, statusCode, StagingStatusStaged)
			if err == nil {
				files = append(files, fileInfo)
			}
		}

		// Check for unstaged changes (worktree vs index)
		if worktreeStatus != ' ' && worktreeStatus != '?' {
			statusCode := porcelainToStatusCode(worktreeStatus)
			fileInfo, err := r.getFileInfoWithGitDiff(repoPath, filePath, statusCode, StagingStatusUnstaged)
			if err == nil {
				files = append(files, fileInfo)
			}
		}

		// Handle untracked files
		if indexStatus == '?' && worktreeStatus == '?' {
			content, err := r.readWorktreeFile(filePath)
			if err != nil {
				continue
			}
			additions := strings.Count(content, "\n")
			if len(content) > 0 && !strings.HasSuffix(content, "\n") {
				additions++
			}
			patch := fmt.Sprintf("diff --git a/%s b/%s\nnew file mode 100644\n--- /dev/null\n+++ b/%s\n@@ -0,0 +1,%d @@\n", filePath, filePath, filePath, additions)
			for _, line := range strings.Split(content, "\n") {
				if line != "" || !strings.HasSuffix(content, "\n") {
					patch += "+" + line + "\n"
				}
			}
			files = append(files, FileInfo{
				Path:          filePath,
				Status:        "added",
				Additions:     additions,
				Deletions:     0,
				Patch:         patch,
				StagingStatus: StagingStatusUnstaged,
			})
		}
	}

	return files, nil
}

// porcelainToStatusCode converts git status porcelain format to go-git StatusCode
func porcelainToStatusCode(c byte) git.StatusCode {
	switch c {
	case 'M':
		return git.Modified
	case 'A':
		return git.Added
	case 'D':
		return git.Deleted
	case 'R':
		return git.Renamed
	case 'C':
		return git.Copied
	default:
		return git.Modified
	}
}

// getFileInfoWithGitDiff uses git diff command for proper unified diff output
func (r *Repo) getFileInfoWithGitDiff(repoPath, filePath string, statusCode git.StatusCode, stagingStatus StagingStatus) (FileInfo, error) {
	status := "modified"
	switch statusCode {
	case git.Added:
		status = "added"
	case git.Deleted:
		status = "deleted"
	case git.Renamed:
		status = "renamed"
	case git.Copied:
		status = "added"
	}

	// Use git diff command for proper unified diff
	var cmd *exec.Cmd
	if stagingStatus == StagingStatusStaged {
		// Staged changes: compare index to HEAD
		cmd = exec.Command("git", "diff", "--cached", "--", filePath)
	} else {
		// Unstaged changes: compare worktree to index
		cmd = exec.Command("git", "diff", "--", filePath)
	}
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		// If git diff fails, return empty patch
		return FileInfo{
			Path:          filePath,
			Status:        status,
			Additions:     0,
			Deletions:     0,
			Patch:         "",
			StagingStatus: stagingStatus,
		}, nil
	}

	patch := string(output)

	// Count additions and deletions
	additions := 0
	deletions := 0
	lines := strings.Split(patch, "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			additions++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			deletions++
		}
	}

	return FileInfo{
		Path:          filePath,
		Status:        status,
		Additions:     additions,
		Deletions:     deletions,
		Patch:         patch,
		StagingStatus: stagingStatus,
	}, nil
}

func (r *Repo) readWorktreeFile(filePath string) (string, error) {
	wt, err := r.repo.Worktree()
	if err != nil {
		return "", err
	}

	fullPath := filepath.Join(wt.Filesystem.Root(), filePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func generateUnifiedDiff(filePath, oldContent, newContent string, status string) string {
	var patch strings.Builder

	patch.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", filePath, filePath))

	if status == "added" {
		patch.WriteString("new file mode 100644\n")
		patch.WriteString("--- /dev/null\n")
		patch.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))
	} else if status == "deleted" {
		patch.WriteString("deleted file mode 100644\n")
		patch.WriteString(fmt.Sprintf("--- a/%s\n", filePath))
		patch.WriteString("+++ /dev/null\n")
	} else {
		patch.WriteString(fmt.Sprintf("--- a/%s\n", filePath))
		patch.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))
	}

	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	// Simple diff: show all old lines as removed, all new lines as added
	// For a more accurate diff, we'd need a proper diff algorithm
	if status == "deleted" {
		if len(oldLines) > 0 {
			patch.WriteString(fmt.Sprintf("@@ -1,%d +0,0 @@\n", len(oldLines)))
			for _, line := range oldLines {
				if line != "" || oldContent != "" {
					patch.WriteString("-" + line + "\n")
				}
			}
		}
	} else if status == "added" {
		if len(newLines) > 0 {
			patch.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(newLines)))
			for _, line := range newLines {
				if line != "" || newContent != "" {
					patch.WriteString("+" + line + "\n")
				}
			}
		}
	} else {
		// For modifications, use a simple line-by-line comparison
		patch.WriteString(fmt.Sprintf("@@ -1,%d +1,%d @@\n", len(oldLines), len(newLines)))
		for _, line := range oldLines {
			patch.WriteString("-" + line + "\n")
		}
		for _, line := range newLines {
			patch.WriteString("+" + line + "\n")
		}
	}

	return patch.String()
}
