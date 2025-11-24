package commands

import (
	"encoding/json"
	"fmt"

	"github.com/tuist/guck/internal/cli/formatters"
	"github.com/tuist/guck/internal/mcp"
	"github.com/urfave/cli/v2"
)

// ListComments handles the "guck comments list" command
func ListComments(c *cli.Context) error {
	repoPath := c.String("repo")
	branch := c.String("branch")
	commit := c.String("commit")
	filePath := c.String("file")
	format := c.String("format")

	// Build params
	params := mcp.ListCommentsParams{
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

	// Handle resolved filter
	if c.Bool("resolved") {
		resolved := true
		params.Resolved = &resolved
	} else if c.Bool("unresolved") {
		resolved := false
		params.Resolved = &resolved
	}

	// Convert to JSON and call MCP function
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}

	result, err := mcp.ListComments(json.RawMessage(paramsJSON))
	if err != nil {
		return err
	}

	return formatters.OutputResult(result, format)
}

// ResolveComment handles the "guck comments resolve" command
func ResolveComment(c *cli.Context) error {
	if c.NArg() != 1 {
		return fmt.Errorf("requires exactly 1 argument: comment-id")
	}

	commentID := c.Args().Get(0)
	repoPath := c.String("repo")
	resolvedBy := c.String("by")
	format := c.String("format")

	params := mcp.ResolveCommentParams{
		RepoPath:   repoPath,
		CommentID:  commentID,
		ResolvedBy: resolvedBy,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return err
	}

	result, err := mcp.ResolveComment(json.RawMessage(paramsJSON))
	if err != nil {
		return err
	}

	return formatters.OutputResult(result, format)
}
