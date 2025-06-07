package prompts

import (
	"context"
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
	assert.Equal(t, "Enhance Go code with better documentation and error handling", prompt.Description)

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
	filePathArg := findArg("filePath")
	require.NotNil(t, filePathArg)
	assert.True(t, filePathArg.Required)
	assert.Equal(t, "Path to the Go file", filePathArg.Description)

	sourceCodeArg := findArg("sourceCode")
	require.NotNil(t, sourceCodeArg)
	assert.True(t, sourceCodeArg.Required)
	assert.Equal(t, "Raw Go code", sourceCodeArg.Description)

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
				"filePath":   "test.go",
				"sourceCode": "package main\n\nfunc main() {}\n",
			},
			wantErr: false,
		},
		{
			name: "missing required args",
			args: map[string]string{
				"filePath": "test.go",
			},
			wantErr:     true,
			errContains: "filePath and sourceCode are required",
		},
		{
			name: "with focus symbol",
			args: map[string]string{
				"filePath":    "test.go",
				"sourceCode":  "package main\n\nfunc main() {}\n",
				"focusSymbol": "main",
			},
			wantErr: false,
		},
		{
			name: "with minify",
			args: map[string]string{
				"filePath":   "test.go",
				"sourceCode": "package main\n\nfunc main() {}\n",
				"minify":     "true",
			},
			wantErr: false,
		},
	}

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
			assert.Contains(t, textContent.Text, "Go code enhancement assistant")

			// Verify user message with source code
			userMsg := result.Messages[1]
			assert.Equal(t, mcp.Role("user"), userMsg.Role)
			textContent, ok = userMsg.Content.(mcp.TextContent)
			require.True(t, ok)
			assert.Contains(t, textContent.Text, tt.args["sourceCode"])
		})
	}
}

func TestRegisterPrompts(t *testing.T) {
	// Initialize parser and server
	p := parser.New()
	s := server.NewMCPServer("Test Server", "1.0.0")

	// Test registration
	err := RegisterPrompts(s, p)
	require.NoError(t, err)

	// Verify prompt is registered by checking if we can get a handler for it
	handler := EnhancePromptHandler(p)
	require.NotNil(t, handler)

	// Test the handler with a basic request
	request := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Arguments: map[string]string{
				"filePath":   "test.go",
				"sourceCode": "package main\n\nfunc main() {}\n",
			},
		},
	}

	result, err := handler(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "Enhance Go code with better documentation and error handling", result.Description)
}
