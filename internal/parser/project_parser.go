package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	gotypes "go/types" // Alias go/types to avoid conflict
	"log"
	"strings"

	ourtypes "github.com/vlad/ast2llm-go/internal/types" // Alias our types
	"golang.org/x/tools/go/packages"
)

// ProjectInfo containes all usefull information about project
type ProjectInfo = map[string]*ourtypes.FileInfo

// ProjectParser handles parsing of Go projects using go/packages and go/types
type ProjectParser struct {
	fset *token.FileSet
}

// New creates a new ProjectParser instance
func New() *ProjectParser {
	return &ProjectParser{
		fset: token.NewFileSet(),
	}
}

// ParseProject loads a Go project and extracts detailed information for all Go files within it.
// It returns a map where keys are absolute file paths and values are their corresponding FileInfo.
func (p *ProjectParser) ParseProject(projectPath string) (ProjectInfo, error) {
	cfg := &packages.Config{
		Mode: packages.LoadSyntax | packages.LoadTypes | packages.LoadImports | packages.LoadFiles,
		Fset: p.fset,
		Dir:  projectPath,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("failed to load packages: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found in %s", projectPath)
	}

	fileInfos := make(ProjectInfo)

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			for _, err := range pkg.Errors {
				log.Printf("Package error in %s: %v", pkg.PkgPath, err)
			}
			// Decide whether to return an error or continue with partial results
			// For now, let's continue processing even with package errors, but log them.
		}

		for _, file := range pkg.Syntax {
			absolutePath := p.fset.File(file.Pos()).Name()
			fileInfo := p.extractFileInfoForFile(file, pkg)
			fileInfos[absolutePath] = fileInfo
		}
	}

	return fileInfos, nil
}

// extractFileInfoForFile extracts detailed information for a single AST file within a package.
func (p *ProjectParser) extractFileInfoForFile(file *ast.File, pkg *packages.Package) *ourtypes.FileInfo {
	fileInfo := ourtypes.NewFileInfo()
	fileInfo.PackageName = file.Name.Name

	// Extract imports specific to this file
	for _, imp := range file.Imports {
		fileInfo.Imports = append(fileInfo.Imports, strings.Trim(imp.Path.Value, "\""))
	}

	// Extract functions and detailed struct info from this file
	localStructsMap := make(map[string]*ourtypes.StructInfo) // To prevent duplicates for methods

	// Iterate over the AST nodes of the current file to find declarations
	ast.Inspect(file, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, isTypeSpec := spec.(*ast.TypeSpec); isTypeSpec {
					// Check if this typeSpec corresponds to a named type that is a struct
					if obj := pkg.TypesInfo.Defs[typeSpec.Name]; obj != nil {
						if namedType, ok := obj.Type().(*gotypes.Named); ok {
							if structType, ok := namedType.Underlying().(*gotypes.Struct); ok {
								// This is a struct definition within the current file
								structInfo := p.extractDetailedStructInfo(obj, namedType, structType, pkg, file)
								localStructsMap[structInfo.Name] = structInfo
							}
						}
					}
				}
			}
		} else if funcDecl, ok := n.(*ast.FuncDecl); ok {
			// Functions and methods are added to fileInfo.Functions
			fileInfo.Functions = append(fileInfo.Functions, funcDecl.Name.Name)
		}
		return true
	})

	// Convert local structs map to slice
	for _, sInfo := range localStructsMap {
		fileInfo.Structs = append(fileInfo.Structs, sInfo)
	}

	// Extract used imported structs from this file
	fileInfo.UsedImportedStructs = p.extractUsedImportedStructInfoFromFile(file, pkg)

	return fileInfo
}

// extractDetailedStructInfo extracts comprehensive details about a struct
func (p *ProjectParser) extractDetailedStructInfo(obj gotypes.Object, namedType *gotypes.Named, structType *gotypes.Struct, pkg *packages.Package, targetFile *ast.File) *ourtypes.StructInfo {
	structInfo := ourtypes.NewStructInfo()
	structInfo.Name = namedType.String() // Use the fully qualified name

	// Extract struct comment (requires traversing AST nodes directly within the target file)
	structComment := ""
	pos := obj.Pos()
	ast.Inspect(targetFile, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Pos() == pos {
					if genDecl.Doc != nil {
						structComment = strings.TrimSpace(genDecl.Doc.Text())
					} else if typeSpec.Doc != nil {
						structComment = strings.TrimSpace(typeSpec.Doc.Text())
					}
					return false // Found it, stop inspecting
				}
			}
		}
		return true
	})
	structInfo.Comment = structComment

	// Extract fields
	for i := 0; i < structType.NumFields(); i++ {
		fieldVar := structType.Field(i)
		fieldTypeName := fieldVar.Type().String() // Use types.Type.String() for canonical name
		fieldName := fieldVar.Name()
		structInfo.Fields = append(structInfo.Fields, &ourtypes.StructField{Name: fieldName, Type: fieldTypeName})
	}

	// Extract methods
	for i := 0; i < namedType.NumMethods(); i++ {
		methodObj := namedType.Method(i)
		sig := methodObj.Type().(*gotypes.Signature)

		params := []string{}
		if sig.Params() != nil {
			for j := 0; j < sig.Params().Len(); j++ {
				params = append(params, sig.Params().At(j).Type().String())
			}
		}

		results := []string{}
		if sig.Results() != nil {
			for j := 0; j < sig.Results().Len(); j++ {
				results = append(results, sig.Results().At(j).Type().String())
			}
		}

		// Method comments also require mapping back to AST if not available directly from types.Object
		methodComment := ""
		methodPos := methodObj.Pos()
		ast.Inspect(targetFile, func(n ast.Node) bool {
			if funcDecl, ok := n.(*ast.FuncDecl); ok && funcDecl.Name.Pos() == methodPos {
				if funcDecl.Doc != nil {
					methodComment = strings.TrimSpace(funcDecl.Doc.Text())
				}
				return false // Found it, stop inspecting
			}
			return true
		})

		structInfo.Methods = append(structInfo.Methods, &ourtypes.StructMethod{
			Name:        methodObj.Name(),
			Comment:     methodComment,
			Parameters:  params,
			ReturnTypes: results,
		})
	}

	return structInfo
}

// extractUsedImportedStructInfoFromFile extracts names of structs imported from other packages and used in the current file.
func (p *ProjectParser) extractUsedImportedStructInfoFromFile(file *ast.File, pkg *packages.Package) []*ourtypes.StructInfo {
	usedImportedStructs := make(map[string]*ourtypes.StructInfo)

	ast.Inspect(file, func(n ast.Node) bool {
		var typeExpr ast.Expr

		switch node := n.(type) {
		case *ast.CompositeLit:
			typeExpr = node.Type
		case *ast.DeclStmt:
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
			typeExpr = node.Type
			// Handle slice and map types for fields
			if arrayType, isArray := typeExpr.(*ast.ArrayType); isArray {
				typeExpr = arrayType.Elt
			} else if mapType, isMap := typeExpr.(*ast.MapType); isMap {
				typeExpr = mapType.Value // We only care about the value type for maps
			}
		case *ast.Ident: // Check for direct identifier usage that might refer to an imported type
			if obj := pkg.TypesInfo.Uses[node]; obj != nil {
				if namedType, ok := obj.Type().(*gotypes.Named); ok {
					if namedType.Obj().Pkg() != nil && namedType.Obj().Pkg() != pkg.Types { // Check if it's from another package
						structName := namedType.String() // Full qualified name (e.g., "context.Context")
						if _, exists := usedImportedStructs[structName]; !exists {
							usedImportedStructs[structName] = &ourtypes.StructInfo{Name: structName}
						}
					}
				}
			}
			return true
		default:
			return true
		}

		if typeExpr == nil {
			return true
		}

		// Dereference pointer if applicable
		if starExpr, ok := typeExpr.(*ast.StarExpr); ok {
			typeExpr = starExpr.X
		}

		if selExpr, ok := typeExpr.(*ast.SelectorExpr); ok {
			if obj := pkg.TypesInfo.Uses[selExpr.Sel]; obj != nil { // Check if the selector refers to a type
				if namedType, ok := obj.Type().(*gotypes.Named); ok {
					if namedType.Obj().Pkg() != nil && namedType.Obj().Pkg() != pkg.Types { // Check if it's from another package
						structName := namedType.String() // Full qualified name (e.g., "context.Context")
						if _, exists := usedImportedStructs[structName]; !exists {
							usedImportedStructs[structName] = &ourtypes.StructInfo{Name: structName}
						}
					}
				}
			}
		}
		return true
	})

	result := make([]*ourtypes.StructInfo, 0, len(usedImportedStructs))
	for _, s := range usedImportedStructs {
		result = append(result, s)
	}
	return result
}
