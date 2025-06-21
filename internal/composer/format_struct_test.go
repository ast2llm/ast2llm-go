package composer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vlad/ast2llm-go/internal/composer"
	"github.com/vlad/ast2llm-go/internal/parser"
	"github.com/vlad/ast2llm-go/internal/types"
)

// TestProjectComposer_Compose_Struct tests composing output for local and imported structs.
func TestProjectComposer_Compose_Struct(t *testing.T) {
	// Create a temporary directory for the test project
	tmpDir, err := os.MkdirTemp("", "testproject_struct")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a dummy go.mod file
	goModContent := `
module example.com/testproject

go 1.22
`
	err = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644)
	assert.NoError(t, err)

	// Create dummy files
	mainGoContent := `
package main

import (
	"example.com/testproject/internal/mypkg"
)

func main() {
	var _ mypkg.MyPkgStruct
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGoContent), 0644)
	assert.NoError(t, err)

	mypkgDir := filepath.Join(tmpDir, "internal", "mypkg")
	err = os.MkdirAll(mypkgDir, 0755)
	assert.NoError(t, err)

	mypkgGoContent := `
package mypkg

// MyPkgStruct is a test struct.
type MyPkgStruct struct {
	ID   int
	Name string
}

// LocalStruct is only used in mypkg.
type LocalStruct struct{
	Value bool
}

func (s *MyPkgStruct) MyMethod() string {
	return "hello"
}
`
	err = os.WriteFile(filepath.Join(mypkgDir, "mypkg.go"), []byte(mypkgGoContent), 0644)
	assert.NoError(t, err)

	// Parse the project
	p := parser.New()
	projectInfo, err := p.ParseProject(tmpDir)
	assert.NoError(t, err)
	assert.NotNil(t, projectInfo)

	// Create a ProjectComposer
	composer := composer.New(projectInfo)

	// Test Compose for main.go (used imported struct)
	mainGoPath := filepath.Join(tmpDir, "main.go")
	composedOutput, err := composer.Compose(mainGoPath)
	assert.NoError(t, err)

	assert.Contains(t, composedOutput, "Used Items From Other Packages:")
	assert.Contains(t, composedOutput, "Struct: example.com/testproject/internal/mypkg.MyPkgStruct")
	assert.Contains(t, composedOutput, "  Comment: MyPkgStruct is a test struct.")
	assert.Contains(t, composedOutput, "  Fields:")
	assert.Contains(t, composedOutput, "    - ID int")
	assert.Contains(t, composedOutput, "    - Name string")
	assert.Contains(t, composedOutput, "  Methods:")
	assert.Contains(t, composedOutput, "    - MyMethod() (string)")

	// Test Compose for mypkg.go (local struct)
	mypkgGoPath := filepath.Join(mypkgDir, "mypkg.go")
	composedOutputPkg, err := composer.Compose(mypkgGoPath)
	assert.NoError(t, err)

	assert.Contains(t, composedOutputPkg, "Local Structs:")
	assert.Contains(t, composedOutputPkg, "Struct: example.com/testproject/internal/mypkg.MyPkgStruct")
	assert.Contains(t, composedOutputPkg, "Struct: example.com/testproject/internal/mypkg.LocalStruct")
	assert.Contains(t, composedOutputPkg, "  Fields:")
	assert.Contains(t, composedOutputPkg, "    - Value bool")
}

// TestProjectComposer_Format_Struct tests composing output from manually created ProjectInfo.
func TestProjectComposer_Format_Struct(t *testing.T) {
	projectInfo := map[string]*types.FileInfo{
		"/project/other.go": {
			PackageName: "other",
			Imports:     []string{},
			Functions:   []*types.FunctionInfo{},
			Structs: []*types.StructInfo{
				{
					Name:    "testme/dto.MyStruct",
					Comment: "A test struct.",
					Fields: []*types.StructField{
						{Name: "FieldA", Type: "string"},
						{Name: "FieldB", Type: "int"},
					},
					Methods: []*types.StructMethod{
						{
							Name:        "GetA",
							Parameters:  []string{},
							ReturnTypes: []string{"string"},
						},
					},
				},
			},
			Interfaces:          []*types.InterfaceInfo{},
			UsedImportedStructs: []*types.StructInfo{},
		},
		"/project/file.go": {
			PackageName: "main",
			Imports:     []string{},
			Functions:   []*types.FunctionInfo{},
			Structs:     []*types.StructInfo{},
			Interfaces:  []*types.InterfaceInfo{},
			UsedImportedStructs: []*types.StructInfo{
				{Name: "testme/dto.MyStruct"},
			},
		},
	}
	composer := composer.New(projectInfo)
	output, err := composer.Compose("/project/file.go")
	assert.NoError(t, err)
	assert.Contains(t, output, "Used Items From Other Packages:")
	assert.Contains(t, output, "Struct: testme/dto.MyStruct")
	assert.Contains(t, output, "  Comment: A test struct.")
	assert.Contains(t, output, "  Fields:")
	assert.Contains(t, output, "    - FieldA string")
	assert.Contains(t, output, "    - FieldB int")
	assert.Contains(t, output, "  Methods:")
	assert.Contains(t, output, "    - GetA() (string)")
}
