package prompts

import (
	"context"
	"os"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vlad/ast2llm-go/internal/parser"
)

func TestNewEnhancePrompt(t *testing.T) {
	prompt := NewEnhancePrompt()

	assert.Equal(t, "enhance", prompt.Name)
	assert.Equal(t, "Enhance Go project code with better documentation and error handling", prompt.Description)

	// Helper function to find argument by name
	findArg := func(name string) *mcp.PromptArgument {
		for _, arg := range prompt.Arguments {
			if arg.Name == name {
				return &arg
			}
		}
		return nil
	}

	// Check required arguments
	projectPathArg := findArg("projectPath")
	require.NotNil(t, projectPathArg)
	assert.True(t, projectPathArg.Required)
	assert.Equal(t, "Path to the Go project", projectPathArg.Description)

	// Check optional arguments
	focusSymbolArg := findArg("focusSymbol")
	require.NotNil(t, focusSymbolArg)
	assert.False(t, focusSymbolArg.Required)
	assert.Equal(t, "Symbol to prioritize in context", focusSymbolArg.Description)

	minifyArg := findArg("minify")
	require.NotNil(t, minifyArg)
	assert.False(t, minifyArg.Required)
	assert.Equal(t, "Remove comments and formatting", minifyArg.Description)
}

func TestEnhancePromptHandler(t *testing.T) {
	// Initialize parser and server
	p := parser.New()
	s := server.NewMCPServer("Test Server", "1.0.0")

	// Register the prompt
	err := RegisterPrompts(s, p)
	require.NoError(t, err)

	// Test cases
	tests := []struct {
		name        string
		args        map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid request",
			args: map[string]string{
				"projectPath": "./testdata/validproject",
			},
			wantErr: false,
		},
		{
			name: "missing required args",
			args: map[string]string{
				"focusSymbol": "main",
			},
			wantErr:     true,
			errContains: "projectPath is required",
		},
		{
			name: "with focus symbol",
			args: map[string]string{
				"projectPath": "./testdata/validproject",
				"focusSymbol": "MyStruct",
			},
			wantErr: false,
		},
		{
			name: "with minify",
			args: map[string]string{
				"projectPath": "./testdata/validproject",
				"minify":      "true",
			},
			wantErr: false,
		},
	}

	// Create dummy testdata directory and files
	err = os.MkdirAll("testdata/validproject", 0755)
	require.NoError(t, err)
	err = os.WriteFile("testdata/validproject/main.go", []byte("package main\n\n// MyStruct is a struct\ntype MyStruct struct{}\nfunc main(){}\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile("testdata/validproject/go.mod", []byte("module testproject\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	defer os.RemoveAll("testdata")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.GetPromptRequest{
				Params: mcp.GetPromptParams{
					Arguments: tt.args,
				},
			}

			handler := EnhancePromptHandler(p)
			result, err := handler(context.Background(), request)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			// Verify result structure
			assert.NotEmpty(t, result.Description)
			assert.NotEmpty(t, result.Messages)

			// Verify system message
			systemMsg := result.Messages[0]
			assert.Equal(t, mcp.Role("system"), systemMsg.Role)
			textContent, ok := systemMsg.Content.(mcp.TextContent)
			require.True(t, ok)
			assert.Contains(t, textContent.Text, "Go project code enhancement assistant")

			// Verify user message with project info
			userMsg := result.Messages[1]
			assert.Equal(t, mcp.Role("user"), userMsg.Role)
			textContent, ok = userMsg.Content.(mcp.TextContent)
			require.True(t, ok)
			assert.Contains(t, textContent.Text, "project structure and parsed AST information")
			assert.Contains(t, textContent.Text, "MyStruct") // Check for some expected content
			assert.Contains(t, textContent.Text, "main.go")

			if tt.name == "with focus symbol" {
				assert.Contains(t, textContent.Text, tt.args["focusSymbol"])
			}

			// Check for minify message if applicable
			if tt.args["minify"] == "true" {
				// The minify message is the last one added if minify is true
				lastMsg := result.Messages[len(result.Messages)-1]
				assert.Contains(t, lastMsg.Content.(mcp.TextContent).Text, "remove all comments and format the code to be more concise.")
			}

		})
	}
}

func TestRegisterPrompts(t *testing.T) {
	// Initialize parser and server
	p := parser.New()
	s := server.NewMCPServer("Test Server", "1.0.0")

	// Register the prompt
	err := RegisterPrompts(s, p)
	require.NoError(t, err)

	// Verify prompt is registered by checking if we can get a handler for it
	handler := EnhancePromptHandler(p)
	require.NotNil(t, handler)

	// Create dummy testdata directory and files for the handler to use
	err = os.MkdirAll("testdata/validproject", 0755)
	require.NoError(t, err)
	err = os.WriteFile("testdata/validproject/main.go", []byte("package main\n\nfunc main(){}\n"), 0644)
	require.NoError(t, err)
	err = os.WriteFile("testdata/validproject/go.mod", []byte("module testproject\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	defer os.RemoveAll("testdata")

	// Test the handler with a basic request
	request := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Arguments: map[string]string{
				"projectPath": "./testdata/validproject",
			},
		},
	}

	result, err := handler(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Enhance Go project code with better documentation and error handling", result.Description)
}
