package parser

import (
	"bytes"
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
	importMap := make(map[string]string)
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		if imp.Name != nil {
			importMap[imp.Name.Name] = path
		} else {
			parts := strings.Split(path, "/")
			importMap[parts[len(parts)-1]] = path
		}
	}

	// Convert map to slice (for direct imports in FileInfo)
	for _, imp := range file.Imports {
		info.Imports = append(info.Imports, strings.Trim(imp.Path.Value, "\""))
	}

	// Extract function names
	for _, decl := range file.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			info.Functions = append(info.Functions, funcDecl.Name.Name)
		}
	}

	// Extract local struct names, comments, fields, and methods
	info.Structs = p.ExtractLocalStructInfo(file)

	// Extract used imported struct names (only name for now)
	info.UsedImportedStructs = p.ExtractUsedImportedStructInfo(file)

	return info
}

// ExtractLocalStructInfo extracts detailed information about structs declared in the file
func (p *FileParser) ExtractLocalStructInfo(file *ast.File) []*types.StructInfo {
	var structs []*types.StructInfo

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

			if structType, isStructType := typeSpec.Type.(*ast.StructType); isStructType {
				structInfo := types.NewStructInfo()
				structInfo.Name = typeSpec.Name.Name

				// Extract struct comment
				if genDecl.Doc != nil {
					structInfo.Comment = strings.TrimSpace(genDecl.Doc.Text())
				} else if typeSpec.Doc != nil {
					structInfo.Comment = strings.TrimSpace(typeSpec.Doc.Text())
				}

				// Extract fields
				structInfo.Fields = p.extractFields(structType)

				// Extract methods
				structInfo.Methods = p.extractMethods(file, structInfo.Name)

				structs = append(structs, structInfo)
			}
		}
	}
	return structs
}

// extractFields extracts fields from an ast.StructType
func (p *FileParser) extractFields(structType *ast.StructType) []*types.StructField {
	fields := make([]*types.StructField, 0) // Initialize as empty slice
	if structType.Fields == nil || len(structType.Fields.List) == 0 {
		return fields // Return empty slice
	}
	for _, field := range structType.Fields.List {
		fieldName := ""
		if len(field.Names) > 0 {
			fieldName = field.Names[0].Name // Assuming single name for simplicity
		}
		fieldType := p.exprToString(field.Type)
		fields = append(fields, &types.StructField{Name: fieldName, Type: fieldType})
	}
	return fields
}

// extractMethods extracts methods associated with a given struct name from the file
func (p *FileParser) extractMethods(file *ast.File, structName string) []*types.StructMethod {
	methods := make([]*types.StructMethod, 0) // Initialize as empty slice
	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
				recvTypeExpr := funcDecl.Recv.List[0].Type
				// Dereference pointer receiver if applicable
				if starExpr, isStar := recvTypeExpr.(*ast.StarExpr); isStar {
					recvTypeExpr = starExpr.X
				}
				if ident, isIdent := recvTypeExpr.(*ast.Ident); isIdent {
					if ident.Name == structName {
						method := &types.StructMethod{
							Name:        funcDecl.Name.Name,
							Comment:     strings.TrimSpace(funcDecl.Doc.Text()),
							Parameters:  p.extractParams(funcDecl.Type.Params),
							ReturnTypes: p.extractResults(funcDecl.Type.Results),
						}
						methods = append(methods, method)
					}
				}
			}
		}
		return true
	})
	return methods
}

// exprToString converts an ast.Expr to its string representation
func (p *FileParser) exprToString(expr ast.Expr) string {
	// Handle basic identifiers directly (e.g., string, int, error)
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	// Handle selector expressions (e.g., pkg.Type)
	if selExpr, ok := expr.(*ast.SelectorExpr); ok {
		return fmt.Sprintf("%s.%s", p.exprToString(selExpr.X), selExpr.Sel.Name)
	}
	// Handle array types (e.g., []Type)
	if arrayType, ok := expr.(*ast.ArrayType); ok {
		return fmt.Sprintf("[]%s", p.exprToString(arrayType.Elt))
	}
	// Handle map types (e.g., map[KeyType]ValueType)
	if mapType, ok := expr.(*ast.MapType); ok {
		return fmt.Sprintf("map[%s]%s", p.exprToString(mapType.Key), p.exprToString(mapType.Value))
	}
	// Handle pointer types (e.g., *Type)
	if starExpr, ok := expr.(*ast.StarExpr); ok {
		return fmt.Sprintf("*%s", p.exprToString(starExpr.X))
	}
	// Fallback for any other complex expressions using ast.Fprint
	var buf bytes.Buffer
	fset := token.NewFileSet()
	ast.Fprint(&buf, fset, expr, nil)
	return buf.String()
}

// extractParams extracts parameter types from a FieldList
func (p *FileParser) extractParams(fl *ast.FieldList) []string {
	if fl == nil || len(fl.List) == 0 {
		return []string{}
	}
	var params []string
	for _, field := range fl.List {
		params = append(params, p.exprToString(field.Type))
	}
	return params
}

// extractResults extracts return types from a FieldList
func (p *FileParser) extractResults(fl *ast.FieldList) []string {
	if fl == nil || len(fl.List) == 0 {
		return []string{}
	}
	var results []string
	for _, field := range fl.List {
		results = append(results, p.exprToString(field.Type))
	}
	return results
}

// ExtractUsedImportedStructInfo extracts names of structs imported from other packages and used in the file
func (p *FileParser) ExtractUsedImportedStructInfo(file *ast.File) []*types.StructInfo {
	usedStructs := make(map[string]*types.StructInfo)

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
					structName := fmt.Sprintf("%s.%s", ident.Name, selExpr.Sel.Name)
					if _, exists := usedStructs[structName]; !exists {
						usedStructs[structName] = &types.StructInfo{Name: structName}
					}
				}
			}
		}
		return true
	})

	result := make([]*types.StructInfo, 0, len(usedStructs))
	for _, s := range usedStructs {
		result = append(result, s)
	}
	return result
}
