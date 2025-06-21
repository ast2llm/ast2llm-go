package parser

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
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
			fileInfo := p.extractFileInfoForFile(file, pkg, pkgs)
			fileInfos[absolutePath] = fileInfo
		}
	}

	return fileInfos, nil
}

// extractFileInfoForFile extracts detailed information for a single AST file within a package.
func (p *ProjectParser) extractFileInfoForFile(file *ast.File, pkg *packages.Package, projectPkgs []*packages.Package) *ourtypes.FileInfo {
	fileInfo := ourtypes.NewFileInfo()
	fileInfo.PackageName = file.Name.Name

	// Extract imports specific to this file
	for _, imp := range file.Imports {
		fileInfo.Imports = append(fileInfo.Imports, strings.Trim(imp.Path.Value, "\""))
	}

	// Extract functions and detailed struct info from this file
	localStructsMap := make(map[string]*ourtypes.StructInfo)       // To prevent duplicates for methods
	localInterfacesMap := make(map[string]*ourtypes.InterfaceInfo) // To prevent duplicates for interfaces

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
							} else if ifaceType, ok := namedType.Underlying().(*gotypes.Interface); ok {
								// This is an interface definition within the current file
								ifaceInfo := p.extractDetailedInterfaceInfo(obj, namedType, ifaceType, pkg, file)
								localInterfacesMap[ifaceInfo.Name] = ifaceInfo
							}
						}
					}
				} else if valSpec, isValueSpec := spec.(*ast.ValueSpec); isValueSpec {
					// This is a var or const declaration
					for i, name := range valSpec.Names {
						if obj := pkg.TypesInfo.Defs[name]; obj != nil {
							varInfo := p.extractGlobalVarInfo(obj, genDecl, valSpec, i, pkg)
							fileInfo.GlobalVars = append(fileInfo.GlobalVars, varInfo)
						}
					}
				}
			}
		} else if funcDecl, ok := n.(*ast.FuncDecl); ok {
			// Only top-level (non-method) functions
			if funcDecl.Recv == nil {
				fnInfo := p.extractFunctionInfo(funcDecl, pkg)
				fileInfo.Functions = append(fileInfo.Functions, fnInfo)
			}
		}
		return true
	})

	// Convert local structs map to slice
	for _, sInfo := range localStructsMap {
		fileInfo.Structs = append(fileInfo.Structs, sInfo)
	}

	// Convert local interfaces map to slice
	for _, iInfo := range localInterfacesMap {
		fileInfo.Interfaces = append(fileInfo.Interfaces, iInfo)
	}

	// Extract used imported structs from this file
	fileInfo.UsedImportedStructs = p.extractUsedImportedStructInfoFromFile(file, pkg)

	// Collect used imported functions (by fully qualified name)
	fileInfo.UsedImportedFunctions = p.extractUsedImportedFunctions(file, pkg, projectPkgs)

	// Collect used imported global vars (by fully qualified name)
	fileInfo.UsedImportedGlobalVars = p.extractUsedImportedGlobalVars(file, pkg, projectPkgs)

	return fileInfo
}

// extractUsedImportedGlobalVars extracts detailed information about imported global variables used in the file.
func (p *ProjectParser) extractUsedImportedGlobalVars(file *ast.File, pkg *packages.Package, projectPkgs []*packages.Package) []*ourtypes.GlobalVarInfo {
	usedVars := make(map[string]*ourtypes.GlobalVarInfo)

	ast.Inspect(file, func(n ast.Node) bool {
		if selExpr, ok := n.(*ast.SelectorExpr); ok {
			if obj := pkg.TypesInfo.Uses[selExpr.Sel]; obj != nil {
				// We are looking for either a variable or a constant.
				_, isVar := obj.(*gotypes.Var)
				_, isConst := obj.(*gotypes.Const)

				if isVar || isConst {
					if obj.Pkg() != nil && obj.Pkg() != pkg.Types { // Check if it's from another package
						varName := obj.Pkg().Path() + "." + obj.Name()
						if _, exists := usedVars[varName]; !exists {
							var foundVar *ourtypes.GlobalVarInfo
							// Search for original declaration in project packages
							for _, pPkg := range projectPkgs {
								if pPkg.PkgPath != obj.Pkg().Path() {
									continue
								}

								for _, fAst := range pPkg.Syntax {
									ast.Inspect(fAst, func(node ast.Node) bool {
										if genDecl, ok := node.(*ast.GenDecl); ok {
											for _, spec := range genDecl.Specs {
												if valSpec, ok := spec.(*ast.ValueSpec); ok {
													for i, name := range valSpec.Names {
														if name.Name == obj.Name() {
															if defObj := pPkg.TypesInfo.Defs[name]; defObj != nil {
																foundVar = p.extractGlobalVarInfo(defObj, genDecl, valSpec, i, pPkg)
																// We need to set the fully qualified name
																foundVar.Name = varName
																return false // stop inner inspect
															}
														}
													}
												}
											}
										}
										return foundVar == nil // continue if not found
									})
									if foundVar != nil {
										break
									}
								}
								if foundVar != nil {
									break
								}
							}

							if foundVar != nil {
								usedVars[varName] = foundVar
							} else {
								// Fallback for stdlib or not found
								if cnst, isConst := obj.(*gotypes.Const); isConst {
									usedVars[varName] = &ourtypes.GlobalVarInfo{
										Name:    varName,
										Type:    cnst.Type().String(),
										Value:   cnst.Val().String(),
										IsConst: true,
									}
								} else {
									usedVars[varName] = &ourtypes.GlobalVarInfo{
										Name:    varName,
										Type:    obj.Type().String(),
										IsConst: false,
									}
								}
							}
						}
					}
				}
			}
		}
		return true
	})

	result := make([]*ourtypes.GlobalVarInfo, 0, len(usedVars))
	for _, v := range usedVars {
		result = append(result, v)
	}
	return result
}

// extractUsedImportedFunctions extracts detailed information about imported functions used in the file.
func (p *ProjectParser) extractUsedImportedFunctions(file *ast.File, pkg *packages.Package, projectPkgs []*packages.Package) []*ourtypes.FunctionInfo {
	var usedImportedFunctions []*ourtypes.FunctionInfo
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := call.Fun.(type) {
		case *ast.SelectorExpr:
			if _, ok := fun.X.(*ast.Ident); ok {
				obj := pkg.TypesInfo.Uses[fun.Sel]
				if obj != nil {
					if fn, ok := obj.(*gotypes.Func); ok {
						// Only functions from other packages
						if fn.Pkg() != nil && fn.Pkg().Path() != pkg.PkgPath {
							fqName := fn.String()
							found := false
							for _, pkg2 := range projectPkgs {
								for _, fileAst := range pkg2.Syntax {
									ast.Inspect(fileAst, func(n ast.Node) bool {
										funcDecl, ok := n.(*ast.FuncDecl)
										if !ok || funcDecl.Recv != nil {
											return true
										}
										obj2 := pkg2.TypesInfo.Defs[funcDecl.Name]
										if obj2 == nil {
											return true
										}
										if fn2, ok := obj2.(*gotypes.Func); ok && fn2.String() == fqName {
											params := []string{}
											if funcDecl.Type.Params != nil {
												for _, field := range funcDecl.Type.Params.List {
													typeStr := pkg2.TypesInfo.TypeOf(field.Type).String()
													for _, name := range field.Names {
														params = append(params, name.Name+" "+typeStr)
													}
													if len(field.Names) == 0 {
														params = append(params, typeStr)
													}
												}
											}
											returns := []string{}
											if funcDecl.Type.Results != nil {
												for _, field := range funcDecl.Type.Results.List {
													typeStr := pkg2.TypesInfo.TypeOf(field.Type).String()
													for range field.Names {
														returns = append(returns, typeStr)
													}
													if len(field.Names) == 0 {
														returns = append(returns, typeStr)
													}
												}
											}
											comment := ""
											if funcDecl.Doc != nil {
												comment = strings.TrimSpace(funcDecl.Doc.Text())
											}
											usedImportedFunctions = append(usedImportedFunctions, &ourtypes.FunctionInfo{
												Name:    fn2.Pkg().Path() + "." + fn2.Name(),
												Comment: comment,
												Params:  params,
												Returns: returns,
											})
											found = true
											return false
										}
										return true
									})
									if found {
										break
									}
								}
								if found {
									break
								}
							}
						}
					}
				}
			}
		}
		return true
	})
	return usedImportedFunctions
}

// extractFunctionInfo extracts detailed information about a function.
func (p *ProjectParser) extractFunctionInfo(funcDecl *ast.FuncDecl, pkg *packages.Package) *ourtypes.FunctionInfo {
	fnInfo := ourtypes.NewFunctionInfo()
	fnInfo.Name = funcDecl.Name.Name
	fnInfo.Comment = ""
	// Extract comment
	if funcDecl.Doc != nil {
		fnInfo.Comment = strings.TrimSpace(funcDecl.Doc.Text())
	}
	// Extract parameters
	if funcDecl.Type.Params != nil {
		for _, field := range funcDecl.Type.Params.List {
			typeStr := pkg.TypesInfo.TypeOf(field.Type).String()
			for _, name := range field.Names {
				fnInfo.Params = append(fnInfo.Params, name.Name+" "+typeStr)
			}
			// Anonymous parameter (e.g. func(int))
			if len(field.Names) == 0 {
				fnInfo.Params = append(fnInfo.Params, typeStr)
			}
		}
	}
	// Extract return types
	if funcDecl.Type.Results != nil {
		for _, field := range funcDecl.Type.Results.List {
			typeStr := pkg.TypesInfo.TypeOf(field.Type).String()
			// Named return values
			for range field.Names {
				fnInfo.Returns = append(fnInfo.Returns, typeStr)
			}
			// Anonymous return value
			if len(field.Names) == 0 {
				fnInfo.Returns = append(fnInfo.Returns, typeStr)
			}
		}
	}
	return fnInfo
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
		field := ourtypes.NewStructField()
		field.Name = fieldName
		field.Type = fieldTypeName
		structInfo.Fields = append(structInfo.Fields, field)
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

		method := ourtypes.NewStructMethod()
		method.Name = methodObj.Name()
		method.Comment = methodComment
		method.Parameters = params
		method.ReturnTypes = results
		structInfo.Methods = append(structInfo.Methods, method)
	}

	return structInfo
}

// extractDetailedInterfaceInfo extracts comprehensive details about an interface
func (p *ProjectParser) extractDetailedInterfaceInfo(obj gotypes.Object, namedType *gotypes.Named, ifaceType *gotypes.Interface, pkg *packages.Package, targetFile *ast.File) *ourtypes.InterfaceInfo {
	ifaceInfo := ourtypes.NewInterfaceInfo()
	ifaceInfo.Name = namedType.String() // Use the fully qualified name

	// Extract interface comment (requires traversing AST nodes directly within the target file)
	ifaceComment := ""
	pos := obj.Pos()
	ast.Inspect(targetFile, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Pos() == pos {
					if genDecl.Doc != nil {
						ifaceComment = strings.TrimSpace(genDecl.Doc.Text())
					} else if typeSpec.Doc != nil {
						ifaceComment = strings.TrimSpace(typeSpec.Doc.Text())
					}
					return false // Found it, stop inspecting
				}
			}
		}
		return true
	})
	ifaceInfo.Comment = ifaceComment

	// Extract methods (including those from embedded interfaces)
	numMethods := ifaceType.NumExplicitMethods()
	for i := 0; i < numMethods; i++ {
		methodObj := ifaceType.ExplicitMethod(i)
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

		ifaceInfo.Methods = append(ifaceInfo.Methods, &ourtypes.InterfaceMethod{
			Name:        methodObj.Name(),
			Comment:     methodComment,
			Parameters:  params,
			ReturnTypes: results,
		})
	}

	// Embedded interfaces
	numEmbeddeds := ifaceType.NumEmbeddeds()
	for i := 0; i < numEmbeddeds; i++ {
		emb := ifaceType.EmbeddedType(i)
		ifaceInfo.Embeddeds = append(ifaceInfo.Embeddeds, emb.String())
	}

	return ifaceInfo
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

// extractGlobalVarInfo extracts information about a global variable or constant.
func (p *ProjectParser) extractGlobalVarInfo(obj gotypes.Object, genDecl *ast.GenDecl, valSpec *ast.ValueSpec, specIndex int, pkg *packages.Package) *ourtypes.GlobalVarInfo {
	comment := ""
	if valSpec.Doc != nil {
		comment = valSpec.Doc.Text()
	} else if genDecl.Doc != nil {
		comment = genDecl.Doc.Text()
	}

	value := ""
	isConst := false

	if c, ok := obj.(*gotypes.Const); ok {
		isConst = true
		value = c.Val().String()
	} else if _, ok := obj.(*gotypes.Var); ok {
		// For vars, try to get the value from the AST if available
		if specIndex < len(valSpec.Values) {
			valueExpr := valSpec.Values[specIndex]

			var buf bytes.Buffer
			if err := printer.Fprint(&buf, pkg.Fset, valueExpr); err == nil {
				value = buf.String()
			}
		}
	}

	varInfo := ourtypes.NewGlobalVarInfo()
	varInfo.Name = obj.Name()
	varInfo.Comment = strings.TrimSpace(comment)
	varInfo.Type = obj.Type().String()
	varInfo.Value = value
	varInfo.IsConst = isConst
	return varInfo
}
