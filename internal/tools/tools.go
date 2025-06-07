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
	FilePath   string `json:"filePath" jsonschema:"required,description=Path to the Go file"`
	SourceCode string `json:"sourceCode" jsonschema:"required,description=Raw Go code"`
}

// NewParseGoTool returns the mcp.Tool for parsing Go code
func NewParseGoTool() mcp.Tool {
	return mcp.NewTool("parse_go",
		mcp.WithDescription("Parse Go code and return its AST"),
		mcp.WithString("filePath",
			mcp.Required(),
			mcp.Description("Path to the Go file"),
		),
		mcp.WithString("sourceCode",
			mcp.Required(),
			mcp.Description("Raw Go code"),
		),
	)
}

// ParseGoToolHandler returns a handler for the parse_go tool
func ParseGoToolHandler(p *parser.FileParser) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filePath, err := request.RequireString("filePath")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		sourceCode, err := request.RequireString("sourceCode")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		file, err := p.Parse(filePath, []byte(sourceCode))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse file: %v", err)), nil
		}

		fileInfo := p.ExtractFileInfo(file)
		astJSON, err := json.Marshal(fileInfo)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal AST: %v", err)), nil
		}

		return mcp.NewToolResultText(string(astJSON)), nil
	}
}

// RegisterTools registers all tools with the MCP server
func RegisterTools(s *server.MCPServer, p *parser.FileParser) error {
	s.AddTool(NewParseGoTool(), ParseGoToolHandler(p))
	return nil
}
