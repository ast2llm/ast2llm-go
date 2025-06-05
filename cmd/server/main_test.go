package main

import (
	"testing"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
	"github.com/stretchr/testify/assert"
)

// TestToolArgs defines arguments for test tool
type TestToolArgs struct {
	Test string `json:"test" jsonschema:"required,description=Test parameter"`
}

func TestToolsRegistration(t *testing.T) {
	// Mock test for tool registration
	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())
	assert.NotPanics(t, func() {
		_ = server.RegisterTool("test_tool", "Test tool", func(args TestToolArgs) (*mcp_golang.ToolResponse, error) {
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("OK")), nil
		})
	})
}

func TestServerInitialization(t *testing.T) {
	// Test server initialization
	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())
	assert.NotNil(t, server, "Server should not be nil")
}
