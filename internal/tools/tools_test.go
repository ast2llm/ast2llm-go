package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vlad/ast2llm-go/internal/parser"
	ourtypes "github.com/vlad/ast2llm-go/internal/types" // Alias ourtypes
)

func TestNewParseGoTool(t *testing.T) {
	tool := NewParseGoTool()

	assert.Equal(t, "parse_go", tool.Name)
	assert.Equal(t, "Parse Go project and return its detailed information", tool.Description)

	// Проверяем, что tool сериализуется с нужными аргументами
	b, err := json.Marshal(tool)
	require.NoError(t, err)
	js := string(b)
	assert.Contains(t, js, "filePath")
	assert.NotContains(t, js, "sourceCode") // sourceCode is removed
	assert.Contains(t, js, "Path to the Go project")
	assert.NotContains(t, js, "Path to the Go file") // Description changed
	assert.NotContains(t, js, "Raw Go code")         // Raw Go code is removed
}

func TestParseGoToolHandler(t *testing.T) {
	p := parser.New()
	handler := ParseGoToolHandler(p)

	// Create a dummy project for testing
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "testproject")
	err := os.MkdirAll(projectPath, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(projectPath, "main.go"), []byte("package main\nfunc main(){}\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(projectPath, "go.mod"), []byte(fmt.Sprintf("module %s\ngo 1.21\n", "example.com/testproject_tools")), 0644)
	require.NoError(t, err)

	// Run go mod tidy to ensure go.mod is valid
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = projectPath
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	require.NoError(t, err, "go mod tidy failed in test setup")

	tests := []struct {
		name        string
		args        map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "valid request",
			args: map[string]any{
				"filePath": projectPath,
			},
			wantErr: false,
		},
		{
			name:        "missing filePath",
			args:        map[string]any{},
			wantErr:     true,
			errContains: "filePath",
		},
		{
			name: "invalid project path",
			args: map[string]any{
				"filePath": "/non/existent/path",
			},
			wantErr:     true,
			errContains: "no packages found",
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
				require.NoError(t, err) // Error from handler is expected to be wrapped in mcp.CallToolResultError
				assert.NotNil(t, result)
				assert.True(t, result.IsError)
				assert.Contains(t, result.Content[0].(mcp.TextContent).Text, tt.errContains)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.False(t, result.IsError)
			assert.NotEmpty(t, result.Content)

			// Verify the content is a JSON string representing FileInfo map
			jsonContent := result.Content[0].(mcp.TextContent).Text
			assert.True(t, json.Valid([]byte(jsonContent)), "Invalid JSON: %s", jsonContent)

			var parsedFileInfos map[string]*ourtypes.FileInfo
			err = json.Unmarshal([]byte(jsonContent), &parsedFileInfos)
			require.NoError(t, err, "Failed to unmarshal JSON content")

			// Basic check for the parsed content. Detailed checks are in parser_test.
			assert.Contains(t, jsonContent, "\"main.go\"")
			assert.Contains(t, jsonContent, "\"PackageName\":\"main\"")
			assert.Contains(t, jsonContent, "\"Functions\":[\"main\"]")
			assert.Contains(t, jsonContent, "\"Structs\":[]")
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

	// Create a dummy project for testing the handler
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "testproject_reg")
	err = os.MkdirAll(projectPath, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(projectPath, "main.go"), []byte("package main\nfunc init(){}\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(projectPath, "go.mod"), []byte(fmt.Sprintf("module %s\ngo 1.21\n", "example.com/testproject_reg")), 0644)
	require.NoError(t, err)

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = projectPath
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	require.NoError(t, err, "go mod tidy failed in test setup for registration")

	// Тестируем обработчик с базовым запросом
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"filePath": projectPath,
			},
		},
	}

	result, err := handler(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.NotEmpty(t, result.Content)
	jsonContent := result.Content[0].(mcp.TextContent).Text
	assert.True(t, json.Valid([]byte(jsonContent)), "Invalid JSON: %s", jsonContent)
	assert.Contains(t, jsonContent, "\"PackageName\":\"main\"")
}
