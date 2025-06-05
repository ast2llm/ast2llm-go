package parser

import (
	"log"
	"path/filepath"

	"github.com/vlad/ast2llm-go/internal/types"
	"golang.org/x/tools/go/packages"
)

// BuildGraph constructs a dependency graph for the project
func (p *FileParser) BuildGraph(rootPath string) (*types.DependencyGraph, error) {
	log.Printf("Building graph for %s", rootPath)

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedFiles | packages.NeedSyntax,
		Dir:  rootPath,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, err
	}

	graph := types.NewDependencyGraph()

	// First pass: create nodes for all packages
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			log.Printf("Warning: package %s has errors: %v", pkg.PkgPath, pkg.Errors)
			continue
		}

		node := &types.Node{
			PkgPath:   pkg.PkgPath,
			Files:     pkg.GoFiles,
			DependsOn: make([]string, 0),
		}

		// Add dependencies
		for impPath := range pkg.Imports {
			node.DependsOn = append(node.DependsOn, impPath)
		}

		graph.Nodes[pkg.PkgPath] = node
	}

	// Second pass: extract exported functions
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			continue
		}

		node := graph.Nodes[pkg.PkgPath]
		for _, file := range pkg.Syntax {
			functions := p.ExtractExportedFunctions(file)
			node.Functions = append(node.Functions, functions...)
		}
	}

	return graph, nil
}

// GetRelativePath returns the relative path between two packages
func (p *FileParser) GetRelativePath(from, to string) string {
	fromDir := filepath.Dir(from)
	toDir := filepath.Dir(to)
	rel, err := filepath.Rel(fromDir, toDir)
	if err != nil {
		log.Printf("Warning: could not get relative path from %s to %s: %v", from, to, err)
		return to
	}
	return rel
}
