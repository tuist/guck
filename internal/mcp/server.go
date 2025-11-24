package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

// JSON-RPC 2.0 message types
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id,omitempty"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP Protocol types
type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
	Capabilities    Capabilities `json:"capabilities"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Capabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct {
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type CallToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// StartStdioServer starts the MCP server using stdio transport
func StartStdioServer() error {
	// Configure logging to stderr (stdout is reserved for JSON-RPC)
	log.SetOutput(os.Stderr)
	log.SetPrefix("[guck-mcp] ")

	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	log.Println("MCP server started")

	for {
		var request JSONRPCRequest
		if err := decoder.Decode(&request); err != nil {
			if err == io.EOF {
				log.Println("Client disconnected")
				return nil
			}
			log.Printf("Error decoding request: %v", err)
			continue
		}

		log.Printf("Received request: %s (id=%v)", request.Method, request.ID)

		var response *JSONRPCResponse

		switch request.Method {
		case "initialize":
			response = handleInitialize(request)

		case "notifications/initialized", "initialized":
			log.Println("Server initialized successfully")
			continue // Notifications don't need responses

		case "tools/list":
			response = handleToolsList(request)

		case "tools/call":
			response = handleToolsCall(request)

		default:
			response = &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      request.ID,
				Error: &JSONRPCError{
					Code:    -32601,
					Message: fmt.Sprintf("Method not found: %s", request.Method),
				},
			}
		}

		if response != nil {
			if err := encoder.Encode(response); err != nil {
				log.Printf("Error encoding response: %v", err)
			}
		}
	}
}

func handleInitialize(request JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: InitializeResult{
			ProtocolVersion: "2024-11-05",
			ServerInfo: ServerInfo{
				Name:    "guck",
				Version: "0.5.0",
			},
			Capabilities: Capabilities{
				Tools: &ToolsCapability{},
			},
		},
	}
}

func handleToolsList(request JSONRPCRequest) *JSONRPCResponse {
	tools := []Tool{
		{
			Name:        "list_comments",
			Description: "List all code review comments for a repository with optional filtering by branch, commit, file path, or resolution status",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the Git repository (defaults to current working directory)",
					},
					"branch": map[string]interface{}{
						"type":        "string",
						"description": "Filter comments by branch name",
					},
					"commit": map[string]interface{}{
						"type":        "string",
						"description": "Filter comments by commit hash",
					},
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Filter comments by file path",
					},
					"resolved": map[string]interface{}{
						"type":        "boolean",
						"description": "Filter comments by resolution status (true for resolved, false for unresolved)",
					},
				},
			},
		},
		{
			Name:        "resolve_comment",
			Description: "Mark a code review comment as resolved, recording who resolved it and when",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"comment_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the comment to resolve",
					},
					"resolved_by": map[string]interface{}{
						"type":        "string",
						"description": "Identifier of who/what is resolving the comment (e.g., 'claude', 'copilot', user name)",
					},
					"repo_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the Git repository (defaults to current working directory)",
					},
				},
				"required": []string{"comment_id", "resolved_by"},
			},
		},
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: ListToolsResult{
			Tools: tools,
		},
	}
}

func handleToolsCall(request JSONRPCRequest) *JSONRPCResponse {
	params, ok := request.Params.(map[string]interface{})
	if !ok {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Invalid params",
			},
		}
	}

	toolName, ok := params["name"].(string)
	if !ok {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Missing tool name",
			},
		}
	}

	arguments, ok := params["arguments"].(map[string]interface{})
	if !ok {
		arguments = make(map[string]interface{})
	}

	// Convert arguments to JSON for existing functions
	argsJSON, err := json.Marshal(arguments)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: fmt.Sprintf("Failed to marshal arguments: %v", err),
			},
		}
	}

	var result interface{}
	var toolErr error

	switch toolName {
	case "list_comments":
		result, toolErr = ListComments(json.RawMessage(argsJSON))

	case "resolve_comment":
		result, toolErr = ResolveComment(json.RawMessage(argsJSON))

	case "add_note":
		result, toolErr = AddNote(json.RawMessage(argsJSON))

	case "list_notes":
		result, toolErr = ListNotes(json.RawMessage(argsJSON))

	case "dismiss_note":
		result, toolErr = DismissNote(json.RawMessage(argsJSON))

	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Unknown tool: %s", toolName),
			},
		}
	}

	if toolErr != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result: CallToolResult{
				Content: []ToolContent{
					{
						Type: "text",
						Text: fmt.Sprintf("Error: %v", toolErr),
					},
				},
				IsError: true,
			},
		}
	}

	// Convert result to JSON text
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &JSONRPCError{
				Code:    -32603,
				Message: fmt.Sprintf("Failed to marshal result: %v", err),
			},
		}
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: CallToolResult{
			Content: []ToolContent{
				{
					Type: "text",
					Text: string(resultJSON),
				},
			},
		},
	}
}
