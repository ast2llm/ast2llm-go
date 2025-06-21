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

func TestProjectComposer_Compose_Function(t *testing.T) {
	// Create a temporary directory for the test project
	tmpDir, err := os.MkdirTemp("", "testproject")
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
	mypkg.MyPkgFunction(4, 8)
}
`
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGoContent), 0644)
	assert.NoError(t, err)

	mypkgDir := filepath.Join(tmpDir, "internal", "mypkg")
	err = os.MkdirAll(mypkgDir, 0755)
	assert.NoError(t, err)

	mypkgGoContent := `
package mypkg

// MyPkgFunction creates return int.
func MyPkgFunction(a, b int) (int, error) {
	return a + b, nil
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

	// Test Compose for main.go
	mainGoPath := filepath.Join(tmpDir, "main.go")
	composedOutput, err := composer.Compose(mainGoPath)
	assert.NoError(t, err)

	assert.Contains(t, composedOutput, "Used Items From Other Packages:")
	assert.Contains(t, composedOutput, "Function: example.com/testproject/internal/mypkg.MyPkgFunction")
	assert.Contains(t, composedOutput, "  Comment: MyPkgFunction creates return int.")
	assert.Contains(t, composedOutput, "  Signature: (a int, b int) -> (int, error)")
}

func TestProjectComposer_Format_Function(t *testing.T) {
	projectInfo := map[string]*types.FileInfo{
		"/project/other.go": {
			PackageName: "other",
			Imports:     []string{},
			Functions: []*types.FunctionInfo{
				{
					Name:    "testme/dto.MyFunction",
					Comment: "Help to calculate",
					Params:  []string{"a int", "b string"},
					Returns: []string{"int", "error"},
				},
			},
			Structs:             []*types.StructInfo{},
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
				{Name: "testme/dto.MyFunction"},
			},
		},
	}
	composer := composer.New(projectInfo)
	output, err := composer.Compose("/project/file.go")
	assert.NoError(t, err)
	assert.Contains(t, output, "Function: testme/dto.MyFunction")
	assert.Contains(t, output, "Comment: Help to calculate")
	assert.Contains(t, output, "Signature: (a int, b string) -> (int, error)")
}
