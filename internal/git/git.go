package git

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type Repo struct {
	repo *git.Repository
}

type FileInfo struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch"`
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

func (r *Repo) GetDiffFiles(baseBranch string) ([]FileInfo, error) {
	// Get the base branch reference
	baseBranchRef, err := r.repo.Reference(plumbing.NewBranchReferenceName(baseBranch), true)
	if err != nil {
		return nil, fmt.Errorf("failed to find branch %s: %w", baseBranch, err)
	}

	baseCommit, err := r.repo.CommitObject(baseBranchRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get base commit: %w", err)
	}

	baseTree, err := baseCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get base tree: %w", err)
	}

	// Get the current HEAD commit
	head, err := r.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	headCommit, err := r.repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD commit: %w", err)
	}

	headTree, err := headCommit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD tree: %w", err)
	}

	// Get the diff
	changes, err := baseTree.Diff(headTree)
	if err != nil {
		return nil, fmt.Errorf("failed to create diff: %w", err)
	}

	files := []FileInfo{}

	for _, change := range changes {
		patch, err := change.Patch()
		if err != nil {
			continue
		}

		filePath := change.To.Name
		if filePath == "" {
			filePath = change.From.Name
		}

		status := "modified"
		switch {
		case change.From.Name == "":
			status = "added"
		case change.To.Name == "":
			status = "deleted"
		case change.From.Name != change.To.Name:
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
			Status:    status,
			Additions: additions,
			Deletions: deletions,
			Patch:     patchStr,
		})
	}

	return files, nil
}
