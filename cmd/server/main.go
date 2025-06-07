package main

import (
	"log"

	"github.com/mark3labs/mcp-go/server"
	"github.com/vlad/ast2llm-go/internal/parser"
	"github.com/vlad/ast2llm-go/internal/prompts"
	"github.com/vlad/ast2llm-go/internal/tools"
)

func main() {
	// Initialize components
	s := server.NewMCPServer(
		"AST2LLM",
		"1.0.0",
		server.WithToolCapabilities(false),
	)
	p := parser.New()

	// Register tools
	if err := tools.RegisterTools(s, p); err != nil {
		log.Fatalf("Failed to register tools: %v", err)
	}

	// Register prompts
	if err := prompts.RegisterPrompts(s, p); err != nil {
		log.Fatalf("Failed to register prompts: %v", err)
	}

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v\n", err)
	}
}
