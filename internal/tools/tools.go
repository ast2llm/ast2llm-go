package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/vlad/ast2llm-go/internal/composer"
	"github.com/vlad/ast2llm-go/internal/parser"
)

// NewParseGoTool returns the mcp.Tool for parsing Go code
func NewParseGoTool() mcp.Tool {
	return mcp.NewTool("parse_go",
		mcp.WithDescription("Parse Go project and return its detailed information"),
		mcp.WithString("projectPath",
			mcp.Required(),
			mcp.Description("Path to the Go project"),
		),
		mcp.WithString("filePath",
			mcp.Required(),
			mcp.Description("Path to the current file"),
		),
	)
}

// ParseGoToolHandler returns a handler for the parse_go tool
func ParseGoToolHandler(p *parser.ProjectParser) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		projectPath, err := request.RequireString("projectPath")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		filePath, err := request.RequireString("filePath")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		projectInfo, err := p.ParseProject(projectPath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse project: %v", err)), nil
		}

		fullFilePath := fmt.Sprintf("%s/%s", projectPath, filePath)
		projectComposer := composer.New(projectInfo)

		info, err := projectComposer.Compose(fullFilePath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to compose project info: %v", err)), nil
		}

		return mcp.NewToolResultText(info), nil
	}
}

// RegisterTools registers all tools with the MCP server
func RegisterTools(s *server.MCPServer, p *parser.ProjectParser) error {
	s.AddTool(NewParseGoTool(), ParseGoToolHandler(p))
	return nil
}
