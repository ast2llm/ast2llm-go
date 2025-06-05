package mcp

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/klauspost/compress/zstd"
	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/vlad/ast2llm-go/internal/parser"
	"github.com/vlad/ast2llm-go/internal/types"
)

// compressJSON compresses JSON data using zstd
func compressJSON(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd encoder: %v", err)
	}
	defer encoder.Close()
	return encoder.EncodeAll(data, nil), nil
}

// GetASTDepsArgs defines arguments for get_ast_deps tool
type GetASTDepsArgs struct {
	FilePath   string `json:"filePath" jsonschema:"required,description=Path to the Go file"`
	SourceCode string `json:"sourceCode" jsonschema:"required,description=Raw Go code"`
}

// BuildDepGraphArgs defines arguments for build_dep_graph tool
type BuildDepGraphArgs struct {
	RootDir string `json:"rootDir" jsonschema:"required,description=Project root directory"`
}

// RegisterTools registers all MCP tools with the server
func RegisterTools(server *mcp_golang.Server, p *parser.FileParser) error {
	// Register get_ast_deps tool
	err := server.RegisterTool("get_ast_deps", "Extract Go file dependencies", func(args GetASTDepsArgs) (*mcp_golang.ToolResponse, error) {
		log.Printf("Processing get_ast_deps request for file: %s", args.FilePath)

		// Parse file
		file, err := p.Parse(args.FilePath, []byte(args.SourceCode))
		if err != nil {
			return nil, fmt.Errorf("parse error: %v", err)
		}

		// Extract dependencies
		deps := p.ExtractDeps(file)
		jsonDeps, err := json.Marshal(deps)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal dependencies: %v", err)
		}

		// Compress response
		compressed, err := compressJSON(jsonDeps)
		if err != nil {
			return nil, fmt.Errorf("failed to compress response: %v", err)
		}

		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(string(compressed))), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register get_ast_deps tool: %v", err)
	}

	// Register build_dep_graph tool
	err = server.RegisterTool("build_dep_graph", "Build project dependency graph", func(args BuildDepGraphArgs) (*mcp_golang.ToolResponse, error) {
		log.Printf("Processing build_dep_graph request for directory: %s", args.RootDir)

		// Build graph with timeout
		done := make(chan struct{})
		var graph *types.DependencyGraph
		var graphErr error

		go func() {
			graph, graphErr = p.BuildGraph(args.RootDir)
			close(done)
		}()

		select {
		case <-done:
			if graphErr != nil {
				return nil, fmt.Errorf("graph build failed: %v", graphErr)
			}
		case <-time.After(30 * time.Second):
			return nil, fmt.Errorf("graph build timed out after 30 seconds")
		}

		// Marshal and compress response
		jsonGraph, err := json.Marshal(graph.Nodes)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal graph: %v", err)
		}

		compressed, err := compressJSON(jsonGraph)
		if err != nil {
			return nil, fmt.Errorf("failed to compress response: %v", err)
		}

		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(string(compressed))), nil
	})
	if err != nil {
		return fmt.Errorf("failed to register build_dep_graph tool: %v", err)
	}

	return nil
}
