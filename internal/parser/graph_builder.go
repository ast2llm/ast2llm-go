package parser

import (
	"go/ast"
	"log"

	"github.com/vlad/ast2llm-go/internal/types"
	"golang.org/x/tools/go/packages"
)

// BuildGraph constructs a dependency graph for the project
func (p *ProjectParser) BuildGraph(rootPath string) (*types.DependencyGraph, error) {
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
			for _, err := range pkg.Errors {
				log.Printf("Warning: package %s has errors: %v", pkg.PkgPath, err)
			}
			continue
		}

		node := &types.Node{
			PkgPath:   pkg.PkgPath,
			Files:     pkg.GoFiles,
			DependsOn: make([]string, 0),
			Functions: make([]string, 0), // Initialize Functions slice
		}

		// Add dependencies
		for impPath := range pkg.Imports {
			node.DependsOn = append(node.DependsOn, impPath)
		}

		// Extract exported functions for this package
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				if funcDecl, ok := n.(*ast.FuncDecl); ok {
					// Check if the function is exported (starts with uppercase)
					if funcDecl.Name.IsExported() {
						node.Functions = append(node.Functions, funcDecl.Name.Name)
					}
				}
				return true
			})
		}
		graph.Nodes[pkg.PkgPath] = node
	}

	return graph, nil
}
