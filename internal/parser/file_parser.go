package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"strings"

	"github.com/vlad/ast2llm-go/internal/types"
)

// FileParser handles parsing of Go source files
type FileParser struct {
	fset *token.FileSet
}

// New creates a new FileParser instance
func New() *FileParser {
	return &FileParser{
		fset: token.NewFileSet(),
	}
}

// Parse loads a file and returns its AST
func (p *FileParser) Parse(filePath string, src []byte) (*ast.File, error) {
	file, err := parser.ParseFile(p.fset, filePath, src, parser.ParseComments)
	if err != nil {
		log.Printf("Error parsing file %s: %v", filePath, err)
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}
	return file, nil
}

// ExtractFileInfo extracts basic information from the AST
func (p *FileParser) ExtractFileInfo(file *ast.File) *types.FileInfo {
	info := types.NewFileInfo()

	// Extract package name
	info.PackageName = file.Name.Name

	// Extract imports
	importMap := make(map[string]struct{})
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		importMap[path] = struct{}{}
	}

	// Convert map to slice
	for imp := range importMap {
		info.Imports = append(info.Imports, imp)
	}

	// Extract function names
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			info.Functions = append(info.Functions, funcDecl.Name.Name)
		}
	}

	// Extract struct names and comments
	info.Structs = p.ExtractStructsWithComments(file)

	// Extract used imported struct names
	info.UsedImportedStructs = p.ExtractUsedImportedStructs(file)

	return info
}

// ExtractStructsWithComments extracts struct names and their associated comments
func (p *FileParser) ExtractStructsWithComments(file *ast.File) []string {
	var structs []string

	for _, decl := range file.Decls {
		genDecl, isGenDecl := decl.(*ast.GenDecl)
		if !isGenDecl || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, isTypeSpec := spec.(*ast.TypeSpec)
			if !isTypeSpec {
				continue
			}

			if _, isStructType := typeSpec.Type.(*ast.StructType); isStructType {
				structName := typeSpec.Name.Name
				description := ""
				if genDecl.Doc != nil {
					// Use the doc comment associated with the GenDecl (type declaration)
					description = strings.TrimSpace(genDecl.Doc.Text())
				} else if typeSpec.Doc != nil {
					// Fallback to doc comment associated with the TypeSpec if available
					description = strings.TrimSpace(typeSpec.Doc.Text())
				}

				if description != "" {
					structs = append(structs, fmt.Sprintf("%s: %s", structName, description))
				} else {
					structs = append(structs, structName)
				}
			}
		}
	}
	return structs
}

// ExtractUsedImportedStructs extracts names of structs imported from other packages and used in the file
func (p *FileParser) ExtractUsedImportedStructs(file *ast.File) []string {
	usedStructs := make(map[string]struct{})

	// Map import paths to their local names (aliases)
	importMap := make(map[string]string)
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		if imp.Name != nil {
			// If there's an alias (e.g., import myalias "path/to/pkg")
			importMap[imp.Name.Name] = path
		} else {
			// No alias, use the last component of the import path
			parts := strings.Split(path, "/")
			importMap[parts[len(parts)-1]] = path
		}
	}

	ast.Inspect(file, func(n ast.Node) bool {
		var typeExpr ast.Expr

		switch node := n.(type) {
		case *ast.CompositeLit:
			// Handles struct literal instantiation (e.g., pkg.StructName{})
			typeExpr = node.Type
		case *ast.DeclStmt:
			// Handles variable declarations with imported types (e.g., var v pkg.StructName)
			if genDecl, ok := node.Decl.(*ast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						typeExpr = valueSpec.Type

						// Handle slice and map types
						if arrayType, isArray := typeExpr.(*ast.ArrayType); isArray {
							typeExpr = arrayType.Elt
						} else if mapType, isMap := typeExpr.(*ast.MapType); isMap {
							typeExpr = mapType.Value // We only care about the value type for maps
						}

					}
				}
			}
		case *ast.Field:
			// Handles struct fields, function parameters, and return types
			typeExpr = node.Type

			// Handle slice and map types for fields
			if arrayType, isArray := typeExpr.(*ast.ArrayType); isArray {
				typeExpr = arrayType.Elt
			} else if mapType, isMap := typeExpr.(*ast.MapType); isMap {
				typeExpr = mapType.Value // We only care about the value type for maps
			}

		case *ast.CallExpr:
			// Handle type conversions like `io.Reader(nil)`
			if selExpr, ok := node.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := selExpr.X.(*ast.Ident); ok {
					// Check if the identifier is an imported package
					if _, isImported := importMap[ident.Name]; isImported {
						usedStructs[fmt.Sprintf("%s.%s", ident.Name, selExpr.Sel.Name)] = struct{}{}
					}
				}
			}
			return true // Continue traversal for arguments of the call
		default:
			return true // Continue traversal for other nodes
		}

		if typeExpr == nil {
			return true // No type expression found for this node type or it's not a relevant node
		}

		// Handle pointer types: *pkg.StructName
		if starExpr, ok := typeExpr.(*ast.StarExpr); ok {
			typeExpr = starExpr.X
		}

		// Check if the type expression is a selector expression (pkg.StructName)
		if selExpr, ok := typeExpr.(*ast.SelectorExpr); ok {
			if ident, ok := selExpr.X.(*ast.Ident); ok {
				// Check if the identifier is an imported package
				if _, isImported := importMap[ident.Name]; isImported {
					usedStructs[fmt.Sprintf("%s.%s", ident.Name, selExpr.Sel.Name)] = struct{}{}
				}
			}
		}
		return true
	})

	result := make([]string, 0, len(usedStructs))
	for s := range usedStructs {
		result = append(result, s)
	}
	return result
}
