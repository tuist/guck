package commands

import (
	"encoding/json"
	"fmt"

	"github.com/tuist/guck/internal/cli/formatters"
	"github.com/tuist/guck/internal/cli/helpers"
	"github.com/tuist/guck/internal/git"
	"github.com/tuist/guck/internal/mcp"
	"github.com/urfave/cli/v2"
)

// AddNote handles the "guck notes add" command
func AddNote(c *cli.Context) error {
	repoPath := c.String("repo")
	filePath := c.String("file")
	text := c.String("text")
	author := c.String("author")
	noteType := c.String("type")
	format := c.String("format")

	// Get current branch and commit
	gitRepo, err := git.Open(repoPath)
	if err != nil {
		return err
	}

	branch, err := gitRepo.CurrentBranch()
	if err != nil {
		return err
	}

	commit, err := gitRepo.CurrentCommit()
	if err != nil {
		return err
	}

	params := mcp.AddNoteParams{
		RepoPath: repoPath,
		Branch:   branch,
		Commit:   commit,
		FilePath: filePath,
		Text:     text,
		Author:   author,
		Type:     noteType,
	}

	// Handle line number
	if c.IsSet("line") {
		line := c.Int("line")
		params.LineNumber = &line
	}

	// Handle metadata
	if c.IsSet("metadata") {
		metadata := make(map[string]string)
		for _, pair := range c.StringSlice("metadata") {
			parts := helpers.SplitKeyValue(pair)
			if len(parts) == 2 {
				metadata[parts[0]] = parts[1]
			}
		}
		if len(metadata) > 0 {
			params.Metadata = metadata
		}
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}

	result, err := mcp.AddNote(json.RawMessage(paramsJSON))
	if err != nil {
		return err
	}

	return formatters.OutputResult(result, format)
}

// ListNotes handles the "guck notes list" command
func ListNotes(c *cli.Context) error {
	repoPath := c.String("repo")
	branch := c.String("branch")
	commit := c.String("commit")
	filePath := c.String("file")
	author := c.String("author")
	format := c.String("format")

	params := mcp.ListNotesParams{
		RepoPath: repoPath,
	}

	if branch != "" {
		params.Branch = &branch
	}
	if commit != "" {
		params.Commit = &commit
	}
	if filePath != "" {
		params.FilePath = &filePath
	}
	if author != "" {
		params.Author = &author
	}

	// Handle dismissed filter
	if c.Bool("dismissed") {
		dismissed := true
		params.Dismissed = &dismissed
	} else if c.Bool("active") {
		dismissed := false
		params.Dismissed = &dismissed
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}

	result, err := mcp.ListNotes(json.RawMessage(paramsJSON))
	if err != nil {
		return err
	}

	return formatters.OutputResult(result, format)
}

// DismissNote handles the "guck notes dismiss" command
func DismissNote(c *cli.Context) error {
	if c.NArg() != 1 {
		return fmt.Errorf("requires exactly 1 argument: note-id")
	}

	noteID := c.Args().Get(0)
	repoPath := c.String("repo")
	dismissedBy := c.String("by")
	format := c.String("format")

	params := mcp.DismissNoteParams{
		RepoPath:    repoPath,
		NoteID:      noteID,
		DismissedBy: dismissedBy,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}

	result, err := mcp.DismissNote(json.RawMessage(paramsJSON))
	if err != nil {
		return err
	}

	return formatters.OutputResult(result, format)
}
