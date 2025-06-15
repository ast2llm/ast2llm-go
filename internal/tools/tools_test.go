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
	// Alias ourtypes
)

func TestNewParseGoTool(t *testing.T) {
	tool := NewParseGoTool()

	assert.Equal(t, "parse_go", tool.Name)
	assert.Equal(t, "Parse Go project and return its detailed information", tool.Description)

	// Проверяем, что tool сериализуется с нужными аргументами
	b, err := json.Marshal(tool)
	require.NoError(t, err)
	js := string(b)
	assert.Contains(t, js, "projectPath")
	assert.Contains(t, js, "filePath")
	assert.Contains(t, js, "Path to the Go project")
	assert.Contains(t, js, "Path to the current file")
	assert.NotContains(t, js, "Raw Go code")
}

func TestParseGoToolHandler(t *testing.T) {
	p := parser.New()
	handler := ParseGoToolHandler(p)

	// Create a dummy project for testing
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "testproject")
	err := os.MkdirAll(projectPath, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(projectPath, "main.go"), []byte(`package main

import "fmt"

// MyStruct is a simple struct
type MyStruct struct{}

func main(){
	fmt.Println("Hello")
	_ = MyStruct{}
}
`), 0644)
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
				"projectPath": projectPath,
				"filePath":    "main.go",
			},
			wantErr: false,
		},
		{
			name:        "missing filePath",
			args:        map[string]any{},
			wantErr:     true,
			errContains: "projectPath",
		},
		{
			name: "invalid project path",
			args: map[string]any{
				"projectPath": "/non/existent/path",
				"filePath":    "main.go",
			},
			wantErr:     true,
			errContains: "failed to parse project",
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
			composedOutput := result.Content[0].(mcp.TextContent).Text
			assert.Contains(t, composedOutput, "--- File: "+filepath.Join(projectPath, "main.go")+" ---")
			assert.Contains(t, composedOutput, "Package: main")
			assert.Contains(t, composedOutput, "Functions:\n- main")
			assert.Contains(t, composedOutput, "Local Structs:\n  Struct: example.com/testproject_tools.MyStruct")
			assert.NotContains(t, composedOutput, "Used Imported Structs (from this project, if available):\n- fmt")
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
				"projectPath": projectPath,
				"filePath":    "main.go",
			},
		},
	}

	result, err := handler(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	assert.NotEmpty(t, result.Content)
	composedOutput := result.Content[0].(mcp.TextContent).Text
	assert.Contains(t, composedOutput, "Package: main")
	assert.Contains(t, composedOutput, "Functions:\n- init")
	assert.NotContains(t, composedOutput, "Local Structs:\n  Struct:")
	assert.NotContains(t, composedOutput, "Used Imported Structs (from this project, if available):\n")
}
