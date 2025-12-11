// ABOUTME: Package export handles JSON export of comments and notes for agent consumption.
// ABOUTME: Exports are organized per-repository in separate directories.

package export

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Comment struct {
	ID         string `json:"id"`
	FilePath   string `json:"file_path"`
	LineNumber *int   `json:"line_number,omitempty"`
	Text       string `json:"text"`
	Timestamp  int64  `json:"timestamp"`
	Branch     string `json:"branch"`
	Commit     string `json:"commit"`
	Resolved   bool   `json:"resolved"`
	ResolvedBy string `json:"resolved_by,omitempty"`
	ResolvedAt int64  `json:"resolved_at,omitempty"`
}

type Note struct {
	ID          string            `json:"id"`
	FilePath    string            `json:"file_path"`
	LineNumber  *int              `json:"line_number,omitempty"`
	Text        string            `json:"text"`
	Timestamp   int64             `json:"timestamp"`
	Branch      string            `json:"branch"`
	Commit      string            `json:"commit"`
	Author      string            `json:"author"`
	Type        string            `json:"type"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Dismissed   bool              `json:"dismissed"`
	DismissedBy string            `json:"dismissed_by,omitempty"`
	DismissedAt int64             `json:"dismissed_at,omitempty"`
}

type ExportData struct {
	GeneratedAt string        `json:"generated_at"`
	RepoPath    string        `json:"repo_path"`
	Comments    []*Comment    `json:"comments"`
	Notes       []*Note       `json:"notes"`
	Summary     ExportSummary `json:"summary"`
}

type ExportSummary struct {
	TotalComments      int `json:"total_comments"`
	UnresolvedComments int `json:"unresolved_comments"`
	TotalNotes         int `json:"total_notes"`
	ActiveNotes        int `json:"active_notes"`
}

func Export(repoPath string, comments []*Comment, notes []*Note, outputPath string) error {
	if comments == nil {
		comments = []*Comment{}
	}
	if notes == nil {
		notes = []*Note{}
	}

	exportData := &ExportData{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		RepoPath:    repoPath,
		Comments:    comments,
		Notes:       notes,
		Summary:     calculateSummary(comments, notes),
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize export data: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	return nil
}

func GetExportPathForRepo(repoPath string) (string, error) {
	return GetExportPathForRepoWithBase(repoPath, "")
}

func GetExportPathForRepoWithBase(repoPath, customBaseDir string) (string, error) {
	var baseDir string
	var err error

	if customBaseDir != "" {
		baseDir = customBaseDir
	} else {
		baseDir, err = getExportsDir()
		if err != nil {
			return "", err
		}
	}

	repoHash := hashRepoPath(repoPath)
	return filepath.Join(baseDir, repoHash, "comments_export.json"), nil
}

func hashRepoPath(repoPath string) string {
	hash := sha256.Sum256([]byte(repoPath))
	return hex.EncodeToString(hash[:8])
}

func calculateSummary(comments []*Comment, notes []*Note) ExportSummary {
	unresolvedComments := 0
	for _, c := range comments {
		if !c.Resolved {
			unresolvedComments++
		}
	}

	activeNotes := 0
	for _, n := range notes {
		if !n.Dismissed {
			activeNotes++
		}
	}

	return ExportSummary{
		TotalComments:      len(comments),
		UnresolvedComments: unresolvedComments,
		TotalNotes:         len(notes),
		ActiveNotes:        activeNotes,
	}
}

func getExportsDir() (string, error) {
	if stateHome := os.Getenv("XDG_STATE_HOME"); stateHome != "" {
		return filepath.Join(stateHome, "guck", "exports"), nil
	}

	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "guck", "exports"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}

	return filepath.Join(home, ".local", "state", "guck", "exports"), nil
}
