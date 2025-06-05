package parser

import (
	"go/ast"
	"strings"
)

// ExtractDeps returns all unique dependencies of a file
func (p *FileParser) ExtractDeps(file *ast.File) []string {
	deps := make(map[string]struct{})

	// 1. Collect imports
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		deps[path] = struct{}{}
	}

	// 2. Collect function calls
	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			switch fn := call.Fun.(type) {
			case *ast.SelectorExpr: // e.g., fmt.Println
				if ident, ok := fn.X.(*ast.Ident); ok {
					deps[ident.Name] = struct{}{}
				}
			case *ast.Ident: // local calls
				deps[fn.Name] = struct{}{}
			}
		}
		return true
	})

	// Convert to slice
	result := make([]string, 0, len(deps))
	for dep := range deps {
		result = append(result, dep)
	}
	return result
}

// ExtractExportedFunctions returns all exported functions from a file
func (p *FileParser) ExtractExportedFunctions(file *ast.File) []string {
	var functions []string

	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			// Check if the function is exported (starts with uppercase)
			if funcDecl.Name.IsExported() {
				functions = append(functions, funcDecl.Name.Name)
			}
		}
	}

	return functions
}
