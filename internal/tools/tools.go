package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/vlad/ast2llm-go/internal/parser"
)

// ParseGoArgs defines arguments for the parse_go tool
type ParseGoArgs struct {
	FilePath string `json:"filePath" jsonschema:"required,description=Path to the Go project"`
}

// NewParseGoTool returns the mcp.Tool for parsing Go code
func NewParseGoTool() mcp.Tool {
	return mcp.NewTool("parse_go",
		mcp.WithDescription("Parse Go project and return its detailed information"),
		mcp.WithString("filePath",
			mcp.Required(),
			mcp.Description("Path to the Go project"),
		),
	)
}

// ParseGoToolHandler returns a handler for the parse_go tool
func ParseGoToolHandler(p *parser.ProjectParser) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		projectPath, err := request.RequireString("filePath")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		fileInfo, err := p.ParseProject(projectPath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse project: %v", err)), nil
		}

		fileInfoJSON, err := json.Marshal(fileInfo)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal project info: %v", err)), nil
		}

		return mcp.NewToolResultText(string(fileInfoJSON)), nil
	}
}

// RegisterTools registers all tools with the MCP server
func RegisterTools(s *server.MCPServer, p *parser.ProjectParser) error {
	s.AddTool(NewParseGoTool(), ParseGoToolHandler(p))
	return nil
}
