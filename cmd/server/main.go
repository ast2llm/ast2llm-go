package main

import (
	"log"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
	"github.com/vlad/ast2llm-go/internal/mcp"
	"github.com/vlad/ast2llm-go/internal/parser"
)

func main() {
	log.Println("[MCP] Starting MCP server initialization...")
	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())
	log.Println("[MCP] Server created")
	p := parser.New()
	log.Println("[MCP] Parser created")

	// Register tools
	if err := mcp.RegisterTools(server, p); err != nil {
		log.Fatalf("[MCP] Failed to register tools: %v", err)
	}
	log.Println("[MCP] Tools registered")

	// Start server
	log.Println("[MCP] Starting MCP server...")
	if err := server.Serve(); err != nil {
		log.Fatalf("[MCP] Server failed: %v", err)
	}
}
