package state

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/tuist/guck/internal/config"
	"github.com/tuist/guck/internal/export"
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
	Author      string            `json:"author"` // e.g., "claude", "copilot", "human:username"
	Type        string            `json:"type"`   // e.g., "explanation", "rationale", "suggestion"
	Metadata    map[string]string `json:"metadata,omitempty"`
	Dismissed   bool              `json:"dismissed"`
	DismissedBy string            `json:"dismissed_by,omitempty"`
	DismissedAt int64             `json:"dismissed_at,omitempty"`
}

type RepoState struct {
	ViewedFiles []string   `json:"viewed_files"`
	Comments    []*Comment `json:"comments"`
	Notes       []*Note    `json:"notes"`
}

type ViewedState struct {
	Repos map[string]map[string]map[string]*RepoState `json:"repos"`
}

type Manager struct {
	stateFile      string
	state          *ViewedState
	exportBasePath string
}

func NewManager() (*Manager, error) {
	stateDir, err := getStateDir()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	stateFile := filepath.Join(stateDir, "viewed.json")

	state := &ViewedState{
		Repos: make(map[string]map[string]map[string]*RepoState),
	}

	if _, err := os.Stat(stateFile); err == nil {
		data, err := os.ReadFile(stateFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read state file: %w", err)
		}

		if err := json.Unmarshal(data, state); err != nil {
			// If unmarshal fails, start with empty state
			state = &ViewedState{
				Repos: make(map[string]map[string]map[string]*RepoState),
			}
		}
	}

	// Load config to get custom export path
	var exportBasePath string
	cfg, err := config.Load()
	if err == nil && cfg.ExportPath != "" {
		exportBasePath = cfg.ExportPath
	}

	return &Manager{
		stateFile:      stateFile,
		state:          state,
		exportBasePath: exportBasePath,
	}, nil
}

func (m *Manager) IsFileViewed(repoPath, branch, commit, filePath string) bool {
	if branches, ok := m.state.Repos[repoPath]; ok {
		if commits, ok := branches[branch]; ok {
			if repoState, ok := commits[commit]; ok {
				for _, viewed := range repoState.ViewedFiles {
					if viewed == filePath {
						return true
					}
				}
			}
		}
	}
	return false
}

func (m *Manager) MarkFileViewed(repoPath, branch, commit, filePath string) error {
	if m.state.Repos[repoPath] == nil {
		m.state.Repos[repoPath] = make(map[string]map[string]*RepoState)
	}

	if m.state.Repos[repoPath][branch] == nil {
		m.state.Repos[repoPath][branch] = make(map[string]*RepoState)
	}

	if m.state.Repos[repoPath][branch][commit] == nil {
		m.state.Repos[repoPath][branch][commit] = &RepoState{
			ViewedFiles: []string{},
			Comments:    []*Comment{},
			Notes:       []*Note{},
		}
	}

	repoState := m.state.Repos[repoPath][branch][commit]

	// Check if already viewed
	for _, viewed := range repoState.ViewedFiles {
		if viewed == filePath {
			return m.save()
		}
	}

	repoState.ViewedFiles = append(repoState.ViewedFiles, filePath)
	return m.save()
}

func (m *Manager) UnmarkFileViewed(repoPath, branch, commit, filePath string) error {
	if branches, ok := m.state.Repos[repoPath]; ok {
		if commits, ok := branches[branch]; ok {
			if repoState, ok := commits[commit]; ok {
				filtered := []string{}
				for _, viewed := range repoState.ViewedFiles {
					if viewed != filePath {
						filtered = append(filtered, viewed)
					}
				}
				repoState.ViewedFiles = filtered
			}
		}
	}

	return m.save()
}

func (m *Manager) AddComment(repoPath, branch, commit, filePath string, lineNumber *int, text string) (*Comment, error) {
	if m.state.Repos[repoPath] == nil {
		m.state.Repos[repoPath] = make(map[string]map[string]*RepoState)
	}

	if m.state.Repos[repoPath][branch] == nil {
		m.state.Repos[repoPath][branch] = make(map[string]*RepoState)
	}

	if m.state.Repos[repoPath][branch][commit] == nil {
		m.state.Repos[repoPath][branch][commit] = &RepoState{
			ViewedFiles: []string{},
			Comments:    []*Comment{},
			Notes:       []*Note{},
		}
	}

	repoState := m.state.Repos[repoPath][branch][commit]

	timestamp := time.Now().Unix()
	comment := &Comment{
		ID:         fmt.Sprintf("%d-%d", timestamp, len(repoState.Comments)),
		FilePath:   filePath,
		LineNumber: lineNumber,
		Text:       text,
		Timestamp:  timestamp,
		Branch:     branch,
		Commit:     commit,
		Resolved:   false,
	}

	repoState.Comments = append(repoState.Comments, comment)

	if err := m.save(); err != nil {
		return nil, err
	}

	return comment, nil
}

func (m *Manager) GetComments(repoPath, branch, commit string, filePath *string) []*Comment {
	if branches, ok := m.state.Repos[repoPath]; ok {
		if commits, ok := branches[branch]; ok {
			if repoState, ok := commits[commit]; ok {
				if filePath == nil {
					return repoState.Comments
				}

				filtered := []*Comment{}
				for _, comment := range repoState.Comments {
					if comment.FilePath == *filePath {
						filtered = append(filtered, comment)
					}
				}
				return filtered
			}
		}
	}

	return []*Comment{}
}

func (m *Manager) ResolveComment(repoPath, branch, commit, commentID, resolvedBy string) error {
	if branches, ok := m.state.Repos[repoPath]; ok {
		if commits, ok := branches[branch]; ok {
			if repoState, ok := commits[commit]; ok {
				for _, comment := range repoState.Comments {
					if comment.ID == commentID {
						comment.Resolved = true
						comment.ResolvedBy = resolvedBy
						comment.ResolvedAt = time.Now().Unix()
						return m.save()
					}
				}
			}
		}
	}

	return fmt.Errorf("comment not found")
}

func (m *Manager) GetAllComments(repoPath string) []*Comment {
	var allComments []*Comment

	if branches, ok := m.state.Repos[repoPath]; ok {
		for _, commits := range branches {
			for _, repoState := range commits {
				allComments = append(allComments, repoState.Comments...)
			}
		}
	}

	return allComments
}

func (m *Manager) AddNote(repoPath, branch, commit, filePath string, lineNumber *int, text, author, noteType string, metadata map[string]string) (*Note, error) {
	if m.state.Repos[repoPath] == nil {
		m.state.Repos[repoPath] = make(map[string]map[string]*RepoState)
	}

	if m.state.Repos[repoPath][branch] == nil {
		m.state.Repos[repoPath][branch] = make(map[string]*RepoState)
	}

	if m.state.Repos[repoPath][branch][commit] == nil {
		m.state.Repos[repoPath][branch][commit] = &RepoState{
			ViewedFiles: []string{},
			Comments:    []*Comment{},
			Notes:       []*Note{},
		}
	}

	repoState := m.state.Repos[repoPath][branch][commit]

	timestamp := time.Now().Unix()
	note := &Note{
		ID:         fmt.Sprintf("%d-%d", timestamp, len(repoState.Notes)),
		FilePath:   filePath,
		LineNumber: lineNumber,
		Text:       text,
		Timestamp:  timestamp,
		Branch:     branch,
		Commit:     commit,
		Author:     author,
		Type:       noteType,
		Metadata:   metadata,
		Dismissed:  false,
	}

	repoState.Notes = append(repoState.Notes, note)

	if err := m.save(); err != nil {
		return nil, err
	}

	return note, nil
}

func (m *Manager) GetNotes(repoPath, branch, commit string, filePath *string) []*Note {
	if branches, ok := m.state.Repos[repoPath]; ok {
		if commits, ok := branches[branch]; ok {
			if repoState, ok := commits[commit]; ok {
				if filePath == nil {
					return repoState.Notes
				}

				filtered := []*Note{}
				for _, note := range repoState.Notes {
					if note.FilePath == *filePath {
						filtered = append(filtered, note)
					}
				}
				return filtered
			}
		}
	}

	return []*Note{}
}

func (m *Manager) GetAllNotes(repoPath string) []*Note {
	var allNotes []*Note

	if branches, ok := m.state.Repos[repoPath]; ok {
		for _, commits := range branches {
			for _, repoState := range commits {
				allNotes = append(allNotes, repoState.Notes...)
			}
		}
	}

	return allNotes
}

func (m *Manager) DismissNote(repoPath, branch, commit, noteID, dismissedBy string) error {
	if branches, ok := m.state.Repos[repoPath]; ok {
		if commits, ok := branches[branch]; ok {
			if repoState, ok := commits[commit]; ok {
				for _, note := range repoState.Notes {
					if note.ID == noteID {
						note.Dismissed = true
						note.DismissedBy = dismissedBy
						note.DismissedAt = time.Now().Unix()
						return m.save()
					}
				}
			}
		}
	}

	return fmt.Errorf("note not found")
}

func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize state: %w", err)
	}

	if err := os.WriteFile(m.stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	if err := m.exportToJSON(); err != nil {
		log.Printf("Warning: failed to export JSON: %v", err)
	}

	return nil
}

func (m *Manager) exportToJSON() error {
	for repoPath, branches := range m.state.Repos {
		if err := m.exportRepoToJSON(repoPath, branches); err != nil {
			log.Printf("Warning: failed to export repo %s: %v", repoPath, err)
		}
	}
	return nil
}

func (m *Manager) exportRepoToJSON(repoPath string, branches map[string]map[string]*RepoState) error {
	exportPath, err := export.GetExportPathForRepoWithBase(repoPath, m.exportBasePath)
	if err != nil {
		return err
	}

	var comments []*export.Comment
	var notes []*export.Note

	for _, commits := range branches {
		for _, repoState := range commits {
			for _, c := range repoState.Comments {
				comments = append(comments, &export.Comment{
					ID:         c.ID,
					FilePath:   c.FilePath,
					LineNumber: c.LineNumber,
					Text:       c.Text,
					Timestamp:  c.Timestamp,
					Branch:     c.Branch,
					Commit:     c.Commit,
					Resolved:   c.Resolved,
					ResolvedBy: c.ResolvedBy,
					ResolvedAt: c.ResolvedAt,
				})
			}
			for _, n := range repoState.Notes {
				notes = append(notes, &export.Note{
					ID:          n.ID,
					FilePath:    n.FilePath,
					LineNumber:  n.LineNumber,
					Text:        n.Text,
					Timestamp:   n.Timestamp,
					Branch:      n.Branch,
					Commit:      n.Commit,
					Author:      n.Author,
					Type:        n.Type,
					Metadata:    n.Metadata,
					Dismissed:   n.Dismissed,
					DismissedBy: n.DismissedBy,
					DismissedAt: n.DismissedAt,
				})
			}
		}
	}

	return export.Export(repoPath, comments, notes, exportPath)
}

func getStateDir() (string, error) {
	// Use XDG_STATE_HOME on Unix, or fallback to XDG_DATA_HOME/LocalAppData
	if stateHome := os.Getenv("XDG_STATE_HOME"); stateHome != "" {
		return filepath.Join(stateHome, "guck"), nil
	}

	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return filepath.Join(dataHome, "guck"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine home directory: %w", err)
	}

	// Platform-specific defaults
	return filepath.Join(home, ".local", "state", "guck"), nil
}
