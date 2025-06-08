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
