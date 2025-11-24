package formatters

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/tuist/guck/internal/mcp"
)

var (
	successColor = color.New(color.FgGreen, color.Bold)
	infoColor    = color.New(color.FgCyan)
	warningColor = color.New(color.FgYellow)
	urlColor     = color.New(color.FgBlue, color.Underline)
)

// OutputResult formats and outputs the result based on the specified format
func OutputResult(result interface{}, format string) error {
	switch format {
	case "json":
		return OutputJSON(result)
	case "toon":
		return OutputToon(result)
	default:
		return OutputHumanReadable(result)
	}
}

// OutputJSON outputs the result as formatted JSON
func OutputJSON(result interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// OutputToon outputs the result in Toon format (tab-separated tables)
func OutputToon(result interface{}) error {
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return fmt.Errorf("cannot convert result to toon format")
	}

	// Check if it's a list result with typed comments
	if commentsRaw, ok := resultMap["comments"]; ok {
		// Try typed slice first
		if comments, ok := commentsRaw.([]mcp.CommentResult); ok {
			return OutputCommentResultsAsToon(comments)
		}
		// Try interface slice
		if comments, ok := commentsRaw.([]interface{}); ok {
			return outputCommentsAsToon(comments)
		}
	}

	// Check if it's a list result with typed notes
	if notesRaw, ok := resultMap["notes"]; ok {
		// Try typed slice first
		if notes, ok := notesRaw.([]mcp.NoteResult); ok {
			return OutputNoteResultsAsToon(notes)
		}
		// Try interface slice
		if notes, ok := notesRaw.([]interface{}); ok {
			return outputNotesAsToon(notes)
		}
	}

	// For simple results, just output as key-value pairs
	for k, v := range resultMap {
		fmt.Printf("%s\t%v\n", k, v)
	}
	return nil
}

// OutputHumanReadable outputs the result in a human-friendly format with colors
func OutputHumanReadable(result interface{}) error {
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return OutputJSON(result)
	}

	// Check if it's a list result with comments
	if comments, ok := resultMap["comments"].([]mcp.CommentResult); ok {
		count := resultMap["count"]
		infoColor.Printf("Found %v comment(s):\n\n", count)

		for _, comment := range comments {
			if comment.Resolved {
				successColor.Print("✓ ")
			} else {
				warningColor.Print("• ")
			}

			fmt.Printf("[%s] ", comment.ID[:8])
			urlColor.Print(comment.FilePath)
			if comment.LineNumber != nil {
				fmt.Printf(":%d", *comment.LineNumber)
			}
			fmt.Println()

			fmt.Printf("  %s\n", comment.Text)

			if comment.Resolved {
				infoColor.Printf("  Resolved by %s\n", comment.ResolvedBy)
			}
			fmt.Println()
		}
		return nil
	}

	// Check if it's a list result with notes
	if notes, ok := resultMap["notes"].([]mcp.NoteResult); ok {
		count := resultMap["count"]
		infoColor.Printf("Found %v note(s):\n\n", count)

		for _, note := range notes {
			if note.Dismissed {
				color.New(color.Faint).Print("✗ ")
			} else {
				infoColor.Print("📝 ")
			}

			fmt.Printf("[%s] ", note.ID[:8])
			urlColor.Print(note.FilePath)
			if note.LineNumber != nil {
				fmt.Printf(":%d", *note.LineNumber)
			}
			fmt.Printf(" (%s)\n", note.Author)

			fmt.Printf("  Type: %s\n", note.Type)
			fmt.Printf("  %s\n", note.Text)

			if note.Dismissed {
				infoColor.Printf("  Dismissed by %s\n", note.DismissedBy)
			}
			fmt.Println()
		}
		return nil
	}

	// For simple success results
	if success, ok := resultMap["success"].(bool); ok && success {
		successColor.Println("✓ Operation completed successfully")
		for k, v := range resultMap {
			if k != "success" {
				infoColor.Printf("  %s: %v\n", k, v)
			}
		}
		return nil
	}

	// Default: output as JSON
	return OutputJSON(result)
}

// OutputCommentResultsAsToon outputs typed comments in Toon format
func OutputCommentResultsAsToon(comments []mcp.CommentResult) error {
	if len(comments) == 0 {
		fmt.Println("# No comments found")
		return nil
	}

	fmt.Println("id\tfile\tline\tresolved\ttext")
	for _, comment := range comments {
		id := comment.ID
		file := comment.FilePath
		line := ""
		if comment.LineNumber != nil {
			line = fmt.Sprintf("%d", *comment.LineNumber)
		}
		resolved := comment.Resolved
		text := truncate(comment.Text, 50)

		fmt.Printf("%s\t%s\t%s\t%v\t%s\n", id, file, line, resolved, text)
	}
	return nil
}

func outputCommentsAsToon(comments []interface{}) error {
	if len(comments) == 0 {
		fmt.Println("# No comments found")
		return nil
	}

	fmt.Println("id\tfile\tline\tresolved\ttext")
	for _, item := range comments {
		comment, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		id := comment["id"]
		file := comment["file_path"]
		line := ""
		if ln, ok := comment["line_number"]; ok && ln != nil {
			line = fmt.Sprintf("%v", ln)
		}
		resolved := comment["resolved"]
		text := truncate(fmt.Sprintf("%v", comment["text"]), 50)

		fmt.Printf("%s\t%s\t%s\t%v\t%s\n", id, file, line, resolved, text)
	}
	return nil
}

// OutputNoteResultsAsToon outputs typed notes in Toon format
func OutputNoteResultsAsToon(notes []mcp.NoteResult) error {
	if len(notes) == 0 {
		fmt.Println("# No notes found")
		return nil
	}

	fmt.Println("id\tfile\tline\tauthor\ttype\tdismissed\ttext")
	for _, note := range notes {
		id := note.ID
		file := note.FilePath
		line := ""
		if note.LineNumber != nil {
			line = fmt.Sprintf("%d", *note.LineNumber)
		}
		author := note.Author
		noteType := note.Type
		dismissed := note.Dismissed
		text := truncate(note.Text, 50)

		fmt.Printf("%s\t%s\t%s\t%s\t%s\t%v\t%s\n", id, file, line, author, noteType, dismissed, text)
	}
	return nil
}

func outputNotesAsToon(notes []interface{}) error {
	if len(notes) == 0 {
		fmt.Println("# No notes found")
		return nil
	}

	fmt.Println("id\tfile\tline\tauthor\ttype\tdismissed\ttext")
	for _, item := range notes {
		note, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		id := note["id"]
		file := note["file_path"]
		line := ""
		if ln, ok := note["line_number"]; ok && ln != nil {
			line = fmt.Sprintf("%v", ln)
		}
		author := note["author"]
		noteType := note["type"]
		dismissed := note["dismissed"]
		text := truncate(fmt.Sprintf("%v", note["text"]), 50)

		fmt.Printf("%s\t%s\t%s\t%s\t%s\t%v\t%s\n", id, file, line, author, noteType, dismissed, text)
	}
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
