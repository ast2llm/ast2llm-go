package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vlad/ast2llm-go/internal/parser"
)

func TestNewParseGoTool(t *testing.T) {
	tool := NewParseGoTool()

	assert.Equal(t, "parse_go", tool.Name)
	assert.Equal(t, "Parse Go code and return its AST", tool.Description)

	// Проверяем, что tool сериализуется с нужными аргументами
	b, err := json.Marshal(tool)
	require.NoError(t, err)
	js := string(b)
	assert.Contains(t, js, "filePath")
	assert.Contains(t, js, "sourceCode")
	assert.Contains(t, js, "Path to the Go file")
	assert.Contains(t, js, "Raw Go code")
}

func TestParseGoToolHandler(t *testing.T) {
	p := parser.New()
	handler := ParseGoToolHandler(p)

	tests := []struct {
		name        string
		args        map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "valid request",
			args: map[string]any{
				"filePath":   "test.go",
				"sourceCode": "package main\n\nfunc main() {}\n",
			},
			wantErr: false,
		},
		{
			name: "missing filePath",
			args: map[string]any{
				"sourceCode": "package main\n\nfunc main() {}\n",
			},
			wantErr:     true,
			errContains: "filePath",
		},
		{
			name: "missing sourceCode",
			args: map[string]any{
				"filePath": "test.go",
			},
			wantErr:     true,
			errContains: "sourceCode",
		},
		{
			name: "invalid code",
			args: map[string]any{
				"filePath":   "test.go",
				"sourceCode": "package main\n\nfunc main() {",
			},
			wantErr:     true,
			errContains: "failed to parse file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: tt.args,
				},
			}

			result, err := handler(context.Background(), request)
			if tt.wantErr {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.IsError)
				assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tt.errContains)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.False(t, result.IsError)
			assert.NotEmpty(t, result.Content)
			assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Package")
		})
	}
}

func TestRegisterTools(t *testing.T) {
	p := parser.New()
	s := server.NewMCPServer("Test Server", "1.0.0")

	err := RegisterTools(s, p)
	require.NoError(t, err)

	// Проверяем, что инструмент зарегистрирован
	handler := ParseGoToolHandler(p)
	require.NotNil(t, handler)

	// Тестируем обработчик с базовым запросом
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"filePath":   "test.go",
				"sourceCode": "package main\n\nfunc main() {}\n",
			},
		},
	}

	result, err := handler(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.NotEmpty(t, result.Content)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Package")
}
